package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// mockStore satisfies UserStore for unit tests.
type mockStore struct {
	user *User
	err  error
}

func (m *mockStore) UpsertUser(_ context.Context, _ *User, _ string) (*User, error) {
	return m.user, m.err
}

func (m *mockStore) GetUserByID(_ context.Context, _ uuid.UUID) (*User, error) {
	return m.user, m.err
}

// newTestHandler creates a Handler wired with a mockStore and fixed secrets.
func newTestHandler(store UserStore) *Handler {
	return NewHandler("client-id", "client-secret", testSecret, "http://localhost:5173", store)
}

// --- HandleMe ---

func TestHandleMe_authenticatedUser(t *testing.T) {
	userID := uuid.New()
	login := "octocat"
	store := &mockStore{user: &User{ID: userID, Login: login, GitHubID: 1}}

	h := newTestHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	ctx := context.WithValue(req.Context(), UserIDKey, userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.HandleMe(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
	}

	var got User
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.Login != login {
		t.Errorf("got login %q, want %q", got.Login, login)
	}
	if got.ID != userID {
		t.Errorf("got ID %v, want %v", got.ID, userID)
	}
}

func TestHandleMe_missingUserID(t *testing.T) {
	h := newTestHandler(&mockStore{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	// Context has no UserIDKey set.
	w := httptest.NewRecorder()
	h.HandleMe(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleMe_userNotFound(t *testing.T) {
	userID := uuid.New()
	store := &mockStore{err: errors.New("not found")}

	h := newTestHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	ctx := context.WithValue(req.Context(), UserIDKey, userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.HandleMe(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("got status %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- HandleRefresh ---

func TestHandleRefresh_validToken(t *testing.T) {
	userID := uuid.New()
	pair, err := IssueTokenPair(testSecret, userID)
	if err != nil {
		t.Fatalf("IssueTokenPair error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"refresh_token": pair.RefreshToken})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h := newTestHandler(&mockStore{})
	h.HandleRefresh(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
	}

	var got TokenPair
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode token pair: %v", err)
	}
	if got.AccessToken == "" {
		t.Error("returned AccessToken is empty")
	}
	if got.RefreshToken == "" {
		t.Error("returned RefreshToken is empty")
	}
}

func TestHandleRefresh_invalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString("not-json"))
	w := httptest.NewRecorder()
	h := newTestHandler(&mockStore{})
	h.HandleRefresh(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRefresh_invalidToken(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"refresh_token": "this.is.garbage"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h := newTestHandler(&mockStore{})
	h.HandleRefresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// --- HandleLogin ---

func TestHandleLogin_redirectsToGitHub(t *testing.T) {
	h := newTestHandler(&mockStore{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	w := httptest.NewRecorder()
	h.HandleLogin(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusTemporaryRedirect)
	}

	location := w.Header().Get("Location")
	if location == "" {
		t.Fatal("Location header is empty")
	}

	// The redirect must go to GitHub and carry a state param.
	if len(location) < 10 {
		t.Fatalf("Location header too short: %q", location)
	}

	// Verify the state query param is present (OAuth CSRF protection).
	// We don't hard-code the GitHub URL to avoid coupling to oauth2 internals.
	if !containsParam(location, "state") {
		t.Errorf("Location %q does not contain 'state' param", location)
	}
}

func containsParam(rawURL, param string) bool {
	// Simple check: the query string contains the param name followed by '='.
	needle := param + "="
	for i := 0; i < len(rawURL)-len(needle); i++ {
		if rawURL[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
