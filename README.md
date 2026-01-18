REBUILDING PROJECT


make venv          # crea .venv
make install       # instala dependencias
make dev           # instala + pre-commit (si lo usas)

make start         # levanta el servidor
make status
make scan
make send          # ejemplo básico

make build         # compila versión carpeta
make exe           # compila versión onefile (el .exe / binario final)
make exe-debug     # útil para ver errores al ejecutar

make clean         # limpia todo
make all           # hace limpieza + install + build + exe