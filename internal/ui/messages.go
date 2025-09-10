package ui

import "codectl/internal/tools"

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

