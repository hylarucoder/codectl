package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	appver "codectl/internal/version"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print codectl version",
	Run: func(cmd *cobra.Command, args []string) {
		// keep output simple for scripting
		fmt.Println(appver.AppVersion)
	},
}
