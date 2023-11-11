package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var printCmd = &cobra.Command{
	Use:   "print",
	Short: "Print the currently configured tokens",
	Run: func(cmd *cobra.Command, args []string) {
		if viper.IsSet("credentials") {
			fmt.Println("=> credentials:")
			fmt.Println(viper.GetString("credentials"))
		} else {
			fmt.Println("no credentials configured")
		}

		if viper.IsSet("gmail_token") {
			fmt.Println("=> gmail token:")
			fmt.Println(viper.GetString("gmail_token"))
		} else {
			fmt.Println("no gmail token configured")
		}

		if viper.IsSet("sheets_token") {
			fmt.Println("=> sheets token:")
			fmt.Println(viper.GetString("sheets_token"))
		} else {
			fmt.Println("no sheets token configured")
		}

		fmt.Println("=> token can be updated here: " + viper.GetString("secrets_url"))
	},
}

func init() {
	tokenCmd.AddCommand(printCmd)
}
