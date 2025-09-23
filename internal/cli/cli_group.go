package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"codectl/internal/system"
	"codectl/internal/webui/server"
)

// cliCmd is kept for backward compatibility; it opens the Web UI.
var cliCmd = &cobra.Command{
	Use:   "cli",
	Short: "打开 CLI 工具管理（Web UI）",
	Long:  "打开 Web UI 管理受支持的开发者 CLI 工具（安装/卸载/升级/状态）。",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := "127.0.0.1:8787"
		srv := &server.Server{Addr: addr}
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()
		url := fmt.Sprintf("http://%s/_/", addr)
		system.Logger.Info("opening Web UI", "url", url)
		if err := server.OpenBrowser(url); err != nil {
			system.Logger.Warn("failed to open browser", "err", err)
		}
		if err := srv.Start(ctx); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err
		}
		return nil
	},
}

func init() { rootCmd.AddCommand(cliCmd) }
