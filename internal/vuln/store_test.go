package vuln

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// compile-time check: *Store must satisfy VulnStore.
var _ VulnStore = (*Store)(nil)

// compile-time check: *mockVulnStore must satisfy VulnStore.
var _ VulnStore = (*mockVulnStore)(nil)

// mockVulnStore is a test double for VulnStore used in unit tests.
type mockVulnStore struct {
	// upsertVulnErr controls the error returned by UpsertVulnerability.
	upsertVulnErr error
	// upsertDepVulnErr controls the error returned by UpsertDepVuln.
	upsertDepVulnErr error
	// deleteDepVulnsErr controls the error returned by DeleteDepVulnsByOrg.
	deleteDepVulnsErr error
	// listByOrgResult is the list of vulns returned by ListByOrg.
	listByOrgResult []VulnWithCounts
	// listByOrgTotal is the total count returned by ListByOrg.
	listByOrgTotal int
	// listByOrgErr controls the error returned by ListByOrg.
	listByOrgErr error
	// getDetailResult is returned by GetDetail (nil means not found).
	getDetailResult *VulnDetail
	// getDetailErr controls the error returned by GetDetail.
	getDetailErr error
	// listDepPairsResult is returned by ListOrgDepPairs.
	listDepPairsResult []DepPair
	// listDepPairsErr controls the error returned by ListOrgDepPairs.
	listDepPairsErr error

	// call trackers
	upsertVulnCalls    []*Vulnerability
	upsertDepVulnCalls [][2]uuid.UUID // [depID, vulnID]
	deleteOrgIDCalls   []uuid.UUID
}

func (m *mockVulnStore) UpsertVulnerability(_ context.Context, v *Vulnerability) error {
	m.upsertVulnCalls = append(m.upsertVulnCalls, v)
	return m.upsertVulnErr
}

func (m *mockVulnStore) UpsertDepVuln(_ context.Context, depID, vulnID uuid.UUID) error {
	m.upsertDepVulnCalls = append(m.upsertDepVulnCalls, [2]uuid.UUID{depID, vulnID})
	return m.upsertDepVulnErr
}

func (m *mockVulnStore) DeleteDepVulnsByOrg(_ context.Context, orgID uuid.UUID) error {
	m.deleteOrgIDCalls = append(m.deleteOrgIDCalls, orgID)
	return m.deleteDepVulnsErr
}

func (m *mockVulnStore) ListByOrg(_ context.Context, _ uuid.UUID, _, _ string, _, _ int) ([]VulnWithCounts, int, error) {
	return m.listByOrgResult, m.listByOrgTotal, m.listByOrgErr
}

func (m *mockVulnStore) GetDetail(_ context.Context, _, _ uuid.UUID) (*VulnDetail, error) {
	return m.getDetailResult, m.getDetailErr
}

func (m *mockVulnStore) ListOrgDepPairs(_ context.Context, _ uuid.UUID) ([]DepPair, error) {
	return m.listDepPairsResult, m.listDepPairsErr
}

// TestMockVulnStore_UpsertVulnerability verifies the mock tracks calls correctly.
func TestMockVulnStore_UpsertVulnerability(t *testing.T) {
	m := &mockVulnStore{}
	v := &Vulnerability{ID: uuid.New(), OsvID: "GHSA-test"}

	if err := m.UpsertVulnerability(context.Background(), v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.upsertVulnCalls) != 1 {
		t.Errorf("expected 1 call, got %d", len(m.upsertVulnCalls))
	}
}

// TestMockVulnStore_UpsertVulnerability_error verifies error propagation.
func TestMockVulnStore_UpsertVulnerability_error(t *testing.T) {
	m := &mockVulnStore{upsertVulnErr: errors.New("db error")}
	v := &Vulnerability{ID: uuid.New(), OsvID: "GHSA-test"}

	err := m.UpsertVulnerability(context.Background(), v)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestMockVulnStore_ListByOrg_empty verifies empty result.
func TestMockVulnStore_ListByOrg_empty(t *testing.T) {
	m := &mockVulnStore{
		listByOrgResult: []VulnWithCounts{},
		listByOrgTotal:  0,
	}

	result, total, err := m.ListByOrg(context.Background(), uuid.New(), "", "", 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
}

// TestMockVulnStore_GetDetail_notFound verifies nil result for missing vuln.
func TestMockVulnStore_GetDetail_notFound(t *testing.T) {
	m := &mockVulnStore{getDetailResult: nil}

	result, err := m.GetDetail(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

// TestMockVulnStore_DeleteDepVulnsByOrg_tracksOrgID verifies call tracking.
func TestMockVulnStore_DeleteDepVulnsByOrg_tracksOrgID(t *testing.T) {
	m := &mockVulnStore{}
	orgID := uuid.New()

	if err := m.DeleteDepVulnsByOrg(context.Background(), orgID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.deleteOrgIDCalls) != 1 || m.deleteOrgIDCalls[0] != orgID {
		t.Errorf("expected orgID %s tracked, got %v", orgID, m.deleteOrgIDCalls)
	}
}

// TestMockVulnStore_ListOrgDepPairs_returnsData verifies dep pairs returned.
func TestMockVulnStore_ListOrgDepPairs_returnsData(t *testing.T) {
	depID := uuid.New()
	m := &mockVulnStore{
		listDepPairsResult: []DepPair{
			{DepID: depID, Ecosystem: "npm", Name: "lodash", Version: "^4.17.21"},
		},
	}

	pairs, err := m.ListOrgDepPairs(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].Name != "lodash" {
		t.Errorf("expected pair name 'lodash', got %q", pairs[0].Name)
	}
}
