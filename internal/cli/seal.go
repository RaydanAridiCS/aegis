package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var sealCmd = &cobra.Command{
	Use:   "seal [directory]",
	Short: "Encrypt a directory",
	Long:  `Seal (encrypt) a directory and all its contents with a password-derived key.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Sealing directory...")
		fmt.Printf("Directory: %s\n", args[0])
		// TODO: Implement seal functionality
	},
}

func init() {
	RootCmd.AddCommand(sealCmd)
}
