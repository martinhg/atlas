package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type Handler struct {
	oauthConfig *oauth2.Config
	store       *Store
	jwtSecret   string
	webURL      string
	states      sync.Map
}

func NewHandler(clientID, clientSecret, jwtSecret, webURL string, store *Store) *Handler {
	return &Handler{
		oauthConfig: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       []string{"read:user", "user:email"},
			Endpoint:     github.Endpoint,
		},
		store:     store,
		jwtSecret: jwtSecret,
		webURL:    webURL,
	}
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		slog.Error("failed to generate oauth state", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.states.Store(state, time.Now())
	url := h.oauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *Handler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "missing state", http.StatusBadRequest)
		return
	}

	stored, ok := h.states.LoadAndDelete(state)
	if !ok {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	if time.Since(stored.(time.Time)) > 10*time.Minute {
		http.Error(w, "state expired", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	token, err := h.oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		slog.Error("oauth exchange failed", "error", err)
		http.Error(w, "oauth exchange failed", http.StatusInternalServerError)
		return
	}

	ghUser, err := fetchGitHubUser(r.Context(), token.AccessToken)
	if err != nil {
		slog.Error("failed to fetch github user", "error", err)
		http.Error(w, "failed to fetch user info", http.StatusInternalServerError)
		return
	}

	user, err := h.store.UpsertUser(r.Context(), ghUser, token.AccessToken)
	if err != nil {
		slog.Error("failed to upsert user", "error", err)
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	tokens, err := IssueTokenPair(h.jwtSecret, user.ID)
	if err != nil {
		slog.Error("failed to issue tokens", "error", err)
		http.Error(w, "failed to issue tokens", http.StatusInternalServerError)
		return
	}

	redirectURL := h.webURL + "/#access_token=" + tokens.AccessToken + "&refresh_token=" + tokens.RefreshToken
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func (h *Handler) HandleMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *Handler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	claims, err := ValidateToken(h.jwtSecret, body.RefreshToken)
	if err != nil {
		http.Error(w, `{"error":"invalid refresh token"}`, http.StatusUnauthorized)
		return
	}

	tokens, err := IssueTokenPair(h.jwtSecret, claims.UserID)
	if err != nil {
		http.Error(w, `{"error":"failed to issue tokens"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokens)
}

type githubAPIUser struct {
	ID        int64   `json:"id"`
	Login     string  `json:"login"`
	Name      *string `json:"name"`
	Email     *string `json:"email"`
	AvatarURL *string `json:"avatar_url"`
}

func fetchGitHubUser(ctx context.Context, accessToken string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ghUser githubAPIUser
	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		return nil, err
	}

	return &User{
		GitHubID:  ghUser.ID,
		Login:     ghUser.Login,
		Name:      ghUser.Name,
		Email:     ghUser.Email,
		AvatarURL: ghUser.AvatarURL,
	}, nil
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
