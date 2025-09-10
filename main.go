package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "os/exec"
    "regexp"
    "strings"
    "time"

    tea "github.com/charmbracelet/bubbletea"
)

// Tool identifiers and metadata
type ToolID string

const (
    ToolCodex  ToolID = "Codex"
    ToolClaude ToolID = "Claude"
    ToolGemini ToolID = "Gemini"
)

type ToolInfo struct {
    ID          ToolID
    DisplayName string
    Package     string   // npm package name for fallback detection
    Binaries    []string // candidate binary names in PATH
    VersionArgs [][]string
}

var tools = []ToolInfo{
    {
        ID:          ToolCodex,
        DisplayName: "Codex (@openai/codex)",
        Package:     "@openai/codex",
        Binaries:    []string{"codex", "openai-codex"},
        VersionArgs: [][]string{{"--version"}, {"-v"}, {"version"}},
    },
    {
        ID:          ToolClaude,
        DisplayName: "Claude Code (@anthropic-ai/claude-code)",
        Package:     "@anthropic-ai/claude-code",
        Binaries:    []string{"claude", "claude-code"},
        VersionArgs: [][]string{{"--version"}, {"-v"}, {"version"}},
    },
    {
        ID:          ToolGemini,
        DisplayName: "Gemini CLI (@google/gemini-cli)",
        Package:     "@google/gemini-cli",
        Binaries:    []string{"gemini"},
        VersionArgs: [][]string{{"--version"}, {"-v"}, {"version"}},
    },
}

// Check results
type CheckResult struct {
    Installed bool
    Version   string
    Source    string // which method produced version (binary/npm)
    Err       string
}

// Bubble Tea messages
type versionMsg struct {
    id     ToolID
    result CheckResult
}

// Model for TUI
type model struct {
    results   map[ToolID]CheckResult
    checking  bool
    updatedAt time.Time
    quitting  bool
}

func initialModel() model {
    return model{
        results:  make(map[ToolID]CheckResult, len(tools)),
        checking: true,
    }
}

func (m model) Init() tea.Cmd {
    return checkAllCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            m.quitting = true
            return m, tea.Quit
        case "r":
            // re-run checks
            m.results = make(map[ToolID]CheckResult, len(tools))
            m.checking = true
            return m, checkAllCmd()
        }
    case versionMsg:
        m.results[msg.id] = msg.result
        if len(m.results) == len(tools) {
            m.checking = false
            m.updatedAt = time.Now()
        }
        return m, nil
    }
    return m, nil
}

func (m model) View() string {
    if m.quitting {
        return "Goodbye!\n"
    }

    b := &strings.Builder{}
    fmt.Fprintf(b, "\n  codectl — CLI 版本检测\n\n")
    for _, t := range tools {
        res, ok := m.results[t.ID]
        if !ok && m.checking {
            fmt.Fprintf(b, "  • %-12s: 检测中…\n", t.ID)
            continue
        }
        if !res.Installed {
            if res.Err != "" {
                fmt.Fprintf(b, "  • %-12s: 未安装 (%s)\n", t.ID, res.Err)
            } else {
                fmt.Fprintf(b, "  • %-12s: 未安装\n", t.ID)
            }
            continue
        }
        // Installed
        ver := res.Version
        if ver == "" {
            ver = "(未知版本)"
        }
        fmt.Fprintf(b, "  • %-12s: %s  [%s]\n", t.ID, ver, res.Source)
    }

    if !m.updatedAt.IsZero() {
        fmt.Fprintf(b, "\n  上次更新: %s\n", m.updatedAt.Format("2006-01-02 15:04:05"))
    }
    fmt.Fprintf(b, "\n  操作: r 重新检测 · q 退出\n\n")
    return b.String()
}

// Commands
func checkAllCmd() tea.Cmd {
    cmds := make([]tea.Cmd, 0, len(tools))
    for _, t := range tools {
        t := t
        cmds = append(cmds, func() tea.Msg {
            res := checkTool(t)
            return versionMsg{id: t.ID, result: res}
        })
    }
    return tea.Batch(cmds...)
}

// checkTool attempts to detect tool version via PATH binaries, then falls back to npm global list.
func checkTool(t ToolInfo) CheckResult {
    // Try binaries in PATH
    for _, bin := range t.Binaries {
        if path, err := exec.LookPath(bin); err == nil {
            for _, args := range t.VersionArgs {
                ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
                out, err := runCmd(ctx, path, args...)
                cancel()
                if err == nil && strings.TrimSpace(out) != "" {
                    ver := parseVersion(out)
                    if ver == "" {
                        ver = strings.Split(strings.TrimSpace(out), "\n")[0]
                    }
                    return CheckResult{Installed: true, Version: ver, Source: fmt.Sprintf("%s %s", bin, strings.Join(args, " "))}
                }
            }
            // Found binary but no version output; still consider installed
            return CheckResult{Installed: true, Version: "", Source: bin}
        }
    }

    // Fallback: npm global list
    if t.Package != "" {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        ver, err := npmGlobalVersion(ctx, t.Package)
        if err == nil && ver != "" {
            return CheckResult{Installed: true, Version: ver, Source: "npm -g"}
        }
        if err != nil && !errors.Is(err, exec.ErrNotFound) {
            return CheckResult{Installed: false, Err: err.Error()}
        }
    }

    return CheckResult{Installed: false, Err: "未找到可执行文件或 npm 记录"}
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
    cmd := exec.CommandContext(ctx, name, args...)
    // Avoid opening pager or interactive prompts
    cmd.Env = append(os.Environ(), "NO_COLOR=1")
    out, err := cmd.CombinedOutput()
    if ctx.Err() == context.DeadlineExceeded {
        return "", ctx.Err()
    }
    return string(out), err
}

var verRe = regexp.MustCompile(`(?i)\bv?(\d+\.\d+\.\d+(?:[\w\.-]+)?)\b`)

func parseVersion(s string) string {
    s = strings.TrimSpace(s)
    if s == "" {
        return ""
    }
    // Take first line
    line := strings.Split(s, "\n")[0]
    if m := verRe.FindStringSubmatch(line); len(m) > 1 {
        return m[1]
    }
    // Fallback: try on full string
    if m := verRe.FindStringSubmatch(s); len(m) > 1 {
        return m[1]
    }
    return ""
}

// npmGlobalVersion queries npm for globally installed package version.
func npmGlobalVersion(ctx context.Context, pkg string) (string, error) {
    // `npm ls -g --depth=0 <pkg> --json`
    out, err := runCmd(ctx, "npm", "ls", "-g", "--depth=0", pkg, "--json")
    if err != nil && out == "" {
        return "", err
    }
    var data struct {
        Dependencies map[string]struct {
            Version string `json:"version"`
        } `json:"dependencies"`
    }
    if err := json.Unmarshal([]byte(out), &data); err != nil {
        return "", err
    }
    if d, ok := data.Dependencies[pkg]; ok {
        return d.Version, nil
    }
    return "", fmt.Errorf("package not found: %s", pkg)
}

func main() {
    if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}
