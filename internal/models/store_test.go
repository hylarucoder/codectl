package models

import (
	"testing"

	tu "codectl/internal/testutil"
)

// withEnv removed (unused; use internal/testutil.WithEnv)

func TestModels_SaveLoad_AddRemove(t *testing.T) {
	tmp := t.TempDir()
	// direct UserConfigDir to temp
	defer tu.WithEnv(t, "XDG_CONFIG_HOME", tmp)()
	defer tu.WithEnv(t, "HOME", tmp)() // fallback

	// initial load -> empty
	got, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty models, got %v", got)
	}

	// save + load normalization
	if err := Save([]string{"b", "a", "a"}); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	got, err = Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected models after save+load: %v", got)
	}

	// Add with duplicates in input
	added, existed, err := Add([]string{"m1", "m2", "m1"})
	if err != nil {
		t.Fatalf("Add error: %v", err)
	}
	// order not guaranteed before sort; check membership
	if !hasAll(added, []string{"m1", "m2"}) {
		t.Fatalf("unexpected added: %v", added)
	}
	if !hasAll(existed, []string{"m1"}) {
		t.Fatalf("unexpected existed: %v", existed)
	}

	// Remove one present and one missing
	removed, missing, err := Remove([]string{"m1", "m3"})
	if err != nil {
		t.Fatalf("Remove error: %v", err)
	}
	if !hasAll(removed, []string{"m1"}) || !hasAll(missing, []string{"m3"}) {
		t.Fatalf("unexpected removed/missing: %v / %v", removed, missing)
	}

	// final store should contain m2 plus a,b from earlier
	final, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if !hasAll(final, []string{"a", "b", "m2"}) || len(final) != 3 {
		t.Fatalf("unexpected final models: %v", final)
	}

	// sanity: Load succeeds after operations
	if _, err := Load(); err != nil {
		t.Fatalf("final Load error: %v", err)
	}
}

func hasAll(set []string, want []string) bool {
	m := map[string]bool{}
	for _, s := range set {
		m[s] = true
	}
	for _, w := range want {
		if !m[w] {
			return false
		}
	}
	return true
}
