<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://img.shields.io/badge/Atlas-Engineering%20Intelligence-00d4aa?style=for-the-badge&labelColor=0a0e1a" />
    <img src="https://img.shields.io/badge/Atlas-Engineering%20Intelligence-009d7e?style=for-the-badge&labelColor=f0f2f5" alt="Atlas" />
  </picture>
</p>

<h1 align="center">Atlas</h1>

<p align="center">
  <strong>Map your entire software ecosystem.<br />Know what breaks before you break it.</strong>
</p>

<p align="center">
  <a href="https://github.com/martinhg/atlas/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/martinhg/atlas/ci.yml?style=flat-square&label=CI&logo=githubactions&logoColor=white" alt="CI" /></a>
  <a href="https://github.com/martinhg/atlas/releases/latest"><img src="https://img.shields.io/github/v/release/martinhg/atlas?style=flat-square&color=00d4aa&label=Release" alt="Release" /></a>
  <a href="https://github.com/martinhg/atlas/blob/main/go.mod"><img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.26" /></a>
  <a href="https://github.com/martinhg/atlas/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-BSL_1.1_+_Apache_2.0-3b82f6?style=flat-square" alt="License" /></a>
  <a href="https://github.com/martinhg/atlas/blob/main/CONTRIBUTING.md"><img src="https://img.shields.io/badge/PRs-welcome-22c55e?style=flat-square" alt="PRs Welcome" /></a>
</p>

<p align="center">
  <a href="#-cli">CLI</a>&ensp;&bull;&ensp;<a href="#-platform-features">Platform</a>&ensp;&bull;&ensp;<a href="#-quick-start">Quick Start</a>&ensp;&bull;&ensp;<a href="ARCHITECTURE.md">Architecture</a>&ensp;&bull;&ensp;<a href="CONTRIBUTING.md">Contributing</a>
</p>

<br />

## What Atlas Does

Atlas gives engineering teams a **living map** of their software ecosystem — repositories, dependencies, services, teams, and risk — all connected.

| Question | Atlas answers it with |
|---|---|
| **What do we use?** | Complete dependency inventory across all repos and 8 ecosystems |
| **Who owns it?** | Automatic ownership detection from CODEOWNERS |
| **What depends on what?** | Interactive dependency graph across the entire org |
| **What breaks if we change X?** | Impact analysis with blast radius, affected repos and teams |
| **What should we fix first?** | Vulnerability dashboard with severity prioritization (OSV.dev) |

<br />

## CLI

The Atlas CLI is **open source** (Apache 2.0) and works standalone — no server, no database, no config.

```bash
go install github.com/nesbite/atlas/cmd/atlas@latest
```

Point it at any directory and get a full dependency report:

```
$ atlas scan --format table

ECOSYSTEM   NAME                          VERSION     TYPE     SOURCE
npm         react                         ^19.2.6     direct   package.json
npm         vite                          ^8.0.12     dev      package.json
Go          github.com/go-chi/chi/v5      v5.2.1      direct   go.mod
Go          github.com/jackc/pgx/v5       v5.7.4      direct   go.mod
PyPI        django                        >=4.2       direct   requirements.txt
Maven       org.springframework:spring    3.4.1       direct   pom.xml
crates.io   serde                         1.0         direct   Cargo.toml
Packagist   laravel/framework             ^11.0       direct   composer.json

42 dependencies found across 5 ecosystems
```

JSON output works too — pipe it wherever you need:

```bash
atlas scan --format json --path ./my-project | jq '.dependencies | length'
```

### Supported Ecosystems

<table>
  <tr>
    <td><img src="https://img.shields.io/badge/npm-package.json-CB3837?style=flat-square&logo=npm&logoColor=white" alt="npm" /></td>
    <td><img src="https://img.shields.io/badge/Go-go.mod-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go" /></td>
    <td><img src="https://img.shields.io/badge/pip-requirements.txt-3776AB?style=flat-square&logo=python&logoColor=white" alt="pip" /></td>
    <td><img src="https://img.shields.io/badge/Maven-pom.xml-C71A36?style=flat-square&logo=apachemaven&logoColor=white" alt="Maven" /></td>
  </tr>
  <tr>
    <td><img src="https://img.shields.io/badge/Cargo-Cargo.toml-DEA584?style=flat-square&logo=rust&logoColor=white" alt="Cargo" /></td>
    <td><img src="https://img.shields.io/badge/PyProject-pyproject.toml-3776AB?style=flat-square&logo=python&logoColor=white" alt="PyProject" /></td>
    <td><img src="https://img.shields.io/badge/Composer-composer.json-885630?style=flat-square&logo=composer&logoColor=white" alt="Composer" /></td>
    <td><img src="https://img.shields.io/badge/Gemfile-Gemfile-CC342D?style=flat-square&logo=rubygems&logoColor=white" alt="Gemfile" /></td>
  </tr>
</table>

Every parser is a **pure function** — takes bytes in, returns structured dependencies out. No I/O, no state, trivially testable.

<br />

## Platform Features

The CLI scans locally. The **platform** connects everything across your organization.

