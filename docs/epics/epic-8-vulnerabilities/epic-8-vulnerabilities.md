# Epic 8 — Vulnerabilities & Risk Dashboard

**Status:** Complete & archived
**Delivered in:** PR1 → PR4 (stacked-to-main)
**Closed:** 2026-06-18

Atlas tracks dependencies and their blast radius. Epic 8 completes the
deps → impact → risk chain by answering: *which dependencies have known
security vulnerabilities?* Vulnerability data is sourced from
[OSV.dev](https://osv.dev) at org-sync time and surfaced through a dashboard and
on the existing dependency pages.

---

## Scope

**In scope (delivered):** OSV.dev batch integration, `vulnerabilities` +
`dependency_vulnerabilities` tables, `internal/vuln/` domain, list/detail API
endpoints, vulnerability dashboard (list + detail pages), severity badges on
dependency pages, vuln sync hooked into the org sync pipeline.

**Out of scope (deliberate):** lock-file parsing, CVSS vector calculation,
periodic re-scans / scheduler, remediation suggestions, multi-ecosystem
(non-npm), auto-PR creation. See [archived.md](./archived.md) for the deferred
backlog and technical debt.

---

## PR breakdown

| PR | Slice | Key files |
|----|-------|-----------|
| PR1 | Foundation: schema + domain types + semver utils | `migrations/000007_create_vulnerabilities.up.sql`, `internal/vuln/{model,semver,store}.go` |
| PR2 | Sync pipeline: OSV client + service + org-sync hook | `internal/vuln/{osv,service}.go`, `internal/org/{sync,handler}.go`, `cmd/atlas-server/main.go` |
| PR3 | API + dashboard frontend | `internal/vuln/handler.go`, `web/src/features/vulnerabilities/*`, `web/src/lib/api.ts`, `web/src/router.tsx` |
| PR4 | Dependency ↔ vuln integration | `internal/dependency/{model,store}.go`, `internal/vuln/{store,handler}.go`, `web/src/features/dependencies/*` |

---

## Verification against spec

Final verification run on 2026-06-18. Backend integration tests executed against
a real PostgreSQL 16 instance (`make dev-up`).

- `go build ./...` — clean
- `go test ./...` (incl. integration tests against real DB) — **all pass**
- `golangci-lint run` on touched packages — **0 issues** (one pre-existing
  `errcheck` finding in `internal/dependency/migration_test.go` is unrelated)
- `pnpm typecheck` — clean
- `pnpm lint` — 0 errors (3 warnings live only in generated `coverage/`)
- `pnpm test` — **255 tests pass** (37 files)
- `pnpm build` (tsc + vite) — clean

### Requirement coverage

| Capability | Requirement | Status |
|------------|-------------|--------|
| vulnerability-sync | OSV batch query on dep sync (chunk at 100) | ✅ `osv.go` |
| vulnerability-sync | Vulnerability upsert (CVE from aliases, CVSS V3>V2>V4, severity from score) | ✅ `service.go` |
| vulnerability-sync | Version matching at sync time (`stripRange`/`compareSemver`/`isAffected`) | ✅ `semver.go` |
| vulnerability-sync | VulnSyncer hook in org sync, non-blocking | ✅ `org/sync.go` |
| vulnerability-dashboard | List endpoint (`page`/`perPage`/`severity`, 400 on invalid severity) | ✅ `handler.go` |
| vulnerability-dashboard | Detail endpoint (404 on missing/malformed id) | ✅ `handler.go` |
| vulnerability-dashboard | List + detail pages with severity badges | ✅ `features/vulnerabilities/` |
| dependency-display | Vuln count column + highest-severity badge on `DependencyTable` | ✅ PR4 |
| dependency-display | "Known Vulnerabilities" section on `DependencyDetailPage` | ✅ PR4 |

---

## Architecture decisions

### Planned (from design)

1. **OSV.dev as sole source** — no auth, batch support, aggregates GHSA+NVD.
   Rejected the GitHub Advisory DB (REST/GraphQL) because of PAT/auth overhead and
   no batch support.
2. **Version matching in Go at sync time, not in SQL** — keeps dashboard SQL
   simple (JOIN through the junction table); stdlib semver handling suffices for
   the MVP `X.Y.Z` case. Trade-off: data is stale until the next sync.
3. **Single vuln row per `osv_id`, ranges as JSONB** — avoids a child table and
   per-query joins on `affected_ranges`.
4. **VulnSyncer per-org (not per-repo) after the repo loop** — one batched OSV
   call per sync instead of N small calls; deduplicates packages across repos.
5. **Severity derived from CVSS numeric score** — consistent thresholds
   (≥9.0 critical, ≥7.0 high, ≥4.0 medium, >0 low, none → unknown).

### Made during implementation (deviations / refinements)

6. **OSV `querybatch` is a two-phase call.** The spec/design assumed
   `POST /v1/querybatch` returns full vulnerability objects. It does **not** — it
   returns only vulnerability IDs per package. `OSVClient.QueryBatch` therefore
   does a second hydration step: `GET /v1/vulns/{id}` for each unique ID (deduped),
   producing full records. Without this, severity would always be `unknown` and
   `isAffected` always `false`, so **zero** `dependency_vulnerabilities` rows would
   ever be written. This is the single most important correctness decision in the
   epic.
7. **CVSS score parsed as numeric only; vector→score calculation is out of scope.**
   `extractCVSS` prefers `CVSS_V3 > CVSS_V2 > CVSS_V4` and only `ParseFloat`s a
   numeric score string. Real OSV typically returns `CVSS_V3` as a *vector string*,
   which does not parse to a float → severity falls back to `unknown`. This is a
   documented MVP limitation (see [archived.md](./archived.md), TD-1).
8. **Cross-domain coupling kept at the SQL layer only.**
   - PR4 (A): `dependency.ListByOrg` LEFT JOINs the vuln tables to compute
     `vuln_count` + `max_severity`. The `dependency` package does **not** import
     the `vuln` package — only shared DB tables.
   - PR4 (B): the "Known Vulnerabilities" section on the dependency detail page
     reuses the vuln list endpoint via a new `?package=` filter, so the backend
     never couples `dependency → vuln` in Go.
9. **`max_severity` ranked and mapped in SQL.** `ListByOrg` ranks severities
   numerically via `MAX(CASE ...)` then maps the rank back to its label, returning
   `""` for dependencies with no vulnerabilities.
10. **`SeverityBadge` is a feature component, not a shadcn primitive** (no Badge
    primitive exists in `components/ui/`). It is imported cross-feature by
    `DependencyTable`. The severity filter UI uses a `Button` row (no `Select`
    primitive available).
11. **`DependencyWithCount` TS fields are required.** `vuln_count` / `max_severity`
    are always returned by the backend, so the TS interface marks them required;
    the one typed test literal was updated, cast-based mocks were unaffected. The
    `DependencyTable` still guards `vuln_count ?? 0` for resilience.

---

## Key gotchas (for future maintainers)

- `semver.go` handles a compound range with a space between operator and version
  (`">=  1.0.0"`) by taking `parts[1]` when the first token is a bare operator.
- `store.go GetDetail` returns `nil, nil` on `pgx.ErrNoRows` (not-found is not an
  error), wrapped with `//nolint:nilerr`.
- `ListOrgDepPairs` uses `DISTINCT ON (d.id) ORDER BY d.id, rd.updated_at DESC` to
  pick the most recent version per dependency.
- Vuln list query params are camelCase (`perPage`), matching the handler.
