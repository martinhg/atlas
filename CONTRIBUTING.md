# Contributing to Atlas

Thank you for your interest in contributing to Atlas! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

## License

Atlas uses a dual-license model:

- **CLI and parsers** (`cmd/atlas/`, `internal/ingest/parsers/`) are licensed under [Apache License 2.0](cmd/atlas/LICENSE).
- **Server and web application** (everything else) are licensed under [Business Source License 1.1](LICENSE).

By contributing, you agree that your contributions will be licensed under the applicable license for the component you are contributing to.

## Getting Started

### Prerequisites

- Go 1.23+
- Node.js 22+
- pnpm 11+ (`npm i -g pnpm`)
- Docker and Docker Compose
- Make

> **Important:** This project uses **pnpm exclusively**. Do not use npm or yarn — the preinstall script will block them.

### Setup

```bash
# Clone the repository
git clone https://github.com/nesbite/atlas.git
cd atlas

# Start dependencies
make dev-up

# Run the backend
make run-server

# In another terminal, install frontend deps and run
cd web && pnpm install
make run-web
```

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/nesbite/atlas/issues).
2. If not, create a new issue using the **Bug Report** template.
3. Include reproduction steps, expected behavior, and actual behavior.

### Suggesting Features

1. Check if the feature has already been requested in [Issues](https://github.com/nesbite/atlas/issues).
2. If not, create a new issue using the **Feature Request** template.
3. Describe the problem you're trying to solve, not just the solution you want.

### Submitting Changes

1. Fork the repository.
2. Create a feature branch from `main`:
   ```bash
   git checkout -b feat/your-feature-name
   ```
3. Make your changes following the coding standards below.
4. Write or update tests as needed.
5. Commit your changes following the [commit convention](#commit-convention).
6. Push to your fork and open a Pull Request.

## Commit Convention

We use [Conventional Commits](https://www.conventionalcommits.org/). Every commit message must follow this format:

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Types

| Type | Description |
|------|-------------|
| `feat` | A new feature |
| `fix` | A bug fix |
| `docs` | Documentation only changes |
| `style` | Code style changes (formatting, semicolons, etc.) |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `perf` | A code change that improves performance |
| `test` | Adding or updating tests |
| `build` | Changes to build system or dependencies |
| `ci` | Changes to CI configuration |
| `chore` | Other changes that don't modify src or test files |

### Scopes

| Scope | Description |
|-------|-------------|
| `auth` | Authentication module |
| `org` | Organization module |
| `catalog` | Catalog module |
| `ingest` | Data ingestion pipeline |
| `impact` | Impact analysis engine |
| `search` | Search module |
| `web` | Frontend application |
| `cli` | CLI tool |
| `db` | Database migrations |
| `docker` | Docker/infrastructure |
| `deps` | Dependency updates |

### Examples

```
feat(ingest): add npm workspace detection for monorepos
fix(catalog): handle repos with no default branch
docs: update setup instructions in README
refactor(auth): extract JWT validation into middleware
test(impact): add blast radius calculation tests
ci: add Go lint step to PR workflow
```

### Rules

- Use imperative mood in the description ("add" not "added", "fix" not "fixed").
- Do not capitalize the first letter of the description.
- Do not end the description with a period.
- Keep the first line under 72 characters.
- Use the body to explain **what** and **why**, not **how**.

## Coding Standards

### Go

- Follow [Effective Go](https://go.dev/doc/effective_go) and the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).
- Run `make lint` before submitting. We use `golangci-lint`.
- Write table-driven tests where applicable.
- No global state. Pass dependencies explicitly.
- Error messages should be lowercase and not end with punctuation.
- No comments unless explaining WHY, not WHAT.

### TypeScript / React

- Use functional components and hooks.
- Use TypeScript strict mode.
- Follow the feature-based folder structure.
- Use TanStack Query for server state — no Redux or Zustand for API data.
- Run `make lint-web` before submitting.

### SQL

- Migrations are sequential and immutable once merged to `main`.
- Use `snake_case` for table and column names.
- Every table must have `id`, `created_at`, and `updated_at` columns.
- Add indexes for foreign keys and frequently queried columns.

## Pull Request Process

1. Fill in the PR template completely.
2. Ensure all CI checks pass.
3. PRs require at least one approving review.
4. Keep PRs focused — one feature or fix per PR.
5. If the PR is large (400+ lines), consider splitting it.
6. Update documentation if your change affects public APIs or behavior.

## Dependency Policy

- **pnpm only** — npm and yarn are blocked via preinstall script.
- **Official registry only** — All packages must come from `registry.npmjs.org`. CI verifies this.
- **Frozen lockfile in CI** — The lockfile cannot be modified during CI builds.
- **Audit on every PR** — `pnpm audit --audit-level=high` runs in CI and blocks merges with known vulnerabilities.
- **No install scripts by default** — `.npmrc` sets `ignore-scripts=true` to prevent supply chain attacks.

If you need to add a new dependency, include a justification in the PR description.

## Development Commands

```bash
make dev-up        # Start PostgreSQL via Docker Compose
make dev-down      # Stop PostgreSQL
make run-server    # Run the API server
make run-web       # Run the frontend dev server
make test          # Run all Go tests
make test-web      # Run frontend tests
make lint          # Lint Go code
make lint-web      # Lint frontend code
make audit-web     # Audit frontend deps for vulnerabilities
make migrate-up    # Run database migrations
make migrate-down  # Rollback last migration
make build         # Build all binaries
make build-cli     # Build CLI only
```

## Questions?

If you have questions about contributing, open a [Discussion](https://github.com/nesbite/atlas/discussions) or reach out at oss@nesbite.com.
