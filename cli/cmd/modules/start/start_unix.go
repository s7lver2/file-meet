//go:build !windows

package start

import "os/exec"

func configureBackground(cmd *exec.Cmd) {
	// En Linux/macOS no necesitamos nada especial
	// El proceso ya se lanza sin ventana asociada cuando se usa Start()
	// y salida redirigida a /dev/null
}