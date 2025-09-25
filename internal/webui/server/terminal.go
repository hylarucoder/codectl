package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

// wsUpgrader upgrades HTTP connections to WebSocket.
var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// Allow all origins for local dev; the server typically binds to localhost.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// terminalWSHandler launches a system shell in a PTY and bridges it over WebSocket.
//
// Client protocol:
// - Send plain text messages as input to the shell.
// - Control messages are JSON: {"type":"resize","cols":<int>,"rows":<int>}.
// - Server sends PTY output as text messages.
func terminalWSHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("upgrade failed: %v", err), http.StatusBadRequest)
		return
	}
	defer conn.Close()

	// Pick shell
	sh, shArgs := defaultShell()
	cmd := exec.Command(sh, shArgs...)

	// Inherit working dir from server process; allow overriding via ?cwd=
	if q := r.URL.Query().Get("cwd"); q != "" {
		cmd.Dir = q
	}

	// Create PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("failed to start shell: "+err.Error()))
		return
	}
	defer func() { _ = ptmx.Close() }() // Best-effort close; will kill the child

	// Optional initial size from query
	if cols, _ := strconv.Atoi(r.URL.Query().Get("cols")); cols > 0 {
		if rows, _ := strconv.Atoi(r.URL.Query().Get("rows")); rows > 0 {
			_ = pty.Setsize(ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
		}
	}

	// Writer: PTY -> WS
	go func() {
		reader := bufio.NewReader(ptmx)
		buf := make([]byte, 4096)
		for {
			n, readErr := reader.Read(buf)
			if n > 0 {
				// Send as text message. xterm expects UTF-8; PTY typically produces UTF-8 on Unix.
				_ = conn.WriteMessage(websocket.TextMessage, buf[:n])
			}
			if readErr != nil {
				if !errors.Is(readErr, io.EOF) {
					_ = conn.WriteMessage(websocket.TextMessage, []byte("\r\n[pty closed]: "+readErr.Error()+"\r\n"))
				}
				// If the command is still running, give it a moment to exit.
				time.Sleep(50 * time.Millisecond)
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "pty closed"))
				_ = conn.Close()
				return
			}
		}
	}()

	// Reader: WS -> PTY
	type resizeMsg struct {
		Type string `json:"type"`
		Cols int    `json:"cols"`
		Rows int    `json:"rows"`
		Data string `json:"data"`
	}

	for {
		mt, data, err := conn.ReadMessage()
		if err != nil {
			// client closed
			break
		}
		switch mt {
		case websocket.TextMessage, websocket.BinaryMessage:
			// Try JSON first for control frames
			var rm resizeMsg
			if json.Unmarshal(data, &rm) == nil && rm.Type != "" {
				if rm.Type == "resize" && rm.Cols > 0 && rm.Rows > 0 {
					_ = pty.Setsize(ptmx, &pty.Winsize{Cols: uint16(rm.Cols), Rows: uint16(rm.Rows)})
					continue
				}
				if rm.Type == "input" && rm.Data != "" {
					_, _ = ptmx.Write([]byte(rm.Data))
					continue
				}
			}
			// Treat as raw input
			if len(data) > 0 {
				_, _ = ptmx.Write(data)
			}
		case websocket.CloseMessage:
			return
		default:
			// ignore other frames
		}
	}
}

// defaultShell returns the platform-appropriate shell and arguments.
func defaultShell() (string, []string) {
	if runtime.GOOS == "windows" {
		// Fallback to powershell if available
		pwsh := os.Getenv("COMSPEC")
		if pwsh == "" {
			pwsh = "powershell.exe"
		}
		return pwsh, []string{}
	}
	// Respect $SHELL, default to /bin/bash then /bin/sh
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh, []string{"-l"}
	}
	if _, err := os.Stat("/bin/bash"); err == nil {
		return "/bin/bash", []string{"-l"}
	}
	return "/bin/sh", []string{"-l"}
}
