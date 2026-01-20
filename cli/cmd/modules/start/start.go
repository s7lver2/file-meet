package start

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"path/filepath"
	"syscall"

	"meet/cmd/lib/process"
)

const PORT = 42532
const BACKEND_MODULE = "main:app"
const PROJECT_ROOT = "C:/Users/NICKE/Desktop/Projects/file-meet"

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

	serverCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,  // 0x08000000 - No crea consola visible
	}

	// python path injection for fix import bugs
	env := os.Environ()
	env = append(env, fmt.Sprintf("PYTHONPATH=%s", projectRoot))
	serverCmd.Env = env

	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr

	if err := serverCmd.Start(); err != nil {
		fmt.Printf("❌ Error al ejecutar: %v\n", err)
	} else {
		fmt.Printf("✓ Servidor iniciado → http://0.0.0.0:%v\n", PORT)
	}
}