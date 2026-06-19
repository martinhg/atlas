package vuln

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// mockOrgResolver is a test double for the OrgResolver interface.
type mockOrgResolver struct {
	orgID uuid.UUID
	found bool
	err   error
}

func (m *mockOrgResolver) GetOrgIDBySlug(_ context.Context, _ string) (uuid.UUID, bool, error) {
	return m.orgID, m.found, m.err
}

// mockVulnReader is a test double for the handler's read interface.
type mockVulnReader struct {
	listResult   []VulnWithCounts
	listTotal    int
	listErr      error
	gotSeverity  string
	gotPackage   string
	gotPage      int
	gotPerPage   int
	detailResult *VulnDetail
	detailErr    error
}

func (m *mockVulnReader) ListByOrg(_ context.Context, _ uuid.UUID, severity, packageName string, page, perPage int) ([]VulnWithCounts, int, error) {
	m.gotSeverity = severity
	m.gotPackage = packageName
	m.gotPage = page
	m.gotPerPage = perPage
	return m.listResult, m.listTotal, m.listErr
}

func (m *mockVulnReader) GetDetail(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*VulnDetail, error) {
	return m.detailResult, m.detailErr
}

func newTestRouter(h *Handler) chi.Router {
	r := chi.NewRouter()
	r.Get("/orgs/{slug}/vulnerabilities", h.HandleListVulnerabilities)
	r.Get("/orgs/{slug}/vulnerabilities/{id}", h.HandleGetVulnerability)
	return r
}

func TestHandleList_200WithEnvelope(t *testing.T) {
	resolver := &mockOrgResolver{orgID: uuid.New(), found: true}
	store := &mockVulnReader{
		listResult: []VulnWithCounts{
			{Vulnerability: Vulnerability{OsvID: "GHSA-aaaa", Severity: SeverityCritical}, AffectedRepoCount: 3},
		},
		listTotal: 1,
	}
	r := newTestRouter(NewHandler(store, resolver))

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/vulnerabilities?page=1&perPage=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp listResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != 1 || len(resp.Data) != 1 || resp.Page != 1 || resp.PerPage != 20 {
		t.Errorf("unexpected envelope: %#v", resp)
	}
}

func TestHandleList_severityFilterPassedToStore(t *testing.T) {
	resolver := &mockOrgResolver{orgID: uuid.New(), found: true}
	store := &mockVulnReader{listResult: []VulnWithCounts{}, listTotal: 0}
	r := newTestRouter(NewHandler(store, resolver))

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/vulnerabilities?severity=critical", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if store.gotSeverity != "critical" {
		t.Errorf("severity passed to store = %q, want critical", store.gotSeverity)
	}
}

func TestHandleList_packageFilterPassedToStore(t *testing.T) {
	resolver := &mockOrgResolver{orgID: uuid.New(), found: true}
	store := &mockVulnReader{listResult: []VulnWithCounts{}, listTotal: 0}
	r := newTestRouter(NewHandler(store, resolver))

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/vulnerabilities?package=lodash", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if store.gotPackage != "lodash" {
		t.Errorf("package passed to store = %q, want lodash", store.gotPackage)
	}
}

func TestHandleList_invalidSeverity_400(t *testing.T) {
	resolver := &mockOrgResolver{orgID: uuid.New(), found: true}
	store := &mockVulnReader{}
	r := newTestRouter(NewHandler(store, resolver))

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/vulnerabilities?severity=banana", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var body map[string]string
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "invalid severity value" {
		t.Errorf("error = %q", body["error"])
	}
}

func TestHandleList_emptyOrg_200EmptyData(t *testing.T) {
	resolver := &mockOrgResolver{orgID: uuid.New(), found: true}
	store := &mockVulnReader{listResult: nil, listTotal: 0}
	r := newTestRouter(NewHandler(store, resolver))

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/vulnerabilities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// data must serialize as [] not null.
	if !strings.Contains(w.Body.String(), `"data":[]`) {
		t.Errorf("expected empty data array, got %s", w.Body.String())
	}
}

func TestHandleList_orgNotFound_404(t *testing.T) {
	resolver := &mockOrgResolver{found: false}
	r := newTestRouter(NewHandler(&mockVulnReader{}, resolver))

	req := httptest.NewRequest(http.MethodGet, "/orgs/ghost/vulnerabilities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleList_invalidPage_400(t *testing.T) {
	resolver := &mockOrgResolver{orgID: uuid.New(), found: true}
	r := newTestRouter(NewHandler(&mockVulnReader{}, resolver))

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/vulnerabilities?page=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleDetail_200(t *testing.T) {
	vulnID := uuid.New()
	resolver := &mockOrgResolver{orgID: uuid.New(), found: true}
	store := &mockVulnReader{
		detailResult: &VulnDetail{
			Vulnerability: Vulnerability{ID: vulnID, OsvID: "GHSA-aaaa", Severity: SeverityHigh},
			AffectedRepos: []AffectedRepo{{RepoName: "acme/web"}},
		},
	}
	r := newTestRouter(NewHandler(store, resolver))

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/vulnerabilities/"+vulnID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var detail VulnDetail
	if err := json.NewDecoder(w.Body).Decode(&detail); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if detail.OsvID != "GHSA-aaaa" || len(detail.AffectedRepos) != 1 {
		t.Errorf("unexpected detail: %#v", detail)
	}
}

func TestHandleDetail_notFound_404(t *testing.T) {
	resolver := &mockOrgResolver{orgID: uuid.New(), found: true}
	store := &mockVulnReader{detailResult: nil} // not found
	r := newTestRouter(NewHandler(store, resolver))

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/vulnerabilities/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var body map[string]string
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "vulnerability not found" {
		t.Errorf("error = %q", body["error"])
	}
}

func TestHandleDetail_invalidID_404(t *testing.T) {
	resolver := &mockOrgResolver{orgID: uuid.New(), found: true}
	r := newTestRouter(NewHandler(&mockVulnReader{}, resolver))

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/vulnerabilities/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for malformed id, got %d", w.Code)
	}
}

func TestHandleList_storeError_500(t *testing.T) {
	resolver := &mockOrgResolver{orgID: uuid.New(), found: true}
	store := &mockVulnReader{listErr: errors.New("db down")}
	r := newTestRouter(NewHandler(store, resolver))

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/vulnerabilities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
