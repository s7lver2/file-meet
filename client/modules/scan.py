import socket
import threading
from queue import Queue
from ipaddress import ip_network


def probe(ip: str, port: int = 42532, timeout: float = 0.9):
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.settimeout(timeout)
    result = sock.connect_ex((ip, port))
    sock.close()
    return ip, result == 0


def worker(queue: Queue, results: list, port: int):
    while not queue.empty():
        ip = queue.get()
        ip, abierto = probe(ip, port)
        if abierto:
            print(f"  → ABIERTO → {ip}:{port}")
            results.append(ip)
        queue.task_done()


def scan(network: str = "192.168.1.0/24", port: int = 42532, threads: int = 80):
    net = ip_network(network, strict=False)
    ips = [str(ip) for ip in net.hosts()]
    
    print(f"Escaneando {len(ips)} hosts (puerto {port}) ...")
    
    queue = Queue()
    for ip in ips:
        queue.put(ip)
    
    results = []
    threads_list = []
    
    for _ in range(threads):
        t = threading.Thread(target=worker, args=(queue, results, port), daemon=True)
        t.start()
        threads_list.append(t)
    
    queue.join()
    
    print(f"\nEncontradas {len(results)} IPs con puerto {port} abierto:")
    for ip in sorted(results):
        print(f"  • {ip}")
    
    return results