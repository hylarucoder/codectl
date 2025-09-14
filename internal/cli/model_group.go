package cli

import "github.com/spf13/cobra"

// modelCmd is a group command to organize model management subcommands.
var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "管理模型清单",
	Long:  "模型相关命令入口（占位，当前无子命令）。",
}

func init() {
	rootCmd.AddCommand(modelCmd)
}
