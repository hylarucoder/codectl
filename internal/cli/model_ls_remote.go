package cli

import (
    "fmt"

    "github.com/spf13/cobra"

    mdl "codectl/internal/models"
)

func init() {
    modelCmd.AddCommand(modelLsRemoteCmd)
}

var modelLsRemoteCmd = &cobra.Command{
    Use:   "ls-remote",
    Short: "列出远端可用模型清单（占位）",
    Long:  "展示已知的远端可用模型清单；当前为占位实现（静态清单）。",
    RunE: func(cmd *cobra.Command, args []string) error {
        list := mdl.ListRemote()
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

