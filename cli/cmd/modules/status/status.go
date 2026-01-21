package status

import (
	"fmt"
	"strconv"

	"meet/cmd/lib/process"
)

const PORT = 42532

func Status() {
	pid, err := process.FindServerProcess(strconv.Itoa(PORT))

	if err != nil {
		fmt.Println(err)
		return
	}

	if pid != 0 {
		fmt.Printf("🟢 ACTIVO (PID %d) → http://0.0.0.0:%d\n", pid, PORT)
	}
}