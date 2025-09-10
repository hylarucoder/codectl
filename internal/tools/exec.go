package tools

import (
    "context"
    "os"
    "os/exec"
)

// runCmd executes a command and returns combined output as string.
func runCmd(ctx context.Context, name string, args ...string) (string, error) {
    cmd := exec.CommandContext(ctx, name, args...)
    // Avoid opening pager or interactive prompts
    cmd.Env = append(os.Environ(), "NO_COLOR=1")
    out, err := cmd.CombinedOutput()
    if ctx.Err() == context.DeadlineExceeded {
        return "", ctx.Err()
    }
    return string(out), err
}

