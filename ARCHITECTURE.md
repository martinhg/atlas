# Architecture

This document describes the high-level architecture of Atlas. It is intended for contributors and anyone who wants to understand how the system fits together.

## System Overview

Atlas is a full-stack application with a Go API server, a React SPA frontend, and PostgreSQL for persistence. GitHub is the primary external integration — Atlas authenticates users via GitHub OAuth and discovers repositories via a GitHub App.

```
┌─────────────┐     ┌──────────────┐     ┌────────────┐
│  React SPA   │────▶│  Go API      │────▶│ PostgreSQL │
│  (Vite)      │◀────│  (chi)       │◀────│            │
└─────────────┘     └──────┬───────┘     └────────────┘
                           │
                    ┌──────┴───────┐
                    │  GitHub API  │
                    │  (App + OAuth)│
                    └──────────────┘
```

## Backend

### Domain Packages

Each domain lives in its own package under `internal/` and follows the same structure: **model → store → handler**.

| Package | Responsibility |
|---------|---------------|
| `auth` | GitHub OAuth flow, JWT issuance/refresh, auth middleware |
| `org` | Organization CRUD, GitHub App installation, repository sync orchestration |
| `catalog` | Repository storage and listing |
| `dependency` | Dependency parsing (npm), storage, and querying |
| `ownership` | CODEOWNERS parsing, ownership storage, and querying |
| `impact` | Blast radius analysis: dependency → affected repos → affected teams, risk scoring |
| `vuln` | OSV.dev vulnerability sync (batch query + hydrate), semver range matching, dashboard list/detail |
| `search` (via existing packages) | ILIKE filtering on repos and dependencies via `?q=` query param |
| `platform/config` | Environment variable loading via godotenv |
| `platform/database` | pgxpool connection, custom migration runner with advisory locking |
| `platform/github` | GitHub App client factory (JWT → installation token) |

### Handler/Store Pattern

Every domain follows the same dependency injection pattern:

```
Handler(store Interface, resolver Interface, ...) → HTTP handlers
Store(pgxpool.Pool) → database operations
```

- **Handlers** receive dependencies via constructor (`NewHandler`), never via globals.
- **Stores** implement interfaces declared in the same package (Go interface segregation).
- **Cross-domain dependencies** use narrow interfaces. For example, the catalog handler declares its own `OrgResolver` interface rather than importing the full `org.OrgStore`.

### Routing

All API routes live under `/api/v1` and are registered in `cmd/atlas-server/main.go` using chi groups:

```
/api/v1/auth/github/login       GET   — OAuth redirect
/api/v1/auth/github/callback    GET   — OAuth callback
/api/v1/auth/refresh            POST  — JWT refresh
/api/v1/auth/me                 GET   — Current user (protected)
/api/v1/webhooks/github         POST  — GitHub App webhooks
/api/v1/orgs                    (group) — Org CRUD + connect
/api/v1/orgs/{slug}/repos                GET  — List repos by org slug
/api/v1/orgs/{slug}/repos/{name}         GET  — Repository detail
/api/v1/orgs/{slug}/repos/{name}/dependencies GET — Dependencies for a repo
/api/v1/orgs/{slug}/dependencies         GET  — List dependencies
/api/v1/orgs/{slug}/dependencies/{eco}/* GET  — Dependency detail
/api/v1/orgs/{slug}/ownership            GET  — List ownership (paginated)
/api/v1/orgs/{slug}/ownership/{repo}     GET  — Ownership detail for a repo
/api/v1/orgs/{slug}/impact               POST — Impact analysis (blast radius)
/api/v1/orgs/{slug}/vulnerabilities      GET  — List vulnerabilities (?severity, ?package)
/api/v1/orgs/{slug}/vulnerabilities/{id} GET  — Vulnerability detail + affected repos
```

All org-scoped routes use `{slug}` (human-readable) as the org identifier. Handlers resolve slug → UUID internally via `OrgResolver`.

### Authentication Flow

1. Frontend redirects to `/api/v1/auth/github/login`
2. Server redirects to GitHub OAuth with client ID
3. GitHub redirects back to `/api/v1/auth/github/callback` with code
4. Server exchanges code for GitHub access token, fetches user profile
5. Server upserts user, issues HS256 JWT (access + refresh)
6. Frontend stores tokens, attaches JWT to all API requests via `apiFetch`
7. On 401, frontend auto-refreshes the access token using the refresh token

### Sync Flow

