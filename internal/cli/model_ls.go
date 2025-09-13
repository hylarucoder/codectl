package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	mdl "codectl/internal/models"
)

func init() {
	modelCmd.AddCommand(modelLsCmd)
}

var modelLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "列出本地模型清单",
	Long:  "打印当前本地模型清单。",
	RunE: func(cmd *cobra.Command, args []string) error {
		list, err := mdl.Load()
		if err != nil {
			return err
		}
		if len(list) == 0 {
			fmt.Println("(空)")
			return nil
		}
		for _, s := range list {
			fmt.Println(s)
		}
		return nil
	},
}
