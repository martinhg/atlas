package vuln

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VulnStore defines the persistence contract for the vulnerability domain.
type VulnStore interface {
	// UpsertVulnerability inserts or updates a vulnerability record keyed on osv_id.
	UpsertVulnerability(ctx context.Context, v *Vulnerability) error

	// UpsertDepVuln creates a dependency_vulnerabilities link if it does not
	// already exist (ON CONFLICT DO NOTHING).
	UpsertDepVuln(ctx context.Context, depID, vulnID uuid.UUID) error

	// DeleteDepVulnsByOrg removes all dependency_vulnerabilities rows for
	// dependencies belonging to the given org. Called before a full re-sync to
	// avoid stale links.
	DeleteDepVulnsByOrg(ctx context.Context, orgID uuid.UUID) error

	// ListByOrg returns a paginated list of vulnerabilities affecting any
	// dependency used by org repositories, along with the total count.
	// severity may be empty (no filter) or one of: critical, high, medium, low, unknown.
	// packageName may be empty (no filter) or an exact package name to filter by.
	// page and perPage are 1-based.
	ListByOrg(ctx context.Context, orgID uuid.UUID, severity, packageName string, page, perPage int) ([]VulnWithCounts, int, error)

	// GetDetail returns full vulnerability details including the list of
	// affected repositories within the org. Returns nil when not found.
	GetDetail(ctx context.Context, orgID uuid.UUID, vulnID uuid.UUID) (*VulnDetail, error)

	// ListOrgDepPairs returns all (depID, ecosystem, name, version) pairs for
	// dependencies used by at least one repository in the org. Used by
	// VulnService to build the OSV batch query.
	ListOrgDepPairs(ctx context.Context, orgID uuid.UUID) ([]DepPair, error)
}

// Store is the pgxpool-backed implementation of VulnStore.
type Store struct {
	db *pgxpool.Pool
}

// NewStore constructs a Store using the provided connection pool.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// UpsertVulnerability inserts a new vulnerability or updates all mutable fields
// when the osv_id already exists.
func (s *Store) UpsertVulnerability(ctx context.Context, v *Vulnerability) error {
	rangesJSON, err := json.Marshal(v.AffectedRanges)
	if err != nil {
		return err
	}

	return s.db.QueryRow(ctx, `
		INSERT INTO vulnerabilities (
			osv_id, cve_id, ecosystem, package_name,
			severity, cvss_score, cvss_vector,
			summary, details,
			published_at, modified_at,
			fixed_version, introduced_version,
			affected_ranges, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9,
			$10, $11,
			$12, $13,
			$14, NOW()
		)
		ON CONFLICT (osv_id) DO UPDATE SET
			cve_id             = EXCLUDED.cve_id,
			severity           = EXCLUDED.severity,
			cvss_score         = EXCLUDED.cvss_score,
			cvss_vector        = EXCLUDED.cvss_vector,
			summary            = EXCLUDED.summary,
			details            = EXCLUDED.details,
			published_at       = EXCLUDED.published_at,
			modified_at        = EXCLUDED.modified_at,
			fixed_version      = EXCLUDED.fixed_version,
			introduced_version = EXCLUDED.introduced_version,
			affected_ranges    = EXCLUDED.affected_ranges,
			updated_at         = NOW()
		RETURNING id
	`,
		v.OsvID, v.CveID, v.Ecosystem, v.PackageName,
		string(v.Severity), v.CvssScore, v.CvssVector,
		v.Summary, v.Details,
		v.PublishedAt, v.ModifiedAt,
		v.FixedVersion, v.IntroducedVersion,
		rangesJSON,
	).Scan(&v.ID)
}

// UpsertDepVuln creates a dependency_vulnerabilities row if it does not exist.
func (s *Store) UpsertDepVuln(ctx context.Context, depID, vulnID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO dependency_vulnerabilities (dep_id, vuln_id)
		VALUES ($1, $2)
		ON CONFLICT (dep_id, vuln_id) DO NOTHING
	`, depID, vulnID)
	return err
}

// DeleteDepVulnsByOrg removes all dependency_vulnerabilities rows for the org
// before a fresh sync to avoid accumulating stale data.
func (s *Store) DeleteDepVulnsByOrg(ctx context.Context, orgID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM dependency_vulnerabilities dv
		USING dependencies d
		JOIN repo_dependencies rd ON rd.dep_id = d.id
		JOIN repositories r       ON r.id = rd.repo_id
		WHERE dv.dep_id = d.id
		  AND r.org_id = $1
	`, orgID)
	return err
}

