package server

import (
    "bufio"
    "encoding/json"
    "errors"
    "net/http"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strings"
)

type specDocMeta struct {
    Path     string            `json:"path"`
    Title    string            `json:"title,omitempty"`
    Status   string            `json:"status,omitempty"`
    Fields   map[string]string `json:"fields,omitempty"`
    Errors   []string          `json:"errors,omitempty"`
    Warnings []string          `json:"warnings,omitempty"`
}

func specListHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet { w.WriteHeader(http.StatusMethodNotAllowed); return }
    // base fixed to vibe-spec for listing
    base, err := resolveBase(r, "vibe-spec")
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    entries := make([]specDocMeta, 0, 16)
    _ = filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
        if err != nil { return nil }
        if d.IsDir() { return nil }
        name := strings.ToLower(d.Name())
        if !strings.HasSuffix(name, ".spec.mdx") { return nil }
        it := checkMDXFile(p)
        it.Path = filepath.ToSlash(relSafe(base, p))
        if it.Fields != nil {
            it.Title = it.Fields["title"]
            it.Status = it.Fields["status"]
        }
        entries = append(entries, it)
        return nil
    })
    sort.Slice(entries, func(i, j int) bool { return strings.ToLower(entries[i].Path) < strings.ToLower(entries[j].Path) })
    writeJSON(w, http.StatusOK, entries)
}

func specDocHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        base, err := resolveBase(r, r.URL.Query().Get("base"))
        if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
        p := r.URL.Query().Get("path")
        if strings.TrimSpace(p) == "" { writeJSON(w, http.StatusBadRequest, errJSON(errors.New("missing path"))); return }
        full, err := secureJoin(base, p)
        if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
        b, err := os.ReadFile(full)
        if err != nil { writeJSON(w, http.StatusNotFound, errJSON(err)); return }
        it := checkMDXBytes(b)
        it.Path = filepath.ToSlash(p)
        writeJSON(w, http.StatusOK, map[string]any{
            "path": it.Path,
            "fields": it.Fields,
            "errors": it.Errors,
            "warnings": it.Warnings,
            "content": string(b),
        })
    case http.MethodPut:
        var in struct{
            Base    string            `json:"base"`
            Path    string            `json:"path"`
            Content string            `json:"content"`
            // For MVP we ignore separate frontmatter updates and accept full content text
            Front   map[string]string `json:"frontmatter"`
        }
        if err := json.NewDecoder(r.Body).Decode(&in); err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
        base, err := resolveBase(r, in.Base)
        if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
        full, err := secureJoin(base, in.Path)
        if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
        if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil { writeJSON(w, http.StatusInternalServerError, errJSON(err)); return }
        if err := os.WriteFile(full, []byte(in.Content), 0o644); err != nil { writeJSON(w, http.StatusInternalServerError, errJSON(err)); return }
        it := checkMDXFile(full)
        writeJSON(w, http.StatusOK, it)
    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
}

func specValidateHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost { w.WriteHeader(http.StatusMethodNotAllowed); return }
    var in struct{
        Base    string `json:"base"`
        Path    string `json:"path"`
        Content string `json:"content"`
    }
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    if strings.TrimSpace(in.Content) == "" && strings.TrimSpace(in.Path) == "" {
        writeJSON(w, http.StatusBadRequest, errJSON(errors.New("must provide content or path")))
        return
    }
    if strings.TrimSpace(in.Content) != "" {
        it := checkMDXBytes([]byte(in.Content))
        writeJSON(w, http.StatusOK, it)
        return
    }
    base, err := resolveBase(r, in.Base)
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    full, err := secureJoin(base, in.Path)
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    it := checkMDXFile(full)
    writeJSON(w, http.StatusOK, it)
}

// checkMDXFile parses an MDX file's frontmatter with minimal checks.
func checkMDXFile(p string) specDocMeta {
    b, err := os.ReadFile(p)
    if err != nil { return specDocMeta{Path: p, Errors: []string{err.Error()}} }
    return checkMDXBytes(b)
}

func checkMDXBytes(b []byte) specDocMeta {
    it := specDocMeta{}
    s := string(b)
    rd := bufio.NewReader(strings.NewReader(s))
    first, _ := rd.ReadString('\n')
    first = strings.TrimRight(first, "\r\n")
    if first != "---" {
        it.Errors = append(it.Errors, "missing frontmatter start '---' on first line")
        return it
    }
    lines := strings.Split(s, "\n")
    endIdx := -1
    for i := 1; i < len(lines); i++ {
        if strings.TrimRight(lines[i], "\r") == "---" { endIdx = i; break }
    }
    if endIdx < 0 {
        it.Errors = append(it.Errors, "missing frontmatter end '---'")
        return it
    }
    it.Fields = map[string]string{}
    fm := lines[1:endIdx]
    keyRe := regexp.MustCompile(`^([A-Za-z0-9_-]+)\s*:\s*(.*)$`)
    for _, ln := range fm {
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
            it.Fields[k] = v
        }
    }
    if strings.TrimSpace(it.Fields["title"]) == "" {
        it.Errors = append(it.Errors, "missing required field 'title'")
    }
    // Heuristic: recommend specVersion for .spec.mdx
    it.Warnings = []string{}
    // (The caller knows path; here we can't reliably infer suffix)
    if _, ok := it.Fields["specversion"]; !ok {
        it.Warnings = append(it.Warnings, "recommended field 'specVersion' is missing")
    }
    return it
}
