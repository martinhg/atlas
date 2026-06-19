package vuln

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestSeverityLevel_constants verifies all SeverityLevel constants are defined.
func TestSeverityLevel_constants(t *testing.T) {
	tests := []struct {
		name  string
		value SeverityLevel
		want  string
	}{
		{"critical", SeverityCritical, "critical"},
		{"high", SeverityHigh, "high"},
		{"medium", SeverityMedium, "medium"},
		{"low", SeverityLow, "low"},
		{"unknown", SeverityUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.want {
				t.Errorf("SeverityLevel constant %s = %q, want %q", tt.name, string(tt.value), tt.want)
			}
		})
	}
}

// TestVulnerability_fields verifies the Vulnerability struct has expected exported fields.
func TestVulnerability_fields(t *testing.T) {
	v := Vulnerability{
		ID:          uuid.New(),
		OsvID:       "GHSA-xxxx-xxxx-xxxx",
		CveID:       strPtr("CVE-2021-12345"),
		Ecosystem:   "npm",
		PackageName: "lodash",
		Severity:    SeverityCritical,
		CvssScore:   float64Ptr(9.8),
		CvssVector:  strPtr("CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"),
		Summary:     strPtr("Prototype pollution"),
		Details:     strPtr("Full details here"),
		PublishedAt: timePtr(time.Now()),
		ModifiedAt:  timePtr(time.Now()),
		AffectedRanges: []AffectedRange{
			{Introduced: "4.0.0", Fixed: "4.17.21"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if v.ID == uuid.Nil {
		t.Error("ID must be settable")
	}
	if v.OsvID == "" {
		t.Error("OsvID must be settable")
	}
	if v.Severity != SeverityCritical {
		t.Error("Severity must be settable")
	}
}

// TestVulnWithCounts_embedsVulnerability verifies VulnWithCounts embeds Vulnerability
// and adds count fields.
func TestVulnWithCounts_embedsVulnerability(t *testing.T) {
	v := VulnWithCounts{
		Vulnerability: Vulnerability{
			ID:          uuid.New(),
			OsvID:       "GHSA-test",
			Ecosystem:   "npm",
			PackageName: "express",
			Severity:    SeverityHigh,
		},
		AffectedRepoCount: 5,
		AffectedTeamCount: 2,
	}

	if v.OsvID != "GHSA-test" {
		t.Error("VulnWithCounts must embed Vulnerability fields")
	}
	if v.AffectedRepoCount != 5 {
		t.Error("AffectedRepoCount must be settable")
	}
	if v.AffectedTeamCount != 2 {
		t.Error("AffectedTeamCount must be settable")
	}
}

// TestVulnDetail_includesAffectedRepos verifies VulnDetail has AffectedRepos.
func TestVulnDetail_includesAffectedRepos(t *testing.T) {
	d := VulnDetail{
		Vulnerability: Vulnerability{
			ID:          uuid.New(),
			OsvID:       "GHSA-detail-test",
			Ecosystem:   "npm",
			PackageName: "axios",
			Severity:    SeverityMedium,
		},
		AffectedRepos: []AffectedRepo{
			{
				RepoID:     uuid.New(),
				RepoName:   "org/repo",
				DepVersion: "^1.0.0",
				DepType:    "dep",
				Teams:      []string{"platform"},
			},
		},
	}

	if len(d.AffectedRepos) != 1 {
		t.Error("VulnDetail must include AffectedRepos")
	}
	if d.AffectedRepos[0].RepoName != "org/repo" {
		t.Error("AffectedRepo.RepoName must be settable")
	}
}

// TestDepPair_fields verifies DepPair has the expected shape for OSV queries.
func TestDepPair_fields(t *testing.T) {
	dp := DepPair{
		DepID:     uuid.New(),
		Ecosystem: "npm",
		Name:      "lodash",
		Version:   "^4.17.21",
	}

	if dp.Name == "" {
		t.Error("DepPair.Name must be settable")
	}
}

// TestAffectedRange_fields verifies AffectedRange for semver matching.
func TestAffectedRange_fields(t *testing.T) {
	ar := AffectedRange{
		Introduced: "1.0.0",
		Fixed:      "2.0.0",
	}

	if ar.Introduced != "1.0.0" {
		t.Error("AffectedRange.Introduced must be settable")
	}
	if ar.Fixed != "2.0.0" {
		t.Error("AffectedRange.Fixed must be settable")
	}
}

// helpers

func strPtr(s string) *string    { return &s }
func float64Ptr(f float64) *float64 { return &f }
func timePtr(t time.Time) *time.Time { return &t }
