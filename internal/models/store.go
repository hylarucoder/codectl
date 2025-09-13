package models

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"codectl/internal/provider"
)

// configDir returns the codectl-specific config directory.
func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(base) == "" {
		// Fallback to home directory if needed
		if home, herr := os.UserHomeDir(); herr == nil {
			base = home
		} else {
			return "", errors.New("cannot determine config directory")
		}
	}
	return filepath.Join(base, "codectl"), nil
}

// filePath returns the models storage file path.
func filePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "models.json"), nil
}

// Load returns the current model list from disk. Missing file yields empty list.
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
	// normalize and dedupe
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

// Save writes the model list to disk, creating the directory if needed.
func Save(models []string) error {
	p, err := filePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	// normalize and sort for stable output
	m := map[string]bool{}
	for _, s := range models {
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

// Add adds the given models to the store, returning which were added and which already existed.
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

// Remove removes the given models, returning which were removed and which were missing.
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

// ListRemote returns a static list of known remote models as a placeholder.
func ListRemote() []string {
	// Load from ~/.codectl/provider.yaml (with built-in fallback)
	return provider.Models()
}
