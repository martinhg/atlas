# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Conventional Commits](https://www.conventionalcommits.org/).

## [Unreleased]

### Added

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
