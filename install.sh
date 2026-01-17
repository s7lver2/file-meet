#!/bin/bash

# setup-file-meet.sh - Automatiza descarga, instalación, compilación, servicio y PATH para file-meet

# Parámetros opcionales
INSTALL_DIR="${2:-$HOME/file-meet}"
PORT=42532
NSSM_URL=""  # No aplica en Linux, usamos systemd

# Función para errores
handle_error() {
    echo "✗ Error: $1" >&2
    exit 1
}

# 2. Descargar/clonar repo
if [ -d "$INSTALL_DIR" ]; then
    echo "Directorio $INSTALL_DIR existe. Actualizando..."
    cd "$INSTALL_DIR"
    git pull
else

# 3. Crear y activar venv
if [ ! -d ".venv" ]; then
    python3 -m venv .venv
fi
source .venv/bin/activate

# 4. Instalar dependencias + Nuitka
pip install -r requirements.txt
pip install nuitka

# 5. Compilar con Nuitka (onefile)
echo "Compilando con Nuitka..."
python -m nuitka \
    --standalone \
    --onefile \
    --output-dir=dist \
    --include-package=backend \
    --include-package=client \
    --include-module=click \
    --include-module=fastapi \
    --include-module=uvicorn \
    --include-module=requests \
    --include-module=http.server \
    --include-module=socketserver \
    --include-module=asyncio \
    --include-module=configparser \
    --nofollow-imports \
    --follow-import-to=backend,client,modules \
    main.py  # Ajusta si tu entry es otro

EXE_PATH="$INSTALL_DIR/dist/file-meet"
if [ ! -f "$EXE_PATH" ]; then
    handle_error "Compilación falló. Revisa logs."
fi
chmod +x "$EXE_PATH"

# 6. Instalar como servicio systemd
SERVICE_FILE="/etc/systemd/system/file-meet.service"
echo "Creando servicio systemd..."
cat <<EOF | sudo tee $SERVICE_FILE
[Unit]
Description=file-meet - Backend para compartir archivos seguros
After=network.target

[Service]
User=$(whoami)
WorkingDirectory=$INSTALL_DIR
ExecStart=$EXE_PATH start
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable file-meet
sudo systemctl start file-meet

# 7. Añadir al PATH del usuario
LOCAL_BIN="$HOME/.local/bin"
mkdir -p "$LOCAL_BIN"
cp "$EXE_PATH" "$LOCAL_BIN/file-meet"
chmod +x "$LOCAL_BIN/file-meet"

if ! grep -q "$LOCAL_BIN" ~/.bashrc; then
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
    source ~/.bashrc
fi

echo "✓ Proceso completado. Usa 'file-meet status' para verificar."