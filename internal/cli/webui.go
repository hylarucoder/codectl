package cli

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "errors"

	"github.com/spf13/cobra"

	"codectl/internal/system"
	"codectl/internal/webui/server"
)

func init() {
	rootCmd.AddCommand(webuiCmd)
	webuiCmd.Flags().StringP("addr", "a", "127.0.0.1:8787", "address to bind (host:port)")
	webuiCmd.Flags().BoolP("open", "o", false, "open the browser after start")
}

var webuiCmd = &cobra.Command{
	Use:   "webui",
	Short: "Start the local Web UI server",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")
		open, _ := cmd.Flags().GetBool("open")
		srv := &server.Server{Addr: addr}

		// Handle Ctrl+C
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

        url := fmt.Sprintf("http://%s/_/", addr)
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
}
