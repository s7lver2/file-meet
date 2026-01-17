# setup-file-meet.ps1 - Instala y configura file-meet SIN compilar (usa venv)
param (
    [string]$InstallDir = "$env:USERPROFILE\file-meet",
    [int]$Port = 42532,
    [string]$NssmUrl = "https://nssm.cc/release/nssm-2.24.zip"
)

function Handle-Error {
    param ([string]$Message)
    Write-Host "✗ Error: $Message" -ForegroundColor Red
    exit 1
}

Write-Host "Iniciando instalación de file-meet (sin compilación)..."

# 1. Clonar o actualizar repositorio
if (Test-Path $InstallDir) {
    Write-Host "Directorio ya existe → actualizando..."
    Set-Location $InstallDir
    git pull
}

# 2. Crear y activar venv
if (-not (Test-Path ".venv")) {
    Write-Host "Creando entorno virtual..."
    python -m venv .venv
}

# Activamos para instalar dependencias (pero el servicio usará el python del venv directamente)
& ".\.venv\Scripts\Activate.ps1"

# 4. Ruta al python del venv y al main.py
$PythonExe = Join-Path $InstallDir ".venv\Scripts\python.exe"
$MainScript = Join-Path $InstallDir "main.py"

# 3. Instalar dependencias (sin nuitka)

Write-Host "Instalando dependencias..."
python -m pip install --upgrade pip setuptools wheel
pip install -r requirements.txt -q  # -q para menos ruido, quita si quieres ver todo

Write-Host "✓ Dependencias instaladas"

if (-not (Test-Path $PythonExe)) { Handle-Error "No se encuentra python en el venv" }
if (-not (Test-Path $MainScript)) { Handle-Error "No se encuentra main.py" }

# 5. Descargar NSSM si no existe
$NssmDir = "$env:USERPROFILE\Tools\nssm"
$NssmZip = "$NssmDir\nssm.zip"
$NssmExe = "$NssmDir\nssm-2.24\win64\nssm.exe"   # ruta típica en zip

if (-not (Test-Path $NssmExe)) {
    Write-Host "Descargando NSSM..."
    New-Item -ItemType Directory -Force -Path $NssmDir | Out-Null
    Invoke-WebRequest -Uri $NssmUrl -OutFile $NssmZip
    Expand-Archive $NssmZip -DestinationPath $NssmDir -Force
    Remove-Item $NssmZip -Force
}

# 6. Instalar/actualizar servicio con NSSM
Write-Host "Configurando servicio con NSSM..."

& $NssmExe remove file-meet confirm   # Borra si ya existe (para actualizar)
& $NssmExe install file-meet $PythonExe
& $NssmExe set file-meet AppParameters "main.py start --host 0.0.0.0 --port $Port"
& $NssmExe set file-meet AppDirectory $InstallDir
& $NssmExe set file-meet AppStdout "$InstallDir\service.log"
& $NssmExe set file-meet AppStderr "$InstallDir\service-error.log"
& $NssmExe set file-meet Description "file-meet - Compartir archivos en LAN"
& $NssmExe set file-meet Start SERVICE_AUTO_START

# Iniciar servicio
& $NssmExe start file-meet

Write-Host "Servicio instalado e iniciado. Revisa logs en:"
Write-Host "  $InstallDir\service.log"
Write-Host "  $InstallDir\service-error.log"

# 7. Añadir al PATH (la carpeta del proyecto, para poder ejecutar manualmente si quieres)
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "Añadido $InstallDir al PATH de usuario (reinicia PowerShell)"
}

Write-Host ""
Write-Host "✓ Instalación completada"
Write-Host "   - Ejecutar manual:   .\.venv\Scripts\Activate.ps1 ; python main.py start"
Write-Host "   - Servicio status:   nssm status file-meet   (o services.msc)"
Write-Host "   - Logs:              $InstallDir\service*.log"
Write-Host "   - Detener:           nssm stop file-meet"