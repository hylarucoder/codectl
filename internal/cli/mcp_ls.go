package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	store "codectl/internal/mcp"
)

func init() {
	mcpCmd.AddCommand(mcpLsCmd)
}

var mcpLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "列出已配置的 MCP 服务端",
	Long:  "打印当前本地配置的 MCP 服务端清单。",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := store.Load()
		if err != nil {
			return err
		}
		if len(c) == 0 {
			fmt.Println("(空)")
			return nil
		}
		// print names and commands
		for name, srv := range c {
			if len(srv.Args) > 0 {
				fmt.Printf("%s: %s %v\n", name, srv.Command, srv.Args)
			} else {
				fmt.Printf("%s: %s\n", name, srv.Command)
			}
		}
		return nil
	},
}
