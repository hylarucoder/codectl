package tools

var Tools = []ToolInfo{
	{
		ID:          ToolCodex,
		DisplayName: "Codex (@openai/codex)",
		Package:     "@openai/codex",
		Binaries:    []string{"codex", "openai-codex"},
		VersionArgs: [][]string{{"--version"}, {"-v"}, {"version"}},
	},
	{
		ID:          ToolClaude,
		DisplayName: "Claude Code (@anthropic-ai/claude-code)",
		Package:     "@anthropic-ai/claude-code",
		Binaries:    []string{"claude", "claude-code"},
		VersionArgs: [][]string{{"--version"}, {"-v"}, {"version"}},
	},
	{
		ID:          ToolGemini,
		DisplayName: "Gemini CLI (@google/gemini-cli)",
		Package:     "@google/gemini-cli",
		Binaries:    []string{"gemini"},
		VersionArgs: [][]string{{"--version"}, {"-v"}, {"version"}},
	},
}
