package provider

import (
    "encoding/json"
    "errors"
    "os"
    "path/filepath"
    "sort"
    "strings"
)

// Catalog represents a simple provider catalog file.
// Default JSON path: ~/.codectl/provider.json
type Catalog struct {
    Models []string `json:"models"`
    MCP    []string `json:"mcp"`
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

// Path returns the JSON catalog path (~/.codectl/provider.json).
func Path() (string, error) { return pathJSON() }

// pathJSON returns ~/.codectl/provider.json
func pathJSON() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil || strings.TrimSpace(home) == "" {
        return "", errors.New("cannot determine user home directory")
    }
    return filepath.Join(home, ".codectl", "provider.json"), nil
}

//

// Load reads the catalog from ~/.codectl/provider.json.
// If the file does not exist, returns defaultCatalog and no error.
func Load() (Catalog, error) {
    p, err := pathJSON()
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
    if err := json.Unmarshal(b, &cfg); err != nil {
        return defaultCatalog, err
    }
    cfg = normalizeCatalog(cfg)
    if len(cfg.Models) == 0 {
        cfg.Models = append([]string(nil), defaultCatalog.Models...)
    }
    if len(cfg.MCP) == 0 {
        cfg.MCP = append([]string(nil), defaultCatalog.MCP...)
    }
    return cfg, nil
}

func normalizeCatalog(cfg Catalog) Catalog {
    cfg.Models = normalizeList(cfg.Models)
    cfg.MCP = normalizeList(cfg.MCP)
    return cfg
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

// Save writes the catalog to ~/.codectl/provider.json, creating parent dirs as needed.
func Save(c Catalog) error {
    // normalize and sort
    c = normalizeCatalog(c)

    p, err := pathJSON()
    if err != nil {
        return err
    }
    if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
        return err
    }
    b, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(p, b, 0o644)
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
