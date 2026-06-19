package vuln

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
)

// recordingStore is a mock VulnStore that records call order and arguments.
type recordingStore struct {
	calls         []string
	pairs         []DepPair
	pairsErr      error
	deleteErr     error
	upsertVulnErr error
	upsertedVulns []*Vulnerability
	depVulnLinks  []depVulnLink
}

type depVulnLink struct {
	depID  uuid.UUID
	vulnID uuid.UUID
}

func (s *recordingStore) UpsertVulnerability(_ context.Context, v *Vulnerability) error {
	s.calls = append(s.calls, "UpsertVulnerability")
	if s.upsertVulnErr != nil {
		return s.upsertVulnErr
	}
	v.ID = uuid.New() // simulate RETURNING id
	s.upsertedVulns = append(s.upsertedVulns, v)
	return nil
}

func (s *recordingStore) UpsertDepVuln(_ context.Context, depID, vulnID uuid.UUID) error {
	s.calls = append(s.calls, "UpsertDepVuln")
	s.depVulnLinks = append(s.depVulnLinks, depVulnLink{depID: depID, vulnID: vulnID})
	return nil
}

func (s *recordingStore) DeleteDepVulnsByOrg(_ context.Context, _ uuid.UUID) error {
	s.calls = append(s.calls, "DeleteDepVulnsByOrg")
	return s.deleteErr
}

func (s *recordingStore) ListByOrg(_ context.Context, _ uuid.UUID, _, _ string, _, _ int) ([]VulnWithCounts, int, error) {
	return nil, 0, nil
}

func (s *recordingStore) GetDetail(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*VulnDetail, error) {
	return nil, nil
}

func (s *recordingStore) ListOrgDepPairs(_ context.Context, _ uuid.UUID) ([]DepPair, error) {
	s.calls = append(s.calls, "ListOrgDepPairs")
	return s.pairs, s.pairsErr
}

// stubOSV is a mock osvQuerier.
type stubOSV struct {
	called  bool
	results []OSVResult
	err     error
}

func (o *stubOSV) QueryBatch(_ context.Context, _ []DepPair) ([]OSVResult, error) {
	o.called = true
	return o.results, o.err
}

func TestSyncOrgVulns_noDeps_skipsOSV(t *testing.T) {
	store := &recordingStore{pairs: nil}
	osv := &stubOSV{}
	svc := NewService(store, osv)

	if err := svc.SyncOrgVulns(context.Background(), uuid.New()); err != nil {
		t.Fatalf("SyncOrgVulns: %v", err)
	}
	if osv.called {
		t.Error("expected OSV not called when there are no deps")
	}
}

func TestSyncOrgVulns_upsertsAndLinksAffected(t *testing.T) {
	dep := DepPair{DepID: uuid.New(), Ecosystem: "npm", Name: "express", Version: "4.17.21"}
	store := &recordingStore{pairs: []DepPair{dep}}
	osv := &stubOSV{
		results: []OSVResult{
			{
				Dep: dep,
				Vulns: []OSVVuln{
					{
						ID:       "GHSA-aaaa",
						Aliases:  []string{"CVE-2021-1234"},
						Severity: []OSVSeverity{{Type: "CVSS_V3", Score: "9.8"}},
						Affected: []OSVAffected{
							{
								Package: OSVPackage{Ecosystem: "npm", Name: "express"},
								Ranges: []OSVRange{
									{Type: "SEMVER", Events: []OSVEvent{{Introduced: "0"}, {Fixed: "4.18.0"}}},
								},
							},
						},
					},
				},
			},
		},
	}
	svc := NewService(store, osv)

	if err := svc.SyncOrgVulns(context.Background(), uuid.New()); err != nil {
		t.Fatalf("SyncOrgVulns: %v", err)
	}

	if len(store.upsertedVulns) != 1 {
		t.Fatalf("expected 1 vuln upserted, got %d", len(store.upsertedVulns))
	}
	v := store.upsertedVulns[0]
	if v.Severity != SeverityCritical {
		t.Errorf("severity = %q, want critical", v.Severity)
	}
	if v.CvssScore == nil || *v.CvssScore != 9.8 {
		t.Errorf("cvss score = %v, want 9.8", v.CvssScore)
	}
	if v.CveID == nil || *v.CveID != "CVE-2021-1234" {
		t.Errorf("cve = %v, want CVE-2021-1234", v.CveID)
	}
	if len(store.depVulnLinks) != 1 || store.depVulnLinks[0].depID != dep.DepID {
		t.Errorf("expected 1 dep-vuln link for affected dep, got %#v", store.depVulnLinks)
	}
}

