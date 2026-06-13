package org

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nesbite/atlas/internal/auth"
)

func newTestHandler(orgS OrgStore) *Handler {
	return NewHandler(orgS, nil, 12345, []byte("fake-key"), "test-webhook-secret")
}

func reqWithUserID(r *http.Request, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), auth.UserIDKey, userID)
	return r.WithContext(ctx)
}

func TestHandleListOrgs_200(t *testing.T) {
	userID := uuid.New()
	orgS := &mockOrgStore{}

	h := newTestHandler(orgS)
	req := httptest.NewRequest(http.MethodGet, "/orgs", nil)
	req = reqWithUserID(req, userID)
	w := httptest.NewRecorder()

	h.HandleListOrgs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestHandleListOrgs_401_no_jwt(t *testing.T) {
	h := newTestHandler(&mockOrgStore{})
	req := httptest.NewRequest(http.MethodGet, "/orgs", nil)
	w := httptest.NewRecorder()

	h.HandleListOrgs(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleGetOrg_200(t *testing.T) {
	orgS := &handlerMockOrgStore{
		getBySlug: &Organization{
			ID:       uuid.New(),
			GitHubID: 1001,
			Name:     "Test Org",
			Slug:     "test-org",
			OwnerID:  uuid.New(),
		},
	}
	h := newTestHandler(orgS)

	r := chi.NewRouter()
	r.Get("/orgs/{slug}", h.HandleGetOrg)

	req := httptest.NewRequest(http.MethodGet, "/orgs/test-org", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleGetOrg_404_not_found(t *testing.T) {
	orgS := &handlerMockOrgStore{getBySlug: nil}
	h := newTestHandler(orgS)

	r := chi.NewRouter()
	r.Get("/orgs/{slug}", h.HandleGetOrg)

	req := httptest.NewRequest(http.MethodGet, "/orgs/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleConnect_400_missing_installation_id(t *testing.T) {
	h := newTestHandler(&handlerMockOrgStore{})
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/connect", bytes.NewBufferString(body))
	req = reqWithUserID(req, uuid.New())
	w := httptest.NewRecorder()

	h.HandleConnect(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleConnect_409_already_linked(t *testing.T) {
	otherOwner := uuid.New()
	orgS := &handlerMockOrgStore{
		getByInstallation: &Organization{
			ID:      uuid.New(),
			OwnerID: otherOwner,
		},
	}
	h := newTestHandler(orgS)

	body := `{"installation_id": 99999}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/connect", bytes.NewBufferString(body))
	req = reqWithUserID(req, uuid.New())
	w := httptest.NewRecorder()

	h.HandleConnect(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandleGitHubWebhook_403_missing_signature(t *testing.T) {
	h := newTestHandler(&handlerMockOrgStore{})
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewBufferString("{}"))
	w := httptest.NewRecorder()

	h.HandleGitHubWebhook(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleGitHubWebhook_403_invalid_signature(t *testing.T) {
	h := newTestHandler(&handlerMockOrgStore{})
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewBufferString("{}"))
	req.Header.Set("X-Hub-Signature-256", "sha256=invalidhex")
	w := httptest.NewRecorder()

	h.HandleGitHubWebhook(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleGitHubWebhook_200_ignored_event(t *testing.T) {
	h := newTestHandler(&handlerMockOrgStore{})
	payload := []byte(`{"action":"deleted"}`)
	sig := signPayload("test-webhook-secret", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewBuffer(payload))
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "push")
	w := httptest.NewRecorder()

	h.HandleGitHubWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ignored" {
		t.Errorf("status = %q, want %q", resp["status"], "ignored")
	}
}

func TestHandleGitHubWebhook_200_installation_created(t *testing.T) {
	orgS := &handlerMockOrgStore{getByInstallation: nil}
	h := newTestHandler(orgS)

	payload := []byte(`{"action":"created","installation":{"id":12345,"account":{"id":100,"login":"my-org"}},"sender":{"id":1}}`)
	sig := signPayload("test-webhook-secret", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewBuffer(payload))
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "installation")
	w := httptest.NewRecorder()

	h.HandleGitHubWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- Additional coverage tests ---

func TestHandleListOrgs_500_storeError(t *testing.T) {
	orgS := &handlerMockOrgStore{orgsErr: errors.New("db down")}
	h := newTestHandler(orgS)

	req := httptest.NewRequest(http.MethodGet, "/orgs", nil)
	req = reqWithUserID(req, uuid.New())
	w := httptest.NewRecorder()

	h.HandleListOrgs(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleGetOrg_500_storeError(t *testing.T) {
	orgS := &handlerMockOrgStore{getBySlugErr: errors.New("db error")}
	h := newTestHandler(orgS)

	r := chi.NewRouter()
	r.Get("/orgs/{slug}", h.HandleGetOrg)

	req := httptest.NewRequest(http.MethodGet, "/orgs/some-slug", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleConnect_401_noJWT(t *testing.T) {
	h := newTestHandler(&handlerMockOrgStore{})
	body := `{"installation_id": 123}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/connect", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.HandleConnect(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleConnect_400_invalidBody(t *testing.T) {
	h := newTestHandler(&handlerMockOrgStore{})
	req := httptest.NewRequest(http.MethodPost, "/orgs/connect", bytes.NewBufferString("not-json"))
	req = reqWithUserID(req, uuid.New())
	w := httptest.NewRecorder()

	h.HandleConnect(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleConnect_500_installationCheckError(t *testing.T) {
	orgS := &handlerMockOrgStore{getByInstErr: errors.New("db error")}
	h := newTestHandler(orgS)

	body := `{"installation_id": 123}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/connect", bytes.NewBufferString(body))
	req = reqWithUserID(req, uuid.New())
	w := httptest.NewRecorder()

	h.HandleConnect(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleGitHubWebhook_200_installation_action_deleted(t *testing.T) {
	orgS := &handlerMockOrgStore{}
	h := newTestHandler(orgS)

	payload := []byte(`{"action":"deleted","installation":{"id":12345,"account":{"id":100,"login":"my-org"}},"sender":{"id":1}}`)
	sig := signPayload("test-webhook-secret", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewBuffer(payload))
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "installation")
	w := httptest.NewRecorder()

	h.HandleGitHubWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ignored" {
		t.Errorf("status = %q, want %q", resp["status"], "ignored")
	}
}

func TestHandleGitHubWebhook_400_invalidJSON(t *testing.T) {
	h := newTestHandler(&handlerMockOrgStore{})
	payload := []byte(`not valid json at all`)
	sig := signPayload("test-webhook-secret", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewBuffer(payload))
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "installation")
	w := httptest.NewRecorder()

	h.HandleGitHubWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleGitHubWebhook_200_existingOrg_syncing(t *testing.T) {
	existingOrg := &Organization{
		ID:      uuid.New(),
		Name:    "my-org",
		Slug:    "my-org",
		OwnerID: uuid.New(),
	}
	orgS := &handlerMockOrgStore{getByInstallation: existingOrg}
	h := newTestHandler(orgS)

	payload := []byte(`{"action":"created","installation":{"id":12345,"account":{"id":100,"login":"my-org"}},"sender":{"id":1}}`)
	sig := signPayload("test-webhook-secret", payload)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewBuffer(payload))
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "installation")
	w := httptest.NewRecorder()

	h.HandleGitHubWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "syncing" {
		t.Errorf("status = %q, want %q", resp["status"], "syncing")
	}
}

func TestRoutes_returnsRouterFunc(t *testing.T) {
	h := newTestHandler(&handlerMockOrgStore{})
	routerFn := h.Routes()
	if routerFn == nil {
		t.Fatal("Routes() returned nil")
	}

	r := chi.NewRouter()
	r.Route("/orgs", routerFn)

	req := httptest.NewRequest(http.MethodGet, "/orgs/", nil)
	req = reqWithUserID(req, uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func signPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

type handlerMockOrgStore struct {
	getBySlug         *Organization
	getBySlugErr      error
	getByInstallation *Organization
	getByInstErr      error
	orgs              []Organization
	orgsErr           error
	upserted          *Organization
	upsertErr         error
}

func (m *handlerMockOrgStore) UpsertOrg(_ context.Context, o *Organization) (*Organization, error) {
	if m.upsertErr != nil {
		return nil, m.upsertErr
	}
	o.ID = uuid.New()
	m.upserted = o
	return o, nil
}

func (m *handlerMockOrgStore) GetOrgBySlug(_ context.Context, _ string) (*Organization, error) {
	return m.getBySlug, m.getBySlugErr
}

func (m *handlerMockOrgStore) GetOrgsByOwnerID(_ context.Context, _ uuid.UUID) ([]Organization, error) {
	if m.orgsErr != nil {
		return nil, m.orgsErr
	}
	if m.orgs == nil {
		return []Organization{}, nil
	}
	return m.orgs, nil
}

func (m *handlerMockOrgStore) GetOrgByInstallationID(_ context.Context, _ int64) (*Organization, error) {
	return m.getByInstallation, m.getByInstErr
}

func (m *handlerMockOrgStore) SetInstallationID(_ context.Context, _ uuid.UUID, _ int64) error {
	return nil
}

func (m *handlerMockOrgStore) SetLastSyncedAt(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
