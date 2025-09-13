package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	mdl "codectl/internal/models"
)

func init() {
	modelCmd.AddCommand(modelRemoveCmd)
}

var modelRemoveCmd = &cobra.Command{
	Use:   "remove <model>...",
	Short: "移除模型",
	Long:  "从本地模型清单中移除指定模型。",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		items := make([]string, 0, len(args))
		for _, a := range args {
			a = strings.TrimSpace(a)
			if a != "" {
				items = append(items, a)
			}
		}
		if len(items) == 0 {
			fmt.Println("未提供有效模型名")
			return nil
		}
		removed, missing, err := mdl.Remove(items)
		if err != nil {
			return err
		}
		for _, s := range removed {
			fmt.Printf("✓ 已移除：%s\n", s)
		}
		for _, s := range missing {
			fmt.Printf("• 未找到：%s\n", s)
		}
		if len(removed) == 0 && len(missing) == 0 {
			fmt.Println("无变更")
		}
		return nil
	},
}
