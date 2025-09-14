package ui

import (
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	cfg "codectl/internal/config"
	"codectl/internal/mcp"
	mdl "codectl/internal/models"
	"codectl/internal/provider"
)

// ConfigInfo aggregates lightweight configuration health for the dash.
type ConfigInfo struct {
	DotDir       string
	DotDirExists bool

	ProviderPath string
	ProviderOK   bool
	ProviderErr  string

	ModelsCount int
	Models      []string
	ModelsErr   string

	MCPCount int
	MCPNames []string
	MCPErr   string

	// Spec stats (from vibe-docs/spec)
	SpecDir         string
	SpecTotal       int
	SpecDraft       int
	SpecProposal    int
	SpecAccepted    int
	SpecDeprecated  int
	SpecRetired     int
	SpecRecent      []string // recent spec titles or filenames
	SpecRecentPaths []string // matching file paths for SpecRecent

	CheckedAt time.Time
}

// configInfoCmd collects config info in a background command.
func configInfoCmd() tea.Cmd {
	return func() tea.Msg {
		info := ConfigInfo{}
		info.CheckedAt = time.Now()

		if dir, err := cfg.DotDir(); err == nil {
			info.DotDir = dir
			if st, err2 := os.Stat(dir); err2 == nil && st.IsDir() {
				info.DotDirExists = true
			}
		}

		// Provider
		if p, err := provider.Path(); err == nil {
			info.ProviderPath = p
			if _, statErr := os.Stat(p); statErr == nil {
				if _, loadErr := provider.LoadV2(); loadErr == nil {
					info.ProviderOK = true
				} else {
					info.ProviderErr = loadErr.Error()
				}
			} else if os.IsNotExist(statErr) {
				// missing file is acceptable; indicates default in-memory provider
				info.ProviderOK = true
			} else if statErr != nil {
				info.ProviderErr = statErr.Error()
			}
		}

		// Models
		if arr, err := mdl.Load(); err == nil {
			info.ModelsCount = len(arr)
			// keep up to first 5 for display
			if len(arr) > 5 {
				info.Models = append(info.Models, arr[:5]...)
			} else {
				info.Models = append(info.Models, arr...)
			}
		} else {
			info.ModelsErr = err.Error()
		}

		// MCP
		if names, err := mcp.Names(); err == nil {
			info.MCPCount = len(names)
			// keep up to first 6
			if len(names) > 6 {
				info.MCPNames = append(info.MCPNames, names[:6]...)
			} else {
				info.MCPNames = append(info.MCPNames, names...)
			}
		} else if os.IsNotExist(err) {
			info.MCPCount = 0
		} else {
			info.MCPErr = err.Error()
		}

		// Spec stats from local repo (vibe-docs/spec)
		// best-effort; ignore errors silently
		scanSpecStats(&info)
		return configInfoMsg{info: info}
	}
}

// scanSpecStats fills Spec* fields in ConfigInfo by scanning vibe-docs/spec/*.spec.mdx.
func scanSpecStats(info *ConfigInfo) {
	// Prefer project-local path "vibe-docs/spec"
	const specDir = "vibe-docs/spec"
	info.SpecDir = specDir
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return
	}
	type rec struct {
		title   string
		status  string
		mtime   time.Time
		display string
		path    string
	}
	var recent []rec
	for _, de := range entries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if !strings.HasSuffix(name, ".spec.mdx") {
			continue
		}
		info.SpecTotal++
		path := specDir + string(os.PathSeparator) + name
		// mtime
		if fi, err := os.Stat(path); err == nil {
			// capture mod time
			// (without VCS history we approximate "recently marked done" by mtime)
			// ignore errors
			_ = fi
		}
		// Parse frontmatter quickly: read first ~2KB and scan until the second '---'
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// limit to first 4096 bytes for quick scan
		if len(b) > 4096 {
			b = b[:4096]
		}
		content := string(b)
		// find title and status only within the first frontmatter block
		title := ""
		status := ""
		// naive: iterate lines until second '---'
		lines := strings.Split(content, "\n")
		sepCount := 0
		for _, ln := range lines {
			t := strings.TrimSpace(ln)
			if t == "---" {
				sepCount++
				if sepCount >= 2 {
					break
				}
				continue
			}
			if sepCount == 0 {
				// frontmatter not started yet; some files might omit opening fence, but our docs include it
				continue
			}
			// simple key: value parse
			if strings.HasPrefix(t, "title:") {
				title = strings.TrimSpace(strings.TrimPrefix(t, "title:"))
			} else if strings.HasPrefix(t, "status:") {
				status = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(t, "status:")))
			}
		}
		switch status {
		case "draft":
			info.SpecDraft++
		case "proposal":
			info.SpecProposal++
		case "accepted":
			info.SpecAccepted++
		case "deprecated":
			info.SpecDeprecated++
		case "retired":
			info.SpecRetired++
		}
		// compute title (fall back to filename)
		if title == "" {
			title = strings.TrimSuffix(name, ".mdx")
		}
		// mtime
		var mt time.Time
		if fi, err := os.Stat(path); err == nil {
			mt = fi.ModTime()
		}
		// collect all specs for recent table (regardless of status)
		recent = append(recent, rec{title: title, status: status, mtime: mt, display: title, path: path})
	}
	// Sort all by mtime desc and keep top 5
	sort.Slice(recent, func(i, j int) bool { return recent[i].mtime.After(recent[j].mtime) })
	for i := 0; i < len(recent) && i < 5; i++ {
		info.SpecRecent = append(info.SpecRecent, recent[i].display)
		info.SpecRecentPaths = append(info.SpecRecentPaths, recent[i].path)
	}
}
