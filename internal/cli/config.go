package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	cfg "codectl/internal/config"
	"codectl/internal/provider"
)

func init() { rootCmd.AddCommand(configCmd) }

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "初始化并显示配置位置",
	Long:  "创建 codectl 配置目录并初始化 provider/models/mcp 清单文件，然后打印配置目录位置。",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configWizard {
			return runConfigWizard(cmd.Context())
		}
		// Ensure dot config dir exists (~/.codectl)
		dir, err := cfg.DotDir()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}

		// Optional: migrate legacy files from OS config dir to ~/.codectl when present
		if legacy, _ := cfg.Dir(); legacy != "" && legacy != dir {
			// models.json
			if !fileExists(filepath.Join(dir, "models.json")) && fileExists(filepath.Join(legacy, "models.json")) {
				if b, err := os.ReadFile(filepath.Join(legacy, "models.json")); err == nil {
					_ = os.WriteFile(filepath.Join(dir, "models.json"), b, 0o644)
				}
			}
			// mcp.json
			if !fileExists(filepath.Join(dir, "mcp.json")) && fileExists(filepath.Join(legacy, "mcp.json")) {
				if b, err := os.ReadFile(filepath.Join(legacy, "mcp.json")); err == nil {
					_ = os.WriteFile(filepath.Join(dir, "mcp.json"), b, 0o644)
				}
			}
		}

		// 1) provider.json (v2): write defaults when missing (or normalize existing)
		provPath, _ := provider.Path()
		existedProv := fileExists(provPath)
		provV2, _ := provider.LoadV2() // returns default v2 if missing
		if err := provider.SaveV2(provV2); err != nil {
			return err
		}
		if existedProv {
			fmt.Printf("• provider.json 已更新/规范化(v2)：%s\n", provPath)
		} else {
			fmt.Printf("✓ 已创建 provider.json(v2)：%s\n", provPath)
		}

		// 2) models.json: seed from provider when missing
		modelsPath := filepath.Join(dir, "models.json")
		if fileExists(modelsPath) {
			fmt.Printf("• 保持现有 models.json：%s\n", modelsPath)
		} else {
			if err := os.WriteFile(modelsPath, []byte(toJSONString(provider.Models())), 0o644); err != nil {
				return err
			}
			fmt.Printf("✓ 已创建 models.json（来源 provider）：%s\n", modelsPath)
		}

		// 3) mcp.json: create example when missing (v2 moved MCP here)
		mcpPath := filepath.Join(dir, "mcp.json")
		if fileExists(mcpPath) {
			fmt.Printf("• 保持现有 mcp.json：%s\n", mcpPath)
		} else {
			// default example matching user's suggested shape
			example := `{
  "Framelink Figma MCP": {
    "command": "npx",
    "args": ["-y", "figma-developer-mcp", "--figma-api-key=YOUR-KEY", "--stdio"]
  }
}`
			if err := os.WriteFile(mcpPath, []byte(example), 0o644); err != nil {
				return err
			}
			fmt.Printf("✓ 已创建 mcp.json（示例配置）：%s\n", mcpPath)
		}

		fmt.Printf("\n配置目录：%s\n", dir)
		return nil
	},
}

func fileExists(path string) bool {
	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		return true
	}
	return false
}

func toJSONString(list []string) string {
	// Minimal JSON array writer without extra deps here
	// Values are already plain identifiers; no quoting of special chars assumed
	// for simplicity. For robustness, they do not contain quotes.
	b := make([]byte, 0, 2+len(list)*8)
	b = append(b, '[')
	for i, s := range list {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, '"')
		for j := 0; j < len(s); j++ {
			c := s[j]
			// escape simple quotes and backslashes if present
			if c == '"' || c == '\\' {
				b = append(b, '\\', c)
			} else {
				b = append(b, c)
			}
		}
		b = append(b, '"')
	}
	b = append(b, ']')
	return string(b)
}

// wizard flag and implementation
var configWizard bool

func init() {
	configCmd.Flags().BoolVarP(&configWizard, "wizard", "w", false, "运行交互式配置向导")
}

func runConfigWizard(ctx context.Context) error {
	// Ensure dot dir exists
	dot, err := cfg.DotDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dot, 0o755); err != nil {
		return err
	}
	// Normalize provider.json (v2)
	prov, _ := provider.LoadV2()
	if err := provider.SaveV2(prov); err != nil {
		return err
	}
	provPath, _ := provider.Path()
	fmt.Printf("使用 provider.json(v2)：%s\n\n", provPath)

	// Interactive selections
    in := newLiner()
    defer func() { _ = in.Close() }()

	// Models
	models := provider.Models()
	fmt.Println("选择要写入本地 models.json 的模型（回车=全部，输入 none 跳过，或输入序号用逗号分隔）：")
	for i, m := range models {
		fmt.Printf("  %2d) %s\n", i+1, m)
	}
	resp, _ := in.Prompt("> ")
	selModels := pickFromList(resp, models)
	if err := os.WriteFile(filepath.Join(dot, "models.json"), []byte(toJSONString(selModels)), 0o644); err != nil {
		return err
	}
	fmt.Printf("✓ 已写入 models.json（%d 条目）\n", len(selModels))

	// MCP: for v2, write example config and allow manual editing later
	mcpPath := filepath.Join(dot, "mcp.json")
	if !fileExists(mcpPath) {
		example := `{
  "Framelink Figma MCP": {
    "command": "npx",
    "args": ["-y", "figma-developer-mcp", "--figma-api-key=YOUR-KEY", "--stdio"]
  }
}`
		if err := os.WriteFile(mcpPath, []byte(example), 0o644); err != nil {
			return err
		}
		fmt.Printf("✓ 已写入 mcp.json 示例\n")
	} else {
		fmt.Printf("• 保持现有 mcp.json：%s\n", mcpPath)
	}

	fmt.Printf("\n完成。配置目录：%s\n", dot)
	return nil
}
