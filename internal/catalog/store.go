package catalog

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepoStore interface {
	UpsertRepository(ctx context.Context, repo *Repository) (*Repository, error)
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID, q string, page, perPage int) ([]Repository, int, error)
	GetRepoByName(ctx context.Context, orgID uuid.UUID, name string) (*Repository, error)
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) UpsertRepository(ctx context.Context, repo *Repository) (*Repository, error) {
	var r Repository
	err := s.db.QueryRow(ctx, `
		INSERT INTO repositories (org_id, github_id, name, full_name, description, default_branch, language, private, fork, stars)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (org_id, github_id)
		DO UPDATE SET
			name           = EXCLUDED.name,
			full_name      = EXCLUDED.full_name,
			description    = EXCLUDED.description,
			default_branch = EXCLUDED.default_branch,
			language       = EXCLUDED.language,
			private        = EXCLUDED.private,
			fork           = EXCLUDED.fork,
			stars          = EXCLUDED.stars,
			updated_at     = NOW()
		RETURNING id, org_id, github_id, name, full_name, description, default_branch, language, private, fork, stars, last_synced_at, created_at, updated_at
	`, repo.OrgID, repo.GitHubID, repo.Name, repo.FullName, repo.Description, repo.DefaultBranch, repo.Language, repo.Private, repo.Fork, repo.Stars).
		Scan(&r.ID, &r.OrgID, &r.GitHubID, &r.Name, &r.FullName, &r.Description, &r.DefaultBranch, &r.Language, &r.Private, &r.Fork, &r.Stars, &r.LastSyncedAt, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetRepositoriesByOrgID returns a filtered, paginated list of repositories
// for the given org. q is applied as an ILIKE filter across name, full_name,
// and COALESCE(description, ''). page and perPage are 1-based.
// Returns the matching page of repos, the total count across all pages, and any error.
func (s *Store) GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID, q string, page, perPage int) ([]Repository, int, error) {
	// Clamp pagination parameters to safe values.
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	offset := (page - 1) * perPage

	rows, err := s.db.Query(ctx, `
		SELECT id, org_id, github_id, name, full_name, description, default_branch,
		       language, private, fork, stars, last_synced_at, created_at, updated_at,
		       COUNT(*) OVER() AS total
		FROM repositories
		WHERE org_id = $1
		  AND ($2 = '' OR (
		    name ILIKE '%' || $2 || '%'
		    OR full_name ILIKE '%' || $2 || '%'
		    OR COALESCE(description, '') ILIKE '%' || $2 || '%'
		  ))
		ORDER BY full_name
		LIMIT $3 OFFSET $4
	`, orgID, q, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var total int
	repos := make([]Repository, 0)
	for rows.Next() {
		var r Repository
		if err := rows.Scan(
			&r.ID, &r.OrgID, &r.GitHubID, &r.Name, &r.FullName, &r.Description,
			&r.DefaultBranch, &r.Language, &r.Private, &r.Fork, &r.Stars,
			&r.LastSyncedAt, &r.CreatedAt, &r.UpdatedAt, &total,
		); err != nil {
			return nil, 0, err
		}
		repos = append(repos, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return repos, total, nil
}

func (s *Store) GetRepoByName(ctx context.Context, orgID uuid.UUID, name string) (*Repository, error) {
	var r Repository
	err := s.db.QueryRow(ctx, `
		SELECT id, org_id, github_id, name, full_name, description, default_branch,
		       language, private, fork, stars, last_synced_at, created_at, updated_at
		FROM repositories
		WHERE org_id = $1 AND name = $2
	`, orgID, name).Scan(
		&r.ID, &r.OrgID, &r.GitHubID, &r.Name, &r.FullName, &r.Description,
		&r.DefaultBranch, &r.Language, &r.Private, &r.Fork, &r.Stars,
		&r.LastSyncedAt, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
