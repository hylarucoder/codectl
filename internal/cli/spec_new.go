package cli

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    "github.com/spf13/cobra"

    "codectl/internal/system"
)

func init() {
    specCmd.AddCommand(specNewCmd)
}

var specNewCmd = &cobra.Command{
    Use:   "new <说明>",
    Short: "Generate a spec draft via Codex and save to vibe-docs/spec",
    Args:  cobra.MinimumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        prompt := strings.TrimSpace(strings.Join(args, " "))
        if prompt == "" {
            return fmt.Errorf("用法：codectl spec new \"<说明>\"")
        }

        // find codex binary
        bin := ""
        for _, cand := range []string{"codex", "openai-codex"} {
            if p, err := exec.LookPath(cand); err == nil && p != "" {
                bin = p
                break
            }
        }
        if bin == "" {
            return fmt.Errorf("未找到 codex CLI（请先安装 @openai/codex 并确保在 PATH 中）")
        }

        // resolve repo root
        cwd, _ := os.Getwd()
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        root, err := system.GitRoot(ctx, cwd)
        cancel()
        if err != nil || strings.TrimSpace(root) == "" {
            root = cwd
        }
        outDir := filepath.Join(root, "vibe-docs", "spec")
        if err := os.MkdirAll(outDir, 0o755); err != nil {
            return fmt.Errorf("创建目录失败：%w", err)
        }

        // run codex exec (argument, then stdin fallback)
        body, runErr := runCodexOnce(bin, prompt, 120*time.Second)
        if runErr != nil && body == "" {
            return fmt.Errorf("codex exec 失败：%w", runErr)
        }

        content := body
        // wrap with frontmatter if missing
        trimmed := strings.TrimSpace(body)
        if !strings.HasPrefix(trimmed, "---") {
            fm := "---\n" +
                "title: " + prompt + "\n" +
                "specVersion: 0.1.0\n" +
                "status: draft\n" +
                "lastUpdated: {auto}\n" +
                "---\n\n"
            content = fm + body
        }

        ts := time.Now().Format("060102-150405")
        name := fmt.Sprintf("draft-%s-%s.spec.mdx", ts, slugifyCLI(prompt))
        outPath := filepath.Join(outDir, name)
        if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
            return fmt.Errorf("写入失败：%w", err)
        }
        // optionally persist raw output when non-fatal error returned
        if runErr != nil {
            _ = os.WriteFile(outPath+".raw.txt", []byte(body), 0o644)
        }
        fmt.Println(outPath)
        return nil
    },
}

// runCodexOnce mirrors the TUI fallback: try arg then stdin.
func runCodexOnce(bin, prompt string, timeout time.Duration) (string, error) {
    // try with argument
    {
        ctx, cancel := context.WithTimeout(context.Background(), timeout)
        cmd := exec.CommandContext(ctx, bin, "exec", prompt)
        cmd.Env = append(os.Environ(), "NO_COLOR=1")
        out, err := cmd.CombinedOutput()
        cancel()
        if err == nil && len(out) > 0 {
            return string(out), nil
        }
        if ctx.Err() == context.DeadlineExceeded {
            return "", ctx.Err()
        }
        if len(out) > 0 {
            return string(out), err
        }
    }
    // fallback via stdin
    {
        ctx, cancel := context.WithTimeout(context.Background(), timeout)
        cmd := exec.CommandContext(ctx, bin, "exec")
        cmd.Env = append(os.Environ(), "NO_COLOR=1")
        cmd.Stdin = strings.NewReader(prompt)
        out, err := cmd.CombinedOutput()
        cancel()
        if err == nil && len(out) > 0 {
            return string(out), nil
        }
        if len(out) > 0 {
            return string(out), err
        }
        if err != nil {
            return "", err
        }
        return string(out), nil
    }
}

// slugifyCLI is a local copy (no export) of slugify logic for filenames.
func slugifyCLI(s string) string {
    s = strings.ToLower(strings.TrimSpace(s))
    b := make([]rune, 0, len(s))
    lastDash := false
    for _, r := range s {
        if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
            b = append(b, r)
            lastDash = false
            continue
        }
        if r >= 0x4E00 && r <= 0x9FFF { // keep CJK
            b = append(b, r)
            lastDash = false
            continue
        }
        if !lastDash {
            b = append(b, '-')
            lastDash = true
        }
    }
    res := strings.Trim(string(b), "-")
    if res == "" {
        res = "spec"
    }
    return res
}

