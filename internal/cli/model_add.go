package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	mdl "codectl/internal/models"
)

func init() {
	modelCmd.AddCommand(modelAddCmd)
}

var modelAddCmd = &cobra.Command{
	Use:   "add <model>...",
	Short: "添加模型",
	Long:  "将指定模型添加到本地模型清单。",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// normalize
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
		added, existed, err := mdl.Add(items)
		if err != nil {
			return err
		}
		for _, s := range added {
			fmt.Printf("✓ 已添加：%s\n", s)
		}
		for _, s := range existed {
			fmt.Printf("• 已存在：%s\n", s)
		}
		if len(added) == 0 && len(existed) == 0 {
			fmt.Println("无变更")
		}
		return nil
	},
}
