package dependency

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nesbite/atlas/internal/dependency/parser"
)

// DepStore defines the persistence contract for the dependency domain.
type DepStore interface {
	// SyncRepoDependencies replaces all repo_dependencies rows for the given
	// repo with the provided deps slice. The operation runs inside a transaction:
	// DELETE existing rows, then INSERT the new ones.
	SyncRepoDependencies(ctx context.Context, repoID uuid.UUID, deps []parser.ParsedDep) error

	// ListByOrg returns a paginated, optionally filtered list of unique
	// dependencies used by any repository in the given org, along with the
	// total count and a per-dep repo count. q is applied as an ILIKE filter
	// on the dependency name; empty string means no filter. Page and perPage
	// are 1-based.
	ListByOrg(ctx context.Context, orgID uuid.UUID, q string, page, perPage int) ([]DependencyWithCount, int, error)

	// GetDetail returns the list of repos (within the org) that use the given
	// dependency identified by ecosystem and name. Returns an empty slice when
	// the dependency is not found — never returns nil.
	GetDetail(ctx context.Context, orgID uuid.UUID, ecosystem, name string) ([]DepDetail, error)

	// ListByRepoName returns all dependencies for a specific repo within the org.
	ListByRepoName(ctx context.Context, orgID uuid.UUID, repoName string) ([]RepoDepDetail, error)
}

// Store is the pgxpool-backed implementation of DepStore.
type Store struct {
	db *pgxpool.Pool
}

// NewStore constructs a Store using the provided connection pool.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// SyncRepoDependencies replaces all repo_dependencies rows for repoID with deps.
// The entire operation runs in a single transaction:
//  1. DELETE FROM repo_dependencies WHERE repo_id = repoID
//  2. UPSERT deps into dependencies using CTE (ON CONFLICT DO NOTHING)
//  3. INSERT repo_dependencies rows
//
// Steps 2 and 3 use pgx.Batch to avoid N+1 round-trips.
func (s *Store) SyncRepoDependencies(ctx context.Context, repoID uuid.UUID, deps []parser.ParsedDep) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Step 1: remove stale rows for this repo.
	if _, err := tx.Exec(ctx, `
		DELETE FROM repo_dependencies WHERE repo_id = $1
	`, repoID); err != nil {
		return err
	}

	if len(deps) == 0 {
		return tx.Commit(ctx)
	}

	// Step 2: batch-upsert all (ecosystem, name) pairs into dependencies using
	// a CTE that avoids write locks on existing rows.
	batch := &pgx.Batch{}
	for _, dep := range deps {
		batch.Queue(`
			WITH ins AS (
				INSERT INTO dependencies (ecosystem, name)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING
				RETURNING id
			)
			SELECT id FROM ins
			UNION ALL
			SELECT id FROM dependencies WHERE ecosystem = $1 AND name = $2
			LIMIT 1
		`, dep.Ecosystem, dep.Name)
	}

	br := tx.SendBatch(ctx, batch)
	depIDs := make([]uuid.UUID, len(deps))
	for i := range deps {
		if err := br.QueryRow().Scan(&depIDs[i]); err != nil {
			br.Close() //nolint:errcheck
			return err
		}
	}
	if err := br.Close(); err != nil {
		return err
	}

	// Step 3: batch-insert repo_dependency rows.
	rdBatch := &pgx.Batch{}
	for i, dep := range deps {
		rdBatch.Queue(`
			INSERT INTO repo_dependencies (repo_id, dep_id, version, dep_type, source_file)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (repo_id, dep_id, source_file)
			DO UPDATE SET version = EXCLUDED.version, dep_type = EXCLUDED.dep_type, updated_at = NOW()
		`, repoID, depIDs[i], dep.Version, dep.DepType, dep.SourceFile)
	}

	br = tx.SendBatch(ctx, rdBatch)
	for range deps {
		if _, err := br.Exec(); err != nil {
			br.Close() //nolint:errcheck
			return err
		}
	}
	if err := br.Close(); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ListByOrg returns a filtered, paginated list of unique dependencies used by
// repos in orgID. q is applied as an ILIKE filter on the dependency name;
// empty string means no filter. total is the count of all matching rows.
func (s *Store) ListByOrg(ctx context.Context, orgID uuid.UUID, q string, page, perPage int) ([]DependencyWithCount, int, error) {
	// Clamp pagination parameters to safe values.
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 50
	}
	if perPage > 100 {
		perPage = 100
	}

	offset := (page - 1) * perPage

	// Single query with COUNT(*) OVER() to avoid race between COUNT and SELECT.
	// $4 = q: when non-empty, filter dependency name with ILIKE.
	rows, err := s.db.Query(ctx, `
		SELECT d.id, d.ecosystem, d.name,
		       COUNT(DISTINCT r.id) AS repo_count,
		       COUNT(*) OVER() AS total
		FROM dependencies d
		JOIN repo_dependencies rd ON rd.dep_id = d.id
		JOIN repositories r       ON r.id = rd.repo_id
		WHERE r.org_id = $1
		  AND ($4 = '' OR d.name ILIKE '%' || $4 || '%')
		GROUP BY d.id, d.ecosystem, d.name
		ORDER BY d.name
		LIMIT $2 OFFSET $3
	`, orgID, perPage, offset, q)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var total int
	result := make([]DependencyWithCount, 0)
	for rows.Next() {
		var d DependencyWithCount
		if err := rows.Scan(&d.ID, &d.Ecosystem, &d.Name, &d.RepoCount, &total); err != nil {
			return nil, 0, err
		}
		result = append(result, d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

// GetDetail returns per-repo usage details for the dependency identified by
// ecosystem and name within the given org. Returns an empty slice when not found.
func (s *Store) GetDetail(ctx context.Context, orgID uuid.UUID, ecosystem, name string) ([]DepDetail, error) {
	rows, err := s.db.Query(ctx, `
		SELECT r.name, rd.version, rd.dep_type, rd.source_file
		FROM repo_dependencies rd
		JOIN dependencies d ON d.id = rd.dep_id
		JOIN repositories r ON r.id = rd.repo_id
		WHERE r.org_id = $1
		  AND d.ecosystem = $2
		  AND d.name = $3
		ORDER BY r.name, rd.source_file
	`, orgID, ecosystem, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]DepDetail, 0)
	for rows.Next() {
		var d DepDetail
		if err := rows.Scan(&d.RepoName, &d.Version, &d.DepType, &d.SourceFile); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Store) ListByRepoName(ctx context.Context, orgID uuid.UUID, repoName string) ([]RepoDepDetail, error) {
	rows, err := s.db.Query(ctx, `
		SELECT d.ecosystem, d.name, rd.version, rd.dep_type, rd.source_file
		FROM repo_dependencies rd
		JOIN dependencies d ON d.id = rd.dep_id
		JOIN repositories r ON r.id = rd.repo_id
		WHERE r.org_id = $1 AND r.name = $2
		ORDER BY d.ecosystem, d.name, rd.source_file
	`, orgID, repoName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]RepoDepDetail, 0)
	for rows.Next() {
		var d RepoDepDetail
		if err := rows.Scan(&d.Ecosystem, &d.Name, &d.Version, &d.DepType, &d.SourceFile); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
