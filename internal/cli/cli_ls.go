package cli

import (
    "fmt"
    "strings"

    "github.com/spf13/cobra"

    "codectl/internal/tools"
)

func init() {
    cliCmd.AddCommand(cliLsCmd)
}

var cliLsCmd = &cobra.Command{
    Use:   "ls",
    Short: "列出受支持工具的当前状态",
    Long:  "展示每个受支持工具的安装状态、当前版本与最新版本信息。",
    RunE: func(cmd *cobra.Command, args []string) error {
        for _, t := range tools.Tools {
            res := tools.CheckTool(t)
            var line strings.Builder
            line.WriteString(fmt.Sprintf("- %s: ", t.DisplayName))
            if !res.Installed {
                line.WriteString("未安装")
                if strings.TrimSpace(res.Err) != "" {
                    line.WriteString(fmt.Sprintf(" （%s）", res.Err))
                }
                fmt.Println(line.String())
                continue
            }
            ver := strings.TrimSpace(res.Version)
            if ver == "" {
                ver = "?"
            }
            // upgrade hint
            if res.Latest != "" && tools.VersionLess(ver, res.Latest) {
                line.WriteString(fmt.Sprintf("%s → 可升级 %s", ver, res.Latest))
            } else {
                if res.Latest != "" {
                    line.WriteString(fmt.Sprintf("%s（最新 %s）", ver, res.Latest))
                } else {
                    line.WriteString(ver)
                }
            }
            if strings.TrimSpace(res.Source) != "" {
                line.WriteString(fmt.Sprintf(" · 来源 %s", res.Source))
            }
            fmt.Println(line.String())
        }
        return nil
    },
}

