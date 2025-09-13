package cli

import (
    "fmt"
    "strings"

    "github.com/spf13/cobra"

    store "codectl/internal/mcp"
)

func init() {
    mcpCmd.AddCommand(mcpAddCmd)
}

var mcpAddCmd = &cobra.Command{
    Use:   "add <server>...",
    Short: "添加 MCP 服务端",
    Long:  "将 MCP 服务端标识加入到本地配置清单。TODO：后续扩展端点/认证等详细配置。",
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
        added, existed, err := store.Add(items)
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
        fmt.Println("TODO：后续支持配置端点、凭据与健康检查。")
        return nil
    },
}

