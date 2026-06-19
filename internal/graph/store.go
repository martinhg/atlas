package graph

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GraphStore defines the persistence contract for the graph domain.
// The single method runs one aggregating SQL pass per request.
type GraphStore interface {
	// GetGraph returns one depAggregate per unique dependency in the org,
	// each carrying all affected repos and their teams. Returns an empty
	// non-nil slice when the org has no dependencies.
	GetGraph(ctx context.Context, orgID uuid.UUID, f GraphFilters) ([]depAggregate, error)
}

// Store is the pgxpool-backed implementation of GraphStore.
type Store struct {
	db *pgxpool.Pool
}

// NewStore constructs a Store using the provided connection pool.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// GetGraph executes a single GROUP-BY-per-dep SQL pass over the org's
// dependency graph. It LEFT JOINs repo_owners to aggregate teams per repo.
// The result is one depAggregate per unique dependency; each carries the list
// of affected repos with their language, dep_type, and team owners.
//
// The ecosystem and team filters are applied in SQL when set. Risk filtering
// is applied in the handler after risk computation.
func (s *Store) GetGraph(ctx context.Context, orgID uuid.UUID, f GraphFilters) ([]depAggregate, error) {
	// Build the query dynamically based on which optional filters are set.
	// We collect args and append WHERE clauses only when a filter value is present.
	args := []any{orgID}
	paramIdx := 2

	ecosystemClause := ""
	if f.Ecosystem != "" {
		ecosystemClause = fmt.Sprintf(" AND d.ecosystem = $%d", paramIdx)
		args = append(args, f.Ecosystem)
		paramIdx++
	}

	teamClause := ""
	if f.Team != "" {
		teamClause = fmt.Sprintf(" AND EXISTS (SELECT 1 FROM repo_owners ro2 WHERE ro2.repo_id = r.id AND ro2.owner = $%d AND ro2.owner_type = 'team')", paramIdx)
		args = append(args, f.Team)
		// Team is currently the last optional filter, so paramIdx is not
		// advanced again. Increment here when adding further filter clauses.
	}

	query := fmt.Sprintf(`
		SELECT
			d.id          AS dep_id,
			d.ecosystem,
			d.name        AS dep_name,
			r.id          AS repo_id,
			r.name        AS repo_name,
			r.language,
			rd.dep_type,
			COALESCE(
				ARRAY_AGG(DISTINCT ro.owner) FILTER (WHERE ro.owner IS NOT NULL AND ro.owner_type = 'team'),
				ARRAY[]::TEXT[]
			) AS teams
		FROM dependencies d
		JOIN repo_dependencies rd ON rd.dep_id = d.id
		JOIN repositories r       ON r.id = rd.repo_id
		LEFT JOIN repo_owners ro  ON ro.repo_id = r.id
		WHERE r.org_id = $1
		%s
		%s
		GROUP BY d.id, d.ecosystem, d.name, r.id, r.name, r.language, rd.dep_type
		ORDER BY d.name, r.name
	`, ecosystemClause, teamClause)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Collect rows and group by dep_id into depAggregate entries.
	// We iterate rows in order (ORDER BY d.name, r.name), accumulating
	// repoRef entries per dep_id using an index map.
	aggregates := make([]depAggregate, 0)
	indexByDepID := make(map[uuid.UUID]int)

	for rows.Next() {
		var (
			depID    uuid.UUID
			ecosystem string
			depName  string
			repoID   uuid.UUID
			repoName string
			language *string
			depType  string
			teams    []string
		)
		if err := rows.Scan(&depID, &ecosystem, &depName, &repoID, &repoName, &language, &depType, &teams); err != nil {
			return nil, err
		}

		ref := repoRef{
			RepoID:   repoID,
			RepoName: repoName,
			Language: language,
			DepType:  depType,
			Teams:    teams,
		}

		if idx, ok := indexByDepID[depID]; ok {
			aggregates[idx].AffectedRepos = append(aggregates[idx].AffectedRepos, ref)
		} else {
			indexByDepID[depID] = len(aggregates)
			aggregates = append(aggregates, depAggregate{
				DepID:         depID,
				Ecosystem:     ecosystem,
				Name:          depName,
				AffectedRepos: []repoRef{ref},
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return aggregates, nil
}
