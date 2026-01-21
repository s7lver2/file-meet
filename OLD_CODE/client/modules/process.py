import psutil

def find_server_process(port):
    """Busca un proceso uvicorn/fastapi escuchando en el puerto dado"""
    for proc in psutil.process_iter(['pid', 'name', 'cmdline']):
        try:
            cmdline = ' '.join(proc.info['cmdline'] or [])
            if any(server in cmdline.lower() for server in ['uvicorn', 'fastapi']) and str(port) in cmdline:
                return proc.info['pid']
        except (psutil.NoSuchProcess, psutil.AccessDenied):
            pass
    return None

def kill_process(pid: int):
    """Termina un proceso y sus hijos de forma limpia"""
    try:
        proc = psutil.Process(pid)
        for child in proc.children(recursive=True):
            child.terminate()
        proc.terminate()
        proc.wait(timeout=5)
        return True
    except psutil.NoSuchProcess:
        return True
    except psutil.TimeoutExpired:
        proc.kill()
        proc.wait()
        return True
    except Exception:
        return False