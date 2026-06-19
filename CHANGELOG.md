# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Conventional Commits](https://www.conventionalcommits.org/).

## [Unreleased]

## [1.1.0] - 2026-06-19

Deps → Impact → Risk. This release completes the chain: from "what depends on
what" to "what breaks if I change it" to "which dependencies are vulnerable."

### Added

- **Vulnerabilities & Risk Dashboard** (Epic 8) — OSV.dev integration tracking
  known vulnerabilities across org dependencies (#35)
  - Migration 000007: `vulnerabilities` + `dependency_vulnerabilities` tables
  - `internal/vuln/` domain: OSV batch client (two-phase query + hydrate), sync
    service, semver range matching, store, and list/detail handlers
  - Vuln sync hooked into org sync as a non-blocking per-org `VulnSyncer` step
  - Vulnerability dashboard: list page with severity filter + detail page with
    affected repositories and team attribution
  - Dependency pages: vulnerability count column with highest-severity badge, and
    a "Known Vulnerabilities" section on the dependency detail page
  - API: `GET /orgs/{slug}/vulnerabilities` (`?severity=`, `?package=`) and
    `GET /orgs/{slug}/vulnerabilities/{id}`
- **Impact Analysis — Blast Radius** (Epic 6) — answer "what breaks if I change
  dependency X?" (#33, #34)
  - `internal/impact/` domain: single-query blast radius (dependency → repos →
    teams) with heuristic risk scoring and version distribution
  - API: `POST /orgs/{slug}/impact` returning affected repos, teams, version
    spread, and a risk score/level
  - Frontend: `ImpactAnalysisPage` with dependency/ecosystem form and results
    table; "Analyze Impact" deep-link from the dependency detail page

### Changed

- Vulnerability sync runs after dependency sync in the org pipeline and never
  blocks or rolls back the parent sync if OSV is unavailable.

### Security

- Forced `undici` to `>=7.28.0` via pnpm overrides to clear a high-severity
  transitive advisory (GHSA-vmh5-mc38-953g) in the test toolchain.

## [1.0.1] - 2026-06-15

### Fixed

- Sync now works with personal GitHub accounts, not just organizations (#31)
- Dependency sync race condition (`no rows in result set`) on concurrent upserts (#31)
- Dependency sync deadlocks via consistent lock ordering (#31)
- Token refresh on full-page navigation when in-memory access token is lost (#31)
- Private key env var parsing with base64 padding variants and trailing whitespace (#31)

## [1.0.0] - 2026-06-15

MVP Phase 1 complete — Atlas is a usable Engineering Intelligence Platform.

### Added

- Repository detail page composing repo info, dependencies, and ownership (#28)
- Two new API endpoints: `GET /repos/{name}` and `GET /repos/{name}/dependencies` (#28)
- Repo names in list table link to detail page (#28)
- Search (Phase 1): ILIKE filtering (`?q=`) on repos and dependencies endpoints (#26)
- Search (Phase 1): debounced search inputs on RepoListPage and DependencyListPage (#27)
- Pagination on repos endpoint with `{data, total, page, per_page}` envelope (#26)
- Migration 000006: `text_pattern_ops` B-tree indexes on repositories.name and full_name (#26)
- shadcn Input component (#27)
- Ownership detection (Phase 1): CODEOWNERS parser with BOM/CRLF handling and 20 test cases (#22)
- Ownership store with sync, paginated list, and detail queries (#22)
- Ownership service with 3-path CODEOWNERS fetch and error isolation (#23)
- Ownership handler with list and detail API endpoints (#23)
- Ownership sync integration via OwnershipSyncer in org.syncRepos (#23)
- Ownership frontend: list page, detail page, table components with type badges (#24)
- Navigation cross-links: Dashboard, Repos, and Dependencies now link to Ownership (#24)
- Dependency parsing (Phase 1): data layer with npm parser and batched upserts (#14)
- Dependency service, handler, sync integration, and API routes (#17)
- Dependency list and detail pages with TanStack Query hooks (#18)
- Cross-navigation links between Repositories and Dependencies pages (#19)
- ARCHITECTURE.md documenting system design, domain model, and key decisions (#20)
- CI reinforcement: golangci-lint v2, govulncheck, CodeQL SAST, PR size guard, conventional commit validation, dependency review, CODEOWNERS (#15)

### Changed

- Standardized all org-scoped URL params from mixed `{orgID}`/`{slug}` to consistent `{slug}` (#19)
- Go version updated from 1.23 to 1.26
- Dockerfile Node version updated from 20 to 22
- README overhauled with visual identity, updated guides and references (#16)

### Fixed

- Removed invalid `role="link"` from clickable table rows for a11y compliance (#19)

## [0.2.0] - 2026-06-13

### Added

- Repository Discovery & Sync via GitHub App (#8, #9, #11, #12)
  - GitHub App infrastructure with installation client
  - Organization and repository models with PostgreSQL store
  - Webhook receiver for installation events
  - Goroutine-based repository sync
  - Catalog store with repos endpoint
  - Frontend dashboard with repository list views
  - Frontend router with protected routes
- Developer onboarding `.env.example` (#10)
- Coverage reinforcement across Go and web test suites (#13)

## [0.1.0] - 2026-06-12

### Added

- Project scaffold: Go monorepo with CLI and server (#1)
- GitHub OAuth login with JWT session management (#1)
- shadcn/ui design system with dark zinc theme (#2)
- AI assistant skills and project conventions (#3)
- CI pipeline with Go tests, web tests, and coverage enforcement (#5, #6, #7)

### Fixed

- AtlasOS renamed to Atlas across all code, docs, and templates (#4)
- esbuild high-severity vulnerability resolved (#4)
