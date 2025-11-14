package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var unsealCmd = &cobra.Command{
	Use:   "unseal [directory]",
	Short: "Decrypt a directory",
	Long:  `Unseal (decrypt) a directory and all its contents with the correct password.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Unsealing directory...")
		fmt.Printf("Directory: %s\n", args[0])
		// TODO: Implement unseal functionality
	},
}

func init() {
	RootCmd.AddCommand(unsealCmd)
}
