package ui

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    xansi "github.com/charmbracelet/x/ansi"
    fuzzy "github.com/sahilm/fuzzy"

	"codectl/internal/system"
	"codectl/internal/tools"
)

type SlashCmd struct {
	Name    string
	Aliases []string
	Desc    string
}

var slashCmds = []SlashCmd{
    {Name: "/add-dir", Desc: "Add a new working directory"},
    {Name: "/agents", Desc: "Manage agent configurations"},
    {Name: "/bashes", Desc: "List and manage background tasks"},
    {Name: "/clear", Aliases: []string{"/reset", "/new"}, Desc: "Clear conversation history and free up context"},
    {Name: "/compact", Desc: "Clear history but keep a summary"},
    {Name: "/config", Aliases: []string{"/theme"}, Desc: "Open config panel"},
    {Name: "/context", Desc: "Visualize current context usage"},
    {Name: "/cost", Desc: "Show total cost and duration"},
    {Name: "/doctor", Desc: "Diagnose and verify installation"},
    {Name: "/add", Desc: "Install supported CLIs (all|codex|claude|gemini)"},
    {Name: "/remove", Desc: "Uninstall supported CLIs"},
    {Name: "/exit", Aliases: []string{"/quit"}, Desc: "Exit the REPL"},
    {Name: "/upgrade", Aliases: []string{"/update"}, Desc: "Upgrade all supported CLIs to latest"},
    {Name: "/status", Desc: "Show current status for tools"},
    {Name: "/init", Desc: "Initialize vibe-docs/AGENTS.md in current repo"},
    {Name: "/task", Desc: "Create vibe-docs/task/YYMMDD-HHMMSS-<slug>.task.mdx"},
	{Name: "/spec", Desc: "Generate spec via Codex and save to vibe-docs/spec"},
}

func (m *model) refreshSlash() {
	v := m.ti.Value()
	// slash visible only when input starts with '/'
	if !strings.HasPrefix(v, "/") {
		m.slashVisible = false
		m.slashFiltered = nil
		m.slashIndex = 0
		return
	}
	m.slashVisible = true
	// filter by prefix token (first word)
	q := strings.TrimSpace(v)
	want := q
	// if there are spaces, only use the first token for filtering
	if sp := strings.IndexAny(q, " \t"); sp >= 0 {
		want = q[:sp]
	}
	m.slashFiltered = filterSlashCommands(want)
	if m.slashIndex >= len(m.slashFiltered) {
		m.slashIndex = 0
	}
}

func filterSlashCommands(prefix string) []SlashCmd {
    // Show all when prefix is just '/'
    if prefix == "/" {
        return slashCmds
    }
    // Fuzzy match on Name and Aliases (case-insensitive)
    q := strings.ToLower(strings.TrimSpace(prefix))
    if q == "" {
        return slashCmds
    }
    // Build candidate list mapping every name/alias to its command
    cand := make([]string, 0, len(slashCmds)*2)
    idx := make(map[string]SlashCmd, len(slashCmds)*2)
    for _, c := range slashCmds {
        key := strings.ToLower(c.Name)
        cand = append(cand, key)
        idx[key] = c
        for _, a := range c.Aliases {
            ak := strings.ToLower(a)
            cand = append(cand, ak)
            idx[ak] = c
        }
    }
    // Run fuzzy search over lowercased candidates
    matches := fuzzy.Find(q, cand)
    if len(matches) == 0 {
        return nil
    }
    // Deduplicate to canonical command order while preserving match ranking
    out := make([]SlashCmd, 0, len(matches))
    seen := map[string]bool{}
    for _, m := range matches {
        s := cand[m.Index]
        c := idx[s]
        if seen[c.Name] {
            continue
        }
        out = append(out, c)
        seen[c.Name] = true
    }
    return out
}

