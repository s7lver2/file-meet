# setup-file-meet.ps1 - Automatiza descarga, instalación, compilación, servicio y PATH para file-meet

# Parámetros opcionales
param (
    [string]$InstallDir = "$env:USERPROFILE\file-meet",
    [int]$Port = 42532,
    [string]$NssmUrl = "https://nssm.cc/release/nssm-2.24.zip"
)

# Función para manejar errores
function Handle-Error {
    param ([string]$Message)
    Write-Host "✗ Error: $Message" -ForegroundColor Red
    exit 1
}

# 2. Descargar repo
if (Test-Path $InstallDir) {
    Write-Host "Directorio $InstallDir ya existe. Actualizando..."
    cd $InstallDir
    git pull
} 

# 3. Crear y activar venv
if (-not (Test-Path ".venv")) {
    python -m venv .venv
}
. .\.venv\Scripts\Activate.ps1

# 4. Instalar dependencias + Nuitka
pip install -r requirements.txt
pip install nuitka

# 5. Compilar con Nuitka (onefile)
Write-Host "Compilando con Nuitka..."
python -m nuitka `
    --standalone `
    --onefile `
    --output-dir=dist `
    --include-package=backend `
    --include-package=client `
    --include-module=click `
    --include-module=fastapi `
    --include-module=uvicorn `
    --include-module=requests `
    --include-module=http.server `
    --include-module=socketserver `
    --include-module=asyncio `
    --include-module=configparser `
    --nofollow-imports `
    --follow-import-to=backend,client,modules `
    --windows-disable-console `
    main.py  # Ajusta si tu entry es otro

$ExePath = "$InstallDir\dist\file-meet.exe"
if (-not (Test-Path $ExePath)) {
    Handle-Error "Compilación falló. Revisa logs."
}

# 6. Descargar e instalar NSSM si no está
$NssmDir = "$env:USERPROFILE\Tools\nssm"
if (-not (Test-Path "$NssmDir\nssm.exe")) {
    Write-Host "Descargando NSSM..."
    mkdir -p $NssmDir
    Invoke-WebRequest -Uri $NssmUrl -OutFile "$NssmDir\nssm.zip"
    Expand-Archive "$NssmDir\nssm.zip" -DestinationPath $NssmDir
    $NssmExe = "$NssmDir\nssm-2.24\win64\nssm.exe"  # Ajusta si 32-bit
} else {
    $NssmExe = "$NssmDir\nssm.exe"
}

# 7. Instalar como servicio
Write-Host "Instalando servicio con NSSM..."
& $NssmExe install file-meet "$ExePath" start
& $NssmExe set file-meet Start SERVICE_AUTO_START
& $NssmExe set file-meet AppDirectory "$InstallDir"
& $NssmExe set file-meet Description "file-meet - Backend para compartir archivos seguros"
& $NssmExe start file-meet

# 8. Añadir al PATH del usuario
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir\dist*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir\dist", "User")
    Write-Host "Añadido a PATH. Reinicia PowerShell para que surta efecto."
}

Write-Host "✓ Proceso completado. Usa 'file-meet status' para verificar."