package org

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nesbite/atlas/internal/auth"
	"github.com/nesbite/atlas/internal/catalog"
	ghplatform "github.com/nesbite/atlas/internal/platform/github"
)

type Handler struct {
	orgStore      OrgStore
	catalogStore  catalog.RepoStore
	depSyncer     DepSyncer
	ghAppID       int64
	ghPrivateKey  []byte
	webhookSecret string
}

func NewHandler(orgStore OrgStore, catalogStore catalog.RepoStore, depSyncer DepSyncer, ghAppID int64, ghPrivateKey []byte, webhookSecret string) *Handler {
	return &Handler{
		orgStore:      orgStore,
		catalogStore:  catalogStore,
		depSyncer:     depSyncer,
		ghAppID:       ghAppID,
		ghPrivateKey:  ghPrivateKey,
		webhookSecret: webhookSecret,
	}
}

func (h *Handler) HandleListOrgs(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	orgs, err := h.orgStore.GetOrgsByOwnerID(r.Context(), userID)
	if err != nil {
		slog.Error("failed to list orgs", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orgs)
}

func (h *Handler) HandleGetOrg(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		http.Error(w, `{"error":"slug is required"}`, http.StatusBadRequest)
		return
	}

	org, err := h.orgStore.GetOrgBySlug(r.Context(), slug)
	if err != nil {
		slog.Error("failed to get org", "slug", slug, "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if org == nil {
		http.Error(w, `{"error":"organization not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(org)
}

type connectRequest struct {
	InstallationID int64 `json:"installation_id"`
}

func (h *Handler) HandleConnect(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var body connectRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if body.InstallationID == 0 {
		http.Error(w, `{"error":"installation_id is required"}`, http.StatusBadRequest)
		return
	}

	existing, err := h.orgStore.GetOrgByInstallationID(r.Context(), body.InstallationID)
	if err != nil {
		slog.Error("failed to check existing installation", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if existing != nil && existing.OwnerID != userID {
		http.Error(w, `{"error":"installation already linked to another user"}`, http.StatusConflict)
		return
	}

	appClient, err := ghplatform.NewAppClient(h.ghAppID, h.ghPrivateKey)
	if err != nil {
		slog.Error("failed to create GitHub app client", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	installation, _, err := appClient.Apps.GetInstallation(r.Context(), body.InstallationID)
	if err != nil {
		slog.Error("failed to get installation from GitHub", "installation_id", body.InstallationID, "error", err)
		http.Error(w, `{"error":"failed to verify installation"}`, http.StatusBadRequest)
		return
	}

	org, err := h.orgStore.UpsertOrg(r.Context(), &Organization{
		GitHubID:             installation.GetAccount().GetID(),
		Name:                 installation.GetAccount().GetLogin(),
		Slug:                 installation.GetAccount().GetLogin(),
		GitHubInstallationID: &body.InstallationID,
		OwnerID:              userID,
	})
	if err != nil {
		slog.Error("failed to upsert org", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	if h.catalogStore != nil {
		installClient, err := ghplatform.NewInstallationClient(h.ghAppID, body.InstallationID, h.ghPrivateKey)
		if err != nil {
			slog.Error("failed to create installation client for sync", "error", err)
		} else {
			go syncRepos(installClient, h.orgStore, h.catalogStore, h.depSyncer, org.ID, org.Slug)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(org)
}

func (h *Handler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"failed to read body"}`, http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("X-Hub-Signature-256")
	if sig == "" {
		http.Error(w, `{"error":"missing signature"}`, http.StatusForbidden)
		return
	}

	if !ghplatform.VerifyWebhookSignature(h.webhookSecret, body, sig) {
		http.Error(w, `{"error":"invalid signature"}`, http.StatusForbidden)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	if eventType != "installation" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ignored"}`))
		return
	}

	var event struct {
		Action       string `json:"action"`
		Installation struct {
			ID      int64 `json:"id"`
			Account struct {
				ID    int64  `json:"id"`
				Login string `json:"login"`
			} `json:"account"`
		} `json:"installation"`
		Sender struct {
			ID int64 `json:"id"`
		} `json:"sender"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		slog.Error("webhook: failed to parse event", "error", err)
		http.Error(w, `{"error":"invalid event payload"}`, http.StatusBadRequest)
		return
	}

	if event.Action != "created" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ignored"}`))
		return
	}

	existing, err := h.orgStore.GetOrgByInstallationID(r.Context(), event.Installation.ID)
	if err != nil {
		slog.Error("webhook: failed to check installation", "error", err)
	}

	if existing != nil {
		if h.catalogStore != nil {
			installClient, err := ghplatform.NewInstallationClient(h.ghAppID, event.Installation.ID, h.ghPrivateKey)
			if err == nil {
				go syncRepos(installClient, h.orgStore, h.catalogStore, h.depSyncer, existing.ID, existing.Slug)
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"syncing"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"processed"}`))
}

func (h *Handler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.HandleListOrgs)
		r.Get("/{slug}", h.HandleGetOrg)
		r.Post("/connect", h.HandleConnect)
	}
}
