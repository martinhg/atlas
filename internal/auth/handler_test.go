package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
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
	needle := param + "="
	for i := 0; i < len(rawURL)-len(needle); i++ {
		if rawURL[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// --- HandleCallback ---

func newCallbackHandler(store UserStore, tokenURL, ghAPIURL string) *Handler {
	h := NewHandler("client-id", "client-secret", testSecret, "http://localhost:5173", store)
	h.oauthConfig.Endpoint = oauth2.Endpoint{TokenURL: tokenURL}
	h.githubBaseURL = ghAPIURL
	return h
}

func TestHandleCallback_missingState(t *testing.T) {
	h := newTestHandler(&mockStore{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback", nil)
	w := httptest.NewRecorder()

	h.HandleCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCallback_invalidState(t *testing.T) {
	h := newTestHandler(&mockStore{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state=unknown", nil)
	w := httptest.NewRecorder()

	h.HandleCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCallback_expiredState(t *testing.T) {
	h := newTestHandler(&mockStore{})
	h.states.Store("expired-state", time.Now().Add(-11*time.Minute))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state=expired-state", nil)
	w := httptest.NewRecorder()

	h.HandleCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCallback_missingCode(t *testing.T) {
	h := newTestHandler(&mockStore{})
	h.states.Store("valid-state", time.Now())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state=valid-state", nil)
	w := httptest.NewRecorder()

	h.HandleCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCallback_exchangeError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "token error", http.StatusBadRequest)
	}))
	defer tokenServer.Close()

	h := newCallbackHandler(&mockStore{}, tokenServer.URL, "")
	h.states.Store("state1", time.Now())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state=state1&code=badcode", nil)
	w := httptest.NewRecorder()

	h.HandleCallback(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleCallback_fetchUserError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token": "test-token",
			"token_type":   "bearer",
		})
	}))
	defer tokenServer.Close()

	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ghServer.Close()

	h := newCallbackHandler(&mockStore{}, tokenServer.URL, ghServer.URL)
	h.states.Store("state2", time.Now())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state=state2&code=validcode", nil)
	w := httptest.NewRecorder()

	h.HandleCallback(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleCallback_upsertError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token": "test-token",
			"token_type":   "bearer",
		})
	}))
	defer tokenServer.Close()

	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    12345,
			"login": "octocat",
		})
	}))
	defer ghServer.Close()

	store := &mockStore{err: errors.New("db error")}
	h := newCallbackHandler(store, tokenServer.URL, ghServer.URL)
	h.states.Store("state3", time.Now())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state=state3&code=validcode", nil)
	w := httptest.NewRecorder()

	h.HandleCallback(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleCallback_success(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token": "test-token",
			"token_type":   "bearer",
		})
	}))
	defer tokenServer.Close()

	name := "Mona"
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    12345,
			"login": "octocat",
			"name":  name,
		})
	}))
	defer ghServer.Close()

	userID := uuid.New()
	store := &mockStore{user: &User{ID: userID, Login: "octocat", GitHubID: 12345}}
	h := newCallbackHandler(store, tokenServer.URL, ghServer.URL)
	h.states.Store("state4", time.Now())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state=state4&code=validcode", nil)
	w := httptest.NewRecorder()

	h.HandleCallback(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusTemporaryRedirect)
	}

	location := w.Header().Get("Location")
	if !strings.HasPrefix(location, "http://localhost:5173/#access_token=") {
		t.Errorf("unexpected redirect: %q", location)
	}
	if !strings.Contains(location, "refresh_token=") {
		t.Errorf("redirect missing refresh_token: %q", location)
	}
}

// --- fetchGitHubUser ---

func TestFetchGitHubUser_success(t *testing.T) {
	name := "Mona Lisa"
	email := "mona@github.com"
	avatar := "https://example.com/avatar.png"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("path = %q, want /user", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer my-token" {
			t.Errorf("Authorization = %q, want Bearer my-token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         42,
			"login":      "mona",
			"name":       name,
			"email":      email,
			"avatar_url": avatar,
		})
	}))
	defer srv.Close()

	h := newTestHandler(&mockStore{})
	h.githubBaseURL = srv.URL
	h.httpClient = srv.Client()

	user, err := h.fetchGitHubUser(context.Background(), "my-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.GitHubID != 42 {
		t.Errorf("GitHubID = %d, want 42", user.GitHubID)
	}
	if user.Login != "mona" {
		t.Errorf("Login = %q, want mona", user.Login)
	}
	if user.Name == nil || *user.Name != name {
		t.Errorf("Name = %v, want %q", user.Name, name)
	}
	if user.Email == nil || *user.Email != email {
		t.Errorf("Email = %v, want %q", user.Email, email)
	}
	if user.AvatarURL == nil || *user.AvatarURL != avatar {
		t.Errorf("AvatarURL = %v, want %q", user.AvatarURL, avatar)
	}
}

func TestFetchGitHubUser_httpError(t *testing.T) {
	h := newTestHandler(&mockStore{})
	h.githubBaseURL = "http://127.0.0.1:1"
	h.httpClient = &http.Client{Timeout: 50 * time.Millisecond}

	_, err := h.fetchGitHubUser(context.Background(), "token")
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
}
