import configparser
from pathlib import Path

CONFIG_FILE = "config.ini"


def load_config(ruta=CONFIG_FILE):
    config = configparser.ConfigParser()
    # Leemos el archivo
    config.read(ruta, encoding="utf-8")
    return config

def load_allowed_hosts() -> list[str]:
    config = load_config(CONFIG_FILE)
    allowed_str = config.get("security", "allowed_hosts", fallback="")
    if not allowed_str:
        return []

    # Limpiamos y separamos
    hosts = [h.strip() for h in allowed_str.split(",") if h.strip()]
    return hosts

def get_allowed_ips() -> set[str]:
    config = configparser.ConfigParser()
    if not config.read(CONFIG_FILE):
        return set()
    
    allowed_str = config.get("security", "allowed_hosts", fallback="")
    # Separamos y limpiamos (aceptamos tanto IPs como nombres, pero convertimos nombres a IPs si es posible más adelante)
    items = {item.strip() for item in allowed_str.split(",") if item.strip()}
    
    # Por ahora filtramos solo los que parecen IPs (simple)
    allowed_ips = {item for item in items if all(part.isdigit() and 0 <= int(part) <= 255 for part in item.split("."))}
    return allowed_ips