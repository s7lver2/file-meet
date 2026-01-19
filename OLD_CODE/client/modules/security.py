from pathlib import Path
import random

import random

# Lista de palabras cortas, fáciles de escribir y recordar
WORD_LIST = [
    "alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel",
    "india", "juliet", "kilo", "lima", "mike", "november", "oscar", "papa",
    "quebec", "romeo", "sierra", "tango", "uniform", "victor", "whiskey",
    "xray", "yankee", "zulu", "apple", "banana", "cherry", "dragon", "eagle",
    "falcon", "ghost", "honey", "island", "jungle", "koala", "lemon", "mango",
    "night", "ocean", "panda", "quiet", "river", "sunny", "tiger", "ultra",
    "violet", "whale", "xenon", "yellow", "zebra"
]

def get_random_word():
    # Intentar cargar diccionario del sistema
    possible_dicts = [
        Path("/usr/share/dict/words"),           # Linux/macOS
        Path("C:/Windows/System32/drivers/etc/words.txt"),  # si existe
        # Puedes añadir más paths si quieres
    ]
    
    for dict_path in possible_dicts:
        if dict_path.exists():
            try:
                with open(dict_path, encoding="utf-8", errors="ignore") as f:
                    words = [line.strip().lower() for line in f if 5 <= len(line.strip()) <= 10 and line.strip().isalpha()]
                if words:
                    return random.choice(words)
            except:
                pass
    
    # Fallback si no hay diccionario
    return random.choice(WORD_LIST)  # usa la lista de arriba