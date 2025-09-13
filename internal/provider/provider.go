package provider

import (
    "errors"
    "os"
    "path/filepath"
    "sort"
    "strings"

    "gopkg.in/yaml.v3"
)

// Catalog represents a simple provider catalog file.
// Path: ~/.codectl/provider.yaml
type Catalog struct {
    Models []string `yaml:"models"`
    MCP    []string `yaml:"mcp"`
}

// defaultCatalog holds a minimal built-in fallback.
var defaultCatalog = Catalog{
    Models: []string{
        "kimi-k2-0905-preview",
        "kimi-k2-0711-preview",
    },
    MCP: []string{
        "figma-developer-mcp",
    },
}

// path returns ~/.codectl/provider.yaml
func path() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil || strings.TrimSpace(home) == "" {
        return "", errors.New("cannot determine user home directory")
    }
    return filepath.Join(home, ".codectl", "provider.yaml"), nil
}

// Load reads the catalog from ~/.codectl/provider.yaml.
// If the file does not exist, returns defaultCatalog and no error.
func Load() (Catalog, error) {
    p, err := path()
    if err != nil {
        return defaultCatalog, err
    }
    b, err := os.ReadFile(p)
    if err != nil {
        if os.IsNotExist(err) {
            return defaultCatalog, nil
        }
        return defaultCatalog, err
    }
    var cfg Catalog
    if err := yaml.Unmarshal(b, &cfg); err != nil {
        return defaultCatalog, err
    }
    // normalize
    cfg.Models = normalizeList(cfg.Models)
    cfg.MCP = normalizeList(cfg.MCP)
    // if lists empty, fallback to default for that section
    if len(cfg.Models) == 0 {
        cfg.Models = append([]string(nil), defaultCatalog.Models...)
    }
    if len(cfg.MCP) == 0 {
        cfg.MCP = append([]string(nil), defaultCatalog.MCP...)
    }
    return cfg, nil
}

func normalizeList(in []string) []string {
    m := map[string]bool{}
    for _, s := range in {
        s = strings.TrimSpace(s)
        if s == "" {
            continue
        }
        m[s] = true
    }
    out := make([]string, 0, len(m))
    for k := range m {
        out = append(out, k)
    }
    sort.Strings(out)
    return out
}

// Models returns the remote models list.
func Models() []string {
    c, _ := Load()
    return c.Models
}

// MCPServers returns the remote MCP servers list.
func MCPServers() []string {
    c, _ := Load()
    return c.MCP
}

