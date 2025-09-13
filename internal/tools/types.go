package tools

// Tool identifiers and metadata
type ToolID string

const (
	ToolCodex  ToolID = "Codex"
	ToolClaude ToolID = "Claude"
	ToolGemini ToolID = "Gemini"
)

type ToolInfo struct {
	ID          ToolID
	DisplayName string
	Package     string   // npm package name for fallback detection
	Binaries    []string // candidate binary names in PATH
	VersionArgs [][]string
}

// Check results
type CheckResult struct {
	Installed bool
	Version   string
	Source    string // which method produced version (binary/npm)
	Err       string
	Latest    string // latest version from registry
}
