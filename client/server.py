# cli/commands.py
import click
import subprocess
import sys
import time
import psutil
import os
from pathlib import Path
from .modules.process import *

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