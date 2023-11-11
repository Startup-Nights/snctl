package cmd

import (
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Handle tokens",
}

func init() {
	rootCmd.AddCommand(tokenCmd)
}
