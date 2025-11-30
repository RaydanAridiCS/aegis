package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "aegis",
	Short: "Aegis - A secure file encryption tool",
	Long:  `Aegis is a CLI tool for encrypting and decrypting directories with secure key derivation.`,
	// Example block provides immediate user guidance, a crucial CLI usability feature.
	Example: `  # Encrypt a folder named 'secrets' in the current directory
  aegis seal ./secrets

  # Decrypt the sealed folder (all files with .aegis extension)
  aegis unseal ./secrets

  # View help for a specific command
  aegis seal --help`,

	Run: func(cmd *cobra.Command, args []string) {
		// Show help when no subcommand is provided
		cmd.Help()
	},
}

func Execute() error {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
