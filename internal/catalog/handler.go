package catalog

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

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

// repoListResponse is the JSON envelope returned by HandleListRepos.
type repoListResponse struct {
	Data    []Repository `json:"data"`
	Total   int          `json:"total"`
	Page    int          `json:"page"`
	PerPage int          `json:"per_page"`
}

// HandleListRepos handles GET /orgs/{slug}/repos
// Query params: q (default ""), page (default 1, >= 1 required),
// per_page (default 25, 1–100 inclusive).
func (h *Handler) HandleListRepos(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	orgID, found, err := h.orgResolver.GetOrgIDBySlug(r.Context(), slug)
	if err != nil {
		slog.Error("failed to resolve org slug", "slug", slug, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		jsonError(w, "organization not found", http.StatusNotFound)
		return
	}

	// Parse search query — empty string means no filter.
	q := r.URL.Query().Get("q")

	// Parse and validate page param.
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		v, err := strconv.Atoi(p)
		if err != nil || v < 1 {
			jsonError(w, "invalid page parameter", http.StatusBadRequest)
			return
		}
		page = v
	}

	// Parse and validate per_page param.
	perPage := 25
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		v, err := strconv.Atoi(pp)
		if err != nil || v <= 0 {
			jsonError(w, "invalid per_page parameter: must be a positive integer", http.StatusBadRequest)
			return
		}
		if v > 100 {
			jsonError(w, "per_page must not exceed 100", http.StatusBadRequest)
			return
		}
		perPage = v
	}

	repos, total, err := h.repoStore.GetRepositoriesByOrgID(r.Context(), orgID, q, page, perPage)
	if err != nil {
		slog.Error("failed to list repos", "org_id", orgID, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Ensure data is never nil in the JSON response.
	if repos == nil {
		repos = []Repository{}
	}

	jsonOK(w, repoListResponse{
		Data:    repos,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	})
}

// jsonOK writes a 200 JSON response.
func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// jsonError writes an error JSON response with the given status code.
func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
