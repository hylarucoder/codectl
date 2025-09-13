package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show config location",
	Long:  "Show where codectl stores configuration and related files.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgDir, err := os.UserConfigDir()
		if err != nil || cfgDir == "" {
			// fall back to home dir if needed
			if home, herr := os.UserHomeDir(); herr == nil {
				cfgDir = home
			}
		}
		fmt.Println(cfgDir)
		return nil
	},
}
