import configparser
from pathlib import Path

CONFIG_FILE = "config.ini"


def load_config(ruta=CONFIG_FILE):
    config = configparser.ConfigParser()
    # Leemos el archivo
    config.read(ruta, encoding="utf-8")
    return config


def main():
    config = cargar_configuracion()
    
    # Leer valores (con conversión de tipos útil)
    debug       = config.getboolean("General", "debug")       # True/False
    log_level   = config["General"]["log_level"]
    db_port     = config.getint("Database", "port")           # int
    db_password = config.get("Database", "password")
    
    print(f"Aplicación : {app_name}")
    print(f"Debug      : {debug}")
    print(f"Log level  : {log_level}")
    print(f"DB puerto  : {db_port}")
    print(f"DB pass    : {'*' * len(db_password)}")
    
    # Modificar y guardar
    config["General"]["debug"] = "true"
    config["Database"]["host"] = "db.miempresa.com"
    
    with open(CONFIG_FILE, "w", encoding="utf-8") as f:
        config.write(f)
    
    print("\nConfiguración modificada y guardada.")
