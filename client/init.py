from .modules.scan import scan
import os
import time

def cinit():
    os.chdir(os.path.dirname(os.path.abspath(__file__)))
    time.sleep(3)
    print(scan("192.168.1.0/24", 42532))