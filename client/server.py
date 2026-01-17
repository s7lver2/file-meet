# cli/commands.py
from .modules.process import *
from .modules.network import *
from .modules.config import *
from .modules.discover import discover_and_update
from zipfile import ZipFile, ZIP_DEFLATED
from pathlib import Path
import click
import subprocess
import sys
import time
import psutil
import requests
import os
import asyncio
import random
import configparser
import socket
import http.server
import socketserver
import threading


PROJECT_ROOT = Path(__file__).resolve().parents[1]  # sube dos niveles hasta la raíz
BACKEND_MODULE = "backend.backend:app"
PORT = 42532

@click.group(invoke_without_command=True)
@click.pass_context
def cli(ctx):
    """CLI para controlar file-meet"""
    if ctx.invoked_subcommand is None:
        click.echo(ctx.get_help())


@cli.command()
@click.option('--reload', is_flag=True, help="Activar modo desarrollo con recarga automática")
def start(reload):
    """Inicia el servidor FastAPI en segundo plano"""
    if find_server_process(PORT):
        click.echo(f"⚠️  El servidor ya está corriendo en el puerto {PORT}")
        return

    cmd = [
        sys.executable, "-m", "uvicorn",
        BACKEND_MODULE,
        "--host", "0.0.0.0",
        "--port", str(PORT),
    ]
    if reload:
        cmd.append("--reload")

    try:
        subprocess.Popen(cmd, cwd=str(PROJECT_ROOT))
        click.echo(f"✓ Servidor iniciado → http://0.0.0.0:{PORT}")
        if reload:
            click.echo("   (modo reload activado)")
    except Exception as e:
        click.echo(f"✗ Error al iniciar: {e}", err=True)


@cli.command()
def stop():
    """Detiene el servidor FastAPI"""
    pid = find_server_process(PORT)
    if not pid:
        click.echo(f"No hay servidor en el puerto {PORT}")
        return

    click.echo(f"Deteniendo servidor (PID {pid})...")
    if kill_process(pid):
        click.echo("✓ Servidor detenido")
    else:
        click.echo("✗ No se pudo detener", err=True)


@cli.command()
def status():
    """Estado del servidor"""
    pid = find_server_process(PORT)
    if pid:
        click.echo(f"🟢 ACTIVO (PID {pid}) → http://0.0.0.0:{PORT}")
    else:
        click.echo(f"🔴 No está corriendo")


@cli.command()
@click.option('--reload', is_flag=True, help="Modo desarrollo con recarga")
def run(reload):
    """Inicia servidor + cliente"""
    pid = find_server_process(PORT)
    server_process = None

    if not pid:
        click.echo("Iniciando servidor...")
        cmd = [
            sys.executable, "-m", "uvicorn",
            BACKEND_MODULE,
            "--host", "0.0.0.0",
            "--port", str(PORT),
        ]
        if reload:
            cmd.append("--reload")

        server_process = subprocess.Popen(cmd, cwd=str(PROJECT_ROOT))
        time.sleep(3)
        click.echo(f"✓ Servidor listo → http://0.0.0.0:{PORT}")

    try:
        from client.init import cinit
        click.echo("\nIniciando cliente...")
        cinit()
    except KeyboardInterrupt:
        click.echo("\nInterrupción detectada")
    except Exception as e:
        click.echo(f"Error en cliente: {e}", err=True)
    finally:
        if server_process:
            new_pid = find_server_process()
            if new_pid:
                click.echo("Apagando servidor...")
                kill_process(new_pid)
        click.echo("Fin.")

@cli.command()
def scan():
    """Escanea la red, consulta /meet y actualiza hosts.ini"""
    asyncio.run(discover_and_update())

@cli.command()
@click.argument('archivo', type=click.Path(exists=True, file_okay=True, dir_okay=False))
@click.argument('destino', type=str)
def send(archivo, destino):
    """Envía un archivo al destino con compresión protegida"""
    config = load_hosts_ini()
    
    if destino not in config:
        click.echo(f"✗ Destino '{destino}' no encontrado en hosts.ini")
        return
    
    passphrase = config[destino].get('passphrase', '').strip()
    if not passphrase:
        click.echo(f"✗ No se encontró passphrase para '{destino}'")
        return
    
    # IP o hostname del destino (de preferencia IP para conexión directa)
    target_host = config[destino].get('address') or config[destino].get('hostname')
    if not target_host:
        click.echo("✗ No se encontró 'address' o 'hostname' en la sección del destino")
        return
    
    # Generar código
    code = f"{random.randint(0, 999999):06d}"
    password = passphrase + code
    click.echo("\n" + "="*60)
    click.echo(f"   ¡CÓDIGO SECRETO (6 dígitos) PARA EL RECEPTOR: {code}")
    click.echo("="*60 + "\n")
    click.echo("Guárdalo y compártelo SOLO con el receptor por otro canal seguro.")
    
    # Comprimir
    archivo_path = Path(archivo)
    zip_name = f"{archivo_path.stem}_protected.zip"
    zip_path = PROJECT_ROOT / zip_name
    
    with ZipFile(zip_path, 'w', compression=ZIP_DEFLATED) as zf:
        zf.setpassword(password.encode('utf-8'))
        zf.write(archivo_path, arcname=archivo_path.name)
    
    click.echo(f"✓ Archivo comprimido: {zip_path}")
    
    # Levantar servidor temporal (8080) para que el receptor lo descargue
    local_ip = get_local_ip()
    serve_port = 8080
    download_url = f"http://{local_ip}:{serve_port}/{zip_name}"
    
    class FileHandler(http.server.SimpleHTTPRequestHandler):
        def __init__(self, *args, **kwargs):
            super().__init__(*args, directory=str(PROJECT_ROOT), **kwargs)
    
    server_ready = threading.Event()
    
    def start_server():
        with socketserver.TCPServer(("", serve_port), FileHandler) as httpd:
            server_ready.set()
            httpd.serve_forever()
    
    server_thread = threading.Thread(target=start_server, daemon=True)
    server_thread.start()
    server_ready.wait(timeout=2)  # Esperamos que arranque
    
    click.echo(f"Servidor temporal activo en {download_url}")
    
    # Enviar petición al destino
    target_url = f"http://{target_host}:8000/files/get"
    payload = {
        "download_url": download_url,
        "filename": zip_name,
        "code_hint": "El código de 6 dígitos fue compartido por el emisor"
    }
    
    try:
        r = requests.post(target_url, json=payload, timeout=10)
        r.raise_for_status()
        click.echo(f"✓ Petición enviada al destino → {target_host} la recibió correctamente")
        click.echo("El receptor descargará el archivo automáticamente.")
    except Exception as e:
        click.echo(f"✗ Error al notificar al destino: {e}")
        click.echo("   El receptor no recibirá la notificación automática.")
    
    # Mantener el servidor vivo un tiempo razonable (ej: 10 minutos) o hasta Ctrl+C
    click.echo("\nServidor activo. Esperando descarga... (Ctrl+C para detener)")
    try:
        time.sleep(600)  # 10 minutos
    except KeyboardInterrupt:
        pass
    finally:
        click.echo("Deteniendo servidor temporal y limpiando...")
        # Borramos el zip del emisor
        try:
            zip_path.unlink()
            click.echo("Archivo zip local eliminado.")
        except:
            pass