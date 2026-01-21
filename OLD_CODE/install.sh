#!/usr/bin/env bash
# setup-file-meet.sh - Instala file-meet SIN Nuitka (ejecuta con python del venv)

set -euo pipefail

INSTALL_DIR="/home/$USER/file-meet"
PORT=42532

handle_error() {
    echo "✗ Error: $1" >&2
    exit 1
}

echo "Iniciando instalación de file-meet (modo sin compilar)..."

# Opcional: si usas Arch/Manjaro → patchelf no es necesario sin Nuitka
# sudo pacman -S --noconfirm patchelf 2>/dev/null || true

# 1. Clonar o actualizar
if [ -d "$INSTALL_DIR" ]; then
    echo "Actualizando repositorio..."
    cd "$INSTALL_DIR" || handle_error "No se pudo cd"
    git pull || handle_error "git pull falló"
fi

# 2. Entorno virtual
if [ ! -d "venv" ]; then
    echo "Creando venv..."
    python3 -m venv venv || handle_error "venv falló"
fi

echo "Activando venv..."
source venv/bin/activate || handle_error "activate falló"

# 3. Dependencias (sin nuitka)
echo "Instalando dependencias..."
pip install --upgrade pip setuptools wheel
pip install -r requirements.txt || echo "⚠ requirements.txt falló – sigue adelante"

echo "✓ Dependencias listas"

# 4. Rutas importantes
PYTHON_BIN="$INSTALL_DIR/usr/bin/python3"
MAIN_SCRIPT="$INSTALL_DIR/main.py"

[ -f "$PYTHON_BIN" ] || handle_error "No se encuentra python en venv"
[ -f "$MAIN_SCRIPT" ] || handle_error "No se encuentra main.py"

# 5. Servicio systemd
SERVICE_FILE="/etc/systemd/system/file-meet.service"

echo "Creando servicio systemd (necesita sudo)..."

sudo tee "$SERVICE_FILE" >/dev/null << EOF
[Unit]
Description=file-meet - Compartir archivos seguros en LAN
After=network.target

[Service]
User=$(whoami)
WorkingDirectory=$INSTALL_DIR
ExecStart=$PYTHON_BIN $MAIN_SCRIPT start --host 0.0.0.0 --port $PORT
Restart=always
RestartSec=5
Environment=PYTHONUNBUFFERED=1

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable file-meet --now

echo ""
sudo systemctl --no-pager status file-meet | head -n 12

# 6. Comando global (opcional – symlink a un script wrapper)
LOCAL_BIN="$HOME/.local/bin"
mkdir -p "$LOCAL_BIN"

cat > "$LOCAL_BIN/file-meet" << EOF
#!/usr/bin/env bash
cd "$INSTALL_DIR" || exit 1
source .venv/bin/activate
exec python main.py "\$@"
EOF

chmod +x "$LOCAL_BIN/file-meet"

if ! grep -q "$LOCAL_BIN" "$HOME/.bashrc" 2>/dev/null; then
    echo "export PATH=\"$HOME/.local/bin:\$PATH\"" >> "$HOME/.bashrc"
    echo "export PATH=\"$HOME/.local/bin:\$PATH\"" >> "$HOME/.profile" 2>/dev/null || true
    echo "→ Añadido ~/.local/bin al PATH → ejecuta 'source ~/.bashrc'"
fi

echo ""
echo "✓ Instalación completada (sin compilación)"
echo "   - Ejecutar manual:   python main.py start --port 8080"
echo "   - Comando global:    file-meet start"
echo "   - Servicio:          sudo systemctl status file-meet"
echo "   - Logs:              journalctl -u file-meet -f -n 50"
echo "   - Detener:           sudo systemctl stop file-meet"