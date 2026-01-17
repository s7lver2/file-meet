# client/modules/discover.py  (o agrégalo a scan.py si prefieres)
import configparser
import requests
import asyncio
import socket
from ipaddress import ip_network
from pathlib import Path
import time
from typing import Optional, Dict

# Configuración
PORT = 42532
TIMEOUT_CONNECT = 0.8       # segundos para intentar conexión
TIMEOUT_REQUEST = 4.0       # segundos para la petición HTTP
NETWORK = "192.168.0.0/24"  # ← cámbialo según tu red actual
HOSTS_INI = Path(__file__).resolve().parents[2] / "hosts.ini"  # raíz del proyecto

# ────────────────────────────────────────────────
#  Parte 1: Escaneo asíncrono (similar al anterior)
# ────────────────────────────────────────────────

async def probe_port(ip: str, port: int = PORT, timeout: float = TIMEOUT_CONNECT) -> bool:
    try:
        _, writer = await asyncio.wait_for(
            asyncio.open_connection(ip, port),
            timeout=timeout
        )
        writer.close()
        await writer.wait_closed()
        return True
    except (asyncio.TimeoutError, ConnectionRefusedError, OSError):
        return False


async def find_open_hosts(network_str: str = NETWORK) -> list[str]:
    network = ip_network(network_str, strict=False)
    hosts = [str(ip) for ip in network.hosts()]

    print(f"Escaneando {len(hosts)} hosts en {network_str} puerto {PORT}...")

    semaphore = asyncio.Semaphore(100)  # límite de concurrencia

    async def bounded_probe(ip):
        async with semaphore:
            return ip, await probe_port(ip)

    tasks = [bounded_probe(ip) for ip in hosts]
    results = await asyncio.gather(*tasks, return_exceptions=True)

    open_ips = []
    for result in results:
        if isinstance(result, Exception):
            continue
        ip, is_open = result
        if is_open:
            print(f"  ABIERTO → {ip}:{PORT}")
            open_ips.append(ip)

    print(f"Encontradas {len(open_ips)} IPs con puerto abierto.")
    return open_ips


# ────────────────────────────────────────────────
#  Parte 2: Obtener datos vía HTTP y guardar en .ini
# ────────────────────────────────────────────────

def fetch_meet_data(ip: str) -> Optional[Dict[str, str]]:
    url = f"http://{ip}:{PORT}/meet"
    try:
        r = requests.get(url, timeout=TIMEOUT_REQUEST)
        r.raise_for_status()
        data = r.json()

        # Esperamos al menos estos campos (ajusta nombres si son diferentes)
        required = {"hostname", "passphrase", "address"}
        if not required.issubset(data.keys()):
            print(f"  Respuesta incompleta de {ip}: faltan campos")
            return None

        return {
            "hostname": data["hostname"].strip(),
            "passphrase": data["passphrase"].strip(),
            "address": data["address"].strip(),
        }
    except (requests.RequestException, ValueError, KeyError) as e:
        print(f"  No se pudo obtener datos válidos de {ip}: {e}")
        return None


def update_hosts_ini(new_entries: Dict[str, Dict[str, str]]):
    config = configparser.ConfigParser(allow_no_value=True)

    # Cargar existente (si existe)
    if HOSTS_INI.is_file():
        config.read(HOSTS_INI)

    # Preservar [Local] si existe
    if "Local" in config:
        local_data = dict(config["Local"])
    else:
        local_data = {"hostname": "local", "passphrase": "test"}

    # Limpiar secciones antiguas (excepto Local)
    for section in list(config.sections()):
        if section != "Local":
            config.remove_section(section)

    # Restaurar Local
    config["Local"] = local_data

    # Añadir/actualizar las nuevas
    for ip, data in new_entries.items():
        section_name = data["hostname"] if data["hostname"] else f"Host_{ip.replace('.', '_')}"
        config[section_name] = {
            "hostname": data["hostname"],
            "passphrase": data["passphrase"],
            "address": data["address"],
            "ip": ip,  # comentario con la IP real (útil para depuración)
        }

    # Guardar
    with open(HOSTS_INI, "w", encoding="utf-8") as f:
        config.write(f)

    print(f"\nArchivo hosts.ini actualizado ({HOSTS_INI})")
    print(f"Se encontraron y añadieron {len(new_entries)} hosts remotos.")


# ────────────────────────────────────────────────
#  Función principal para ejecutar todo
# ────────────────────────────────────────────────

async def discover_and_update():
    start_time = time.time()

    open_ips = await find_open_hosts(NETWORK)

    if not open_ips:
        print("No se encontraron hosts con puerto abierto.")
        return

    discovered = {}

    for ip in open_ips:
        print(f"  Consultando /meet en {ip}...")
        data = fetch_meet_data(ip)
        if data:
            discovered[ip] = data

    if discovered:
        update_hosts_ini(discovered)
    else:
        print("Ningún host respondió con datos válidos en /meet")

    print(f"Tiempo total: {time.time() - start_time:.2f} segundos")


# Para ejecutar directamente (útil para pruebas)
if __name__ == "__main__":
    asyncio.run(discover_and_update())