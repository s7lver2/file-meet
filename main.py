from backend.init import binit
from client.init import cinit
import subprocess

if __name__ == "__main__":
    subprocess.Popen(["python", "backend/init.py"], shell=False)
    cinit()
