package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var speakerCmd = &cobra.Command{
	Use:   "speaker",
	Short: "Upload one/multiple speaker images",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("speaker called")
	},
}

func init() {
	uploadCmd.AddCommand(speakerCmd)
}
