# Repository Guidelines

This guide helps contributors work effectively in this repository.

## Project Structure & Module Organization
- `cmd/codectl/`: CLI entrypoint wiring; see `main.go`.
- `internal/cli/`: Cobra commands (e.g., `model`, `provider`, `mcp`, `spec`).
- `internal/webui/`: Web UI server (Gin) and embedded assets.
- `internal/provider/`: Provider catalog v2 (JSON at `~/.codectl/provider.json`) and schema helpers.
- `internal/mcp/`, `internal/models/`, `internal/config/`, `internal/settings/`, `internal/tools/`, `internal/version/`: supporting modules and storage utilities.
- `vibe-docs/`: Spec‑driven docs; keep specs and AGENT guidance in sync with behavior.
- Top-level: `Makefile`, `.air.toml`, `Dockerfile.test`, `README.md`.

## Build, Test, and Development Commands
- Run locally: `go run .`  • Build: `go build -o codectl`.
- Hot reload dev: `make start` (requires `go install github.com/air-verse/air@latest`).
- Format: `make format` (`go fmt ./...`).  Lint: `make lint` (`golangci-lint` or `go vet`).
- Tests: `make test` (writes coverage to `coverage.out`). Docker tests: `make docker-test` (e.g., `GO_TEST_FLAGS='-v -run TestLoad'`).

## Coding Style & Naming Conventions
- Go 1.25; always run `go fmt`. Keep diffs minimal.
- Packages: lowercase, short names. Exports: `CamelCase`; unexported: `lowerCamel`.
- Errors: return errors (no panics in libraries); wrap with `%w` when appropriate.
- CLI flags/commands live in `internal/cli/*`; keep command files small and cohesive.

## Testing Guidelines
- Standard `testing` only; files end with `_test.go`, functions `TestXxx`.
- Use `t.TempDir()` and `internal/testutil.WithEnv` to isolate `HOME`/`XDG_CONFIG_HOME` for file-based tests.
- Avoid network I/O; rely on fixtures or in‑memory data.
- Run `make test` (or `make docker-test`) before pushing.

## Commit & Pull Request Guidelines
- Conventional Commits: `feat(scope): ...`, `fix`, `docs`, `chore`, etc. Common scopes: `specui`, `cli`, `provider`, `docs`.
- PRs must include: purpose/motivation, concise changes list, test plan output (`make format lint test`), and screenshots/recordings for WebUI/CLI changes. Link related issues/specs in `vibe-docs/`.

## Security & Configuration Tips
- Never commit secrets. Respect `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GOOGLE_API_KEY`, and proxy env vars.
- Provider catalog lives at `~/.codectl/provider.json`; inspect schema via `codectl provider schema`. Initialize defaults with `codectl config`.
