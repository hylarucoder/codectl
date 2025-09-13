package cli

import (
	"github.com/spf13/cobra"

	"codectl/internal/provider"
	"codectl/internal/system"
)

func init() {
	providerCmd.AddCommand(providerSyncCmd)
}

var providerSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "手动同步 provider.json",
	Long:  "创建或更新 ~/.codectl/provider.json（v2 结构：providers 映射）。若存在则规范化/重写保持 v2 形态。",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := provider.LoadV2()
		if err != nil {
			return err
		}
		if err := provider.SaveV2(cfg); err != nil {
			return err
		}
		p, _ := provider.Path()
		// flatten model count for logging
		totalModels := len(provider.Models())
		system.Logger.Info("同步完成 (v2)", "path", p, "models", totalModels)
		system.Logger.Info("可手动编辑该文件以自定义 providers 与 models。")
		return nil
	},
}
