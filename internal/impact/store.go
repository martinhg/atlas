package impact

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ImpactStore defines the persistence contract for the impact analysis domain.
type ImpactStore interface {
	// GetBlastRadius returns every repo in the given org that declares the
	// dependency identified by ecosystem and name, along with its version,
	// dependency type, and owning teams. Returns an empty slice when the
	// dependency is not used by any repo in the org — never returns nil.
	GetBlastRadius(ctx context.Context, orgID uuid.UUID, ecosystem, name string) ([]AffectedRepo, error)
}

// Store is the pgxpool-backed implementation of ImpactStore.
type Store struct {
	db *pgxpool.Pool
}

// NewStore constructs a Store using the provided connection pool.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// GetBlastRadius joins dependencies -> repo_dependencies -> repositories ->
// repo_owners to find every repo in orgID that declares the dependency
// identified by ecosystem and name. One row is returned per repo, carrying
// its version, dep_type, and a deduplicated (case-insensitive) list of team
// owners. Results are scoped strictly to orgID.
func (s *Store) GetBlastRadius(ctx context.Context, orgID uuid.UUID, ecosystem, name string) ([]AffectedRepo, error) {
	rows, err := s.db.Query(ctx, `
		SELECT r.name, r.full_name, rd.version, rd.dep_type,
		       ARRAY_AGG(DISTINCT ro.owner) FILTER (WHERE ro.owner_type = 'team') AS teams
		FROM dependencies d
		JOIN repo_dependencies rd ON rd.dep_id = d.id
		JOIN repositories r       ON r.id = rd.repo_id
		LEFT JOIN repo_owners ro  ON ro.repo_id = r.id
		WHERE r.org_id = $1
		  AND d.ecosystem = $2
		  AND d.name = $3
		GROUP BY r.id, r.name, r.full_name, rd.version, rd.dep_type
		ORDER BY r.name
	`, orgID, ecosystem, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]AffectedRepo, 0)
	for rows.Next() {
		var repo AffectedRepo
		var teams []string
		if err := rows.Scan(&repo.RepoName, &repo.FullName, &repo.Version, &repo.DepType, &teams); err != nil {
			return nil, err
		}
		repo.Teams = dedupeOwnersCaseInsensitive(teams)
		result = append(result, repo)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// dedupeOwnersCaseInsensitive removes case-insensitive duplicates from owner
// strings (e.g. "@org/Team" and "@org/team" collapse to one entry, keeping
// the first-seen casing) and always returns a non-nil slice, sorted for
// deterministic output.
func dedupeOwnersCaseInsensitive(owners []string) []string {
	seen := make(map[string]struct{}, len(owners))
	result := make([]string, 0, len(owners))
	for _, owner := range owners {
		key := strings.ToLower(owner)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, owner)
	}
	sort.Strings(result)
	return result
}
