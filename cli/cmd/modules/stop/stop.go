package stop

import (
	"fmt"
	"strconv"

	"meet/cmd/lib/process"
)

const PORT = 42532

func Stop() {
	pid, err := process.FindServerProcess(strconv.Itoa(PORT))

	if pid == 0 {
		fmt.Printf("No hay servidor en el puerto %d\n", PORT)
		return
	}

	fmt.Printf("Deteniendo servidor (PID %d)\n", pid)

	err = process.KillProcess(pid)

	if err != nil {
		fmt.Printf("✗ No se pudo detener: %d", err)
		return
	} else {
		fmt.Println("✓ Servidor detenido")
	}
}