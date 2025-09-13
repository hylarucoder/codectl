package provider

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Default JSON path: ~/.codectl/provider.json

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

// ========== V2 Catalog (providers map) ==========

// Model describes a model entry under a provider in v2.
type Model struct {
	Name             string `json:"name,omitempty"`
	ID               string `json:"id,omitempty"`
	ContextWindow    int    `json:"context_window,omitempty"`
	DefaultMaxTokens int    `json:"default_max_tokens,omitempty"`
}

// Provider describes a single provider entry in v2.
type Provider struct {
	Name    string  `json:"name,omitempty"`
	BaseURL string  `json:"base_url,omitempty"`
	Type    string  `json:"type,omitempty"`
	Models  []Model `json:"models,omitempty"`
}

// CatalogV2 represents the v2 shape: top-level providers (arbitrary keys) plus optional MCP list.
// We implement custom marshal/unmarshal to keep the top-level as a single object.
type CatalogV2 struct {
	Providers map[string]Provider `json:"-"`
}

func (c *CatalogV2) UnmarshalJSON(b []byte) error {
	type rawObj = map[string]json.RawMessage
	var root rawObj
	if err := json.Unmarshal(b, &root); err != nil {
		return err
	}
	c.Providers = map[string]Provider{}
	for k, v := range root {
		if k == "mcp" { // legacy: ignore in v2
			continue
		}
		var p Provider
		if err := json.Unmarshal(v, &p); err != nil {
			// skip invalid provider entries
			continue
		}
		c.Providers[k] = p
	}
	c.normalize()
	return nil
}

func (c CatalogV2) MarshalJSON() ([]byte, error) {
	// Build an ordered map to keep stable output
	out := map[string]any{}
	// sort provider keys
	keys := make([]string, 0, len(c.Providers))
	for k := range c.Providers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		// normalize per provider models
		p := c.Providers[k]
		// de-dup models by ID, prefer ID then Name
		seen := map[string]struct{}{}
		cleaned := make([]Model, 0, len(p.Models))
		for _, m := range p.Models {
			id := strings.TrimSpace(m.ID)
			name := strings.TrimSpace(m.Name)
			if id == "" && name == "" {
				continue
			}
			key := id
			if key == "" {
				key = name
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			m.ID = id
			m.Name = name
			cleaned = append(cleaned, m)
		}
		p.Models = cleaned
		out[k] = p
	}
	return json.MarshalIndent(out, "", "  ")
}

func (c *CatalogV2) normalize() {
	// normalize providers models (de-dup IDs)
	for k, p := range c.Providers {
		seen := map[string]struct{}{}
		cleaned := make([]Model, 0, len(p.Models))
		for _, m := range p.Models {
			id := strings.TrimSpace(m.ID)
			name := strings.TrimSpace(m.Name)
			if id == "" && name == "" {
				continue
			}
			key := id
			if key == "" {
				key = name
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			m.ID = id
			m.Name = name
			cleaned = append(cleaned, m)
		}
		p.Models = cleaned
		c.Providers[k] = p
	}
}

// DefaultV2 returns a minimal v2 catalog skeleton.
func DefaultV2() CatalogV2 {
	return CatalogV2{
		Providers: map[string]Provider{
			"ollama": {Name: "Ollama", Type: "openai", BaseURL: "http://localhost:11434/v1/", Models: []Model{}},
		},
	}
}

// LoadV2 reads v2 catalog; on missing file returns DefaultV2.
func LoadV2() (CatalogV2, error) {
	p, err := pathJSON()
	if err != nil {
		return DefaultV2(), err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultV2(), nil
		}
		return DefaultV2(), err
	}
	var cfg CatalogV2
	if err := json.Unmarshal(b, &cfg); err != nil {
		return DefaultV2(), err
	}
	return cfg, nil
}

// SaveV2 writes v2 catalog to ~/.codectl/provider.json.
func SaveV2(c CatalogV2) error {
	c.normalize()
	p, err := pathJSON()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(c) // indent handled by MarshalJSON
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
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

// Models returns the flattened remote model identifiers from provider.json (v2).
func Models() []string {
	cfg, err := LoadV2()
	if err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, 16)
	for _, p := range cfg.Providers {
		for _, m := range p.Models {
			id := strings.TrimSpace(m.ID)
			if id == "" {
				id = strings.TrimSpace(m.Name)
			}
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return normalizeList(out)
}