func renderSlashHelp(width int, cmds []SlashCmd, sel int) string {
	// limit list size for readability
	maxItems := 10
	if len(cmds) > maxItems {
		cmds = cmds[:maxItems]
		if sel >= maxItems {
			sel = maxItems - 1
		}
	}
	// compute widths
	nameWidth := 16
	inner := width - 2
	if inner < 20 {
		inner = 20
	}
	// styles
	hl := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render
	var b strings.Builder
	// top border
	b.WriteString("╭" + strings.Repeat("─", inner) + "╮\n")
	if len(cmds) == 0 {
		line := "  no matches"
		w := xansi.StringWidth(line)
		if w > inner {
			diff := w - inner
			if diff > 0 && diff < len(line) {
				line = line[:len(line)-diff]
			}
		}
		b.WriteString("│")
		b.WriteString(line)
		pad := inner - xansi.StringWidth(line)
		if pad > 0 {
			b.WriteString(strings.Repeat(" ", pad))
		}
		b.WriteString("│\n")
		// bottom border and hint
		b.WriteString("╰" + strings.Repeat("─", inner) + "╯\n")
		b.WriteString("  ↑/↓ 选择 · Tab 补全 · Enter 执行 · Esc 关闭\n")
		return b.String()
	}
	for i, c := range cmds {
		line := fmt.Sprintf("  %-*s  %s", nameWidth, c.Name, dim(c.Desc))
		// trim to inner width
		w := xansi.StringWidth(line)
		if w > inner {
			// naive trim to fit
			diff := w - inner
			if diff > 0 && diff < len(line) {
				line = line[:len(line)-diff]
			}
		}
		if i == sel {
			line = hl(line)
		}
		b.WriteString("│")
		b.WriteString(line)
		// pad to inner
		pad := inner - xansi.StringWidth(line)
		if pad > 0 {
			b.WriteString(strings.Repeat(" ", pad))
		}
		b.WriteString("│\n")
	}
	// bottom border
	b.WriteString("╰" + strings.Repeat("─", inner) + "╯\n")
	// hint line
	b.WriteString("  ↑/↓ 选择 · Tab 补全 · Enter 执行 · Esc 关闭\n")
	return b.String()
}

// execSlashLine parses and executes a typed slash command line.
func (m model) execSlashLine(line string) tea.Cmd {
	s := strings.TrimSpace(line)
	if s == "" || !strings.HasPrefix(s, "/") {
		return nil
	}
	parts := strings.Fields(s)
	cmd := parts[0]
	args := ""
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}
	return m.execSlashCmd(cmd, args)
}

