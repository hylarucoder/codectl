package cli

import (
    "github.com/spf13/cobra"

    "codectl/internal/specui"
)

func init() {
    specCmd.AddCommand(specUICmd)
}

var specUICmd = &cobra.Command{
    Use:   "ui",
    Short: "Open interactive Spec browser (TUI)",
    RunE: func(cmd *cobra.Command, args []string) error {
        return specui.Start()
    },
}

