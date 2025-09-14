package provider

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	tu "codectl/internal/testutil"
)

// withEnv removed (unused; use internal/testutil.WithEnv)

func TestLoadV2_DefaultWhenNoFiles(t *testing.T) {
	tmp := t.TempDir()
	defer tu.WithEnv(t, "HOME", tmp)()

	cfg, err := LoadV2()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(cfg.Providers) == 0 {
		t.Fatalf("expected default v2 providers, got: %+v", cfg)
	}
	if _, ok := cfg.Providers["ollama"]; !ok {
		t.Fatalf("expected default provider 'ollama' present")
	}
}

func TestSaveV2_AndModels(t *testing.T) {
	tmp := t.TempDir()
	defer tu.WithEnv(t, "HOME", tmp)()

	in := CatalogV2{Providers: map[string]Provider{
		"ollama": {Name: "Ollama", Type: "openai", Models: []Model{{ID: "b"}, {ID: "a"}, {ID: "a"}}},
		"openai": {Name: "OpenAI", Type: "openai", Models: []Model{{Name: "gpt"}}},
	}}
	if err := SaveV2(in); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	p, err := Path()
	if err != nil {
		t.Fatalf("Path error: %v", err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read json error: %v", err)
	}
	// ensure file is JSON (starts with '{')
	if strings.TrimSpace(string(b))[0] != '{' {
		t.Fatalf("expected JSON file, got: %s", string(b))
	}
	var got CatalogV2
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}

	// Flatten and verify normalization
	models := Models()
	if len(models) != 3 || models[0] != "a" || models[1] != "b" || models[2] != "gpt" {
		t.Fatalf("unexpected flattened models: %v", models)
	}
}

// No YAML migration test: YAML is not supported.

// contains removed (unused)
