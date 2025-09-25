package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"codectl/internal/system"
	"codectl/internal/webui/server"
)

var rootCmd = &cobra.Command{
	Use:   "codectl",
	Short: "codectl – local Web UI for AI dev tooling",
	Long:  "codectl 启动本地 Web UI（内嵌前端 + 本地后端），整合 Provider/Spec/MCP/Settings 等能力。",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default action: start the embedded Web UI server
		addr, _ := cmd.Flags().GetString("addr")
		open, _ := cmd.Flags().GetBool("open")
		srv := &server.Server{Addr: addr}

		// Handle Ctrl+C for graceful shutdown
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		url := fmt.Sprintf("http://%s/", addr)
		system.Logger.Info("starting webui", "url", url)
		if open {
			if err := server.OpenBrowser(url); err != nil {
				system.Logger.Warn("failed to open browser", "err", err)
			}
		}
		if err := srv.Start(ctx); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err
		}
		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	// Flags to control the default web server
	rootCmd.Flags().StringP("addr", "a", "127.0.0.1:8787", "address to bind (host:port)")
	rootCmd.Flags().BoolP("open", "o", false, "open the browser after start")
}

// Execute runs the CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		system.Logger.Error("command execution failed", "err", err)
		os.Exit(1)
	}
}
