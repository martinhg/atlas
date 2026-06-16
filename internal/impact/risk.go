package impact

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

// computeRiskScore implements the heuristic v1 risk formula:
//
//	score = repo_count * dep_type_weight * team_spread_factor
//
// dep_type_weight is the highest weight among the affected repos' dep types.
// team_spread_factor = distinct_team_count / max(1, repo_count), normalized to [0,1].
// The score is then mapped to a RiskLevel: <2.0 Low, <5.0 Medium, >=5.0 High.
func computeRiskScore(repos []AffectedRepo) (float64, RiskLevel) {
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
