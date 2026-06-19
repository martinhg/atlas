package vuln

import (
	"time"

	"github.com/google/uuid"
)

// SeverityLevel represents the CVSS-derived severity classification of a vulnerability.
type SeverityLevel string

const (
	// SeverityCritical represents CVSS score >= 9.0.
	SeverityCritical SeverityLevel = "critical"
	// SeverityHigh represents CVSS score 7.0–8.9.
	SeverityHigh SeverityLevel = "high"
	// SeverityMedium represents CVSS score 4.0–6.9.
	SeverityMedium SeverityLevel = "medium"
	// SeverityLow represents CVSS score 0.1–3.9.
	SeverityLow SeverityLevel = "low"
	// SeverityUnknown is used when no CVSS score is available.
	SeverityUnknown SeverityLevel = "unknown"
)

// AffectedRange describes a version range [Introduced, Fixed) in which a vulnerability exists.
// Fixed is empty when the vulnerability has no known fix (open upper bound).
type AffectedRange struct {
	Introduced string `json:"introduced"`
	Fixed      string `json:"fixed"`
}

// Vulnerability is the canonical record for a known vulnerability sourced from OSV.dev.
type Vulnerability struct {
	ID                 uuid.UUID      `json:"id"`
	OsvID              string         `json:"osv_id"`
	CveID              *string        `json:"cve_id,omitempty"`
	Ecosystem          string         `json:"ecosystem"`
	PackageName        string         `json:"package_name"`
	Severity           SeverityLevel  `json:"severity"`
	CvssScore          *float64       `json:"cvss_score,omitempty"`
	CvssVector         *string        `json:"cvss_vector,omitempty"`
	Summary            *string        `json:"summary,omitempty"`
	Details            *string        `json:"details,omitempty"`
	PublishedAt        *time.Time     `json:"published_at,omitempty"`
	ModifiedAt         *time.Time     `json:"modified_at,omitempty"`
	FixedVersion       *string        `json:"fixed_version,omitempty"`
	IntroducedVersion  *string        `json:"introduced_version,omitempty"`
	AffectedRanges     []AffectedRange `json:"affected_ranges,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// VulnWithCounts extends Vulnerability with aggregate counts used in the list view.
type VulnWithCounts struct {
	Vulnerability
	AffectedRepoCount int `json:"affected_repo_count"`
	AffectedTeamCount int `json:"affected_team_count"`
}

// AffectedRepo describes one repository affected by a vulnerability, including
// the specific dependency version and team attribution.
type AffectedRepo struct {
	RepoID     uuid.UUID `json:"repo_id"`
	RepoName   string    `json:"repo_name"`
	DepVersion string    `json:"dep_version"`
	DepType    string    `json:"dep_type"`
	Teams      []string  `json:"teams"`
}

// VulnDetail extends Vulnerability with the full list of affected repositories,
// returned by the detail endpoint.
type VulnDetail struct {
	Vulnerability
	AffectedRepos []AffectedRepo `json:"affected_repos"`
}

// DepPair is a (depID, ecosystem, name, version) tuple used when querying OSV
// for vulnerability matches against a specific dependency.
type DepPair struct {
	DepID     uuid.UUID `json:"dep_id"`
	Ecosystem string    `json:"ecosystem"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
}
