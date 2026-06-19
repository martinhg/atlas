package impact

// AffectedRepo describes one repository that declares the queried dependency,
// including the version, dependency type, and owning teams.
type AffectedRepo struct {
	RepoName string   `json:"repo_name"`
	FullName string   `json:"full_name"`
	Version  string   `json:"version"`
	DepType  string   `json:"dep_type"`
	Teams    []string `json:"teams"`
}

// VersionDist describes how many affected repos use a given version of the
// queried dependency.
type VersionDist struct {
	Version   string `json:"version"`
	RepoCount int    `json:"repo_count"`
}

// DependencyRef identifies the dependency that was queried.
type DependencyRef struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
}

// BlastRadius is the full response envelope for an impact analysis query.
type BlastRadius struct {
	Dependency          DependencyRef  `json:"dependency"`
	TotalRepos          int            `json:"total_repos"`
	RiskLevel           RiskLevel      `json:"risk_level"`
	RiskScore           float64        `json:"risk_score"`
	AffectedRepos       []AffectedRepo `json:"affected_repos"`
	VersionDistribution []VersionDist  `json:"version_distribution"`
}
