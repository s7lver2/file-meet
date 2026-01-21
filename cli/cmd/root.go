package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"meet/cmd/modules/start"
	"meet/cmd/modules/stop"
	"meet/cmd/modules/status"
	"meet/cmd/modules/scan"
	"meet/cmd/modules/send"
	"meet/cmd/modules/decrypt"
)

var reload bool

var rootCmd = &cobra.Command{
		Use: "meet",
		Short: "meet cli client for interact with the backend",
		Long: "meet is a cli client than allow you to share files between local network reacheable clients",
}
	
var infoCmd = &cobra.Command{
		Use: "info",
		Short: "Just a way to test if the program is installed (print debug info too)",
		Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("hewooo :3")
		},
}

var startCmd = &cobra.Command{
		Use: "start",
		Short: "start backend server process manually",
		Run: func(cmd *cobra.Command, args []string) {
				start.Start(reload)
		},
}

var stopCmd = &cobra.Command{
		Use: "stop",
		Short: "stop backend server process manually",
		Run: func(cmd *cobra.Command, args []string) {
				stop.Stop()
		},
}

var statusCmd = &cobra.Command{
		Use: "status",
		Short: "shows the current status of the backend",
		Run: func(cmd *cobra.Command, args []string) {
				status.Status()
		},
}

var scanCmd = &cobra.Command{
		Use: "scan",
		Short: "do a network scan to detect other users in network using package",
		Run: func(cmd *cobra.Command, args []string) {
				scan.Scan()
		},
}

var sendCmd = &cobra.Command{
		Use: "send [filename] [target]",
		Short: "send a file to the selected target",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
				file := args[0]
				target := args[1]
				err := send.Send(file, target)

				if err != nil {
					fmt.Println(err)
				}
		},
}

var decryptCmd = &cobra.Command{
	Use:   "decrypt [filename] [code]",
	Short: "Decrypt a file using the 6 digit code",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]
		code := args[1]
		return decrypt.Decrypt(filename, code)
	},
}

func init() {
	/* Flag Config */

	startCmd.Flags().BoolVarP(&reload, "reload", "r", false, "reload server on changes")

	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(decryptCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
			fmt.Println(err)
			os.Exit(1)
	}
}