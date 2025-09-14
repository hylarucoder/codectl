package ui

import (
	"time"

	"codectl/internal/system"
	"codectl/internal/tools"
)

// Bubble Tea messages
type versionMsg struct {
	id     tools.ToolID
	result tools.CheckResult
}

// Upgrade support messages
type upgradeProgressMsg struct {
	id   tools.ToolID
	note string
}

// generic notifications and quit
type noticeMsg string
type quitMsg struct{}

// start upgrade flow
type startUpgradeMsg struct{}

// periodic tick for status bar time
type tickMsg time.Time

// git info updates
type gitInfoMsg struct{ info system.GitInfo }

// external exec finished messages (e.g., /codex)
type codexFinishedMsg struct {
	err   error
	quiet bool
}

// config info
type configInfoMsg struct{ info ConfigInfo }

// external exec finished messages for settings
type settingsFinishedMsg struct {
	err   error
	quiet bool
}
