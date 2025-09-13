package mcp

import (
    "os"
    "testing"
)

func withEnv(t *testing.T, key, val string) func() {
    t.Helper()
    old, had := os.LookupEnv(key)
    if val == "" { _ = os.Unsetenv(key) } else { _ = os.Setenv(key, val) }
    return func() { if had { _ = os.Setenv(key, old) } else { _ = os.Unsetenv(key) } }
}

func TestMCP_SaveLoad_AddRemove(t *testing.T) {
    tmp := t.TempDir()
    defer withEnv(t, "XDG_CONFIG_HOME", tmp)()
    defer withEnv(t, "HOME", tmp)()

    // initial load
    got, err := Load()
    if err != nil { t.Fatalf("Load error: %v", err) }
    if len(got) != 0 { t.Fatalf("expected empty mcp list, got %v", got) }

    if err := Save([]string{"b", "a", "a"}); err != nil { t.Fatalf("Save error: %v", err) }
    got, err = Load()
    if err != nil { t.Fatalf("Load error: %v", err) }
    if len(got) != 2 || got[0] != "a" || got[1] != "b" {
        t.Fatalf("unexpected mcp after save+load: %v", got)
    }

    added, existed, err := Add([]string{"s1", "s2", "s1"})
    if err != nil { t.Fatalf("Add error: %v", err) }
    if !hasAll(added, []string{"s1", "s2"}) { t.Fatalf("unexpected added: %v", added) }
    if !hasAll(existed, []string{"s1"}) { t.Fatalf("unexpected existed: %v", existed) }

    removed, missing, err := Remove([]string{"s1", "s3"})
    if err != nil { t.Fatalf("Remove error: %v", err) }
    if !hasAll(removed, []string{"s1"}) || !hasAll(missing, []string{"s3"}) {
        t.Fatalf("unexpected removed/missing: %v / %v", removed, missing)
    }

    final, err := Load()
    if err != nil { t.Fatalf("Load error: %v", err) }
    if !hasAll(final, []string{"a", "b", "s2"}) || len(final) != 3 {
        t.Fatalf("unexpected final mcp list: %v", final)
    }

    // sanity: Load succeeds after operations
    if _, err := Load(); err != nil {
        t.Fatalf("final Load error: %v", err)
    }
}

func hasAll(set []string, want []string) bool {
    m := map[string]bool{}
    for _, s := range set { m[s] = true }
    for _, w := range want { if !m[w] { return false } }
    return true
}
