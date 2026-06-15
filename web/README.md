# Atlas Web

React frontend for the Atlas Engineering Intelligence Platform.

## Stack

- React 19 with TypeScript
- Vite 8 (dev server + build)
- Tailwind CSS v4 with dark zinc theme
- shadcn/ui primitives
- TanStack Query v5 for server state

## Development

```bash
pnpm install
pnpm dev          # Start dev server on port 5173
pnpm test         # Run Vitest test suite
pnpm lint         # ESLint
pnpm build        # Production build
```

> **pnpm only** — npm and yarn are blocked by a preinstall guard.

The dev server proxies `/api` requests to the Go backend at `http://localhost:8080`.

## Project Structure

```
src/
├── components/          Shared components (DashboardPage, LoginPage, AuthGuard)
│   └── ui/              shadcn primitives (Button, Card, Avatar)
├── features/            Feature modules, each self-contained
│   ├── catalog/         Repository list page, table, and hooks
│   └── dependencies/    Dependency list/detail pages, tables, and hooks
├── hooks/               Shared hooks (useOrgs)
├── lib/                 Utilities
│   ├── api.ts           API types and fetch functions
│   ├── auth.ts          JWT storage, apiFetch with auto-refresh
│   ├── query-client.ts  TanStack Query client
│   └── utils.ts         cn() helper
├── pages/               Standalone pages (GitHubCallbackPage)
├── router.tsx           Route definitions
└── test/                Vitest setup
```

## Conventions

- **UI components**: always use shadcn from `@/components/ui/` — never build custom
- **Data fetching**: one TanStack Query hook per feature (`useRepos`, `useDependencies`)
- **API calls**: use `apiFetch` from `@/lib/auth` — auto-attaches JWT, auto-refreshes on 401
- **Styling**: dark-only zinc palette (`bg-zinc-950`, `text-zinc-100`, `border-zinc-800`)
- **Imports**: always use `@/` path alias, never relative `../../`
- **Testing**: Vitest + React Testing Library, co-located in `__tests__/` directories

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `VITE_API_URL` | Backend API URL | `http://localhost:8080` |
| `VITE_GITHUB_APP_SLUG` | GitHub App slug for install link | `atlas-dev` |
