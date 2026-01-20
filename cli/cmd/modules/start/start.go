package start

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"meet/cmd/lib/process"
)

const PORT = 42532
const BACKEND_MODULE = "main:app"

func Start(reload bool) {
	currentDir, _ := os.Getwd()
	projectRoot := filepath.Join(currentDir, "..")

	pid, _ := process.FindServerProcess(strconv.Itoa(PORT))
	if pid != 0 {
		fmt.Printf("⚠️  El servidor ya está corriendo en el puerto %d\n", PORT)
		return
	}

	cmdArgs := []string{
		"-m", "uvicorn",
		BACKEND_MODULE,
		"--host", "0.0.0.0",
		"--port", strconv.Itoa(PORT),
	}

	if reload {
		cmdArgs = append(cmdArgs, "--reload")
		fmt.Println("(Modo Reload Activado)")
	}

	serverCmd := exec.Command("python", cmdArgs...)
	serverCmd.Dir = filepath.Join(projectRoot, "backend")

	// Llamada multiplataforma
	configureBackground(serverCmd)

	// PYTHONPATH
	env := os.Environ()
	env = append(env, fmt.Sprintf("PYTHONPATH=%s", projectRoot))
	serverCmd.Env = env

	// Redirigir salida a /dev/null en TODAS las plataformas (silencio)
	devNull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	serverCmd.Stdout = devNull
	serverCmd.Stderr = devNull

	if err := serverCmd.Start(); err != nil {
		fmt.Printf("❌ Error al ejecutar: %v\n", err)
	} else {
		fmt.Printf("✓ Servidor iniciado → http://0.0.0.0:%v\n", PORT)
	}
}