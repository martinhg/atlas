package catalog

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	repoStore RepoStore
}

func NewHandler(repoStore RepoStore) *Handler {
	return &Handler{repoStore: repoStore}
}

func (h *Handler) HandleListRepos(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "orgID")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid org_id"}`, http.StatusBadRequest)
		return
	}

	repos, err := h.repoStore.GetRepositoriesByOrgID(r.Context(), orgID)
	if err != nil {
		slog.Error("failed to list repos", "org_id", orgID, "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repos)
}
