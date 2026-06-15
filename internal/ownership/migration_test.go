package ownership_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nesbite/atlas/internal/platform/database"
	"github.com/nesbite/atlas/migrations"
)

func getMigrationTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := database.RunMigrations(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return pool
}

// TestMigration_repoOwners_table_exists verifies that the migration creates
// the repo_owners table with the correct columns.
func TestMigration_repoOwners_table_exists(t *testing.T) {
	pool := getMigrationTestPool(t)
	ctx := context.Background()

	rows, err := pool.Query(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_name = 'repo_owners'
		ORDER BY ordinal_position
	`)
	if err != nil {
		t.Fatalf("query information_schema: %v", err)
	}
	defer rows.Close()

	cols := make(map[string]bool)
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			t.Fatalf("scan column_name: %v", err)
		}
		cols[col] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows error: %v", err)
	}

	required := []string{"id", "repo_id", "pattern", "owner", "owner_type", "source", "line_number", "created_at", "updated_at"}
	for _, c := range required {
		if !cols[c] {
			t.Errorf("repo_owners table missing column %q", c)
		}
	}
}

// TestMigration_repoOwners_unique_constraint verifies UNIQUE(repo_id, pattern, owner).
func TestMigration_repoOwners_unique_constraint(t *testing.T) {
	pool := getMigrationTestPool(t)
	ctx := context.Background()

	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_indexes
		WHERE tablename = 'repo_owners'
		  AND indexdef LIKE '%repo_id%pattern%owner%'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("index query: %v", err)
	}
	if count == 0 {
		t.Error("expected unique index on repo_owners(repo_id, pattern, owner), not found")
	}
}

// TestMigration_repoOwners_repo_id_index verifies the idx_repo_owners_repo_id index.
func TestMigration_repoOwners_repo_id_index(t *testing.T) {
	pool := getMigrationTestPool(t)
	ctx := context.Background()

	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_indexes
		WHERE tablename = 'repo_owners'
		  AND indexname = 'idx_repo_owners_repo_id'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("index query: %v", err)
	}
	if count == 0 {
		t.Error("expected index idx_repo_owners_repo_id on repo_owners, not found")
	}
}

// TestMigration_repoOwners_fk_to_repositories verifies the FK from
// repo_owners.repo_id to repositories.id.
func TestMigration_repoOwners_fk_to_repositories(t *testing.T) {
	pool := getMigrationTestPool(t)
	ctx := context.Background()

	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM information_schema.referential_constraints rc
		JOIN information_schema.key_column_usage kcu
			ON rc.constraint_name = kcu.constraint_name
		WHERE kcu.table_name = 'repo_owners'
		  AND kcu.column_name = 'repo_id'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("FK query: %v", err)
	}
	if count == 0 {
		t.Error("repo_owners.repo_id does not have a foreign key constraint")
	}
}
