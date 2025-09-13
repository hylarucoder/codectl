package cli

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"

    "github.com/spf13/cobra"

    "codectl/internal/system"
)

type checkItem struct {
    Path          string            `json:"path"`
    HasFrontmatter bool             `json:"hasFrontmatter"`
    Fields        map[string]string `json:"fields,omitempty"`
    Errors        []string          `json:"errors,omitempty"`
    Warnings      []string          `json:"warnings,omitempty"`
}

type checkReport struct {
    Root     string      `json:"root"`
    Dirs     []string    `json:"dirs"`
    Items    []checkItem `json:"items"`
    Errors   int         `json:"errors"`
    Warnings int         `json:"warnings"`
}

var (
    checkJSON bool
)

func init() {
    rootCmd.AddCommand(checkCmd)
    checkCmd.Flags().BoolVar(&checkJSON, "json", false, "output JSON report")
}

var checkCmd = &cobra.Command{
    Use:   "check",
    Short: "Check MDX frontmatter under vibe-docs/spec (*.spec.mdx)",
    RunE: func(cmd *cobra.Command, args []string) error {
        // determine repo root (fallback to CWD)
        cwd, _ := os.Getwd()
        root := cwd
        if giRoot, err := system.GitRoot(cmd.Context(), cwd); err == nil && strings.TrimSpace(giRoot) != "" {
            root = giRoot
        }
        dirs := []string{
            filepath.Join(root, "vibe-docs", "spec"),
        }
        rep := checkReport{Root: root, Dirs: make([]string, 0, len(dirs))}
        for _, d := range dirs {
            if st, err := os.Stat(d); err != nil || !st.IsDir() {
                // skip silently if missing
                continue
            }
            rep.Dirs = append(rep.Dirs, d)
            // walk
            filepath.WalkDir(d, func(path string, de os.DirEntry, err error) error {
                if err != nil {
                    rep.Items = append(rep.Items, checkItem{Path: path, Errors: []string{err.Error()}})
                    rep.Errors++
                    return nil
                }
                if de.IsDir() {
                    return nil
                }
                name := strings.ToLower(de.Name())
                if !strings.HasSuffix(name, ".spec.mdx") { // only spec docs
                    return nil
                }
                it := checkMDX(path)
                if len(it.Errors) > 0 {
                    rep.Errors += len(it.Errors)
                }
                if len(it.Warnings) > 0 {
                    rep.Warnings += len(it.Warnings)
                }
                rep.Items = append(rep.Items, it)
                return nil
            })
        }

        if checkJSON {
            // pretty JSON to stdout
            enc := json.NewEncoder(os.Stdout)
            enc.SetIndent("", "  ")
            if err := enc.Encode(rep); err != nil {
                return err
            }
        } else {
            // text summary
            for _, it := range rep.Items {
                if len(it.Errors) > 0 {
                    fmt.Printf("ERR  %s  %s\n", relFrom(root, it.Path), strings.Join(it.Errors, "; "))
                    continue
                }
                if len(it.Warnings) > 0 {
                    fmt.Printf("WARN %s  %s\n", relFrom(root, it.Path), strings.Join(it.Warnings, "; "))
                } else {
                    fmt.Printf("OK   %s\n", relFrom(root, it.Path))
                }
            }
            fmt.Printf("\nSummary: %d file(s), %d error(s), %d warning(s)\n", len(rep.Items), rep.Errors, rep.Warnings)
        }

        if rep.Errors > 0 {
            // non-zero when any error
            return fmt.Errorf("check failed: %d error(s)", rep.Errors)
        }
        return nil
    },
}

func relFrom(root, p string) string {
    if r, err := filepath.Rel(root, p); err == nil {
        return r
    }
    return p
}

// parse MDX frontmatter and validate required keys
func checkMDX(path string) checkItem {
    it := checkItem{Path: path}
    b, err := os.ReadFile(path)
    if err != nil {
        it.Errors = append(it.Errors, err.Error())
        return it
    }
    s := string(b)
    // Support files without BOM only; trim leading whitespace to be lenient
    trimmed := s
    // but ensure '---' is the very first non-empty line
    rd := bufio.NewReader(strings.NewReader(s))
    firstLine, _ := rd.ReadString('\n')
    firstLine = strings.TrimRight(firstLine, "\r\n")
    if firstLine != "---" {
        it.Errors = append(it.Errors, "missing frontmatter start '---' on first line")
        return it
    }
    // find closing '---' on its own line
    // We'll scan lines until the next '---'
    lines := strings.Split(trimmed, "\n")
    endIdx := -1
    for i := 1; i < len(lines); i++ {
        if strings.TrimRight(lines[i], "\r") == "---" { // end delimiter
            endIdx = i
            break
        }
    }
    if endIdx < 0 {
        it.Errors = append(it.Errors, "missing frontmatter end '---'")
        return it
    }
    it.HasFrontmatter = true
    fmLines := lines[1:endIdx]
    fields := make(map[string]string)
    keyRe := regexp.MustCompile(`^([A-Za-z0-9_-]+)\s*:\s*(.*)$`)
    for _, ln := range fmLines {
        l := strings.TrimSpace(ln)
        if l == "" || strings.HasPrefix(l, "#") {
            continue
        }
        if m := keyRe.FindStringSubmatch(l); len(m) == 3 {
            key := strings.ToLower(strings.TrimSpace(m[1]))
            val := strings.TrimSpace(m[2])
            // strip surrounding quotes if present
            if len(val) >= 2 {
                if (val[0] == '\'' && val[len(val)-1] == '\'') || (val[0] == '"' && val[len(val)-1] == '"') {
                    val = val[1 : len(val)-1]
                }
            }
            fields[key] = val
        }
    }
    it.Fields = fields
    // minimal required: title must exist and be non-empty
    if strings.TrimSpace(fields["title"]) == "" {
        it.Errors = append(it.Errors, "missing required field 'title'")
    }
    // heuristics: if filename ends with .spec.mdx, recommend specVersion
    if strings.HasSuffix(strings.ToLower(filepath.Base(path)), ".spec.mdx") {
        if strings.TrimSpace(fields["specversion"]) == "" {
            it.Warnings = append(it.Warnings, "recommended field 'specVersion' is missing")
        }
    }
    return it
}
