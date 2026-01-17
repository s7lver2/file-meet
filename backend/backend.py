from fastapi import FastAPI
from .modules.config import *
from .modules.network import get_ip
from pathlib import Path
import os

config = load_config(f"{os.path.dirname(os.path.abspath(__file__))}/../config.ini")
allowed_hosts = load_allowed_hosts()
app = FastAPI()

pr_ip = get_ip()

@app.get("/")
def read_root():
    return {"Status": "Up"}

@app.get("/meet")
def read_root():
    information = {
        "address": pr_ip,
        "hostname": config.get("meet", "hostname"),
        "passphrase": config.get("meet", "passphrase"),
        "allowed_hosts": allowed_hosts
    }
    return information