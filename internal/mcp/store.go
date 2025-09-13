package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	cfg "codectl/internal/config"
)

// Server represents a configurable MCP server entry.
type Server struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// Catalog is the mapping of display name to server config.
type Catalog map[string]Server

func filePath() (string, error) {
	dir, err := cfg.DotDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mcp.json"), nil
}

// Load reads mcp.json in v2 map shape only.
func Load() (Catalog, error) {
	p, err := filePath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return Catalog{}, nil
		}
		return nil, err
	}
	var m Catalog
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return normalize(m), nil
}

// Save writes the catalog to mcp.json.
func Save(c Catalog) error {
	p, err := filePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	c = normalize(c)
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

// Add adds entries with default NPX launcher for each given name.
// If an entry exists, it's reported in existed.
func Add(names []string) (added []string, existed []string, err error) {
	c, err := Load()
	if err != nil {
		return nil, nil, err
	}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := c[n]; ok {
			existed = append(existed, n)
			continue
		}
		c[n] = Server{Command: "npx", Args: []string{"-y", n, "--stdio"}}
		added = append(added, n)
	}
	if err := Save(c); err != nil {
		return nil, nil, err
	}
	return added, existed, nil
}

// Remove deletes entries by name.
func Remove(names []string) (removed []string, missing []string, err error) {
	c, err := Load()
	if err != nil {
		return nil, nil, err
	}
	for _, n := range names {
		if _, ok := c[n]; ok {
			delete(c, n)
			removed = append(removed, n)
		} else {
			missing = append(missing, n)
		}
	}
	if err := Save(c); err != nil {
		return nil, nil, err
	}
	return removed, missing, nil
}

func normalize(in Catalog) Catalog {
	out := Catalog{}
	// Keys sorted not necessary for map, but we ensure stable content by ordering values in Marshal
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, strings.TrimSpace(k))
	}
	sort.Strings(keys)
	for _, k := range keys {
		if k == "" {
			continue
		}
		s := in[k]
		// minimal cleanup
		s.Command = strings.TrimSpace(s.Command)
		// remove empty args and trim
		aa := make([]string, 0, len(s.Args))
		for _, a := range s.Args {
			a = strings.TrimSpace(a)
			if a != "" {
				aa = append(aa, a)
			}
		}
		s.Args = aa
		out[k] = s
	}
	return out
}

// List names only (sorted) for simple display.
func Names() ([]string, error) {
	c, err := Load()
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

// ListRemote returns a placeholder list of discoverable MCP servers.
// In v2, provider.json no longer carries MCP; this is a static seed.
func ListRemote() []string { return []string{"figma-developer-mcp"} }
