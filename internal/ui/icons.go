package ui

import "os"

// nfEnabled returns true when Nerd Font icons should be rendered.
// Opt-in via environment variable NERDFONT=1 to avoid tofu on systems
// without Nerd Font installed.
// Default to enabled; allow disabling via NERDFONT=0
func nfEnabled() bool {
	return os.Getenv("NERDFONT") != "0"
}

func nf(icon, fallback string) string {
	if nfEnabled() {
		return icon
	}
	return fallback
}

// High-level icons used across the dash
func IconSpecTasks() string { return nf("", "") } // fa-list-alt
func IconConfig() string    { return nf("", "") } // fa-sliders/gear
func IconOps() string       { return nf("", "") } // fa-terminal

// Stat icons
func IconTotal() string      { return nf("", "") } // fa-line-chart
func IconDraft() string      { return nf("", "") } // fa-pencil
func IconProposal() string   { return nf("", "") } // fa-rocket
func IconAccepted() string   { return nf("", "") } // fa-check
func IconDeprecated() string { return nf("", "") } // fa-warning
func IconRetired() string    { return nf("", "") } // fa-archive
func IconDoc() string        { return nf("", "") } // fa-file

// Status bar icons
func IconTerminal() string { return nf("", "") }
func IconClock() string    { return nf("", "") }
func IconVersion() string  { return nf("", "") }
func IconGit() string      { return nf("", "git") } // nf-dev-git
func IconBranch() string   { return nf("", "br") }  // nf-oct-git_branch
func IconCommit() string   { return nf("", "sha") } // nf-oct-git_commit
func IconDirty() string    { return nf("", "*") }   // fa-exclamation-circle

// Workbar helpers
func IconRefresh() string   { return nf("", "") } // fa-refresh
func IconFilter() string    { return nf("", "") } // fa-filter
func IconUser() string      { return nf("", "") } // fa-user
func IconFastBolt() string  { return nf("", "") } // fa-bolt
func IconTasksWork() string { return nf("", "") } // fa-list-alt
func IconDiff() string      { return nf("", "") } // fa-random
func IconSearch() string    { return nf("", "") } // fa-search

// Diff change type icons (Octicons)
func IconDiffAdded() string     { return nf("", "+") } // nf-oct-diff_added
func IconDiffModified() string  { return nf("", "~") } // nf-oct-diff_modified
func IconDiffRemoved() string   { return nf("", "-") } // nf-oct-diff_removed
func IconDiffRenamed() string   { return nf("", ">") } // nf-oct-diff_renamed
func IconDiffCopied() string    { return nf("", "c") } // fa-copy
func IconDiffUnmerged() string  { return nf("", "U") } // nf-dev-git_merge
func IconDiffUntracked() string { return nf("", "?") } // fa-question

// IconDiffByStatus maps porcelain short status (X or Y) to an icon
func IconDiffByStatus(s string) string {
	switch s {
	case "A":
		return IconDiffAdded()
	case "M":
		return IconDiffModified()
	case "D":
		return IconDiffRemoved()
	case "R":
		return IconDiffRenamed()
	case "C":
		return IconDiffCopied()
	case "U":
		return IconDiffUnmerged()
	case "?":
		return IconDiffUntracked()
	default:
		return IconDiff()
	}
}
