from pathlib import Path
import configparser

PROJECT_ROOT = Path(__file__).resolve().parents[2]

def load_hosts_ini():
    config = configparser.ConfigParser()
    hosts_file = PROJECT_ROOT / "hosts.ini"
    if hosts_file.exists():
        config.read(hosts_file)
    return config