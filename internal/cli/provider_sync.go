package cli

import (
    "fmt"

    "github.com/spf13/cobra"

    "codectl/internal/provider"
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
        fmt.Printf("同步完成：%s\n", p)
        fmt.Printf("models: %d, mcp: %d\n", len(cfg.Models), len(cfg.MCP))
        fmt.Println("可手动编辑该文件以自定义清单。")
        return nil
    },
}
