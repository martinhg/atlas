<p align="center">
  <h1 align="center">Atlas</h1>
  <p align="center">
    <strong>Engineering Intelligence Platform</strong>
    <br />
    Map your entire software ecosystem. Know what breaks before you break it.
  </p>
</p>

<p align="center">
  <a href="https://github.com/martinhg/atlas/actions/workflows/ci.yml">
    <img src="https://github.com/martinhg/atlas/actions/workflows/ci.yml/badge.svg" alt="CI" />
  </a>
  <a href="https://github.com/martinhg/atlas/blob/main/go.mod">
    <img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" alt="Go 1.26" />
  </a>
  <a href="https://github.com/martinhg/atlas/blob/main/LICENSE">
    <img src="https://img.shields.io/badge/License-BSL_1.1-blue" alt="License" />
  </a>
  <a href="https://github.com/martinhg/atlas/blob/main/CONTRIBUTING.md">
    <img src="https://img.shields.io/badge/PRs-welcome-brightgreen" alt="PRs Welcome" />
  </a>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> &middot;
  <a href="ARCHITECTURE.md">Architecture</a> &middot;
  <a href="CONTRIBUTING.md">Contributing</a> &middot;
  <a href="SECURITY.md">Security</a> &middot;
  <a href="LICENSE">License</a>
</p>

<hr />

## What Atlas Does

Atlas gives engineering teams a living map of their software ecosystem — repositories, dependencies, services, teams, and risk — all connected.

<table>
  <tr>
    <td><strong>What do we use?</strong></td>
    <td>Complete dependency inventory across all repositories</td>
  </tr>
  <tr>
    <td><strong>Who owns it?</strong></td>
    <td>Automatic ownership detection from CODEOWNERS, commits, and PRs</td>
  </tr>
  <tr>
    <td><strong>What depends on what?</strong></td>
    <td>Dependency graph across the entire organization</td>
  </tr>
  <tr>
    <td><strong>What breaks if we change X?</strong></td>
    <td>Impact analysis with blast radius</td>
  </tr>
  <tr>
    <td><strong>What should we fix first?</strong></td>
    <td>Risk prioritization with organizational context</td>
  </tr>
</table>

<hr />

## Quick Start

### Prerequisites

- Go 1.26+
- Node.js 22+
- pnpm 11+
- Docker

### Setup

```bash
make dev-up          # Start PostgreSQL
make run-server      # Start API server (port 8080)
make run-web         # Start frontend dev server (port 5173)
```

The API runs at `http://localhost:8080` and the frontend at `http://localhost:5173`.

<hr />

## Tech Stack

<table>
  <tr>
    <td><strong>Backend</strong></td>
    <td>Go, chi, pgx/pgxpool</td>
  </tr>
  <tr>
    <td><strong>Frontend</strong></td>
    <td>React 19, Vite 8, TypeScript, Tailwind CSS v4, shadcn/ui</td>
  </tr>
  <tr>
    <td><strong>Database</strong></td>
    <td>PostgreSQL 16</td>
  </tr>
  <tr>
    <td><strong>Auth</strong></td>
    <td>GitHub OAuth + HS256 JWT</td>
  </tr>
  <tr>
    <td><strong>CI</strong></td>
    <td>GitHub Actions, golangci-lint, govulncheck, Vitest</td>
  </tr>
</table>

<hr />

## Project Structure

```
cmd/
  atlas-server/            API server entrypoint
  atlas/                   CLI tool (Apache 2.0)
internal/
  auth/                    GitHub OAuth + JWT authentication
  catalog/                 Repository store and listing
  dependency/              Dependency parsing, storage, and querying
    parser/                npm package.json parser
  ingest/parsers/          CLI parsers (Apache 2.0)
  org/                     Organization management + sync orchestration
  platform/
    config/                Environment configuration (godotenv)
    database/              pgxpool connection + migration runner
    github/                GitHub App client factory
migrations/                SQL migrations (auto-embedded, auto-run on startup)
web/                       React SPA (Vite + Tailwind v4 + shadcn/ui)
  src/features/            Feature modules (catalog, dependencies)
  src/components/          Shared components + shadcn primitives
  src/lib/                 API client, auth, utilities
deploy/                    Docker + Compose
```

For a deeper dive into how these pieces fit together, see [ARCHITECTURE.md](ARCHITECTURE.md).

<hr />

## Roadmap

- [x] **Epic 1** — Authentication (GitHub OAuth + JWT)
- [x] **Epic 2** — Repository Discovery & Sync
- [x] **Epic 3** — Dependency Parsing (npm, Phase 1)
- [ ] **Epic 4** — Ownership Detection
- [ ] **Epic 5** — Search
- [ ] **Epic 6** — Dependency Graph Visualization
- [ ] **Epic 7** — Impact Analysis
- [ ] **Epic 8** — Risk Dashboard

<hr />

## CLI

The Atlas CLI is open source (Apache 2.0) and can be used standalone:

```bash
go install github.com/nesbite/atlas/cmd/atlas@latest
atlas scan
```

<hr />

## License

Atlas uses a dual-license model:

- **CLI and parsers** — [Apache License 2.0](cmd/atlas/LICENSE)
- **Server and web app** — [Business Source License 1.1](LICENSE)

See [LICENSE](LICENSE) for details.

<hr />

<p align="center">
  <a href="ARCHITECTURE.md">Architecture</a> &middot;
  <a href="CONTRIBUTING.md">Contributing</a> &middot;
  <a href="SECURITY.md">Security Policy</a> &middot;
  <a href="CODE_OF_CONDUCT.md">Code of Conduct</a>
  <br />
  Built by <a href="https://github.com/martinhg">Nesbite</a>
</p>
