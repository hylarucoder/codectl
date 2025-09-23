package server

import (
    "encoding/json"
    "errors"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "strconv"
    sys "codectl/internal/system"
)

// Allowed base roots to limit FS access surface.
// "repo": repository root if available, else CWD
// "vibe-spec": repo/vibe-docs/spec
var allowedBases = []string{"repo", "vibe-spec"}

type fsTreeNode struct {
    Path     string        `json:"path"`
    Name     string        `json:"name"`
    Dir      bool          `json:"dir"`
    Children []fsTreeNode  `json:"children,omitempty"`
}

// resolveBase returns the absolute path for a base key.
func resolveBase(r *http.Request, base string) (string, error) {
    cwd, _ := os.Getwd()
    root := cwd
    if gi, err := sys.GitRoot(r.Context(), cwd); err == nil && strings.TrimSpace(gi) != "" {
        root = gi
    }
    switch base {
    case "repo", "":
        return root, nil
    case "vibe-spec":
        return filepath.Join(root, "vibe-docs", "spec"), nil
    default:
        return "", errors.New("invalid base")
    }
}

// secureJoin joins base and p and ensures the result stays within base.
func secureJoin(base, p string) (string, error) {
    // Clean and prevent absolute paths
    if filepath.IsAbs(p) {
        return "", errors.New("absolute path not allowed")
    }
    clean := filepath.Clean(p)
    full := filepath.Join(base, clean)
    // Resolve symlinks best-effort
    baseEval, _ := filepath.EvalSymlinks(base)
    fullEval, _ := filepath.EvalSymlinks(full)
    if baseEval == "" { baseEval = base }
    if fullEval == "" { fullEval = full }
    // Ensure prefix
    rel, err := filepath.Rel(baseEval, fullEval)
    if err != nil || strings.HasPrefix(rel, "..") || strings.Contains(rel, string(os.PathSeparator)+"..") {
        return "", errors.New("path escapes base")
    }
    return full, nil
}

func fsTreeHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet { w.WriteHeader(http.StatusMethodNotAllowed); return }
    baseKey := r.URL.Query().Get("base")
    depth := 2
    if v := strings.TrimSpace(r.URL.Query().Get("depth")); v != "" {
        if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 8 { depth = n }
    }
    base, err := resolveBase(r, baseKey)
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    st, err := os.Stat(base)
    if err != nil || !st.IsDir() { writeJSON(w, http.StatusNotFound, errJSON(errors.New("base not found"))); return }
    // Build shallow tree
    node, err := buildTree(base, base, depth)
    if err != nil { writeJSON(w, http.StatusInternalServerError, errJSON(err)); return }
    writeJSON(w, http.StatusOK, node)
}

func buildTree(root, dir string, depth int) (fsTreeNode, error) {
    node := fsTreeNode{Path: relSafe(root, dir), Name: filepath.Base(dir), Dir: true}
    if depth <= 0 { return node, nil }
    ents, err := os.ReadDir(dir)
    if err != nil { return node, err }
    // sort by name, dirs first
    sort.Slice(ents, func(i, j int) bool {
        if ents[i].IsDir() != ents[j].IsDir() { return ents[i].IsDir() }
        return strings.ToLower(ents[i].Name()) < strings.ToLower(ents[j].Name())
    })
    for _, e := range ents {
        name := e.Name()
        if strings.HasPrefix(name, ".") { continue } // skip dotfiles
        p := filepath.Join(dir, name)
        if e.IsDir() {
            child, err := buildTree(root, p, depth-1)
            if err != nil { continue }
            node.Children = append(node.Children, child)
        } else {
            node.Children = append(node.Children, fsTreeNode{Path: relSafe(root, p), Name: name, Dir: false})
        }
    }
    return node, nil
}

func relSafe(root, p string) string {
    if r, err := filepath.Rel(root, p); err == nil { return filepath.ToSlash(r) }
    return filepath.ToSlash(p)
}

func fsReadHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet { w.WriteHeader(http.StatusMethodNotAllowed); return }
    base, err := resolveBase(r, r.URL.Query().Get("base"))
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    p := r.URL.Query().Get("path")
    if strings.TrimSpace(p) == "" { writeJSON(w, http.StatusBadRequest, errJSON(errors.New("missing path"))); return }
    full, err := secureJoin(base, p)
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    b, err := os.ReadFile(full)
    if err != nil { writeJSON(w, http.StatusNotFound, errJSON(err)); return }
    writeJSON(w, http.StatusOK, map[string]any{"path": filepath.ToSlash(p), "content": string(b)})
}

func fsWriteHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPut { w.WriteHeader(http.StatusMethodNotAllowed); return }
    var in struct{
        Base    string `json:"base"`
        Path    string `json:"path"`
        Content string `json:"content"`
        Create  bool   `json:"create"`
    }
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    base, err := resolveBase(r, in.Base)
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    full, err := secureJoin(base, in.Path)
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil { writeJSON(w, http.StatusInternalServerError, errJSON(err)); return }
    if !in.Create {
        if st, err := os.Stat(full); err != nil || st.IsDir() {
            writeJSON(w, http.StatusNotFound, errJSON(errors.New("file not found")))
            return
        }
    }
    if err := os.WriteFile(full, []byte(in.Content), 0o644); err != nil { writeJSON(w, http.StatusInternalServerError, errJSON(err)); return }
    writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func fsRenameHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost { w.WriteHeader(http.StatusMethodNotAllowed); return }
    var in struct{
        Base   string `json:"base"`
        Path   string `json:"path"`
        NewPath string `json:"newPath"`
    }
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    base, err := resolveBase(r, in.Base)
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    src, err := secureJoin(base, in.Path)
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    dst, err := secureJoin(base, in.NewPath)
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil { writeJSON(w, http.StatusInternalServerError, errJSON(err)); return }
    if err := os.Rename(src, dst); err != nil { writeJSON(w, http.StatusInternalServerError, errJSON(err)); return }
    writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func fsDeleteHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost { w.WriteHeader(http.StatusMethodNotAllowed); return }
    var in struct{
        Base string `json:"base"`
        Path string `json:"path"`
    }
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    base, err := resolveBase(r, in.Base)
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    full, err := secureJoin(base, in.Path)
    if err != nil { writeJSON(w, http.StatusBadRequest, errJSON(err)); return }
    if err := os.Remove(full); err != nil { writeJSON(w, http.StatusInternalServerError, errJSON(err)); return }
    writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func fsPatchHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost { w.WriteHeader(http.StatusMethodNotAllowed); return }
    // MVP: not implemented; reserve endpoint
    // Accept body and return 501
    _, _ = io.Copy(io.Discard, r.Body)
    w.WriteHeader(http.StatusNotImplemented)
}
