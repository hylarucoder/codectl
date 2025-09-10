package tools

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
)

// NpmGlobalVersion queries npm for globally installed package version.
func NpmGlobalVersion(ctx context.Context, pkg string) (string, error) {
    out, err := runCmd(ctx, "npm", "ls", "-g", "--depth=0", pkg, "--json")
    if err != nil && out == "" {
        return "", err
    }
    var data struct {
        Dependencies map[string]struct {
            Version string `json:"version"`
        } `json:"dependencies"`
    }
    if err := json.Unmarshal([]byte(out), &data); err != nil {
        return "", err
    }
    if d, ok := data.Dependencies[pkg]; ok {
        return d.Version, nil
    }
    return "", fmt.Errorf("package not found: %s", pkg)
}

// NpmLatestVersion queries npm registry for latest dist-tag ("version")
func NpmLatestVersion(ctx context.Context, pkg string) (string, error) {
    out, err := runCmd(ctx, "npm", "view", pkg, "version", "--json")
    if err != nil && out == "" {
        return "", err
    }
    s := strings.TrimSpace(out)
    // npm may return a bare JSON string like "1.2.3" or plain 1.2.3
    if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
        return strings.Trim(s, "\""), nil
    }
    // Try parse JSON
    var v string
    if json.Unmarshal([]byte(s), &v) == nil && v != "" {
        return v, nil
    }
    // Fallback: first line
    return strings.Split(s, "\n")[0], nil
}

// NpmUpgradeLatest installs latest version globally.
func NpmUpgradeLatest(ctx context.Context, pkg string) error {
    // Use --no-fund and --no-audit to speed up and reduce noise
    _, err := runCmd(ctx, "npm", "install", "-g", fmt.Sprintf("%s@latest", pkg), "--no-fund", "--no-audit")
    if err != nil && !strings.Contains(err.Error(), "not found") {
        return err
    }
    return nil
}

