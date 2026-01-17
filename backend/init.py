import subprocess
import os

def binit():
    os.chdir(os.path.dirname(os.path.abspath(__file__)))
    subprocess.run(["fastapi", "dev", "backend.py", "--host", "127.0.0.1", "--port", "42532"], check=True)

if __name__ == "__main__":
    binit()