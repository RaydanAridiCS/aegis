package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch [directory]",
	Short: "Watch a directory for changes",
	Long:  `Watch a directory for file changes and automatically re-seal when changes are detected.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Watching directory...")
		fmt.Printf("Directory: %s\n", args[0])
		// TODO: Implement watch functionality
	},
}

func init() {
	RootCmd.AddCommand(watchCmd)
}