// ListByOrg returns a paginated list of vulnerabilities affecting any dependency
// in the given org. The query joins through dependency_vulnerabilities →
// dependencies → repo_dependencies → repositories. Results are ordered by
// cvss_score DESC NULLS LAST, then affected_repo_count DESC.
func (s *Store) ListByOrg(ctx context.Context, orgID uuid.UUID, severity, packageName string, page, perPage int) ([]VulnWithCounts, int, error) {
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
		SELECT
			v.id, v.osv_id, v.cve_id,
			v.ecosystem, v.package_name,
			v.severity, v.cvss_score,
			v.summary,
			v.published_at,
			COUNT(DISTINCT r.id)                AS affected_repo_count,
			COUNT(DISTINCT ro.owner)             AS affected_team_count,
			COUNT(*) OVER()                      AS total
		FROM vulnerabilities v
		JOIN dependency_vulnerabilities dv ON dv.vuln_id = v.id
		JOIN dependencies d                ON d.id = dv.dep_id
		JOIN repo_dependencies rd          ON rd.dep_id = d.id
		JOIN repositories r                ON r.id = rd.repo_id
		LEFT JOIN repo_owners ro           ON ro.repo_id = r.id AND ro.owner_type = 'team'
		WHERE r.org_id = $1
		  AND ($4 = '' OR v.severity = $4)
		  AND ($5 = '' OR v.package_name = $5)
		GROUP BY v.id, v.osv_id, v.cve_id,
		         v.ecosystem, v.package_name,
		         v.severity, v.cvss_score,
		         v.summary, v.published_at
		ORDER BY v.cvss_score DESC NULLS LAST, affected_repo_count DESC
		LIMIT $2 OFFSET $3
	`, orgID, perPage, offset, severity, packageName)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var total int
	result := make([]VulnWithCounts, 0)
	for rows.Next() {
		var item VulnWithCounts
		if err := rows.Scan(
			&item.ID, &item.OsvID, &item.CveID,
			&item.Ecosystem, &item.PackageName,
			&item.Severity, &item.CvssScore,
			&item.Summary,
			&item.PublishedAt,
			&item.AffectedRepoCount,
			&item.AffectedTeamCount,
			&total,
		); err != nil {
			return nil, 0, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

// GetDetail returns full vulnerability details for a specific vuln within the org,
// including the list of affected repositories with team attribution.
// Returns nil when the vulnerability is not found or does not affect the org.
func (s *Store) GetDetail(ctx context.Context, orgID uuid.UUID, vulnID uuid.UUID) (*VulnDetail, error) {
	// First fetch the vulnerability metadata.
	var v VulnDetail
	var rangesJSON []byte

	err := s.db.QueryRow(ctx, `
		SELECT
			v.id, v.osv_id, v.cve_id,
			v.ecosystem, v.package_name,
			v.severity, v.cvss_score, v.cvss_vector,
			v.summary, v.details,
			v.published_at, v.modified_at,
			v.fixed_version, v.introduced_version,
			v.affected_ranges,
			v.created_at, v.updated_at
		FROM vulnerabilities v
		JOIN dependency_vulnerabilities dv ON dv.vuln_id = v.id
		JOIN dependencies d                ON d.id = dv.dep_id
		JOIN repo_dependencies rd          ON rd.dep_id = d.id
		JOIN repositories r                ON r.id = rd.repo_id
		WHERE r.org_id = $1
		  AND v.id = $2
		LIMIT 1
	`, orgID, vulnID).Scan(
		&v.ID, &v.OsvID, &v.CveID,
		&v.Ecosystem, &v.PackageName,
		&v.Severity, &v.CvssScore, &v.CvssVector,
		&v.Summary, &v.Details,
		&v.PublishedAt, &v.ModifiedAt,
		&v.FixedVersion, &v.IntroducedVersion,
		&rangesJSON,
		&v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		// pgx returns pgx.ErrNoRows when not found; treat as nil result.
		return nil, nil //nolint:nilerr
	}

	if len(rangesJSON) > 0 {
		if err := json.Unmarshal(rangesJSON, &v.AffectedRanges); err != nil {
			return nil, err
		}
	}

	// Fetch affected repos with team attribution.
	rows, err := s.db.Query(ctx, `
		SELECT
			r.id, r.name,
			rd.version, rd.dep_type,
			COALESCE(
				ARRAY_AGG(DISTINCT ro.owner) FILTER (WHERE ro.owner IS NOT NULL AND ro.owner_type = 'team'),
				'{}'
			) AS teams
		FROM dependency_vulnerabilities dv
		JOIN dependencies d       ON d.id = dv.dep_id
		JOIN repo_dependencies rd ON rd.dep_id = d.id
		JOIN repositories r       ON r.id = rd.repo_id
		LEFT JOIN repo_owners ro  ON ro.repo_id = r.id AND ro.owner_type = 'team'
		WHERE dv.vuln_id = $1
		  AND r.org_id = $2
		GROUP BY r.id, r.name, rd.version, rd.dep_type
		ORDER BY r.name
	`, vulnID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	v.AffectedRepos = make([]AffectedRepo, 0)
	for rows.Next() {
		var ar AffectedRepo
		if err := rows.Scan(&ar.RepoID, &ar.RepoName, &ar.DepVersion, &ar.DepType, &ar.Teams); err != nil {
			return nil, err
		}
		v.AffectedRepos = append(v.AffectedRepos, ar)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &v, nil
}

// ListOrgDepPairs returns one row per unique (dep_id, ecosystem, name, version)
// combination across all repositories in the given org. Versions are taken from
// repo_dependencies — one row per unique dep within the org is sufficient for
// the MVP batch query (no lock-file deduplication yet).
func (s *Store) ListOrgDepPairs(ctx context.Context, orgID uuid.UUID) ([]DepPair, error) {
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT ON (d.id)
			d.id, d.ecosystem, d.name, rd.version
		FROM dependencies d
		JOIN repo_dependencies rd ON rd.dep_id = d.id
		JOIN repositories r       ON r.id = rd.repo_id
		WHERE r.org_id = $1
		ORDER BY d.id, rd.updated_at DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]DepPair, 0)
	for rows.Next() {
		var dp DepPair
		if err := rows.Scan(&dp.DepID, &dp.Ecosystem, &dp.Name, &dp.Version); err != nil {
			return nil, err
		}
		result = append(result, dp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
