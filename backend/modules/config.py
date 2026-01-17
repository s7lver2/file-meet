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