import socket

def get_ip():
    try:
        # Conectamos a un endpoint externo (no envía datos reales)
        # Esto fuerza al sistema operativo a elegir la interfaz de red principal
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.settimeout(0)
        s.connect(("8.8.8.8", 80))  # Google DNS, da igual cuál uses
        local_ip = s.getsockname()[0]
    except Exception:
        local_ip = "127.0.0.1"  # fallback
    finally:
        s.close()
    return local_ip