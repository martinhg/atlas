package vuln

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// osvQuerier is the subset of the OSV client the service depends on. Declaring
// it locally keeps the service testable with a stub and decoupled from the
// concrete HTTP client.
type osvQuerier interface {
	QueryBatch(ctx context.Context, packages []DepPair) ([]OSVResult, error)
}

// Service orchestrates vulnerability sync: it gathers an org's dependencies,
// queries OSV.dev, and persists vulnerabilities plus their dependency links.
// It satisfies the org package's VulnSyncer interface.
type Service struct {
	store VulnStore
	osv   osvQuerier
}

// NewService constructs a Service from a store and an OSV querier.
func NewService(store VulnStore, osv osvQuerier) *Service {
	return &Service{store: store, osv: osv}
}

// SyncOrgVulns queries OSV.dev for every dependency in the org and rebuilds the
// dependency_vulnerabilities links. OSV failures are logged but never returned —
// vuln sync is additive and MUST NOT block the parent org sync.
func (s *Service) SyncOrgVulns(ctx context.Context, orgID uuid.UUID) error {
	pairs, err := s.store.ListOrgDepPairs(ctx, orgID)
	if err != nil {
		return err
	}
	if len(pairs) == 0 {
		return nil
	}

	results, err := s.osv.QueryBatch(ctx, pairs)
	if err != nil {
		// Non-blocking: log and return nil without touching the store.
		slog.Error("vuln sync: OSV query failed", "org_id", orgID, "error", err)
		return nil
	}

	// Rebuild links from scratch to avoid stale data.
	if err := s.store.DeleteDepVulnsByOrg(ctx, orgID); err != nil {
		return err
	}

	for _, res := range results {
		for _, ov := range res.Vulns {
			v := toVulnerability(ov, res.Dep.Ecosystem, res.Dep.Name)
			if err := s.store.UpsertVulnerability(ctx, v); err != nil {
				slog.Error("vuln sync: upsert vulnerability failed", "osv_id", v.OsvID, "error", err)
				continue
			}
			if !isAffected(res.Dep.Version, v.AffectedRanges) {
				continue
			}
			if err := s.store.UpsertDepVuln(ctx, res.Dep.DepID, v.ID); err != nil {
				slog.Error("vuln sync: upsert dep-vuln link failed", "dep_id", res.Dep.DepID, "vuln_id", v.ID, "error", err)
			}
		}
	}

	return nil
}

// toVulnerability maps a raw OSV advisory to a domain Vulnerability for the
// given dependency's ecosystem and name.
func toVulnerability(ov OSVVuln, ecosystem, name string) *Vulnerability {
	score := extractCVSS(ov.Severity)
	ranges := extractRanges(ov.Affected, ecosystem, name)

	v := &Vulnerability{
		OsvID:          ov.ID,
		CveID:          extractCVE(ov.Aliases),
		Ecosystem:      ecosystem,
		PackageName:    name,
		Severity:       severityFromCVSS(score),
		CvssScore:      score,
		AffectedRanges: ranges,
		PublishedAt:    ov.Published,
		ModifiedAt:     ov.Modified,
	}

	if ov.Summary != "" {
		summary := ov.Summary
		v.Summary = &summary
	}
	if ov.Details != "" {
		details := ov.Details
		v.Details = &details
	}
	if len(ranges) > 0 {
		if ranges[0].Introduced != "" {
			introduced := ranges[0].Introduced
			v.IntroducedVersion = &introduced
		}
		if ranges[0].Fixed != "" {
			fixed := ranges[0].Fixed
			v.FixedVersion = &fixed
		}
	}

	return v
}

// extractCVE returns the first CVE-prefixed alias, or nil when none is present.
func extractCVE(aliases []string) *string {
	for _, a := range aliases {
		if strings.HasPrefix(a, "CVE-") {
			cve := a
			return &cve
		}
	}
	return nil
}

// extractCVSS returns the numeric CVSS base score, preferring V3 > V2 > V4.
// OSV may provide the score as a CVSS vector string; computing a base score
// from a vector is out of scope, so a non-numeric score yields nil (unknown).
func extractCVSS(severities []OSVSeverity) *float64 {
	byType := make(map[string]string, len(severities))
	for _, sev := range severities {
		byType[sev.Type] = sev.Score
	}
	for _, t := range []string{"CVSS_V3", "CVSS_V2", "CVSS_V4"} {
		raw, ok := byType[t]
		if !ok {
			continue
		}
		if f, err := strconv.ParseFloat(raw, 64); err == nil {
			return &f
		}
	}
	return nil
}

// severityFromCVSS maps a CVSS base score to a SeverityLevel.
func severityFromCVSS(score *float64) SeverityLevel {
	if score == nil {
		return SeverityUnknown
	}
	switch {
	case *score >= 9.0:
		return SeverityCritical
	case *score >= 7.0:
		return SeverityHigh
	case *score >= 4.0:
		return SeverityMedium
	case *score > 0:
		return SeverityLow
	default:
		return SeverityUnknown
	}
}

// extractRanges flattens OSV affected ranges for the matching package into
// [introduced, fixed) pairs. GIT ranges are ignored (no semver semantics).
func extractRanges(affected []OSVAffected, ecosystem, name string) []AffectedRange {
	var out []AffectedRange
	for _, a := range affected {
		if !strings.EqualFold(a.Package.Ecosystem, ecosystem) || a.Package.Name != name {
			continue
		}
		for _, rg := range a.Ranges {
			if rg.Type == "GIT" {
				continue
			}
			var cur AffectedRange
			open := false
			for _, e := range rg.Events {
				if e.Introduced != "" {
					if open {
						out = append(out, cur)
					}
					cur = AffectedRange{Introduced: e.Introduced}
					open = true
				}
				if e.Fixed != "" {
					cur.Fixed = e.Fixed
					out = append(out, cur)
					cur = AffectedRange{}
					open = false
				}
			}
			if open {
				out = append(out, cur)
			}
		}
	}
	return out
}
