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
    "regexp"
    "strconv"

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
    format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format"))) // "split" for side-by-side rows
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
    res := map[string]any{"path": filepath.ToSlash(p), "mode": modeOrAll(mode), "diff": out}
    if format == "split" {
        rows := unifiedToSplit(out)
        res["split"] = rows
    }
    writeJSON(w, http.StatusOK, res)
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

// unifiedToSplit converts a unified diff string into side-by-side rows.
// Each row contains left/right text and a type: ctx|del|add|meta|empty, plus optional line numbers.
type splitRow struct {
    Left string `json:"left"`
    Right string `json:"right"`
    LT   string `json:"lt"`
    RT   string `json:"rt"`
    LN   int    `json:"ln,omitempty"`
    RN   int    `json:"rn,omitempty"`
}

var hunkRe = regexp.MustCompile(`^@@ -([0-9]+)(?:,([0-9]+))? \+([0-9]+)(?:,([0-9]+))? @@`)

func unifiedToSplit(diff string) []splitRow {
    rows := make([]splitRow, 0, 256)
    lines := strings.Split(strings.ReplaceAll(diff, "\r\n", "\n"), "\n")
    // pending blocks of deletions/additions (within a hunk)
    type lineNumText struct{ n int; s string }
    var dels, adds []lineNumText
    flush := func() {
        n := len(dels)
        if len(adds) > n { n = len(adds) }
        for i := 0; i < n; i++ {
            var l, r lineNumText
            if i < len(dels) { l = dels[i] } else { l = lineNumText{} }
            if i < len(adds) { r = adds[i] } else { r = lineNumText{} }
            row := splitRow{Left: l.s, Right: r.s}
            if l.s != "" { row.LT = "del"; row.LN = l.n } else { row.LT = "empty" }
            if r.s != "" { row.RT = "add"; row.RN = r.n } else { row.RT = "empty" }
            rows = append(rows, row)
        }
        dels = nil
        adds = nil
    }
    // current hunk line numbers
    var lno, rno int
    for _, raw := range lines {
        if strings.HasPrefix(raw, "diff ") || strings.HasPrefix(raw, "index ") || strings.HasPrefix(raw, "--- ") || strings.HasPrefix(raw, "+++ ") {
            flush()
            if strings.TrimSpace(raw) == "" { continue }
            rows = append(rows, splitRow{Left: raw, Right: raw, LT: "meta", RT: "meta"})
            continue
        }
        if m := hunkRe.FindStringSubmatch(raw); m != nil {
            flush()
            // parse captures: -l,c +r,c
            lno = atoiSafe(m[1])
            rno = atoiSafe(m[3])
            rows = append(rows, splitRow{Left: raw, Right: raw, LT: "meta", RT: "meta"})
            continue
        }
        if raw == "" { // skip trailing empty
            continue
        }
        switch raw[0] {
        case ' ':
            flush()
            s := raw[1:]
            rows = append(rows, splitRow{Left: s, Right: s, LT: "ctx", RT: "ctx", LN: lno, RN: rno})
            lno++
            rno++
        case '-':
            dels = append(dels, lineNumText{n: lno, s: raw[1:]})
            lno++
        case '+':
            adds = append(adds, lineNumText{n: rno, s: raw[1:]})
            rno++
        default:
            flush()
            rows = append(rows, splitRow{Left: raw, Right: raw, LT: "meta", RT: "meta"})
        }
    }
    flush()
    return rows
}

func atoiSafe(s string) int {
    if s == "" { return 0 }
    n, _ := strconv.Atoi(s)
    return n
}
