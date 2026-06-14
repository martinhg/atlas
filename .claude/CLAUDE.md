# Atlas — Project Rules

Atlas is an Engineering Intelligence Platform by Nesbite. Impact Analysis is the wedge feature.

## Stack

- **Backend**: Go 1.26+, chi router, pgx/pgxpool, godotenv, HS256 JWT
- **Frontend**: React 19, Vite 8, Tailwind CSS v4, shadcn/ui, TypeScript 6
- **Database**: PostgreSQL
- **Planned**: Zustand 5, Zod 4, TanStack Query v5, TanStack Table v8, Playwright

## Non-Negotiable Rules

- **pnpm ONLY** — never npm or yarn. A preinstall guard blocks them.
- **Conventional commits** — `feat:`, `fix:`, `chore:`, `docs:`, `test:`, `refactor:`
- **English** in all code, comments, UI copy, commit messages, and docs.
- **Dark theme** — zinc palette: `bg-zinc-950`, `text-zinc-100`, `border-zinc-800`.
- **Path aliases** — always use `@/` imports in the frontend.
- **Tests required** — no feature ships without tests. Frontend: Vitest 3 + RTL. Backend: Go standard testing.

## Project Structure

```
cmd/atlas-server/main.go       — Go server entrypoint (chi, CORS, graceful shutdown)
internal/
  auth/                        — GitHub OAuth + JWT auth domain
  platform/config/             — env var loading (godotenv)
  platform/database/           — pgxpool + custom migration runner
migrations/                    — SQL migrations (auto-embedded, auto-run on startup)
web/                           — React frontend (Vite + Tailwind v4 + shadcn)
  src/components/ui/           — shadcn primitives (Button, Card, Avatar)
  src/components/              — feature/page components
  src/lib/                     — utilities (auth.ts, utils.ts)
  src/test/                    — test setup
skills/                        — AI assistant skills (Claude Code, Gemini)
```

## Backend Conventions

- Handlers receive dependencies via constructor: `NewHandler(store, config)`
- Routes registered with chi groups: `r.Route("/api/v1/auth", handler.Routes)`
- Store pattern: interface + implementation (see `auth/store.go`)
- Migrations: `migrations/{number}_{description}.up.sql`, auto-embedded via `embed.go`
- Custom migration runner — no golang-migrate dependency
- Always use `IF NOT EXISTS` for tables and indexes

## Frontend Conventions

- UI primitives: always use shadcn from `@/components/ui/` — never build custom
- `cn()` from `@/lib/utils` for conditional classes
- Page components: `{Name}Page.tsx`, default export
- Feature components: `{Name}.tsx`, named export
- API calls: use `apiFetch` from `@/lib/auth` (auto-attaches JWT, auto-refreshes on 401)

## Environment

- Backend: port 8080 (`PORT` env var)
- Frontend: port 5173 (Vite dev server, proxies `/api` to backend)
- `.env` in project root — gitignored, loaded by godotenv
- Required vars: `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`, `JWT_SECRET`
- Defaults: `DATABASE_URL=postgres://atlas:atlas@localhost:5432/atlas?sslmode=disable`

## Skills

Run `./skills/setup.sh` to symlink all skills to `.claude/skills/` and `.gemini/skills/`.

When doing backend work, load: `go-handler`, `go-migration`, `go-testing`, `tdd`
When doing frontend work, load: `react-component`, `react-testing`, `react-19`, `vitest`, `tailwind-4`, `tailwind-v4-shadcn`
When doing full-stack work, load: `project-conventions` + relevant backend/frontend skills