When a GitHub App installation event is received (or a user connects an org):

1. `org.Handler` receives the webhook / connect request
2. Spawns a goroutine calling `syncRepos()`
3. `syncRepos` fetches all repos from GitHub API, upserts each via `catalog.RepoStore`
4. For each repo, if a `DepSyncer` is injected, it triggers dependency sync; if an `OwnershipSyncer` is injected, it triggers ownership sync
5. `dependency.Service.SyncRepoDependencies` discovers `package.json` files via GitHub tree API, fetches content, parses, and batch-upserts dependencies
6. `ownership.Service.SyncRepoOwnership` tries 3 CODEOWNERS paths (CODEOWNERS, .github/CODEOWNERS, docs/CODEOWNERS), parses, and batch-upserts ownership rows; errors are isolated per repo
7. After the repo loop, if a `VulnSyncer` is injected, `vuln.Service.SyncOrgVulns` collects the org's unique dependencies, batch-queries OSV.dev (chunked at 100, then hydrates each vuln ID via `GET /v1/vulns/{id}`), matches semver ranges in Go, and rebuilds the `dependency_vulnerabilities` links. This step is non-blocking: an OSV failure is logged and never aborts or rolls back the dependency sync

### Migrations

SQL migrations live in `migrations/` and are auto-embedded via `embed.FS`. They run on server startup with `pg_advisory_lock(42)` to prevent races when multiple test packages run in parallel.

```
000001_create_users.up.sql
000002_create_organizations.up.sql
000003_create_repositories.up.sql
000004_create_dependencies.up.sql
000005_create_repo_owners.up.sql
000006_add_search_indexes.up.sql
000007_create_vulnerabilities.up.sql
```

## Frontend

### Stack

React 19, Vite 8, TypeScript, Tailwind CSS v4, shadcn/ui, TanStack Query v5.

### Structure

```
web/src/
├── components/          Shared components (DashboardPage, LoginPage, AuthGuard)
│   └── ui/              shadcn primitives (Button, Card, Avatar, Input)
├── features/            Feature modules (catalog, dependencies, ownership, impact)
│   ├── catalog/         RepoListPage, RepoDetailPage, RepoTable, useRepos, useRepoDetail, useRepoDeps
│   ├── dependencies/    DependencyListPage, DependencyDetailPage, hooks, tables
│   ├── impact/          ImpactAnalysisPage, ImpactResultTable, useImpactAnalysis
│   ├── ownership/       OwnershipListPage, OwnershipDetailPage, hooks, tables with type badges
│   └── vulnerabilities/ VulnerabilityListPage, VulnerabilityDetailPage, SeverityBadge, table, hooks
├── hooks/               Shared hooks (useOrgs)
├── lib/                 Utilities (api.ts, auth.ts, query-client.ts)
├── pages/               Standalone pages (GitHubCallbackPage)
└── router.tsx           Route definitions
```

### Conventions

- **UI primitives**: always shadcn from `@/components/ui/` — never custom
- **Data fetching**: TanStack Query hooks per feature (`useRepos`, `useDependencies`)
- **API client**: `apiFetch` from `@/lib/auth` auto-attaches JWT, auto-refreshes on 401
- **Routing**: all org-scoped routes use `:slug` param (e.g. `/orgs/:slug/repos`)
- **Styling**: dark-only zinc palette (`bg-zinc-950`, `text-zinc-100`, `border-zinc-800`)
- **Imports**: always use `@/` path alias

### Navigation Graph

```
Login → Dashboard → Repositories (per org) → Repository Detail (deps + ownership)
                  → Dependencies (per org) → Dependency Detail → Impact Analysis (pre-filled)
                  →                                            → Vulnerabilities (filtered by package)
                  → Ownership (per org)    → Ownership Detail (per repo)
                  → Impact Analysis (per org) — form: dependency + ecosystem → blast radius results
                  → Vulnerabilities (per org) → Vulnerability Detail (affected repos + teams)
```

Cross-links exist between Dashboard, Repositories, Dependencies, Ownership, Impact Analysis, and Vulnerabilities pages via breadcrumb navigation. Dependency Detail has an "Analyze Impact" button (navigates to Impact Analysis pre-filled) and a "Known Vulnerabilities" section; the dependency table's vulnerability count links to the vulnerability dashboard filtered by package.

## Data Model

