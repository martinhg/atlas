# Epic 8 — Deferred Work & Technical Debt

Items consciously left out of Epic 8. Each is safe to pick up later as its own
change. Ordered roughly by impact.

---

## TD-1 — CVSS vector → numeric score parsing (HIGH impact)

**What:** `extractCVSS` (`internal/vuln/service.go`) only accepts a numeric CVSS
score string and `ParseFloat`s it. Real OSV advisories usually expose `CVSS_V3`
as a **vector string** (e.g. `CVSS:3.1/AV:N/AC:L/...`), which does not parse to a
float.

**Consequence:** in production most vulnerabilities will resolve to
`severity = unknown` and `cvss_score = NULL`, even when OSV has full CVSS data.
Severity filtering and the "highest severity" badges will be largely populated
with `unknown` until this is addressed.

**Why deferred:** the spec explicitly lists "CVSS vector calculation" as out of
scope for the MVP.

**Follow-up:** add a CVSS v3/v4 base-score calculator (or a small vetted
dependency) that converts the vector to a numeric base score; feed the result
into `extractCVSS`. Also populate the currently-unused `cvss_vector` column while
doing so.

---

## TD-2 — Lock-file parsing / exact resolved versions

**What:** version matching uses `stripRange` on the declared range in
`repo_dependencies.version`. Compound ranges (e.g. `">=1.2.0 <2.0.0"`) keep only
the first token and log a warning.

**Consequence:** false negatives — a dependency may be vulnerable but not flagged
because the declared range doesn't resolve to a concrete version.

**Why deferred:** out of scope; requires parsing lock files
(`package-lock.json`, `pnpm-lock.yaml`, etc.) to obtain exact installed versions.

**Follow-up:** parse lock files during dep sync, store the resolved version, and
match against it instead of the declared range.

---

## TD-3 — Periodic re-scans / scheduler

**What:** vulnerability data only refreshes when an org sync runs (manual connect
or installation webhook). There is no scheduled re-scan.

**Consequence:** newly published advisories for already-synced dependencies are
not picked up until the next org sync.

**Why deferred:** out of scope.

**Follow-up:** add a scheduled job (cron/worker) that re-runs
`Service.SyncOrgVulns` per org on an interval.

---

## TD-4 — Remediation suggestions

**What:** the dashboard reports vulnerabilities and their fixed versions but does
not suggest or automate upgrades.

**Why deferred:** out of scope.

**Follow-up:** surface "upgrade to X" guidance from `fixed_version`; optionally
pair with TD-5 (auto-PR).

---

## TD-5 — Auto-PR creation for fixes

**What:** no automated pull request to bump a vulnerable dependency.

**Why deferred:** out of scope; depends on TD-2 (resolved versions) and TD-4.

---

## TD-6 — Multi-ecosystem support

**What:** the MVP targets npm. OSV queries and version matching assume npm
semver semantics.

**Why deferred:** out of scope.

**Follow-up:** validate `semver.go` and ecosystem strings against other OSV
ecosystems (PyPI, Go, Maven, …) and adjust matching where their version schemes
differ.

---

## TD-7 — Zero-value timestamps in the list payload (LOW / cosmetic)

**What:** `Vulnerability.CreatedAt` / `UpdatedAt` are not `omitempty` and are not
selected by `ListByOrg`, so list responses serialize them as
`0001-01-01T00:00:00Z`.

**Consequence:** harmless noise in the list JSON; the frontend ignores them.

**Follow-up:** either select the columns in `ListByOrg` or add `omitempty` to the
list-item serialization.

---

## TD-8 — Pre-existing lint debt observed (not introduced by Epic 8)

**What:** `golangci-lint` reports unchecked-error (`errcheck`) findings in code
outside this epic — e.g. `internal/dependency/migration_test.go`,
`internal/auth/handler.go` (`resp.Body.Close`), and several
`json.Encode`/`w.Write` calls in `internal/org/handler.go` and `main.go`.

**Why noted here:** Epic 8 code is clean (0 issues in `internal/vuln/`), but these
pre-existing findings mean `golangci-lint run ./...` is not green repo-wide.

**Follow-up:** a small dedicated cleanup change to wrap the unchecked errors
(`_ = ...` or `//nolint` with rationale).
