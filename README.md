# CODECTL

<p align="center">
    <img src="https://github.com/user-attachments/assets/effc6bc1-ef96-49cc-8751-6f9d1052e248" width="800"/>
<p>

> SDD is all you need

English documentation. 中文版请见：README.zh-cn.md

A minimal TUI for Spec‑Driven Development that maximizes the effective use of Codex and other coding agents.

## Features

- Spec‑Driven Development workflow (Spec → Task → Coding)
- Minimal TUI for essential agent monitoring
- Manage CLI coding agents (Codex / Claude Code / Gemini CLI)
- Manage MCP and third‑party models
- TUI + CLI: interactive usage and scriptable integration

## Why Spec‑Driven Development

- Spec: define specs in `vibe-docs/spec/`
- Task: define tasks in `vibe-docs/task/`
- Coding: execute implementation via LLMs

## Quick Start

1) Build and run codectl:

```bash
# Run locally for development
go run .

# Or build a binary
go build -o codectl
./codectl
```

## Usage

```bash
codectl cli                     # Open CLI manager TUI (/add, /remove, /upgrade, ...)
# TODO: optimize this
# shortcut for: codex --dangerously-bypass-approvals-and-sandbox -m gpt-5 -c model_reasoning_effort=high
codectl codex                   # codex + GPT‑5 (high effort)
# TODO: implement this
codectl update                  # Self‑update from GitHub Releases (planned)
codectl version                 # Print codectl version (numeric only; script‑friendly)
# TODO: maybe better TUI
codectl config                  # Init and print config dir (generate provider/models/mcp files)

codectl spec                    # Open interactive Spec UI (table picker + left Markdown + right logs + bottom input)
codectl spec new "<desc>"       # Call codex exec to draft a spec and save to vibe-docs/spec

codectl check                   # Validate frontmatter of .spec.mdx under vibe-docs/spec (title required)
codectl check --json            # Output JSON report

codectl provider sync           # Manually sync/generate ~/.codectl/provider.json (then customize)
codectl provider schema         # Print JSON Schema of provider.json (for validation/completion)
```

## Roadmap

- [ ] 
    1. Prototype
- [ ] 
    2. Better Spec TUI
- [ ] 
    3. Config Wizard (MCP/Custom Provider)

## Development & Build

Requirement: Go 1.25+ (latest stable recommended)

```bash
# Fetch deps (first run will pull automatically)
go mod download

# Run locally
go run .

# Build binary
go build -o codectl
```

This project uses [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI. Contributions welcome: please
consider updating specs in `vibe-docs/spec/` first, then submit implementation + docs.

## Disclaimer

codectl helps you install, check, and configure third‑party tools; it does not provide model inference itself. The
capabilities, stability, and terms of third‑party CLIs/MCPs are governed by their respective providers—review and follow
their usage policies as needed.

## License

MIT License. See `LICENSE`.
