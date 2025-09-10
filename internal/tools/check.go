package tools

import (
    "context"
    "errors"
    "fmt"
    "os/exec"
    "strings"
    "time"
)

// CheckTool attempts to detect tool version via PATH binaries, then falls back to npm global list.
func CheckTool(t ToolInfo) CheckResult {
    // Always try to get latest version from npm
    latest := ""
    if t.Package != "" {
        ctxL, cancelL := context.WithTimeout(context.Background(), 6*time.Second)
        latest, _ = NpmLatestVersion(ctxL, t.Package)
        cancelL()
    }

    // Try binaries in PATH
    for _, bin := range t.Binaries {
        if path, err := exec.LookPath(bin); err == nil {
            for _, args := range t.VersionArgs {
                ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
                out, err := runCmd(ctx, path, args...)
                cancel()
                if err == nil && strings.TrimSpace(out) != "" {
                    ver := ParseVersion(out)
                    if ver == "" {
                        ver = strings.Split(strings.TrimSpace(out), "\n")[0]
                    }
                    return CheckResult{Installed: true, Version: ver, Source: fmt.Sprintf("%s %s", bin, strings.Join(args, " ")), Latest: latest}
                }
            }
            // Found binary but no version output; still consider installed
            return CheckResult{Installed: true, Version: "", Source: bin, Latest: latest}
        }
    }

    // Fallback: npm global list
    if t.Package != "" {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        ver, err := NpmGlobalVersion(ctx, t.Package)
        if err == nil && ver != "" {
            return CheckResult{Installed: true, Version: ver, Source: "npm -g", Latest: latest}
        }
        if err != nil && !errors.Is(err, exec.ErrNotFound) {
            return CheckResult{Installed: false, Err: err.Error(), Latest: latest}
        }
    }

    return CheckResult{Installed: false, Err: "未找到可执行文件或 npm 记录", Latest: latest}
}

