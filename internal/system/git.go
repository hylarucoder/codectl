package system

import (
    "context"
    "os/exec"
    "strings"
    "time"
)

type GitInfo struct {
    InRepo   bool
    Branch   string
    ShortSHA string
    Dirty    bool
}

// GetGitInfo inspects the Git repository at dir and returns basic status.
func GetGitInfo(ctx context.Context, dir string) (GitInfo, error) {
    gi := GitInfo{}

    // Ensure git exists
    if _, err := exec.LookPath("git"); err != nil {
        return gi, nil
    }

    // Provide a short timeout per call to avoid hanging
    withTimeout := func(d time.Duration) (context.Context, context.CancelFunc) {
        return context.WithTimeout(ctx, d)
    }

    // Check if inside a work tree
    {
        cctx, cancel := withTimeout(800 * time.Millisecond)
        out, err := exec.CommandContext(cctx, "git", "-C", dir, "rev-parse", "--is-inside-work-tree").CombinedOutput()
        cancel()
        if err != nil {
            return gi, nil
        }
        if strings.TrimSpace(string(out)) != "true" {
            return gi, nil
        }
        gi.InRepo = true
    }

    // Branch name (short)
    {
        cctx, cancel := withTimeout(800 * time.Millisecond)
        out, err := exec.CommandContext(cctx, "git", "-C", dir, "symbolic-ref", "--quiet", "--short", "HEAD").CombinedOutput()
        cancel()
        if err == nil {
            gi.Branch = strings.TrimSpace(string(out))
        } else {
            // Detached head fallback
            cctx2, cancel2 := withTimeout(800 * time.Millisecond)
            out2, err2 := exec.CommandContext(cctx2, "git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD").CombinedOutput()
            cancel2()
            if err2 == nil {
                gi.Branch = strings.TrimSpace(string(out2))
            }
        }
    }

    // Short SHA
    {
        cctx, cancel := withTimeout(800 * time.Millisecond)
        out, err := exec.CommandContext(cctx, "git", "-C", dir, "rev-parse", "--short", "HEAD").CombinedOutput()
        cancel()
        if err == nil {
            gi.ShortSHA = strings.TrimSpace(string(out))
        }
    }

    // Dirty state
    {
        cctx, cancel := withTimeout(800 * time.Millisecond)
        out, err := exec.CommandContext(cctx, "git", "-C", dir, "status", "--porcelain").CombinedOutput()
        cancel()
        if err == nil {
            gi.Dirty = strings.TrimSpace(string(out)) != ""
        }
    }

    return gi, nil
}

// GitRoot returns the repository top-level directory for dir, if in a Git repo.
func GitRoot(ctx context.Context, dir string) (string, error) {
    if _, err := exec.LookPath("git"); err != nil {
        return "", err
    }
    out, err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--show-toplevel").CombinedOutput()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(out)), nil
}