// execSlashCmd executes a slash command by name and optional args.
func (m model) execSlashCmd(cmd string, args string) tea.Cmd {
    c := canonicalSlash(cmd)
    switch c {
    case "/exit", "/quit":
        return func() tea.Msg { return quitMsg{} }
    case "/clear", "/reset", "/new":
        return func() tea.Msg { return noticeMsg("已清空会话（占位实现）") }
    case "/doctor":
        // Trigger re-check and show notice
        return tea.Batch(
            func() tea.Msg { return noticeMsg("正在运行诊断…") },
            checkAllCmd(),
        )
    case "/add":
        // Install specified or all tools via npm, similar to `codectl cli add`
        sel := selectToolsFromArgString(args)
        if len(sel) == 0 {
            return func() tea.Msg { return noticeMsg("未选择任何工具（用法：/add all|codex|claude|gemini...）") }
        }
        return tea.Batch(
            func() tea.Msg { return noticeMsg("正在安装所选工具…（请稍候）") },
            func() tea.Msg {
                var b strings.Builder
                for i, t := range sel {
                    fmt.Fprintf(&b, "[%d/%d] %s 安装中…\n", i+1, len(sel), t.DisplayName)
                    res := tools.CheckTool(t)
                    if res.Installed {
                        ver := res.Version
                        if strings.TrimSpace(ver) == "" {
                            ver = "已安装"
                        }
                        fmt.Fprintf(&b, "  • 跳过：%s\n", ver)
                        continue
                    }
                    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
                    err := tools.NpmUpgradeLatest(ctx, t.Package)
                    cancel()
                    if err != nil {
                        fmt.Fprintf(&b, "  × 安装失败：%v\n", err)
                        continue
                    }
                    // Recheck and report
                    res2 := tools.CheckTool(t)
                    ver := strings.TrimSpace(res2.Version)
                    if ver == "" {
                        if res2.Latest != "" {
                            ver = res2.Latest
                        } else {
                            ver = "latest"
                        }
                    }
                    fmt.Fprintf(&b, "  ✓ 安装成功 → %s\n", ver)
                }
                return noticeMsg(b.String())
            },
        )
    case "/remove":
        // Uninstall specified or all tools via npm, similar to `codectl cli remove`
        sel := selectToolsFromArgString(args)
        if len(sel) == 0 {
            return func() tea.Msg { return noticeMsg("未选择任何工具（用法：/remove all|codex|claude|gemini...）") }
        }
        return tea.Batch(
            func() tea.Msg { return noticeMsg("正在卸载所选工具…（请稍候）") },
            func() tea.Msg {
                var b strings.Builder
                for i, t := range sel {
                    fmt.Fprintf(&b, "[%d/%d] %s 卸载中…\n", i+1, len(sel), t.DisplayName)
                    res := tools.CheckTool(t)
                    if !res.Installed {
                        fmt.Fprintf(&b, "  • 未安装，跳过\n")
                        continue
                    }
                    if strings.TrimSpace(t.Package) == "" {
                        fmt.Fprintf(&b, "  • 未配置 npm 包名，无法卸载，跳过\n")
                        continue
                    }
                    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
                    err := tools.NpmUninstallGlobal(ctx, t.Package)
                    cancel()
                    if err != nil {
                        fmt.Fprintf(&b, "  × 卸载失败：%v\n", err)
                        continue
                    }
                    // Recheck
                    res2 := tools.CheckTool(t)
                    if res2.Installed {
                        note := "仍检测到已安装"
                        if strings.TrimSpace(res2.Source) != "" {
                            note += fmt.Sprintf("（来源：%s）", res2.Source)
                        }
                        fmt.Fprintf(&b, "  • %s\n", note)
                    } else {
                        fmt.Fprintf(&b, "  ✓ 卸载成功\n")
                    }
                }
                return noticeMsg(b.String())
            },
        )
    case "/status":
        return func() tea.Msg {
            // Build a concise one-line status summary
            parts := make([]string, 0, len(slashCmds))
            for _, t := range tools.Tools {
				res, ok := m.results[t.ID]
				if !ok && m.checking {
					parts = append(parts, fmt.Sprintf("%s: 检测中…", t.ID))
					continue
				}
				if !ok {
					parts = append(parts, fmt.Sprintf("%s: 未知", t.ID))
					continue
				}
				if !res.Installed {
					parts = append(parts, fmt.Sprintf("%s: 未安装", t.ID))
					continue
				}
				ver := res.Version
				if ver == "" {
					ver = "?"
				}
				if res.Latest != "" && tools.VersionLess(res.Version, res.Latest) {
					parts = append(parts, fmt.Sprintf("%s: %s→%s", t.ID, ver, res.Latest))
				} else {
					parts = append(parts, fmt.Sprintf("%s: %s", t.ID, ver))
				}
			}
			if len(parts) == 0 {
				return noticeMsg("暂无状态")
			}
			summary := strings.Join(parts, " · ")
			return noticeMsg(summary)
		}
	case "/upgrade", "/update":
		// Kick off the same upgrade flow as pressing 'u'
		return func() tea.Msg { return startUpgradeMsg{} }
	case "/init":
		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			gi, _ := system.GetGitInfo(ctx, m.cwd)
			if !gi.InRepo {
				return noticeMsg("当前目录不在 Git 仓库内，未进行任何操作")
			}
			root, err := system.GitRoot(ctx, m.cwd)
			if err != nil || strings.TrimSpace(root) == "" {
				root = m.cwd
			}
			dir := filepath.Join(root, "vibe-docs")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return noticeMsg(fmt.Sprintf("创建目录失败：%v", err))
			}
			path := filepath.Join(dir, "AGENTS.md")
			if _, statErr := os.Stat(path); statErr == nil {
				return noticeMsg(fmt.Sprintf("已存在：%s", path))
			} else if !os.IsNotExist(statErr) {
				return noticeMsg(fmt.Sprintf("无法访问 %s：%v", path, statErr))
			}
			content := defaultAgentsMD()
			if writeErr := os.WriteFile(path, []byte(content), 0o644); writeErr != nil {
				return noticeMsg(fmt.Sprintf("写入失败：%v", writeErr))
			}
			return noticeMsg(fmt.Sprintf("已创建：%s", path))
		}
	case "/task":
		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			gi, _ := system.GetGitInfo(ctx, m.cwd)
			if !gi.InRepo {
				return noticeMsg("当前目录不在 Git 仓库内，未进行任何操作")
			}
			root, err := system.GitRoot(ctx, m.cwd)
			if err != nil || strings.TrimSpace(root) == "" {
				root = m.cwd
			}
			dir := filepath.Join(root, "vibe-docs", "task")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return noticeMsg(fmt.Sprintf("创建目录失败：%v", err))
			}
			// Use args as task title when provided; build slug
			title := strings.TrimSpace(args)
			if title == "" {
				title = "未命名任务"
			}
			now := time.Now()
			ts := now.Format("060102-150405") // YYMMDD-HHMMSS
			slug := slugify(title)
			filename := fmt.Sprintf("%s-%s.task.mdx", ts, slug)
			path := filepath.Join(dir, filename)
			content := defaultTaskMD(title, now)
			if writeErr := os.WriteFile(path, []byte(content), 0o644); writeErr != nil {
				return noticeMsg(fmt.Sprintf("写入失败：%v", writeErr))
			}
			return noticeMsg(fmt.Sprintf("已创建：%s", path))
		}
	case "/spec":
		// Generate a spec via Codex CLI and save under vibe-docs/spec
		prompt := strings.TrimSpace(args)
		if prompt == "" {
			return func() tea.Msg { return noticeMsg("用法：/spec <说明>") }
		}
		return tea.Batch(
			func() tea.Msg { return noticeMsg("正在生成 spec（codex exec）…") },
			func() tea.Msg {
				// locate codex binary
				bin := ""
				for _, cand := range []string{"codex", "openai-codex"} {
					if p, err := exec.LookPath(cand); err == nil && p != "" {
						bin = p
						break
					}
				}
				if bin == "" {
					return noticeMsg("未找到 codex CLI（尝试安装 @openai/codex）")
				}
				// resolve repo root
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				root, err := system.GitRoot(ctx, m.cwd)
				cancel()
				if err != nil || strings.TrimSpace(root) == "" {
					root = m.cwd
				}
				outDir := filepath.Join(root, "vibe-docs", "spec")
				if err := os.MkdirAll(outDir, 0o755); err != nil {
					return noticeMsg(fmt.Sprintf("创建目录失败：%v", err))
				}
				// Run codex exec with robust fallback: arg then stdin
				body, runErr := runCodex(bin, prompt, 120*time.Second)
				if runErr != nil && body == "" {
					return noticeMsg(fmt.Sprintf("codex exec 失败：%v", runErr))
				}
				content := body
				// wrap with minimal frontmatter when missing
				trimmed := strings.TrimSpace(body)
				if !strings.HasPrefix(trimmed, "---") {
					title := prompt
					fm := "---\n" +
						"title: " + title + "\n" +
						"specVersion: 0.1.0\n" +
						"status: draft\n" +
						"lastUpdated: {auto}\n" +
						"---\n\n"
					content = fm + body
				}
				// filename
				ts := time.Now().Format("060102-150405")
				name := fmt.Sprintf("draft-%s-%s.spec.mdx", ts, slugify(prompt))
				outPath := filepath.Join(outDir, name)
				if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
					return noticeMsg(fmt.Sprintf("写入失败：%v", err))
				}
				// also persist raw output for debugging if there was a non-fatal error
				if runErr != nil {
					_ = os.WriteFile(outPath+".raw.txt", []byte(body), 0o644)
				}
				return noticeMsg(fmt.Sprintf("已生成：%s（使用 %s）", outPath, filepath.Base(bin)))
			},
		)
	default:
		// not implemented
		return func() tea.Msg {
			// find description if exists
			var desc string
			for _, sc := range slashCmds {
				if sc.Name == c {
					desc = sc.Desc
					break
				}
			}
			if desc == "" {
				desc = "未实现"
			}
			return noticeMsg(fmt.Sprintf("命令 %s：%s (尚未实现)", c, desc))
		}
	}
}

