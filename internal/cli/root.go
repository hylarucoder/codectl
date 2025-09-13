package cli

import (
    "os"

    "github.com/spf13/cobra"

    "codectl/internal/app"
    "codectl/internal/system"
)

var rootCmd = &cobra.Command{
	Use:   "codectl",
	Short: "codectl – AI dev tooling helper",
	Long:  "codectl provides a TUI and subcommands to check and manage AI dev CLIs.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default action: launch the TUI
		return app.Start()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the CLI.
func Execute() {
    if err := rootCmd.Execute(); err != nil {
        system.Logger.Error("command execution failed", "err", err)
        os.Exit(1)
    }
}
