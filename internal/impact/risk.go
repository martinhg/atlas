package impact

import "github.com/nesbite/atlas/internal/risk"

// RiskLevel aliases risk.RiskLevel so Epic 6's public JSON/API surface is unchanged.
type RiskLevel = risk.RiskLevel

// Re-export the risk-level constants as package-level values so existing callers
// inside the impact package compile without modification.
const (
	RiskLow    = risk.RiskLow
	RiskMedium = risk.RiskMedium
	RiskHigh   = risk.RiskHigh
)

// computeRiskScore delegates to the shared risk engine. The signature is kept
// identical (unexported, same input/output) so all existing impact callsites
// continue to work without change.
func computeRiskScore(repos []AffectedRepo) (float64, RiskLevel) {
	affected := make([]risk.Affected, len(repos))
	for i, r := range repos {
		affected[i] = risk.Affected{
			DepType: r.DepType,
			Teams:   r.Teams,
		}
	}
	return risk.ComputeRiskScore(affected)
}
