package process

import (
	_ "fmt"
	"strings"
	"time"
	"github.com/shirou/gopsutil/v3/process"
)

func FindServerProcess(port string) (int32, error) {
	processes, err := process.Processes()

	if err != nil {
			return 0, err
	}

	for _,p := range processes {
		cmdlineSlice, err := p.CmdlineSlice()

		if err != nil {
			continue
		}

		cmdline := strings.ToLower(strings.Join(cmdlineSlice, " "))

		hasServer := strings.Contains(cmdline, "uvicorn") || strings.Contains(cmdline, "fastapi")
		hasPort := strings.Contains(cmdline, port)

		if hasServer && hasPort {
				return p.Pid, nil
		}
	}

	return 0, nil	
}

func KillProcess(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil
	}

	// 1. Obtener hijos recursivamente
	children, err := p.Children()
	if err == nil {
		for _, child := range children {
			child.Terminate() 
		}
	}

	// 2. Intentar terminar el proceso padre de forma limpia
	err = p.Terminate()
	if err != nil {
		return err
	}

	// 3. Esperar a que muera (timeout de 5 segundos)
	gone := make(chan bool, 1)
	go func() {
		for {
			isRunning, _ := p.IsRunning()
			if !isRunning {
				gone <- true
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	select {
	case <-gone:
		return nil
	case <-time.After(5 * time.Second):
		// 4. Si expira el tiempo, forzamos el Kill (como tu TimeoutExpired)
		return p.Kill()
	}
}