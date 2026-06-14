# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Conventional Commits](https://www.conventionalcommits.org/).

## [Unreleased]

### Added

- Dependency data layer: migration, npm parser, and store with batched upserts (#14)
- CI reinforcement: golangci-lint, govulncheck, CodeQL SAST, PR size guard, conventional commit validation, dependency review

### Changed

- Go version updated from 1.23 to 1.26
- Dockerfile Node version updated from 20 to 22

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
