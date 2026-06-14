package dependency_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nesbite/atlas/internal/platform/database"
	"github.com/nesbite/atlas/migrations"
)

// getTestPool opens a pgxpool using DATABASE_URL and runs migrations.
// The test is skipped when DATABASE_URL is not set.
func getTestPool(t *testing.T) *pgxpool.Pool {
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

// TestMigration_dependencies_table_exists verifies that the migration creates
// the dependencies table with the expected columns and unique constraint.
func TestMigration_dependencies_table_exists(t *testing.T) {
	pool := getTestPool(t)
	ctx := context.Background()

	// Verify the table exists by querying its columns from information_schema.
	rows, err := pool.Query(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_name = 'dependencies'
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

	required := []string{"id", "ecosystem", "name", "created_at"}
	for _, c := range required {
		if !cols[c] {
			t.Errorf("dependencies table missing column %q", c)
		}
	}
}

// TestMigration_dependencies_unique_constraint verifies UNIQUE(ecosystem, name).
func TestMigration_dependencies_unique_constraint(t *testing.T) {
	pool := getTestPool(t)
	ctx := context.Background()

	// Insert a dependency, then try inserting the same ecosystem+name again —
	// should fail with a unique violation.
	cleanup := func() {
		pool.Exec(ctx, "DELETE FROM dependencies WHERE name = 'test-dep-migration'")
	}
	t.Cleanup(cleanup)
	cleanup()

	_, err := pool.Exec(ctx, `
		INSERT INTO dependencies (ecosystem, name) VALUES ('npm', 'test-dep-migration')
	`)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO dependencies (ecosystem, name) VALUES ('npm', 'test-dep-migration')
	`)
	if err == nil {
		t.Error("expected unique constraint violation on duplicate (ecosystem, name), got nil error")
	}
}

// TestMigration_repo_dependencies_table_exists verifies that the migration creates
// the repo_dependencies table with the expected columns.
func TestMigration_repo_dependencies_table_exists(t *testing.T) {
	pool := getTestPool(t)
	ctx := context.Background()

	rows, err := pool.Query(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_name = 'repo_dependencies'
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

	required := []string{"id", "repo_id", "dep_id", "version", "dep_type", "source_file", "created_at"}
	for _, c := range required {
		if !cols[c] {
			t.Errorf("repo_dependencies table missing column %q", c)
		}
	}
}

// TestMigration_repo_dependencies_fk_to_repositories verifies the FK from
// repo_dependencies.repo_id to repositories.id (ON DELETE CASCADE).
func TestMigration_repo_dependencies_fk_to_repositories(t *testing.T) {
	pool := getTestPool(t)
	ctx := context.Background()

	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM information_schema.referential_constraints rc
		JOIN information_schema.key_column_usage kcu
			ON rc.constraint_name = kcu.constraint_name
		WHERE kcu.table_name = 'repo_dependencies'
		  AND kcu.column_name = 'repo_id'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("FK query: %v", err)
	}
	if count == 0 {
		t.Error("repo_dependencies.repo_id does not have a foreign key constraint")
	}
}

// TestMigration_repo_dependencies_fk_to_dependencies verifies the FK from
// repo_dependencies.dep_id to dependencies.id.
func TestMigration_repo_dependencies_fk_to_dependencies(t *testing.T) {
	pool := getTestPool(t)
	ctx := context.Background()

	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM information_schema.referential_constraints rc
		JOIN information_schema.key_column_usage kcu
			ON rc.constraint_name = kcu.constraint_name
		WHERE kcu.table_name = 'repo_dependencies'
		  AND kcu.column_name = 'dep_id'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("FK query: %v", err)
	}
	if count == 0 {
		t.Error("repo_dependencies.dep_id does not have a foreign key constraint")
	}
}

// TestMigration_repo_dependencies_unique_constraint verifies UNIQUE(repo_id, dep_id, source_file).
func TestMigration_repo_dependencies_unique_constraint(t *testing.T) {
	pool := getTestPool(t)
	ctx := context.Background()

	// Check that the unique index exists on the table.
	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_indexes
		WHERE tablename = 'repo_dependencies'
		  AND indexdef LIKE '%repo_id%dep_id%source_file%'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("index query: %v", err)
	}
	if count == 0 {
		t.Error("expected unique index on repo_dependencies(repo_id, dep_id, source_file), not found")
	}
}
