package vuln

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// OrgResolver resolves an org slug to its UUID. The vuln handler depends on this
// narrow interface rather than the full org store to stay decoupled.
type OrgResolver interface {
	GetOrgIDBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error)
}

// VulnReader is the read subset of VulnStore the handler needs. Store satisfies it.
type VulnReader interface {
	ListByOrg(ctx context.Context, orgID uuid.UUID, severity, packageName string, page, perPage int) ([]VulnWithCounts, int, error)
	GetDetail(ctx context.Context, orgID uuid.UUID, vulnID uuid.UUID) (*VulnDetail, error)
}

// Handler exposes HTTP endpoints for the vulnerability dashboard.
type Handler struct {
	store       VulnReader
	orgResolver OrgResolver
}

// NewHandler constructs a Handler.
func NewHandler(store VulnReader, orgResolver OrgResolver) *Handler {
	return &Handler{store: store, orgResolver: orgResolver}
}

// listResponse is the JSON envelope for the list endpoint.
type listResponse struct {
	Data    []VulnWithCounts `json:"data"`
	Total   int              `json:"total"`
	Page    int              `json:"page"`
	PerPage int              `json:"per_page"`
}

// validSeverities is the set of accepted severity filter values.
var validSeverities = map[string]bool{
	string(SeverityCritical): true,
	string(SeverityHigh):     true,
	string(SeverityMedium):   true,
	string(SeverityLow):      true,
	string(SeverityUnknown):  true,
}

// HandleListVulnerabilities handles GET /orgs/{slug}/vulnerabilities.
// Query params: page (default 1), perPage (default 20, max 100),
// severity (optional, one of critical|high|medium|low|unknown).
func (h *Handler) HandleListVulnerabilities(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	orgID, found, err := h.orgResolver.GetOrgIDBySlug(r.Context(), slug)
	if err != nil {
		slog.Error("vuln handler: failed to resolve org slug", "slug", slug, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		jsonError(w, "organization not found", http.StatusNotFound)
		return
	}

	page := 1
	perPage := 20

	if p := r.URL.Query().Get("page"); p != "" {
		v, err := strconv.Atoi(p)
		if err != nil || v < 1 {
			jsonError(w, "invalid page parameter", http.StatusBadRequest)
			return
		}
		page = v
	}

	if pp := r.URL.Query().Get("perPage"); pp != "" {
		v, err := strconv.Atoi(pp)
		if err != nil || v <= 0 {
			jsonError(w, "invalid perPage parameter: must be a positive integer", http.StatusBadRequest)
			return
		}
		if v > 100 {
			jsonError(w, "perPage must not exceed 100", http.StatusBadRequest)
			return
		}
		perPage = v
	}

	severity := r.URL.Query().Get("severity")
	if severity != "" && !validSeverities[severity] {
		jsonError(w, "invalid severity value", http.StatusBadRequest)
		return
	}

	// Optional package filter — empty string means no filter.
	packageName := r.URL.Query().Get("package")

	vulns, total, err := h.store.ListByOrg(r.Context(), orgID, severity, packageName, page, perPage)
	if err != nil {
		slog.Error("vuln handler: failed to list vulnerabilities", "org_id", orgID, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	if vulns == nil {
		vulns = []VulnWithCounts{}
	}

	jsonOK(w, listResponse{
		Data:    vulns,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	})
}

// HandleGetVulnerability handles GET /orgs/{slug}/vulnerabilities/{id}.
func (h *Handler) HandleGetVulnerability(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	orgID, found, err := h.orgResolver.GetOrgIDBySlug(r.Context(), slug)
	if err != nil {
		slog.Error("vuln handler: failed to resolve org slug", "slug", slug, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		jsonError(w, "organization not found", http.StatusNotFound)
		return
	}

	vulnID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		// A malformed id cannot match any record — treat as not found.
		jsonError(w, "vulnerability not found", http.StatusNotFound)
		return
	}

	detail, err := h.store.GetDetail(r.Context(), orgID, vulnID)
	if err != nil {
		slog.Error("vuln handler: failed to get vulnerability detail", "org_id", orgID, "vuln_id", vulnID, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if detail == nil {
		jsonError(w, "vulnerability not found", http.StatusNotFound)
		return
	}

	jsonOK(w, detail)
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
