package ownership

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ownershipStore is the local interface for the handler's store dependency.
// It exposes only the methods the handler needs, keeping it testable with mocks.
type ownershipStore interface {
	ListByOrg(ctx context.Context, orgID uuid.UUID, page, perPage int) ([]RepoOwnerSummary, int, error)
	ListByRepo(ctx context.Context, orgID uuid.UUID, repoName string) ([]OwnerRule, error)
}

// orgResolver is the local interface for resolving an org slug to its UUID.
// dependency.OrgResolver (and orgStoreResolver in main.go) satisfy this interface.
type orgResolver interface {
	GetOrgIDBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error)
}

// Handler exposes HTTP endpoints for the ownership domain.
type Handler struct {
	store       ownershipStore
	orgResolver orgResolver
}

// NewHandler constructs an ownership Handler.
func NewHandler(store ownershipStore, resolver orgResolver) *Handler {
	return &Handler{store: store, orgResolver: resolver}
}

// ownershipListResponse is the JSON envelope for the list endpoint.
type ownershipListResponse struct {
	Data    []RepoOwnerSummary `json:"data"`
	Total   int                `json:"total"`
	Page    int                `json:"page"`
	PerPage int                `json:"per_page"`
}

// ownershipDetailResponse is the JSON envelope for the detail endpoint.
type ownershipDetailResponse struct {
	Repo  string      `json:"repo"`
	Rules []OwnerRule `json:"rules"`
}

// HandleListOwnership handles GET /orgs/{slug}/ownership
// Query params: page (default 1, min 1), per_page (default 50, min 1, max 100).
func (h *Handler) HandleListOwnership(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	orgID, found, err := h.orgResolver.GetOrgIDBySlug(r.Context(), slug)
	if err != nil {
		slog.Error("ownership handler: failed to resolve org slug", "slug", slug, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		jsonError(w, "org not found", http.StatusNotFound)
		return
	}

	// Parse and validate pagination params.
	page := 1
	perPage := 50

	if p := r.URL.Query().Get("page"); p != "" {
		v, err := strconv.Atoi(p)
		if err != nil || v < 1 {
			jsonError(w, "page must be >= 1", http.StatusBadRequest)
			return
		}
		page = v
	}

	if pp := r.URL.Query().Get("per_page"); pp != "" {
		v, err := strconv.Atoi(pp)
		if err != nil || v < 1 {
			jsonError(w, "per_page must be between 1 and 100", http.StatusBadRequest)
			return
		}
		if v > 100 {
			jsonError(w, "per_page must be between 1 and 100", http.StatusBadRequest)
			return
		}
		perPage = v
	}

	summaries, total, err := h.store.ListByOrg(r.Context(), orgID, page, perPage)
	if err != nil {
		slog.Error("ownership handler: failed to list ownership", "org_id", orgID, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Ensure data is never nil in the JSON response.
	if summaries == nil {
		summaries = make([]RepoOwnerSummary, 0)
	}

	jsonOK(w, ownershipListResponse{
		Data:    summaries,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	})
}

// HandleGetRepoOwnership handles GET /orgs/{slug}/ownership/{repo}
func (h *Handler) HandleGetRepoOwnership(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	repo := chi.URLParam(r, "repo")

	orgID, found, err := h.orgResolver.GetOrgIDBySlug(r.Context(), slug)
	if err != nil {
		slog.Error("ownership handler: failed to resolve org slug", "slug", slug, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		jsonError(w, "org not found", http.StatusNotFound)
		return
	}

	rules, err := h.store.ListByRepo(r.Context(), orgID, repo)
	if err != nil {
		slog.Error("ownership handler: failed to get repo ownership", "org_id", orgID, "repo", repo, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Ensure rules is never nil in the JSON response.
	if rules == nil {
		rules = make([]OwnerRule, 0)
	}

	jsonOK(w, ownershipDetailResponse{
		Repo:  repo,
		Rules: rules,
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