func TestSyncOrgVulns_unaffectedVersion_noLink(t *testing.T) {
	// dep version 4.18.1 is outside introduced=0 fixed=4.18.0 → not affected.
	dep := DepPair{DepID: uuid.New(), Ecosystem: "npm", Name: "express", Version: "4.18.1"}
	store := &recordingStore{pairs: []DepPair{dep}}
	osv := &stubOSV{
		results: []OSVResult{
			{
				Dep: dep,
				Vulns: []OSVVuln{
					{
						ID: "GHSA-aaaa",
						Affected: []OSVAffected{
							{
								Package: OSVPackage{Ecosystem: "npm", Name: "express"},
								Ranges:  []OSVRange{{Type: "SEMVER", Events: []OSVEvent{{Introduced: "0"}, {Fixed: "4.18.0"}}}},
							},
						},
					},
				},
			},
		},
	}
	svc := NewService(store, osv)

	if err := svc.SyncOrgVulns(context.Background(), uuid.New()); err != nil {
		t.Fatalf("SyncOrgVulns: %v", err)
	}
	if len(store.upsertedVulns) != 1 {
		t.Errorf("vuln should still be upserted, got %d", len(store.upsertedVulns))
	}
	if len(store.depVulnLinks) != 0 {
		t.Errorf("expected no dep-vuln link for unaffected version, got %#v", store.depVulnLinks)
	}
}

func TestSyncOrgVulns_deleteRunsBeforeUpserts(t *testing.T) {
	dep := DepPair{DepID: uuid.New(), Ecosystem: "npm", Name: "express", Version: "4.17.21"}
	store := &recordingStore{pairs: []DepPair{dep}}
	osv := &stubOSV{
		results: []OSVResult{
			{Dep: dep, Vulns: []OSVVuln{{ID: "GHSA-aaaa"}}},
		},
	}
	svc := NewService(store, osv)

	if err := svc.SyncOrgVulns(context.Background(), uuid.New()); err != nil {
		t.Fatalf("SyncOrgVulns: %v", err)
	}

	deleteIdx, upsertIdx := -1, -1
	for i, c := range store.calls {
		if c == "DeleteDepVulnsByOrg" && deleteIdx == -1 {
			deleteIdx = i
		}
		if c == "UpsertVulnerability" && upsertIdx == -1 {
			upsertIdx = i
		}
	}
	if deleteIdx == -1 {
		t.Fatal("DeleteDepVulnsByOrg was not called")
	}
	if upsertIdx != -1 && deleteIdx > upsertIdx {
		t.Errorf("expected delete before upserts, got calls %v", store.calls)
	}
}

func TestSyncOrgVulns_osvError_nonBlocking_noWrites(t *testing.T) {
	dep := DepPair{DepID: uuid.New(), Ecosystem: "npm", Name: "express", Version: "4.17.21"}
	store := &recordingStore{pairs: []DepPair{dep}}
	osv := &stubOSV{err: fmt.Errorf("osv unavailable")}
	svc := NewService(store, osv)

	// OSV failure must not propagate (non-blocking).
	if err := svc.SyncOrgVulns(context.Background(), uuid.New()); err != nil {
		t.Fatalf("expected nil error on OSV failure, got %v", err)
	}
	for _, c := range store.calls {
		if c == "DeleteDepVulnsByOrg" || c == "UpsertVulnerability" || c == "UpsertDepVuln" {
			t.Errorf("expected no store writes on OSV failure, got calls %v", store.calls)
			break
		}
	}
}

func TestSyncOrgVulns_noCVSS_unknownSeverity(t *testing.T) {
	dep := DepPair{DepID: uuid.New(), Ecosystem: "npm", Name: "express", Version: "4.17.21"}
	store := &recordingStore{pairs: []DepPair{dep}}
	osv := &stubOSV{
		results: []OSVResult{
			{Dep: dep, Vulns: []OSVVuln{{ID: "GHSA-aaaa"}}}, // no severity
		},
	}
	svc := NewService(store, osv)

	if err := svc.SyncOrgVulns(context.Background(), uuid.New()); err != nil {
		t.Fatalf("SyncOrgVulns: %v", err)
	}
	if len(store.upsertedVulns) != 1 {
		t.Fatalf("expected 1 vuln, got %d", len(store.upsertedVulns))
	}
	v := store.upsertedVulns[0]
	if v.Severity != SeverityUnknown {
		t.Errorf("severity = %q, want unknown", v.Severity)
	}
	if v.CvssScore != nil {
		t.Errorf("cvss score = %v, want nil", v.CvssScore)
	}
}
