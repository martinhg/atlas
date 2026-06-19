// Package risk provides the shared heuristic v1 risk-scoring engine for Atlas.
// It is imported by internal/impact (blast-radius) and internal/graph (graph viz)
// so the same formula drives both features with a single source of truth.
package risk

// RiskLevel classifies the heuristic blast-radius risk of a dependency.
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// Affected describes one repository that declares a dependency under analysis,
// including its dependency type and owning teams.
type Affected struct {
	// DepType is the declared relationship type (e.g. "direct", "dev", "peer").
	DepType string
	// Teams is the list of team owners for this repository.
	Teams []string
}

// depTypeWeights maps a dependency type to its risk weight. Higher weight
// means a dependency of that type contributes more to the overall risk score.
var depTypeWeights = map[string]float64{
	"direct":   1.0,
	"dep":      1.0,
	"peer":     0.8,
	"optional": 0.5,
	"dev":      0.3,
	"devDep":   0.3,
}

// riskLowThreshold and riskMediumThreshold define the score boundaries that
// map a numeric risk score to a RiskLevel.
const (
	riskLowThreshold    = 2.0
	riskMediumThreshold = 5.0
)

// ComputeRiskScore implements the heuristic v1 risk formula:
//
//	score = repo_count * dep_type_weight * team_spread_factor
//
// dep_type_weight is the highest weight among the affected repos' dep types.
// team_spread_factor = distinct_team_count / max(1, repo_count), normalized to [0,1].
// The score is then mapped to a RiskLevel: <2.0 Low, <5.0 Medium, >=5.0 High.
//
// When repos is empty (no vuln/dep data available) it returns (0, RiskLow),
// gracefully degrading without errors.
func ComputeRiskScore(repos []Affected) (float64, RiskLevel) {
	repoCount := len(repos)
	if repoCount == 0 {
		return 0, RiskLow
	}

	maxWeight := 0.0
	teams := make(map[string]struct{})
	for _, repo := range repos {
		if w, ok := depTypeWeights[repo.DepType]; ok && w > maxWeight {
			maxWeight = w
		}
		for _, team := range repo.Teams {
			teams[team] = struct{}{}
		}
	}

	teamSpread := float64(len(teams)) / float64(repoCount)
	if teamSpread > 1 {
		teamSpread = 1
	}

	score := float64(repoCount) * maxWeight * (0.5 + 0.5*teamSpread)

	return score, riskLevelFromScore(score)
}

// riskLevelFromScore maps a numeric risk score to a RiskLevel bucket.
func riskLevelFromScore(score float64) RiskLevel {
	switch {
	case score < riskLowThreshold:
		return RiskLow
	case score < riskMediumThreshold:
		return RiskMedium
	default:
		return RiskHigh
	}
}
