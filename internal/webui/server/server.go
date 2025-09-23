package server

import (
	"context"
	"errors"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"codectl/internal/system"
	appver "codectl/internal/version"
	webembed "codectl/internal/webui/embed"
)

type Server struct {
	Addr string
}

func (s *Server) Start(ctx context.Context) error {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// API routes: wrap existing net/http handlers for minimal changes
	mountAPIGin(r)
	mountEmbeddedUIGin(r)

	srv := &http.Server{Addr: s.Addr, Handler: r, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	system.Logger.Info("webui server listening", "addr", s.Addr)
	return srv.ListenAndServe()
}

// OpenBrowser tries to open a URL in the system browser.
func OpenBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return runCmd(cmd, args...)
}

func mountAPIGin(r *gin.Engine) {
	api := r.Group("/api")
	api.GET("/health", gin.WrapF(func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))
	api.GET("/version", gin.WrapF(func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"version": appver.AppVersion})
	}))

	// providers
	api.Any("/providers", gin.WrapF(providersHandler))

	// FS
	api.GET("/fs/tree", gin.WrapF(fsTreeHandler))
	api.GET("/fs/read", gin.WrapF(fsReadHandler))
	api.PUT("/fs/write", gin.WrapF(fsWriteHandler))
	api.POST("/fs/rename", gin.WrapF(fsRenameHandler))
	api.POST("/fs/delete", gin.WrapF(fsDeleteHandler))
	api.POST("/fs/patch", gin.WrapF(fsPatchHandler))

	// Spec
	api.GET("/spec/docs", gin.WrapF(specListHandler))
	api.Any("/spec/doc", gin.WrapF(specDocHandler))
	api.POST("/spec/validate", gin.WrapF(specValidateHandler))

	// Diff
	api.GET("/diff/changes", gin.WrapF(diffChangesHandler))
	api.GET("/diff/file", gin.WrapF(diffFileHandler))

	// Tasks
	api.GET("/tasks/list", gin.WrapF(tasksListHandler))
	api.PUT("/tasks/update", gin.WrapF(tasksUpdateHandler))

	// Sessions
	api.Any("/sessions", gin.WrapF(sessionsRootHandler))
	// Catch-all below /api/sessions/* to the http handler
	r.Any("/api/sessions/*any", gin.WrapF(sessionItemHandler))

	// Terminal (WebSocket PTY)
	api.GET("/term/ws", gin.WrapF(terminalWSHandler))
}

// mountEmbeddedUIGin serves embedded SPA at all non-/api GET routes with index fallback.
func mountEmbeddedUIGin(r *gin.Engine) {
	// create sub FS at dist
	dist, err := fs.Sub(webembed.DistFS, "dist")
	if err != nil {
		// Serve helpful 404 for non-API via NoRoute
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api/") || c.Request.URL.Path == "/api" {
				c.Status(http.StatusNotFound)
				return
			}
			c.String(http.StatusNotFound, "webui assets not found. Build frontend into ui/dist and recompile.")
		})
		return
	}
	httpFS := http.FS(dist)
	// Serve embedded UI at root with SPA fallback using NoRoute to avoid wildcard conflicts
	r.NoRoute(func(c *gin.Context) {
		// Do not hijack API routes
		if strings.HasPrefix(c.Request.URL.Path, "/api/") || c.Request.URL.Path == "/api" {
			c.Status(http.StatusNotFound)
			return
		}
		// Only handle GET/HEAD for SPA assets
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}
		p := strings.TrimPrefix(c.Request.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		// Try static file
		f, err := httpFS.Open(p)
		if err == nil {
			_ = f.Close()
			if ct := mime.TypeByExtension(filepath.Ext(p)); ct != "" {
				c.Header("Content-Type", ct)
			}
			c.FileFromFS(p, httpFS)
			return
		}
		// fallback to index.html
		if _, err := httpFS.Open("index.html"); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				c.String(http.StatusNotFound, "index.html not found in embedded dist.")
				return
			}
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.FileFromFS("index.html", httpFS)
	})
}
