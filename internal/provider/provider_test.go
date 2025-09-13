package provider

import (
    "encoding/json"
    "os"
    "strings"
    "testing"
)

func withEnv(t *testing.T, key, val string) func() {
    t.Helper()
    old, had := os.LookupEnv(key)
    if val == "" {
        _ = os.Unsetenv(key)
    } else {
        _ = os.Setenv(key, val)
    }
    return func() {
        if had {
            _ = os.Setenv(key, old)
        } else {
            _ = os.Unsetenv(key)
        }
    }
}

func TestLoad_DefaultWhenNoFiles(t *testing.T) {
    tmp := t.TempDir()
    defer withEnv(t, "HOME", tmp)()

    cfg, err := Load()
    if err != nil {
        t.Fatalf("Load error: %v", err)
    }
    if len(cfg.Models) == 0 || len(cfg.MCP) == 0 {
        t.Fatalf("expected defaultCatalog fallback, got: %+v", cfg)
    }
    // Quick sanity: known defaults present
    if !contains(cfg.Models, "kimi-k2-0905-preview") {
        t.Fatalf("expected default model present, got: %v", cfg.Models)
    }
}

func TestSaveAndLoad_JSON(t *testing.T) {
    tmp := t.TempDir()
    defer withEnv(t, "HOME", tmp)()

    in := Catalog{Models: []string{"b", "a", "a"}, MCP: []string{"x", "y", "x"}}
    if err := Save(in); err != nil {
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
    var got Catalog
    if err := json.Unmarshal(b, &got); err != nil {
        t.Fatalf("json unmarshal error: %v", err)
    }

    // Load and verify normalization
    cfg, err := Load()
    if err != nil {
        t.Fatalf("Load error: %v", err)
    }
    if len(cfg.Models) != 2 || cfg.Models[0] != "a" || cfg.Models[1] != "b" {
        t.Fatalf("unexpected models: %v", cfg.Models)
    }
    if len(cfg.MCP) != 2 || cfg.MCP[0] != "x" || cfg.MCP[1] != "y" {
        t.Fatalf("unexpected mcp: %v", cfg.MCP)
    }
}

// No YAML migration test: YAML is not supported.

func contains(arr []string, s string) bool {
    for _, v := range arr {
        if v == s { return true }
    }
    return false
}
