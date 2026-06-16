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

### Migrations

SQL migrations live in `migrations/` and are auto-embedded via `embed.FS`. They run on server startup with `pg_advisory_lock(42)` to prevent races when multiple test packages run in parallel.

```
000001_create_users.up.sql
000002_create_organizations.up.sql
000003_create_repositories.up.sql
000004_create_dependencies.up.sql
000005_create_repo_owners.up.sql
000006_add_search_indexes.up.sql
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
│   └── ownership/       OwnershipListPage, OwnershipDetailPage, hooks, tables with type badges
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
                  → Ownership (per org)    → Ownership Detail (per repo)
                  → Impact Analysis (per org) — form: dependency + ecosystem → blast radius results
```

Cross-links exist between Dashboard, Repositories, Dependencies, Ownership, and Impact Analysis pages via breadcrumb navigation. Dependency Detail has an "Analyze Impact" button that navigates to Impact Analysis pre-filled.

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
  ├── ecosystem, name, version (unique together)
  └── created_at, updated_at

repo_dependencies
  ├── repo_id → repositories.id
  ├── dependency_id → dependencies.id
  ├── dep_type (direct, dev, peer, optional)
  ├── source_file
  └── created_at, updated_at

repo_owners
  ├── repo_id → repositories.id
  ├── pattern (CODEOWNERS glob pattern)
  ├── owner (team, username, or email)
  ├── owner_type (team, user, email)
  ├── source (codeowners)
  ├── line_number
  └── created_at, updated_at
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

## Releases

- **v1.0.0** (PR #29) — MVP Phase 1: auth, catalog, dependencies, ownership, search
- **v1.0.1** (PR #32) — Hotfixes: personal GitHub account support, auth token refresh, sync race conditions
