package scan

import (
	"fmt"

	"meet/cmd/lib/discovery"
)

func Scan() {
	err := discovery.DiscoverAndUpdate()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
}