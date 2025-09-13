# Repository Guidelines

## Project Structure & Module Organization
- `cmd/codectl/`: CLI entrypoint wiring; see `main.go`.
- `internal/cli/`: Cobra commands (e.g., `model`, `provider`, `mcp`, `spec`).
- `internal/ui/` + `internal/app/`: Bubble Tea TUI models, update/view, and app bootstrap.
- `internal/provider/`: Provider catalog v2 (JSON at `~/.codectl/provider.json`) and schema helpers.
- `internal/mcp/`, `internal/models/`, `internal/config/`, `internal/settings/`, `internal/tools/`, `internal/version/`: supporting modules and storage utilities.
- `vibe-docs/`: Spec‑Driven docs; keep specs and AGENT guidance in sync with behavior.
- Top-level: `Makefile`, `.air.toml`, `Dockerfile.test`, `README.md`.

## Build, Test, and Development Commands
- `go run .` — run CLI locally. `go build -o codectl` to compile.
- `make start` — hot‑reload dev via Air (install with `go install github.com/air-verse/air@latest`).
- `make format` — `go fmt ./...`.
- `make lint` — `golangci-lint run` if available, else `go vet ./...`.
- `make test` — unit tests with coverage to `coverage.out`.
- `make docker-test` — run tests in Docker; example: `GO_TEST_FLAGS='-v -run TestLoad' make docker-test`.

## Coding Style & Naming Conventions
- Go 1.25; always format with `go fmt` and keep diffs minimal.
- Packages: lowercase, short names. Exports: `CamelCase`; unexported: `lowerCamel`.
- Errors: return errors (no panics in libraries); wrap with `%w` when appropriate.
- CLI flags/commands live in `internal/cli/*`; keep command files small and cohesive.

## Testing Guidelines
- Standard `testing` only; files end with `_test.go`, functions `TestXxx`.
- Use `t.TempDir()` and `internal/testutil.WithEnv` to isolate `HOME`/`XDG_CONFIG_HOME` for file‑based tests.
- Avoid network; rely on fixtures or in‑memory data. Ensure new code paths include tests.
- Run `make test` (or `make docker-test`) before pushing.

## Commit & Pull Request Guidelines
- Use Conventional Commits: `feat(scope): ...`, `fix`, `docs`, `chore`, etc. Scopes seen here: `specui`, `cli`, `provider`, `docs`.
- PRs must include: purpose/motivation, concise changes list, test plan (`make format lint test` output), and screenshots/terminal captures for TUI/CLI changes. Link related issues/specs in `vibe-docs/`.

## Security & Configuration Tips
- Do not commit secrets. Respect `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GOOGLE_API_KEY`, and proxy vars.
- Provider catalog lives at `~/.codectl/provider.json`; inspect schema via `codectl provider schema`. Initialize defaults with `codectl config`.