<table>
  <tr>
    <td width="50%">
      <h3>Dependency Graph</h3>
      <p>Interactive visualization with Sigma.js. Repos, dependencies, and teams as a single navigable graph. Filter by ecosystem, risk level, or team.</p>
    </td>
    <td width="50%">
      <h3>Impact Analysis</h3>
      <p>Change one dependency — see every repo and team affected. Blast radius calculation with heuristic risk scoring and version distribution.</p>
    </td>
  </tr>
  <tr>
    <td width="50%">
      <h3>Vulnerability Dashboard</h3>
      <p>OSV.dev integration with batch querying, semver range matching, and severity prioritization. Know what to fix first.</p>
    </td>
    <td width="50%">
      <h3>Ownership Detection</h3>
      <p>Parses CODEOWNERS from 3 standard paths. Automatic team attribution across repos. Know who owns what before you need to ask.</p>
    </td>
  </tr>
  <tr>
    <td width="50%">
      <h3>Multi-Ecosystem Sync</h3>
      <p>Server discovers and parses all 8 manifest types from GitHub repos automatically. One sync, complete inventory.</p>
    </td>
    <td width="50%">
      <h3>GitHub Integration</h3>
      <p>OAuth login, GitHub App for repo discovery, webhook-triggered sync. Connect your org and Atlas does the rest.</p>
    </td>
  </tr>
</table>

<br />

## Quick Start

### CLI only (Apache 2.0)

```bash
go install github.com/nesbite/atlas/cmd/atlas@latest
atlas scan
```

### Full platform

> [!NOTE]
> Requires Go 1.26+, Node.js 22+, pnpm 11+, and Docker.

```bash
make dev-up          # Start PostgreSQL
make run-server      # API server on :8080
make run-web         # Frontend on :5173
```

The API runs at `http://localhost:8080` and the frontend at `http://localhost:5173`.

<br />

## Tech Stack

| Layer | Technologies |
|---|---|
| **Backend** | Go 1.26, chi router, pgx/pgxpool, godotenv |
| **Frontend** | React 19, Vite 8, TypeScript 6, Tailwind CSS v4, shadcn/ui |
| **Database** | PostgreSQL 16 |
| **Auth** | GitHub OAuth + HS256 JWT |
| **CLI** | Standalone Go binary, 8 ecosystem scanners, zero config |
| **CI** | GitHub Actions, golangci-lint, govulncheck, Vitest, CodeQL SAST |

<br />

## Project Structure

```
cmd/
  atlas-server/              API server entrypoint
  atlas/                     CLI tool (Apache 2.0)
internal/
  auth/                      GitHub OAuth + JWT authentication
  catalog/                   Repository store and listing
  dependency/                Multi-ecosystem dependency sync + querying
  ownership/                 CODEOWNERS parsing + ownership detection
  impact/                    Blast radius analysis + risk scoring
  vuln/                      OSV.dev vulnerability sync, matching, and dashboard
  graph/                     Dependency graph aggregation + server-side filters
  risk/                      Shared risk heuristic (reused by impact + graph)
  scan/                      CLI scan engine + ecosystem scanners
  ingest/parsers/            Pure-function parsers for 8 ecosystems (Apache 2.0)
    depmodel/                Shared ParsedDep types + ecosystem constants
    npm/                     package.json parser
    composer/                composer.json parser
    gomod/                   go.mod parser
    pip/                     requirements.txt parser
    maven/                   pom.xml parser
    cargo/                   Cargo.toml parser
    pyproject/               pyproject.toml parser (PEP 621)
    gemfile/                 Gemfile parser
    codeowners/              CODEOWNERS parser
  org/                       Organization management + sync orchestration
  platform/
    config/                  Environment configuration (godotenv)
    database/                pgxpool connection + migration runner
    github/                  GitHub App client factory
migrations/                  SQL migrations (auto-embedded, auto-run on startup)
web/                         React SPA (Vite + Tailwind v4 + shadcn/ui)
  src/features/              Feature modules (catalog, dependencies, ownership,
                             impact, vulnerabilities, graph)
  src/components/            Shared components + shadcn primitives
  src/lib/                   API client, auth, utilities
deploy/                      Docker + Compose
```

For a deeper dive, see [ARCHITECTURE.md](ARCHITECTURE.md).

<br />

## Roadmap

- [x] **Epic 1** — Authentication (GitHub OAuth + JWT)
- [x] **Epic 2** — Repository Discovery & Sync
- [x] **Epic 3** — Dependency Parsing (npm)
- [x] **Epic 4** — Ownership Detection (CODEOWNERS)
- [x] **Epic 5** — Search
- [x] **v1.0.0** — MVP Phase 1 Complete
- [x] **Epic 6** — Impact Analysis (Blast Radius)
- [x] **Epic 8** — Vulnerabilities & Risk Dashboard (OSV.dev)
- [x] **v1.1.0** — Deps → Impact → Risk chain complete
- [x] **Epic 7** — Dependency Graph Visualization (Sigma.js)
- [x] **v1.2.0** — Interactive dependency graph
- [x] **Epic 9** — CLI `atlas scan` + Multi-Ecosystem Parsers
- [x] **v1.3.0** — CLI + 8 ecosystem parsers

<br />

## License

Atlas uses a **dual-license** model:

| Component | License | Paths |
|---|---|---|
| **CLI + Parsers** | [Apache License 2.0](cmd/atlas/LICENSE) | `cmd/atlas/`, `internal/ingest/parsers/` |
| **Server + Web App** | [Business Source License 1.1](LICENSE) | Everything else |

<br />

---

<p align="center">
  <a href="ARCHITECTURE.md">Architecture</a>&ensp;&bull;&ensp;<a href="CONTRIBUTING.md">Contributing</a>&ensp;&bull;&ensp;<a href="SECURITY.md">Security Policy</a>&ensp;&bull;&ensp;<a href="CODE_OF_CONDUCT.md">Code of Conduct</a>&ensp;&bull;&ensp;<a href="CHANGELOG.md">Changelog</a>
  <br /><br />
  Built by <a href="https://github.com/martinhg">Nesbite</a>
</p>
