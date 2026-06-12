---
name: project-conventions
description: "Atlas project-wide conventions: monorepo layout, package manager, commits, env, auth, theme"
metadata:
  keywords:
    - conventions
    - monorepo
    - pnpm
    - commits
    - environment
    - atlas
    - nesbite
    - architecture
license: MIT
---

# Project Conventions

## When to Use

Load this skill at the start of any task involving Atlas. It defines the constraints that all other work must respect.

## Rules

### Identity

- **Product**: Atlas
- **Company**: Nesbite
- **Description**: Engineering Intelligence Platform
- UI copy and all code artifacts use "Atlas", not "AtlasOS" (legacy string in the codebase, being phased out).

### Package manager

**pnpm only.** A `preinstall` script in `web/package.json` blocks npm and yarn:

```json
"scripts": {
  "preinstall": "npx only-allow pnpm"
}
```

Never run `npm install` or `yarn` in this project. Always use `pnpm`.

### Monorepo layout

```
atlas/
  cmd/
    atlas-server/main.go   — HTTP server entrypoint
  internal/
    auth/                  — authentication domain
    catalog/               — repository catalog domain
    org/                   — organization domain
    platform/
      config/config.go     — env loading (godotenv)
      database/            — pgxpool + migration runner
  migrations/
    *.up.sql               — SQL migrations (embedded)
    embed.go               — //go:embed *.sql
  web/
    src/
      components/          — React components
      lib/                 — shared utilities and auth
      test/setup.ts        — Vitest global setup
    vite.config.ts
    vitest.config.ts
  skills/                  — project-specific Claude skills
```

### Commit messages

Conventional commits, always lowercase type:

```
feat: add repository list endpoint
fix: handle missing refresh token on startup
chore: update pnpm lockfile
docs: add env example for GitHub OAuth
test: add handler tests for catalog domain
refactor: extract jsonError helper to httputil
```

No AI attribution lines. No "Co-authored-by" footers.

### Environment variables

- Loaded by `godotenv` from `.env` in the project root at startup.
- `.env` is gitignored — never commit it.
- `.env.example` must list every required variable with empty values:

```dotenv
DATABASE_URL=
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
JWT_SECRET=
WEB_URL=
SERVER_PORT=
```

When adding a new env var:
1. Add to `internal/platform/config/config.go`
2. Add to `.env.example` with an empty value and a comment if non-obvious

### Ports

| Service | Port |
|---------|------|
| Go backend | 8080 |
| Vite dev server | 5173 |

Vite proxies `/api` to `http://localhost:8080` — configured in `web/vite.config.ts`.

### API

- All endpoints under `/api/v1/` prefix.
- Public routes directly on the router.
- Protected routes in a `r.Group` with `auth.Middleware(cfg.JWTSecret)`.
- CORS is configured globally in `main.go` for `cfg.WebURL` — do not add per-route CORS headers.

### Auth flow

```
GitHub OAuth → callback → upsert user → issue JWT pair → redirect to frontend with tokens in hash
```

- Access token: HS256 JWT, 15 min TTL, stored in memory (never localStorage)
- Refresh token: HS256 JWT, 7 days TTL, stored in `localStorage` under key `atlas_refresh_token`
- `apiFetch` in `@/lib/auth` auto-refreshes on 401 — use it for all authenticated API calls

### Database

- PostgreSQL via `pgxpool` (never `database/sql` or GORM).
- All queries are raw SQL — no ORM.
- UUIDs for all primary keys: `DEFAULT gen_random_uuid()`.
- Timestamps: `TIMESTAMPTZ NOT NULL DEFAULT NOW()`.
- Migrations: custom runner, no golang-migrate. See `go-migration` skill.

### UI theme

Dark zinc palette — see `react-component` skill for the full token table. No light mode. No gray/slate tokens.

### Language

All code, comments, UI copy, commit messages, and documentation are in English.

### What NOT to do

- Do not use `npm` or `yarn`.
- Do not use `golang-migrate`, GORM, or `database/sql`.
- Do not store the access token in localStorage.
- Do not add `.down.sql` migration files.
- Do not use `gray-*` or `slate-*` Tailwind tokens.
- Do not use relative imports in the frontend (`../../`) — use `@/` aliases.
- Do not write "AtlasOS" in new UI copy.
