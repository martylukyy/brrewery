# AGENTS.md

Repository guidelines for AI coding agents (Codex, Claude Code, etc.) working on brrewery.

## Project description

brrewery is a web interface for accessing, installing, managing, updating/upgrading and removing packages.
It is driven by a Go backend and a React + Vite + TypeScript + TailwindCSS + TanStack frontend.
It is supposed to be a successor to https://github.com/swizzin/swizzin but only supporting amd64 archiectures.
It's project structure should adhere to related projects in the https://github.com/autobrr org on GitHub like:
- https://github.com/autobrr/autobrr
- https://github.com/autobrr/qui
- https://github.com/autobrr/netronome
- https://github.com/autobrr/dashbrr
- https://github.com/autobrr/librrary
Unlike related projects, brrewery does not use a SQLite or PostgreSQL database due to the nature of the project.
Unlike related projects, brrewery should only log to a logfile and should not use a config.toml since it's dashboard should always be served by a nginx webserver at `/`.
For the nginx configuration use a configuration structure like https://github.com/digitalocean/nginxconfig.io does.
Do not keep track of packages installed via JSON files which could be tampered with but by querying the filesystem if the executables or dependencies for a certain exists.
For setup we need a sh/bash install script that installs all the dependencies for brrewery and autostarts the daemon for brrewery
so we don't need any env vars or CLI flags.
The listen adresses for the brrewery webUI in production Environments should be port 80 for HTTP and port 443 for HTTPS.
When opening http(s)://127.0.0.1 to access brrewery, a login form should be shown which needs the login information from the user that started the initial setup. 
Later on other users can be added since brrewery should be a multi-tenant software solution.

## Security related notes

Do not store any user information in a file or a database.
Prompt the user for passwords or tokens on the frontend in case such information needs to be provided during installation of packages.

## Owner Collaboration Notes

- Do not implement review-suggested or extra changes outside requested scope without explicit user approval first.
- Treat other agent/Codex/CodeRabbit feedback as input to discuss, not automatic action.

## Project Structure & Module Organization

The Go backend lives in `cmd/brrewery` (entrypoint) and `internal/` modules for API routing, configuration, metrics, and packages; shared helpers sit in `pkg/`. 
The React/TailwindCSS/Vite client is in `web/src` with static assets in `web/public`, and its production bundle must stay synced to `internal/web/dist`. 
Reference docs live under `docs/`, while Docker and compose files in the repository root support container workflows.
End-user docs live in the Docusaurus project under `documentation/docs/`. Prefer updating those for user-facing copy; `docs/` is mostly internal/engineering notes.
Keep `README.md` concise. Feature deep-dives belong in `documentation/docs/`, not the root README.

## Build, Test, and Development Commands

```bash
# Build
make build              # Frontend bundle + Go binary with version metadata
make backend            # Go binary only
make frontend           # Frontend bundle only

# Development
make dev                # Starts air (hot-reload) + pnpm dev
make dev-backend        # Backend only with hot-reload
make dev-frontend       # Frontend only

# Testing
make test               # go test -race -count=1 -v ./...
make test-openapi       # Validate OpenAPI spec after touching internal/web/swagger

# Linting
make lint               # Changed files only (fast, use during iteration)
make lint-json          # JSON output to lint-report.json

# Pre-commit
make precommit          # fmt + gofix + lint on changed files

# Formatting
make fmt                # gofmt + frontend eslint --fix on changed files
make gofix-changed      # Apply go fix on changed Go files only
make gofix-check-changed # Check go fix drift on changed Go files only
```

## Linting Strategy

The project uses golangci-lint v2 with strict configuration targeting AI-generated code patterns:

| Linter | Purpose | Threshold |
|--------|---------|-----------|
| dupl | Catch code duplication | 100 tokens |
| gocognit | Cognitive complexity | 15 |
| funlen | Function length | 80 lines |
| interfacebloat | Interface size | 5 methods |
| errcheck | Unchecked errors | All, including type assertions |
| gocritic | Non-idiomatic patterns | diagnostic + style + performance |

**Workflow:**
1. During implementation: `make precommit` (changed files only, fast feedback)
2. To fix issues: `make lint-fix` then address remaining manually

**Guardrail (web formatting):** avoid repo-wide `pnpm format` / `eslint --fix` sweeps unless explicitly requested. Prefer fixing only the files reported by lint for the current task/PR.