// canonicalize command including aliases
func canonicalSlash(name string) string {
	n := strings.ToLower(name)
	for _, c := range slashCmds {
		if strings.ToLower(c.Name) == n {
			return c.Name
		}
		for _, a := range c.Aliases {
			if strings.ToLower(a) == n {
				return c.Name
			}
		}
	}
	return n
}

// defaultAgentsMD returns a minimal template for AGENTS.md
func defaultAgentsMD() string {
	return `# AGENTS.md

This file guides AI coding agents working in this repository.

- Scope: This file applies to the entire repository.
- Conventions: Add code style, naming, and architectural guidelines here.
- How to Run: Document dev setup and commands.
- Testing: Where tests live and how to run them.
- Prohibited: List areas agents must not modify.

You can create more AGENTS.md files in subdirectories for overrides.
`
}

// defaultTaskMDX returns a minimal template for 000-a-task.mdx
func defaultTaskMD(title string, t time.Time) string {
	if title == "" {
		title = "未命名任务"
	}
	// ISO timestamp for createdAt
	created := t.Format(time.RFC3339)
	return "---\n" +
		"title: " + title + "\n" +
		"createdAt: " + created + "\n" +
		"lastUpdated: {auto}\n" +
		"---\n\n" +
		"# 任务说明（草案）\n\n" +
		"> 由 codectl /task 生成。可使用 '/task <标题>' 指定标题。\n\n" +
		"## 背景\n- \n\n" +
		"## 目标\n- \n\n" +
		"## 非目标\n- \n\n" +
		"## 验收标准\n- \n\n" +
		"## 实现要点\n- \n\n" +
		"## 风险与依赖\n- \n\n" +
		"## 参考链接\n- \n"
}

