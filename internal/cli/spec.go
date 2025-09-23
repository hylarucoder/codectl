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

func init() { rootCmd.AddCommand(specCmd) }

var specCmd = &cobra.Command{
    Use:   "spec",
    Short: "Open Spec UI in the Web UI",
    RunE: func(cmd *cobra.Command, args []string) error {
        addr := "127.0.0.1:8787"
        // Start embedded web server and open browser pointing to Spec UI
        srv := &server.Server{Addr: addr}
        // Handle Ctrl+C
        ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
        defer cancel()
        url := fmt.Sprintf("http://%s/_/", addr)
        system.Logger.Info("opening Spec UI", "url", url)
        if err := server.OpenBrowser(url); err != nil {
            system.Logger.Warn("failed to open browser", "err", err)
        }
        if err := srv.Start(ctx); err != nil {
            if errors.Is(err, http.ErrServerClosed) { return nil }
            return err
        }
        return nil
    },
}

// no flags; `codectl spec` starts server and opens browser
