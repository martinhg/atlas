package dependency

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// OrgResolver resolves an org slug to its UUID. The dependency handler depends
// on this interface rather than the full org.OrgStore to stay decoupled.
type OrgResolver interface {
	GetOrgIDBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error)
}

// Handler exposes HTTP endpoints for the dependency domain.
type Handler struct {
	store       DepStore
	orgResolver OrgResolver
}

// NewHandler constructs a Handler.
func NewHandler(store DepStore, orgResolver OrgResolver) *Handler {
	return &Handler{store: store, orgResolver: orgResolver}
}

// listResponse is the JSON envelope for the list endpoint.
type listResponse struct {
	Data    []DependencyWithCount `json:"data"`
	Total   int                   `json:"total"`
	Page    int                   `json:"page"`
	PerPage int                   `json:"per_page"`
}

// detailResponse is the JSON envelope for the detail endpoint.
type detailResponse struct {
	Ecosystem string      `json:"ecosystem"`
	Name      string      `json:"name"`
	Repos     []DepDetail `json:"repos"`
}

// HandleListDependencies handles GET /orgs/{slug}/dependencies
// Query params: page (default 1), per_page (default 50). per_page <= 0 → 400.
func (h *Handler) HandleListDependencies(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	orgID, found, err := h.orgResolver.GetOrgIDBySlug(r.Context(), slug)
	if err != nil {
		slog.Error("dependency handler: failed to resolve org slug", "slug", slug, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		jsonError(w, "organization not found", http.StatusNotFound)
		return
	}

	// Parse and validate pagination params.
	page := 1
	perPage := 50

	if p := r.URL.Query().Get("page"); p != "" {
		v, err := strconv.Atoi(p)
		if err != nil || v < 1 {
			jsonError(w, "invalid page parameter", http.StatusBadRequest)
			return
		}
		page = v
	}

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

	deps, total, err := h.store.ListByOrg(r.Context(), orgID, page, perPage)
	if err != nil {
		slog.Error("dependency handler: failed to list dependencies", "org_id", orgID, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Ensure data is never nil in the JSON response.
	if deps == nil {
		deps = []DependencyWithCount{}
	}

	jsonOK(w, listResponse{
		Data:    deps,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	})
}

// HandleGetDependency handles GET /orgs/{slug}/dependencies/{ecosystem}/{name}
func (h *Handler) HandleGetDependency(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	ecosystem := chi.URLParam(r, "ecosystem")
	name := chi.URLParam(r, "*")

	orgID, found, err := h.orgResolver.GetOrgIDBySlug(r.Context(), slug)
	if err != nil {
		slog.Error("dependency handler: failed to resolve org slug", "slug", slug, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		jsonError(w, "organization not found", http.StatusNotFound)
		return
	}

	repos, err := h.store.GetDetail(r.Context(), orgID, ecosystem, name)
	if err != nil {
		slog.Error("dependency handler: failed to get dependency detail",
			"org_id", orgID, "ecosystem", ecosystem, "name", name, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	if len(repos) == 0 {
		jsonError(w, "dependency not found", http.StatusNotFound)
		return
	}

	jsonOK(w, detailResponse{
		Ecosystem: ecosystem,
		Name:      name,
		Repos:     repos,
	})
}

// jsonOK writes a 200 JSON response.
func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// jsonError writes an error JSON response with the given status code.
func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
