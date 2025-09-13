package store

import (
    "encoding/json"
    "errors"
    "os"
    "path/filepath"
    "sort"
    "strings"
)

// NormalizeStrings trims, deduplicates and sorts a slice of strings.
func NormalizeStrings(in []string) []string {
    m := map[string]struct{}{}
    for _, s := range in {
        s = strings.TrimSpace(s)
        if s == "" {
            continue
        }
        m[s] = struct{}{}
    }
    out := make([]string, 0, len(m))
    for k := range m {
        out = append(out, k)
    }
    sort.Strings(out)
    return out
}

// LoadStringList reads a JSON string array from path.
// Missing file yields an empty list without error. Output is normalized.
func LoadStringList(path string) ([]string, error) {
    b, err := os.ReadFile(path)
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
    return NormalizeStrings(arr), nil
}

// SaveStringList writes a JSON string array to path, creating parent dirs.
// Input is normalized before writing.
func SaveStringList(path string, list []string) error {
    if strings.TrimSpace(path) == "" {
        return errors.New("empty path")
    }
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        return err
    }
    arr := NormalizeStrings(list)
    b, err := json.MarshalIndent(arr, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, b, 0o644)
}

// AddToStringList loads the string list from path, adds items, saves, and returns sets.
func AddToStringList(path string, toAdd []string) (added []string, existed []string, err error) {
    cur, err := LoadStringList(path)
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
    if err := SaveStringList(path, next); err != nil {
        return nil, nil, err
    }
    sort.Strings(added)
    sort.Strings(existed)
    return added, existed, nil
}

// RemoveFromStringList loads, removes items, saves, and returns sets.
func RemoveFromStringList(path string, toRemove []string) (removed []string, missing []string, err error) {
    cur, err := LoadStringList(path)
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
    if err := SaveStringList(path, next); err != nil {
        return nil, nil, err
    }
    sort.Strings(removed)
    sort.Strings(missing)
    return removed, missing, nil
}

