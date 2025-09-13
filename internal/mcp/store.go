package mcp

import (
    "encoding/json"
    "os"
    "path/filepath"
    "sort"
    "strings"

    cfg "codectl/internal/config"
    "codectl/internal/provider"
)

func filePath() (string, error) {
    dir, err := cfg.Dir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dir, "mcp.json"), nil
}

// Load returns configured MCP servers.
func Load() ([]string, error) {
	p, err := filePath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var arr []string
	if err := json.Unmarshal(b, &arr); err != nil {
		return nil, err
	}
	m := map[string]bool{}
	for _, s := range arr {
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
	return out, nil
}

// Save writes configured MCP servers.
func Save(list []string) error {
	p, err := filePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	m := map[string]bool{}
	for _, s := range list {
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
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

// ListRemote returns a placeholder list of known MCP servers.
func ListRemote() []string { return provider.MCPServers() }

// Add adds given servers to the MCP list.
func Add(toAdd []string) (added []string, existed []string, err error) {
	cur, err := Load()
	if err != nil {
		return nil, nil, err
	}
	set := map[string]bool{}
	for _, s := range cur {
		set[s] = true
	}
	for _, s := range toAdd {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if set[s] {
			existed = append(existed, s)
		} else {
			set[s] = true
			added = append(added, s)
		}
	}
	next := make([]string, 0, len(set))
	for k := range set {
		next = append(next, k)
	}
	if err := Save(next); err != nil {
		return nil, nil, err
	}
	sort.Strings(added)
	sort.Strings(existed)
	return added, existed, nil
}

// Remove removes given servers from the MCP list.
func Remove(toRemove []string) (removed []string, missing []string, err error) {
	cur, err := Load()
	if err != nil {
		return nil, nil, err
	}
	set := map[string]bool{}
	for _, s := range cur {
		set[s] = true
	}
	for _, s := range toRemove {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if set[s] {
			delete(set, s)
			removed = append(removed, s)
		} else {
			missing = append(missing, s)
		}
	}
	next := make([]string, 0, len(set))
	for k := range set {
		next = append(next, k)
	}
	if err := Save(next); err != nil {
		return nil, nil, err
	}
	sort.Strings(removed)
	sort.Strings(missing)
	return removed, missing, nil
}
