# Contributing to brrewery

Thanks for taking interest in contributing! We welcome anyone who wants to contribute.

If you have an idea for a bigger feature or a change, then we are happy to discuss it before you start working on it.
It is usually a good idea to make sure it aligns with the project and is a good fit.
Open an issue to start the discussion.

This document is a guide to help you through the process of contributing to brrewery.

## Become a contributor

* Code: new features, bug fixes, improvements
* App playbooks: add or improve Ansible playbooks for installable apps
* Report bugs

## Developer guide

This guide helps you get started developing brrewery.

## Dependencies

Make sure you have the following dependencies installed before setting up your developer environment:

- [Git](https://git-scm.com/)
- [Go](https://golang.org/dl/) (see [go.mod](go.mod#L3) for minimum required version)
- [Node.js](https://nodejs.org) (we usually use the latest Node LTS version - for further information see `@types/node` major version in [package.json](web/package.json))
- [pnpm](https://pnpm.io/installation)
- [air](https://github.com/air-verse/air) (optional, for backend hot-reload via `make dev`)
- [Ansible](https://docs.ansible.com/ansible/latest/installation_guide/index.html) (only needed when working on app playbooks)

## How to contribute

- **Fork and Clone:** [Fork the brrewery repository](https://github.com/autobrr/brrewery/fork) and clone it to start working on your changes.
- **Branching:** Create a new branch for your changes. Use a descriptive name for easy understanding.
  - Checkout a new branch for your fix or feature: `git checkout -b fix/install-progress`
- **Coding:** Comment non-obvious logic - see [AGENTS.md](AGENTS.md) for code style and comment conventions. With Go, use `go fmt`.
- **Commit Guidelines:** We appreciate the use of [Conventional Commit Guidelines](https://www.conventionalcommits.org/en/v1.0.0/#summary) when writing your commits.
  - Examples: `fix(apps): handle missing systemd unit`, `feat(ansible): add sonarr playbook`
  - There is no need for force pushing or rebasing. We squash commits on merge to keep the history clean and manageable.
  - The PR title becomes the squashed commit message, so use the conventional commit format for the title as well.
- **Pull Requests:** Submit a pull request from your fork with a clear description of your changes. Reference any related issues.
  - Target the `develop` branch.
  - Fill out the pull request template, including the AI disclosure section.
  - Mark it as Draft if it's still in progress.
- **Code Review:** Be open to feedback during the code review process.

## Development environment

The backend is written in Go and the frontend is written in TypeScript using React (Vite + TailwindCSS + TanStack).

You need to have the Go toolchain installed and Node.js with `pnpm` as the package manager.

Clone the project and change dir:

```shell
git clone https://github.com/autobrr/brrewery.git && cd brrewery
```

Install all dependencies (Go and web) with:

```shell
make deps
```

> [!TIP]
> `make dev` starts both the backend (with hot-reload via air) and the frontend dev server at once.

## Frontend

First install the web dependencies:

```shell
cd web && pnpm install
```

Run the project:

```shell
pnpm dev
```

Or from the repository root:

```shell
make dev-frontend
```

This starts the Vite dev server, set up to communicate with the backend API at [http://127.0.0.1:8081](http://127.0.0.1:8081).

### Build

To build the frontend and sync the production bundle to `internal/web/dist`, run:

```shell
make frontend
```

## Backend

Install Go dependencies:

```shell
go mod download
```

Run the project with hot-reload:

```shell
make dev-backend
```

This runs the API on [http://127.0.0.1:8081](http://127.0.0.1:8081) with the repository's `ansible/` directory as the playbook root. There is no config file — the dev environment variables are set by the Makefile.

### Build

To build the backend, run:

```shell
make backend
```

This will output a `brrewery` binary in the repository root.

You can also build the frontend and the backend at once with:

```shell
make build
```

## Tests

All tests run per commit with GitHub Actions.

### Run backend tests

```shell
make test
```

This runs the Go suite with `-race -count=1`.

After touching `internal/web/swagger`, validate the OpenAPI spec:

```shell
make test-openapi
```

### Run frontend tests

Tests are colocated as `*.test.ts(x)` under `web/src/`:

```shell
cd web && pnpm test
```

For iterative local work:

```shell
cd web && pnpm test:watch
```

Frontend changes should ship with vitest specs. Behavior jsdom can't render — virtualization, drag-and-drop, scroll — needs a manual smoke test (a passing suite isn't full coverage).

## Linting and formatting

Before committing, run:

```shell
make precommit
```

This formats, applies `go fix`, and lints the changed files only (fast feedback during iteration).

Other useful targets:

- `make lint` — lint changed files (Go + frontend)
- `make lint-full` — lint the whole repository
- `make lint-fix` — auto-fix lint issues, then address the rest manually

Avoid repo-wide `pnpm format` / `eslint --fix` sweeps — prefer fixing only the files reported by lint for your current change.

## App playbooks (Ansible)

App installs are driven by Ansible playbooks under `ansible/playbooks`. To add support for a new app, start from an existing playbook for a similar app and adjust it.

Check playbook syntax with:

```shell
make ansible-syntax-check
```

Do not track installed apps via JSON state files — brrewery detects installs by querying the filesystem for the app's executables and dependencies. Never store app install secrets (API keys, tokens, etc.) in files; they are prompted for in the frontend at install time only.

See [AGENTS.md](AGENTS.md) for the full set of project conventions.
