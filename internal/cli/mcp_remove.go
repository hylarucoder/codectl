package cli

import (
    "fmt"
    "strings"

    "github.com/spf13/cobra"

    store "codectl/internal/mcp"
)

func init() {
    mcpCmd.AddCommand(mcpRemoveCmd)
}

var mcpRemoveCmd = &cobra.Command{
    Use:   "remove <server>...",
    Short: "移除 MCP 服务端",
    Long:  "从本地配置清单中移除 MCP 服务端标识。TODO：后续支持清理相关缓存/凭据。",
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
            fmt.Println("未提供有效服务端标识")
            return nil
        }
        removed, missing, err := store.Remove(items)
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
        fmt.Println("TODO：后续考虑移除关联配置/缓存并做健康检查刷新。")
        return nil
    },
}

