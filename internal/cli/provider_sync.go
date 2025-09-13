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
    Long:  "创建或更新 ~/.codectl/provider.json，写入当前内置/已知清单（models/mcp）。",
    RunE: func(cmd *cobra.Command, args []string) error {
        cfg, err := provider.Load()
        if err != nil {
            return err
        }
        if err := provider.Save(cfg); err != nil {
            return err
        }
        p, _ := provider.Path()
        system.Logger.Info("同步完成", "path", p, "models", len(cfg.Models), "mcp", len(cfg.MCP))
        system.Logger.Info("可手动编辑该文件以自定义清单。")
        return nil
    },
}
