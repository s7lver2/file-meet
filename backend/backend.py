from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse
from pathlib import Path
from .modules.config import *
from .modules.network import *
from pathlib import Path
import os
import requests
import shutil
import json
import time

config = load_config(f"{os.path.dirname(os.path.abspath(__file__))}/../config.ini")
allowed_self_hosts = load_allowed_hosts()
pr_ip = get_ip()
ALLOWED_IPS = get_allowed_ips()
current_zip_path: Path | None = None
TEMP_DIR = Path(__file__).parent / "temp_downloads"
TEMP_DIR.mkdir(exist_ok=True)

app = FastAPI()



@app.get("/")
def read_root():
    return {"Status": "Up"}

@app.get("/meet")
def read_root():
    information = {
        "address": pr_ip,
        "hostname": config.get("meet", "hostname"),
        "passphrase": config.get("meet", "passphrase"),
        "allowed_hosts": allowed_self_hosts
    }
    return information


@app.post("/files/get")
async def receive_file(payload: dict):
    print("\n" + "="*60)
    print("¡POST /files/get recibido!")
    print("Payload recibido:", payload)
    
    download_url = payload.get("download_url")
    filename = payload.get("filename")
    passphrase_enc = payload.get("passphrase_enc")
    
    if not download_url or not filename:
        print("ERROR: Faltan parámetros")
        raise HTTPException(400, "Faltan parámetros: download_url o filename")
    
    print(f"Intentando descargar desde: {download_url}")
    print(f"Guardando como: {filename}")
    
    try:
        r = requests.get(download_url, timeout=30, stream=True)
        print(f"Status de descarga: {r.status_code}")
        r.raise_for_status()
        
        zip_path = TEMP_DIR / filename
        print(f"Guardando en: {zip_path}")
        
        with open(zip_path, "wb") as f:
            shutil.copyfileobj(r.raw, f)
        
        print(f"¡Descarga COMPLETADA! Tamaño: {zip_path.stat().st_size} bytes")
        zip_path.with_suffix(".info").write_text(str(time.time()))
        
        return JSONResponse({
            "status": "success",
            "message": f"Archivo recibido y guardado como {filename}",
            "path": str(zip_path)
        })
    except Exception as e:
        print(f"ERROR en descarga: {str(e)}")
        raise HTTPException(500, f"Error al descargar archivo: {str(e)}")

# Endpoint para que el usuario desencripte (puedes hacer un CLI o endpoint simple)
@app.get("/files/decrypt/{filename}")
async def decrypt_file(filename: str, code: str):
    zip_path = TEMP_DIR / filename
    if not zip_path.exists():
        raise HTTPException(404, "Archivo no encontrado")
    
    # Aquí iría la lógica de desencriptar (usando zipfile con password = passphrase + code)
    # Por ahora solo simulamos
    try:
        # passphrase del propio host (de config.ini)
        config = configparser.ConfigParser()
        config.read(Path(__file__).parent.parent / "config.ini")
        passphrase = config["meet"]["passphrase"]
        
        full_password = passphrase + code
        
        # Prueba de extracción (puedes mover a función separada)
        from zipfile import ZipFile
        with ZipFile(zip_path, 'r') as zf:
            zf.setpassword(full_password.encode('utf-8'))
            zf.extractall(TEMP_DIR / "extracted")
        
        # Borrar el zip después de éxito
        zip_path.unlink()
        zip_path.with_suffix(".info").unlink(missing_ok=True)
        
        return {"status": "success", "message": "Archivo desencriptado y extraído"}
    
    except RuntimeError as e:  # contraseña incorrecta
        raise HTTPException(403, "Código incorrecto o passphrase no coincide")
    except Exception as e:
        raise HTTPException(500, f"Error al desencriptar: {str(e)}")