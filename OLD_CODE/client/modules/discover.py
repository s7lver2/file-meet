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
    
    # 1. Cargar existente (si existe)
    if HOSTS_INI.is_file():
        config.read(HOSTS_INI)
        print(f"Leyendo hosts.ini existente ({len(config.sections())} secciones)")
    else:
        print("hosts.ini no existe → creando nuevo")

    # 2. NO borrar nada automáticamente
    #    Solo actualizamos o añadimos
    actualizados = 0
    nuevos = 0

    for ip, data in new_entries.items():
        hostname = data["hostname"].strip() if data["hostname"] else f"Host_{ip.replace('.', '_')}"
        section_name = hostname

        # Si ya existe la sección → actualizamos campos
        if section_name in config:
            actualizados += 1
            print(f"  Actualizando {section_name} ({ip})")
        else:
            nuevos += 1
            config.add_section(section_name)
            print(f"  Añadiendo nuevo: {section_name} ({ip})")

        config[section_name]["hostname"] = hostname
        config[section_name]["passphrase"] = data["passphrase"].strip()
        config[section_name]["address"] = data["address"].strip()
        config[section_name]["ip"] = ip  # redundante pero útil
        config[section_name]["last_seen"] = str(time.time())

    # 3. Opcional: limpiar hosts muy antiguos (ej: > 48 horas sin verse)
    to_remove = []
    current_time = time.time()
    for section in config.sections():
        if section == "Local":
            continue
        if "last_seen" in config[section]:
            last = float(config[section]["last_seen"])
            if current_time - last > 172800:  # 48 horas = 172800 segundos
                to_remove.append(section)

    for section in to_remove:
        config.remove_section(section)
        print(f"  Eliminado host antiguo: {section}")

    # 4. Guardar solo si hubo cambios
    if nuevos > 0 or actualizados > 0 or to_remove:
        with open(HOSTS_INI, "w", encoding="utf-8") as f:
            config.write(f)
        print(f"\nhosts.ini actualizado correctamente:")
        print(f"  Nuevos: {nuevos}")
        print(f"  Actualizados: {actualizados}")
        print(f"  Eliminados antiguos: {len(to_remove)}")
        print(f"  Total hosts remotos: {len(config.sections()) - (1 if 'Local' in config else 0)}")
    else:
        print("\nNo hubo cambios en hosts.ini")

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