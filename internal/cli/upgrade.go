package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	appver "codectl/internal/version"
)

func init() {
	rootCmd.AddCommand(upgradeCmd)
}

var upgradeCmd = &cobra.Command{
	Use:   "update",
	Short: "更新 codectl 自身（自更新）",
	Long:  "TODO：后续将连接 GitHub Releases，自动下载并替换二进制以完成自更新。",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("codectl 自更新（TODO）")
		fmt.Printf("  当前版本: v%s\n", appver.AppVersion)
		fmt.Printf("  运行平台: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Println()
		fmt.Println("规划：")
		fmt.Println("  - 查询 GitHub Releases 最新版本并对比当前版本")
		fmt.Println("  - 按平台下载二进制（含校验和验证）")
		fmt.Println("  - 备份并替换当前可执行文件，支持回滚")
		fmt.Println()
		fmt.Println("暂未实现：请先手动更新（从源码构建或下载新版本）。")
		return nil
	},
}
