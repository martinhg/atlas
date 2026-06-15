package ownership

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	ownerparser "github.com/nesbite/atlas/internal/ownership/parser"
)

// OwnershipStore defines the persistence contract for the ownership domain.
type OwnershipStore interface {
	// SyncRepoOwners replaces all repo_owners rows for the given repo atomically.
	// Deletes all existing rows for repoID, then inserts owners in a single
	// transaction. If owners is empty, only the delete is performed (idempotent clear).
	SyncRepoOwners(ctx context.Context, repoID uuid.UUID, owners []ownerparser.ParsedOwner) error

	// ListByOrg returns ownership summaries for all repos in an org, paginated.
	// Returns (summaries, totalCount, error).
	// page is 1-based. perPage is clamped to [1, 100] by the store.
	ListByOrg(ctx context.Context, orgID uuid.UUID, page, perPage int) ([]RepoOwnerSummary, int, error)

	// ListByRepo returns all CODEOWNERS rules for a specific repo in an org,
	// ordered by line_number. Returns empty slice (not error) if no ownership data exists.
	ListByRepo(ctx context.Context, orgID uuid.UUID, repoName string) ([]OwnerRule, error)
}

// Store is the pgxpool-backed implementation of OwnershipStore.
type Store struct {
	db *pgxpool.Pool
}

// NewStore constructs a Store using the provided connection pool.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// SyncRepoOwners replaces all repo_owners rows for repoID with the provided owners.
// The entire operation runs in a single transaction:
//  1. DELETE FROM repo_owners WHERE repo_id = repoID
//  2. Batch INSERT all owners (skipped when owners is empty)
func (s *Store) SyncRepoOwners(ctx context.Context, repoID uuid.UUID, owners []ownerparser.ParsedOwner) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Step 1: delete all existing rows for this repo.
	if _, err := tx.Exec(ctx, `
		DELETE FROM repo_owners WHERE repo_id = $1
	`, repoID); err != nil {
		return err
	}

	// If no owners provided, commit the delete and return.
	if len(owners) == 0 {
		return tx.Commit(ctx)
	}

	// Step 2: batch-insert all owner rows.
	batch := &pgx.Batch{}
	for _, o := range owners {
		var lineNum *int
		if o.LineNumber > 0 {
			ln := o.LineNumber
			lineNum = &ln
		}
		batch.Queue(`
			INSERT INTO repo_owners (repo_id, pattern, owner, owner_type, line_number)
			VALUES ($1, $2, $3, $4, $5)
		`, repoID, o.Pattern, o.Owner, o.OwnerType, lineNum)
	}

	br := tx.SendBatch(ctx, batch)
	for range owners {
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

// ListByOrg returns a paginated list of repos in the org with ownership summaries.
// total is the count of all matching rows (not just the current page).
func (s *Store) ListByOrg(ctx context.Context, orgID uuid.UUID, page, perPage int) ([]RepoOwnerSummary, int, error) {
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

	// Single query using COUNT(*) OVER() to avoid a second round-trip for total.
	// ARRAY_AGG FILTER returns NULL when there are no team owners — we handle
	// this by scanning into *[]string and converting nil to empty slice.
	rows, err := s.db.Query(ctx, `
		SELECT r.name AS repo_name,
		       COUNT(DISTINCT ro.owner)::int AS owner_count,
		       COUNT(DISTINCT CASE WHEN ro.owner_type = 'team' THEN ro.owner END)::int AS team_count,
		       ARRAY_AGG(DISTINCT ro.owner) FILTER (WHERE ro.owner_type = 'team') AS teams,
		       COUNT(*) OVER()::int AS total
		FROM repo_owners ro
		JOIN repositories r ON r.id = ro.repo_id
		WHERE r.org_id = $1
		GROUP BY r.id, r.name
		ORDER BY r.name
		LIMIT $2 OFFSET $3
	`, orgID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var total int
	result := make([]RepoOwnerSummary, 0)
	for rows.Next() {
		var s RepoOwnerSummary
		// teams may be NULL when a repo has no team owners — scan into *[]string.
		var teams []string
		if err := rows.Scan(&s.RepoName, &s.OwnerCount, &s.TeamCount, &teams, &total); err != nil {
			return nil, 0, err
		}
		if teams == nil {
			teams = make([]string, 0)
		}
		s.Teams = teams
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

// ListByRepo returns all CODEOWNERS rules for the given repo within the org,
// ordered by line_number ASC NULLS LAST.
func (s *Store) ListByRepo(ctx context.Context, orgID uuid.UUID, repoName string) ([]OwnerRule, error) {
	rows, err := s.db.Query(ctx, `
		SELECT ro.pattern, ro.owner, ro.owner_type, ro.line_number
		FROM repo_owners ro
		JOIN repositories r ON r.id = ro.repo_id
		WHERE r.org_id = $1 AND r.name = $2
		ORDER BY ro.line_number ASC NULLS LAST
	`, orgID, repoName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]OwnerRule, 0)
	for rows.Next() {
		var rule OwnerRule
		if err := rows.Scan(&rule.Pattern, &rule.Owner, &rule.OwnerType, &rule.LineNumber); err != nil {
			return nil, err
		}
		result = append(result, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
