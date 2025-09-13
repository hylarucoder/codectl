package cli

import "github.com/spf13/cobra"

// modelCmd is a group command to organize model management subcommands.
var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "管理模型清单",
	Long:  "添加、移除与列出可用模型（本地与远端清单）。",
}

func init() {
	rootCmd.AddCommand(modelCmd)
}