// slugify converts a title to a safe kebab-case slug
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// replace non-alphanumeric (including spaces) with '-'
	b := make([]rune, 0, len(s))
	lastDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b = append(b, r)
			lastDash = false
			continue
		}
		// keep CJK as-is to allow readable slugs; otherwise dash
		if r >= 0x4E00 && r <= 0x9FFF {
			b = append(b, r)
			lastDash = false
			continue
		}
		if !lastDash {
			b = append(b, '-')
			lastDash = true
		}
	}
	// trim leading/trailing '-'
	res := strings.Trim(btoa(b), "-")
	if res == "" {
		res = "task"
	}
	return res
}

func btoa(r []rune) string { return string(r) }

// runCodex tries to execute `codex exec` with prompt as argument and
// falls back to providing the prompt via STDIN. Returns output body and error.
func runCodex(bin string, prompt string, timeout time.Duration) (string, error) {
	// try with argument
	{
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		cmd := exec.CommandContext(ctx, bin, "exec", prompt)
		cmd.Env = append(os.Environ(), "NO_COLOR=1")
		out, err := cmd.CombinedOutput()
		cancel()
		if err == nil && len(out) > 0 {
			return string(out), nil
		}
		// keep err/out for fallback decision
		if ctx.Err() == context.DeadlineExceeded {
			return "", ctx.Err()
		}
		if len(out) > 0 {
			// if we have output even with error, return it to persist
			return string(out), err
		}
	}
	// fallback: provide prompt via STDIN
	{
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		cmd := exec.CommandContext(ctx, bin, "exec")
		cmd.Env = append(os.Environ(), "NO_COLOR=1")
		cmd.Stdin = strings.NewReader(prompt)
		out, err := cmd.CombinedOutput()
		cancel()
		if err == nil && len(out) > 0 {
			return string(out), nil
		}
		if len(out) > 0 {
			return string(out), err
		}
		if err != nil {
			return "", err
		}
		return string(out), nil
	}
}

// selectToolsFromArgString converts a space-separated args string into a slice of ToolInfo.
// Accepts: empty (defaults to all), or any of: all, codex, claude, gemini.
func selectToolsFromArgString(args string) []tools.ToolInfo {
    s := strings.TrimSpace(args)
    if s == "" {
        return tools.Tools
    }
    parts := strings.Fields(s)
    m := map[string]bool{}
    for _, p := range parts {
        pp := strings.TrimSpace(strings.ToLower(p))
        if pp == "" {
            continue
        }
        m[pp] = true
    }
    if m["all"] {
        return tools.Tools
    }
    sel := make([]tools.ToolInfo, 0, len(tools.Tools))
    for _, t := range tools.Tools {
        id := strings.ToLower(string(t.ID))
        names := []string{
            id,
            strings.ToLower(t.DisplayName),
        }
        switch t.ID {
        case tools.ToolCodex:
            names = append(names, "codex", "openai", "openai-codex")
        case tools.ToolClaude:
            names = append(names, "claude", "claude-code", "anthropic")
        case tools.ToolGemini:
            names = append(names, "gemini", "google")
        }
        matched := false
        for _, n := range names {
            if m[n] {
                matched = true
                break
            }
        }
        if matched {
            sel = append(sel, t)
        }
    }
    return sel
}