## Coding Style & Naming Conventions

Keep Go code `gofmt`-clean with PascalCase exports, camelCase locals, and package-level interfaces grouped by domain inside `internal/<area>`. The frontend follows ESLint @stylistic defaults: two-space indentation, double quotes, trailing commas on multiline literals, and Unix line endings. Organize React modules by feature within `web/src/{pages,routes,components}` and choose descriptive file names (e.g., `torrent-table.tsx`).

**Critical conventions:**
- Prefer explicit error handling over silent failures
- Keep interfaces small (≤5 methods)
- Avoid `map[string]interface{}` — use proper structs
- No backward compatibility shims unless explicitly requested
- **Loop variables (Go 1.22+):** Don't use `tt := tt` in parallel subtests — Go 1.22+ creates a new variable per iteration, so the old workaround is unnecessary and flagged by linters

**Single-user self-hosted context:** brrewery runs on someone's home server, not as a multi-tenant SaaS with untrusted input and complex failure modes. Skip paranoid defensive programming for impossible or purely theoretical scenarios. Code that guards against states that can't happen adds complexity without value. Prioritize readable, maintainable code over excessive robustness.

## Code Shape

- Prefer behavior-bearing branches only. If multiple `switch` cases return the same value as `default`, collapse them.
- In boolean classifiers, list only the exceptional cases (`true` cases or error cases). Let `default` handle the common path.
- Do not add documentation-only branches unless they enforce something mechanically via compiler, linter, or tests.
- When a branch only enumerates known states, ask whether it changes behavior, improves safety, or provides exhaustiveness checking. If not, delete it.

## React Effects

- Use `useEffect` only to sync with external systems (DOM, subscriptions, network).
- Avoid derived state in Effects; calculate during render, or `useMemo` for expensive compute.
- Put user-driven logic in event handlers, not Effects.
- To reset state, prefer a `key` or render-time adjustments instead of Effects.
- Fetch Effects must guard against stale responses (cleanup/abort).
- Source: https://react.dev/learn/you-might-not-need-an-effect

## Testing Guidelines

Place backend tests beside implementations as `*_test.go`.
Prefer table-driven cases and reuse the integration fixtures. 
Run targeted local tests for touched packages and add `make test-openapi` when contracts change. 
CI covers the full `make test` suite unless explicitly requested locally. 
Frontend work should include Vitest + React Testing Library specs named `*.test.tsx` near the component.
When running tests, always use `-race` and `-count=1`.
When adding Go tests that create files with `os.WriteFile`, use `0o600` or tighter permissions unless the test explicitly needs broader mode bits. This avoids `gosec` `G306` lint failures.

## Commit & Pull Request Guidelines

Follow the conventional commit style in history (`feat(scope):`, `fix(scope):`, etc.) and link issues or PR numbers in the body when relevant. Keep commits focused—split backend and frontend changes when practical.

**Never add:**
- "🤖 Generated with [Claude Code](https://claude.com/claude-code)"
- "Co-Authored-By: Claude" or any AI co-author credits
- Any advertising or attribution in commit messages

PRs need a clear summary, testing checklist, and UI screenshots for visual tweaks. Confirm local targeted verification, `make lint`, and a fresh `make build` succeed before requesting review; rely on CI for the full `make test` suite unless explicitly requested.

## Pre-Commit Checklist

1. `make precommit` passes (`fmt` + `gofix-changed` + `lint`, changed files only)
2. Targeted local tests for touched packages pass (full `make test` covered by CI unless explicitly requested)
3. `make build` succeeds
4. If touched `internal/web/swagger`, run `make test-openapi`

## API Change Rules

- API contract changes must update OpenAPI content under `internal/web/swagger` and pass `make test-openapi`.
- Prefer minimal, reviewable diffs in high-churn areas.

## Architecture Quick Reference

```text
cmd/brrewery/main.go         CLI entrypoint (serve, generate-config, create-user, etc.)
internal/api/                HTTP handlers + middleware (chi router)
internal/services/           Domain services (autobrr, plex, prowlarr, qbittorrent, radarr, sabnzbd, sonarr)
internal/models/             Data models + store interfaces
pkg/                         Shared utilities
web/src/                     React + Vite + TypeScript + TailwindCSS
```