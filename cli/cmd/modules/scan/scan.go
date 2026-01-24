package scan

import (
	"fmt"

	"meet/cmd/lib/discovery"
	"github.com/spf13/viper"
)

func Scan(cfg *viper.Viper) error {
	err := discovery.DiscoverAndUpdate(cfg)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
}