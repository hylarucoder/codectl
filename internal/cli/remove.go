package cli

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/spf13/cobra"

    "codectl/internal/tools"
)

func init() {
    cliCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
    Use:   "remove [tool|all]...",
    Short: "卸载受支持的 CLI 工具",
    Long:  "为指定或全部工具执行全局卸载（npm -g uninstall）。支持：all、codex、claude、gemini。",
    Args:  cobra.ArbitraryArgs,
    RunE: func(cmd *cobra.Command, args []string) error {
        selected := selectTools(args)
        if len(selected) == 0 {
            fmt.Println("未选择任何工具")
            return nil
        }
        for i, t := range selected {
            fmt.Printf("[%d/%d] %s 卸载中…\n", i+1, len(selected), t.DisplayName)
            // pre-check
            res := tools.CheckTool(t)
            if !res.Installed {
                fmt.Println("  • 未安装，跳过")
                continue
            }
            if strings.TrimSpace(t.Package) == "" {
                fmt.Println("  • 未配置 npm 包名，无法通过 npm 卸载，跳过")
                continue
            }
            // Uninstall
            ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
            err := tools.NpmUninstallGlobal(ctx, t.Package)
            cancel()
            if err != nil {
                fmt.Printf("  × 卸载失败：%v\n", err)
                continue
            }
            // Recheck
            res2 := tools.CheckTool(t)
            if res2.Installed {
                note := "仍检测到已安装"
                if res2.Source != "" {
                    note += fmt.Sprintf("（来源：%s）", res2.Source)
                }
                fmt.Printf("  • %s\n", note)
            } else {
                fmt.Println("  ✓ 卸载成功")
            }
        }
        return nil
    },
}