```
users
  ├── id (UUID, PK)
  ├── github_id (BIGINT, unique)
  ├── login, name, avatar_url
  └── created_at, updated_at

organizations
  ├── id (UUID, PK)
  ├── github_id (BIGINT, unique)
  ├── name, slug (unique)
  ├── github_installation_id
  ├── owner_id → users.id
  └── last_synced_at, created_at, updated_at

repositories
  ├── id (UUID, PK)
  ├── org_id → organizations.id
  ├── github_id (BIGINT, unique)
  ├── name, full_name, description
  ├── default_branch, language, private, fork, stars
  └── last_synced_at, created_at, updated_at

dependencies
  ├── id (UUID, PK)
  ├── ecosystem, name (unique together)
  └── created_at, updated_at

repo_dependencies
  ├── repo_id → repositories.id
  ├── dep_id → dependencies.id
  ├── version (the declared range, e.g. "^4.17.21")
  ├── dep_type (direct, dev, peer, optional)
  ├── source_file
  └── created_at, updated_at (unique: repo_id, dep_id, source_file)

repo_owners
  ├── repo_id → repositories.id
  ├── pattern (CODEOWNERS glob pattern)
  ├── owner (team, username, or email)
  ├── owner_type (team, user, email)
  ├── source (codeowners)
  ├── line_number
  └── created_at, updated_at

vulnerabilities
  ├── id (UUID, PK)
  ├── osv_id (unique), cve_id
  ├── ecosystem, package_name
  ├── severity (critical/high/medium/low/unknown), cvss_score, cvss_vector
  ├── summary, details, fixed_version, introduced_version
  ├── affected_ranges (JSONB: [{introduced, fixed}])
  └── published_at, modified_at, created_at, updated_at

dependency_vulnerabilities
  ├── dep_id → dependencies.id (ON DELETE CASCADE)
  ├── vuln_id → vulnerabilities.id (ON DELETE CASCADE)
  └── created_at (unique: dep_id, vuln_id)
```

## CI Pipeline

GitHub Actions runs on every PR and push to main:

- **Go**: `go test -race`, `go vet`, `golangci-lint` (only-new-issues), `govulncheck`
- **Frontend**: `pnpm lint`, `tsc --noEmit`, `vitest run`, `pnpm audit`
- **PR checks**: conventional commit validation, PR size guard (warns >400 lines), dependency review
- **Security**: CodeQL SAST (Go + TypeScript, weekly schedule)

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Slug-based org URLs | Human-readable URLs; handlers resolve slug → UUID internally |
| Custom migration runner | Avoids golang-migrate dependency; advisory lock prevents CI race conditions |
| Interface segregation | Each package declares only the interfaces it needs (e.g. `catalog.OrgResolver` vs `dependency.OrgResolver`) |
| No ORM | pgx/pgxpool direct queries for full control and performance |
| Feature-based frontend | Each domain (catalog, dependencies) is self-contained with its own pages, hooks, and components |
| pnpm only | Security-first: frozen lockfile, `ignore-scripts=true`, registry enforcement |
| ILIKE over FTS for search | At <1000 repos per org, ILIKE is <5ms; `text_pattern_ops` indexes accelerate prefix queries; pg_trgm is the documented upgrade path |
| Apps.ListRepos over ListByOrg | Works for both organization and personal GitHub accounts; ListByOrg 404s on personal accounts |
| Normalized dependency model | `dependencies` + `repo_dependencies` junction avoids duplicate rows, enables `repo_count` aggregation |
| OSV.dev as vulnerability source | No auth, batch endpoint, aggregates GHSA+NVD; querybatch returns only IDs so `vuln.OSVClient` hydrates each via `GET /v1/vulns/{id}` |
| Sync-time version matching in Go | `dependency_vulnerabilities` is populated at sync time via `stripRange`/`compareSemver`/`isAffected`, keeping dashboard SQL simple (JOIN through the junction) |
| Cross-domain coupling at the SQL layer only | `dependency.ListByOrg` LEFT JOINs the vuln tables for counts/max-severity without importing the `vuln` package; the dependency detail page reuses the vuln list endpoint via `?package=` |

## Releases

- **v1.0.0** (PR #29) — MVP Phase 1: auth, catalog, dependencies, ownership, search
- **v1.0.1** (PR #32) — Hotfixes: personal GitHub account support, auth token refresh, sync race conditions
- **v1.1.0** (PR #33, #34, #35) — Impact Analysis (blast radius) + Vulnerabilities & Risk Dashboard (OSV.dev); completes the deps → impact → risk chain
