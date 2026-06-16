package impact

import "testing"

// TestComputeRiskScore covers the heuristic v1 formula:
// score = repo_count * dep_type_weight * team_spread_factor
// where dep_type_weight maps direct=1.0, peer=0.8, optional=0.5, dev=0.3,
// and team_spread_factor = distinct_team_count / max(1, repo_count), normalized [0,1].
// Thresholds: <2.0 Low, <5.0 Medium, >=5.0 High.
func TestComputeRiskScore(t *testing.T) {
	tests := []struct {
		name      string
		repos     []AffectedRepo
		wantLevel RiskLevel
	}{
		{
			name:      "zero repos yields low risk and zero score",
			repos:     []AffectedRepo{},
			wantLevel: RiskLow,
		},
		{
			name: "single repo dev dependency yields low risk",
			repos: []AffectedRepo{
				{RepoName: "svc-a", DepType: "dev", Teams: []string{"@acme/team-x"}},
			},
			wantLevel: RiskLow,
		},
		{
			name:      "wide direct dependency across many teams yields high risk",
			wantLevel: RiskHigh,
			repos: []AffectedRepo{
				{RepoName: "svc-1", DepType: "direct", Teams: []string{"@acme/team-1"}},
				{RepoName: "svc-2", DepType: "direct", Teams: []string{"@acme/team-2"}},
				{RepoName: "svc-3", DepType: "direct", Teams: []string{"@acme/team-3"}},
				{RepoName: "svc-4", DepType: "direct", Teams: []string{"@acme/team-4"}},
				{RepoName: "svc-5", DepType: "direct", Teams: []string{"@acme/team-5"}},
				{RepoName: "svc-6", DepType: "direct", Teams: []string{"@acme/team-1"}},
				{RepoName: "svc-7", DepType: "direct", Teams: []string{"@acme/team-2"}},
				{RepoName: "svc-8", DepType: "direct", Teams: []string{"@acme/team-3"}},
				{RepoName: "svc-9", DepType: "direct", Teams: []string{"@acme/team-4"}},
				{RepoName: "svc-10", DepType: "direct", Teams: []string{"@acme/team-5"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, level := computeRiskScore(tt.repos)
			if level != tt.wantLevel {
				t.Errorf("computeRiskScore() level = %q, want %q", level, tt.wantLevel)
			}
		})
	}
}

// TestComputeRiskScore_zeroReposReturnsZeroScore verifies the exact score value
// for the empty-input scenario per spec AC "Zero repos affected".
func TestComputeRiskScore_zeroReposReturnsZeroScore(t *testing.T) {
	score, _ := computeRiskScore([]AffectedRepo{})
	if score != 0 {
		t.Errorf("computeRiskScore() score = %v, want 0", score)
	}
}

// TestComputeRiskScore_boundaryValues triangulates the threshold boundaries
// to force the real weighted formula rather than a hardcoded mapping.
func TestComputeRiskScore_boundaryValues(t *testing.T) {
	tests := []struct {
		name      string
		repos     []AffectedRepo
		wantLevel RiskLevel
	}{
		{
			name: "two repos same team direct dependency stays low",
			repos: []AffectedRepo{
				{RepoName: "svc-a", DepType: "direct", Teams: []string{"@acme/team-x"}},
				{RepoName: "svc-b", DepType: "direct", Teams: []string{"@acme/team-x"}},
			},
			wantLevel: RiskLow,
		},
		{
			name: "moderate spread direct dependency yields medium risk",
			repos: []AffectedRepo{
				{RepoName: "svc-a", DepType: "direct", Teams: []string{"@acme/team-x"}},
				{RepoName: "svc-b", DepType: "direct", Teams: []string{"@acme/team-y"}},
				{RepoName: "svc-c", DepType: "direct", Teams: []string{"@acme/team-z"}},
			},
			wantLevel: RiskMedium,
		},
		{
			name: "optional dependency type dampens risk vs direct",
			repos: []AffectedRepo{
				{RepoName: "svc-a", DepType: "optional", Teams: []string{"@acme/team-x"}},
				{RepoName: "svc-b", DepType: "optional", Teams: []string{"@acme/team-y"}},
				{RepoName: "svc-c", DepType: "optional", Teams: []string{"@acme/team-z"}},
			},
			wantLevel: RiskLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, level := computeRiskScore(tt.repos)
			if level != tt.wantLevel {
				t.Errorf("computeRiskScore() level = %q, want %q", level, tt.wantLevel)
			}
		})
	}
}
