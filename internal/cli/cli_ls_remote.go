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
    cliCmd.AddCommand(cliLsRemoteCmd)
}

var cliLsRemoteCmd = &cobra.Command{
    Use:   "ls-remote",
    Short: "列出受支持工具的远端最新版本",
    Long:  "从 npm registry 查询各受支持工具的最新版本（若配置了 npm 包名）。",
    RunE: func(cmd *cobra.Command, args []string) error {
        for _, t := range tools.Tools {
            var latest, note string
            if strings.TrimSpace(t.Package) == "" {
                note = "（未配置 npm 包名）"
            } else {
                ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
                v, err := tools.NpmLatestVersion(ctx, t.Package)
                cancel()
                if err != nil {
                    note = fmt.Sprintf("（获取失败：%v）", err)
                } else {
                    latest = v
                }
            }
            if latest != "" {
                fmt.Printf("- %s: %s%s\n", t.DisplayName, latest, note)
            } else {
                fmt.Printf("- %s%s\n", t.DisplayName, note)
            }
        }
        return nil
    },
}

