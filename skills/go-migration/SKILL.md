---
name: go-migration
description: "How to add database migrations in Atlas (custom embedded runner, pgx, no golang-migrate)"
metadata:
  keywords:
    - go
    - migration
    - sql
    - postgres
    - database
    - schema
license: MIT
---

# Go Migration

## When to Use

Load this skill when:
- Adding a new table or column to the Atlas database
- Creating or modifying indexes
- Renaming columns or adding constraints
- Any DDL change to the PostgreSQL schema

## Rules

### Naming

```
{zero-padded-6-digit-number}_{description}.up.sql
```

Examples:
- `000002_create_repositories.up.sql`
- `000003_add_repo_description.up.sql`
- `000004_create_idx_repos_owner.up.sql`

- Always zero-pad to 6 digits so lexicographic sort equals chronological order.
- Use lowercase with underscores for the description.
- Only `.up.sql` files exist — there are no down migrations.

### Location

All migration files go in `migrations/` (project root). They are auto-embedded via `migrations/embed.go` and auto-applied at server startup via `database.RunMigrations`.

Do NOT create migrations anywhere else. Do NOT add new embed files — the existing `//go:embed *.sql` glob picks up every `.sql` file automatically.

### Runner behavior

- Custom runner in `internal/platform/database/migrate.go`.
- Tracks applied versions in the `schema_migrations` table.
- Runs each file inside a pgx transaction — if any statement fails, the whole migration rolls back.
- Idempotent: skips already-applied versions on every restart.
- No dependency on `golang-migrate` or any other migration library.

### SQL conventions

Always use `IF NOT EXISTS` to make migrations safe to re-inspect:

```sql
CREATE TABLE IF NOT EXISTS repositories (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_repositories_owner_id ON repositories (owner_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_repositories_owner_name ON repositories (owner_id, name);
```

Standard column conventions:
- `id UUID PRIMARY KEY DEFAULT gen_random_uuid()` — always UUID, never serial/int
- `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- Foreign keys: `REFERENCES {table}(id) ON DELETE CASCADE` unless soft-delete is intended

### One concern per file

Each migration file does exactly one logical change. Do not bundle unrelated table changes in the same file.

### No down migrations

The project does not maintain `.down.sql` files. If a schema change needs to be reverted:
1. Write a new migration that undoes the change.
2. Never delete or edit a previously applied migration file.

### Verifying the next number

```bash
ls migrations/*.up.sql | tail -1
```

Increment the number by 1 for the new file.

## Example

`migrations/000002_create_repositories.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS repositories (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    url         TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_repositories_owner_id ON repositories (owner_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_repositories_owner_name ON repositories (owner_id, name);
```

Canonical reference: `migrations/000001_create_users.up.sql`.
