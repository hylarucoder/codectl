package cli

import (
    "fmt"

    "github.com/spf13/cobra"

    store "codectl/internal/mcp"
)

func init() {
    mcpCmd.AddCommand(mcpLsRemoteCmd)
}

var mcpLsRemoteCmd = &cobra.Command{
    Use:   "ls-remote",
    Short: "列出远端可用的 MCP 服务端（占位）",
    Long:  "展示已知的远端可用 MCP 服务端清单；当前为占位实现（静态清单）。",
    RunE: func(cmd *cobra.Command, args []string) error {
        list := store.ListRemote()
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

