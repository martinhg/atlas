package catalog

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type OrgResolver interface {
	GetOrgIDBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error)
}

type Handler struct {
	repoStore   RepoStore
	orgResolver OrgResolver
}

func NewHandler(repoStore RepoStore, orgResolver OrgResolver) *Handler {
	return &Handler{repoStore: repoStore, orgResolver: orgResolver}
}

func (h *Handler) HandleListRepos(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	orgID, found, err := h.orgResolver.GetOrgIDBySlug(r.Context(), slug)
	if err != nil {
		slog.Error("failed to resolve org slug", "slug", slug, "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, `{"error":"organization not found"}`, http.StatusNotFound)
		return
	}

	repos, err := h.repoStore.GetRepositoriesByOrgID(r.Context(), orgID)
	if err != nil {
		slog.Error("failed to list repos", "org_id", orgID, "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(repos); err != nil {
		slog.Error("failed to encode repos response", "error", err)
	}
}
