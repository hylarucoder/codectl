package main

import (
	"os"

	"codectl/internal/cli"
	"codectl/internal/demo"
)

// If CODECTL_DEMO=chat is set, run the chat demo via `go run main.go`.
// Otherwise, default to the standard CLI entrypoint.
func main() {
	if os.Getenv("CODECTL_DEMO") == "chat" {
		_ = demo.Chat()
		return
	}
	cli.Execute()
}
