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
    """Envía un archivo comprimido con contraseña al destino especificado"""
    config = load_hosts_ini()
    
    if destino not in config:
        click.echo(f"✗ Destino '{destino}' no encontrado en hosts.ini")
        sys.exit(1)
    
    passphrase = config[destino].get('passphrase')
    if not passphrase:
        click.echo(f"✗ No se encontró passphrase para '{destino}'")
        sys.exit(1)
    
    # Generar código aleatorio de 6 dígitos
    code = f"{random.randint(0, 999999):06d}"
    password = passphrase + code
    
    click.echo(f"Código de verificación (solo para ti): {code}")
    click.echo("Guárdalo seguro - se usará para descomprimir en el destino.")
    
    # Comprimir el archivo
    archivo_path = Path(archivo)
    zip_name = f"{archivo_path.stem}_enviado.zip"
    zip_path = PROJECT_ROOT / zip_name
    
    try:
        with ZipFile(zip_path, 'w', ZIP_DEFLATED) as zf:
            zf.setpassword(password.encode('utf-8'))
            zf.write(archivo, arcname=archivo_path.name)
        click.echo(f"✓ Archivo comprimido como {zip_name}")
    except Exception as e:
        click.echo(f"✗ Error al comprimir: {e}")
        sys.exit(1)
    
    # Levantar servidor básico HTTP en puerto disponible (ej. 8080)
    serve_port = 8080
    local_ip = get_local_ip()
    serve_dir = str(PROJECT_ROOT)
    
    class Handler(http.server.SimpleHTTPRequestHandler):
        def __init__(self, *args, **kwargs):
            super().__init__(*args, directory=serve_dir, **kwargs)
    
    try:
        with socketserver.TCPServer(("", serve_port), Handler) as httpd:
            server_thread = threading.Thread(target=httpd.serve_forever, daemon=True)
            server_thread.start()
            click.echo(f"✓ Servidor básico iniciado en http://{local_ip}:{serve_port}/{zip_name}")
            click.echo("Presiona Ctrl+C para detener el servidor cuando termines.")
            
            # Mantener vivo hasta Ctrl+C (para el resto de la lógica más adelante)
            while True:
                time.sleep(1)
    except KeyboardInterrupt:
        click.echo("\nServidor detenido por usuario.")
    except Exception as e:
        click.echo(f"✗ Error al iniciar servidor: {e}")
    finally:
        # Limpieza: borrar zip temporal si quieres
        zip_path.unlink()
        pass