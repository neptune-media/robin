package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	VERSION = "0.6.1"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the program version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("robin version %s\n", VERSION)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
