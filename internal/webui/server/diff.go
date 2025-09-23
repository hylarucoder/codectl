package server

import (
    "bytes"
    "context"
    "errors"
    "net/http"
    "os/exec"
    "path/filepath"
    "sort"
    "strings"
    "time"

    sys "codectl/internal/system"
)

type changeItem struct {
    Path   string `json:"path"`
    Status string `json:"status"` // XY from porcelain or simplified
    Group  string `json:"group"`  // Staged|Unstaged|Untracked
}

// diffChangesHandler lists working tree changes similar to TUI Diff tab.
// GET /api/diff/changes?mode=all|staged|worktree&specOnly=0|1
func diffChangesHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet { w.WriteHeader(http.StatusMethodNotAllowed); return }
    ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
    defer cancel()
    root, err := sys.GitRoot(ctx, ".")
    if err != nil || strings.TrimSpace(root) == "" {
        writeJSON(w, http.StatusBadRequest, errJSON(errors.New("not in a git repository")))
        return
    }
    mode := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mode")))
    specOnly := strings.TrimSpace(r.URL.Query().Get("specOnly")) == "1" || strings.ToLower(strings.TrimSpace(r.URL.Query().Get("specOnly"))) == "true"

    // Always use porcelain to build list. Mode only affects grouping semantics a bit; we keep full list.
    out, err := runGitOutput(ctx, root, "status", "--porcelain=1")
    if err != nil { writeJSON(w, http.StatusInternalServerError, errJSON(err)); return }
    lines := strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n")
    items := make([]changeItem, 0, len(lines))
    for _, ln := range lines {
        l := strings.TrimRight(ln, "\r\n")
        if strings.TrimSpace(l) == "" { continue }
        if len(l) < 3 { continue }
        // Format: XY<space>path or XY<space>old -> new (rename)
        xy := l[:2]
        rest := strings.TrimSpace(l[2:])
        p := rest
        if i := strings.Index(rest, " -> "); i >= 0 {
            p = strings.TrimSpace(rest[i+4:])
        }
        // Normalize to forward slashes
        p = filepath.ToSlash(p)
        group := ""
        switch {
        case xy == "??":
            group = "Untracked"
        case xy[0] != ' ':
            group = "Staged"
        case xy[1] != ' ':
            group = "Unstaged"
        default:
            group = "Unstaged"
        }
        if mode == "staged" && group != "Staged" { continue }
        if mode == "worktree" && group == "Staged" && xy != "??" { continue }
        if specOnly {
            if !(strings.HasPrefix(p, "vibe-docs/spec/") || strings.HasSuffix(strings.ToLower(p), ".spec.mdx")) {
                continue
            }
        }
        items = append(items, changeItem{Path: p, Status: xy, Group: group})
    }
    sort.Slice(items, func(i, j int) bool {
        gi := orderGroup(items[i].Group)
        gj := orderGroup(items[j].Group)
        if gi != gj { return gi < gj }
        if items[i].Path != items[j].Path { return strings.ToLower(items[i].Path) < strings.ToLower(items[j].Path) }
        return items[i].Status < items[j].Status
    })
    writeJSON(w, http.StatusOK, items)
}

func orderGroup(g string) int {
    switch g {
    case "Staged":
        return 0
    case "Unstaged":
        return 1
    case "Untracked":
        return 2
    default:
        return 3
    }
}

// diffFileHandler returns a unified diff for a file/path.
// GET /api/diff/file?path=...&mode=all|staged|worktree
func diffFileHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet { w.WriteHeader(http.StatusMethodNotAllowed); return }
    p := strings.TrimSpace(r.URL.Query().Get("path"))
    if p == "" { writeJSON(w, http.StatusBadRequest, errJSON(errors.New("missing path"))); return }
    mode := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mode")))
    ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
    defer cancel()
    root, err := sys.GitRoot(ctx, ".")
    if err != nil || strings.TrimSpace(root) == "" {
        writeJSON(w, http.StatusBadRequest, errJSON(errors.New("not in a git repository")))
        return
    }
    args := []string{"-c", "color.ui=false", "-c", "core.pager=cat", "diff", "--no-ext-diff"}
    switch mode {
    case "staged":
        args = append(args, "--cached")
    case "worktree":
        // default diff against index; keep as-is
    default: // all
        args = append(args, "HEAD")
    }
    args = append(args, "--", p)
    out, err := runGitOutput(ctx, root, args...)
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, errJSON(err))
        return
    }
    // If empty and file is untracked, provide a simple hint
    if strings.TrimSpace(out) == "" {
        out = "(no diff) — file might be untracked or unchanged)"
    }
    writeJSON(w, http.StatusOK, map[string]any{"path": filepath.ToSlash(p), "mode": modeOrAll(mode), "diff": out})
}

func modeOrAll(m string) string { if m == "" { return "all" }; return m }

// runGitOutput runs `git -C root <args...>` and returns stdout as string.
func runGitOutput(ctx context.Context, root string, args ...string) (string, error) {
    full := append([]string{"-C", root}, args...)
    cmd := exec.CommandContext(ctx, "git", full...)
    var buf bytes.Buffer
    cmd.Stdout = &buf
    cmd.Stderr = &buf
    if err := cmd.Run(); err != nil {
        return "", errors.New(strings.TrimSpace(buf.String()))
    }
    return buf.String(), nil
}

