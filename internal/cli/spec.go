package cli

import (
	"github.com/spf13/cobra"

	"codectl/internal/specui"
)

func init() { rootCmd.AddCommand(specCmd) }

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Open interactive Spec UI",
	RunE:  func(cmd *cobra.Command, args []string) error { return specui.Start() },
}

// no flags; spec opens UI by default
