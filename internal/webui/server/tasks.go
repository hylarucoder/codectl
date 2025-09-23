package server

import (
    "bufio"
    "encoding/json"
    "net/http"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strings"
)

type taskItem struct {
    Path     string            `json:"path"`
    Title    string            `json:"title,omitempty"`
    Status   string            `json:"status,omitempty"`
    Owner    string            `json:"owner,omitempty"`
    Priority string            `json:"priority,omitempty"`
    Due      string            `json:"due,omitempty"`
    Fields   map[string]string `json:"fields,omitempty"`
}

// GET /api/tasks/list?status=&owner=&priority=&q=
func tasksListHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet { w.WriteHeader(http.StatusMethodNotAllowed); return }
    base, err := resolveBase(r, "repo")
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    root := filepath.Join(base, "vibe-docs", "task")
    st, err := os.Stat(root)
    if err != nil || !st.IsDir() {
        writeJSON(w, http.StatusOK, []taskItem{})
        return
    }
    qStatus := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
    qOwner := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("owner")))
    qPri := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("priority")))
    q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))

    items := make([]taskItem, 0, 16)
    _ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
        if err != nil { return nil }
        if d.IsDir() { return nil }
        name := strings.ToLower(d.Name())
        if !strings.HasSuffix(name, ".task.mdx") { return nil }
        it := parseTaskMDX(p)
        it.Path = filepath.ToSlash(relSafe(root, p))
        // filters
        if qStatus != "" && strings.ToLower(it.Status) != qStatus { return nil }
        if qOwner != "" && strings.ToLower(it.Owner) != qOwner { return nil }
        if qPri != "" && strings.ToUpper(it.Priority) != qPri { return nil }
        if q != "" {
            if !(strings.Contains(strings.ToLower(it.Title), q) || strings.Contains(strings.ToLower(it.Path), q)) {
                return nil
            }
        }
        items = append(items, it)
        return nil
    })
    sort.Slice(items, func(i, j int) bool { return strings.ToLower(items[i].Path) < strings.ToLower(items[j].Path) })
    writeJSON(w, http.StatusOK, items)
}

func parseTaskMDX(p string) taskItem {
    b, err := os.ReadFile(p)
    if err != nil { return taskItem{Path: filepath.ToSlash(p)} }
    s := string(b)
    rd := bufio.NewReader(strings.NewReader(s))
    first, _ := rd.ReadString('\n')
    first = strings.TrimRight(first, "\r\n")
    if first != "---" {
        return taskItem{Path: filepath.ToSlash(p)}
    }
    lines := strings.Split(s, "\n")
    endIdx := -1
    for i := 1; i < len(lines); i++ {
        if strings.TrimRight(lines[i], "\r") == "---" { endIdx = i; break }
    }
    fm := map[string]string{}
    if endIdx > 1 {
        keyRe := regexp.MustCompile(`^([A-Za-z0-9_-]+)\s*:\s*(.*)$`)
        for _, ln := range lines[1:endIdx] {
            l := strings.TrimSpace(ln)
            if l == "" || strings.HasPrefix(l, "#") { continue }
            if m := keyRe.FindStringSubmatch(l); len(m) == 3 {
                k := strings.ToLower(strings.TrimSpace(m[1]))
                v := strings.TrimSpace(m[2])
                if len(v) >= 2 {
                    if (v[0] == '\'' && v[len(v)-1] == '\'') || (v[0] == '"' && v[len(v)-1] == '"') {
                        v = v[1:len(v)-1]
                    }
                }
                fm[k] = v
            }
        }
    }
    it := taskItem{Fields: fm}
    it.Title = fm["title"]
    it.Status = fm["status"]
    it.Owner = fm["owner"]
    it.Priority = fm["priority"]
    it.Due = fm["due"]
    return it
}

// PUT /api/tasks/update { path, content }
func tasksUpdateHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPut { w.WriteHeader(http.StatusMethodNotAllowed); return }
    var in struct{ Path, Content string }
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    base, err := resolveBase(r, "repo")
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    root := filepath.Join(base, "vibe-docs", "task")
    full, err := secureJoin(root, in.Path)
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil { writeJSON(w, http.StatusInternalServerError, errJSON(err)); return }
    if err := os.WriteFile(full, []byte(in.Content), 0o644); err != nil { writeJSON(w, http.StatusInternalServerError, errJSON(err)); return }
    writeJSON(w, http.StatusOK, parseTaskMDX(full))
}

