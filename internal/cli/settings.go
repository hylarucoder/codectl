package cli

import (
	"github.com/spf13/cobra"

	"codectl/internal/settings"
)

func init() {
	rootCmd.AddCommand(settingsCmd)
}

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "打开设置界面（选择要写入 models.json 的模型）",
	RunE: func(cmd *cobra.Command, args []string) error {
		return settings.Run()
	},
}
