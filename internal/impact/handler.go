package impact

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// OrgResolver resolves an org slug to its UUID. The impact handler depends on
// this interface rather than the full org.OrgStore to stay decoupled.
type OrgResolver interface {
	GetOrgIDBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error)
}

// Handler exposes HTTP endpoints for the impact analysis domain.
type Handler struct {
	store       ImpactStore
	orgResolver OrgResolver
}

// NewHandler constructs an impact Handler.
func NewHandler(store ImpactStore, orgResolver OrgResolver) *Handler {
	return &Handler{store: store, orgResolver: orgResolver}
}

// analyzeImpactRequest is the JSON request body for HandleAnalyzeImpact.
type analyzeImpactRequest struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
}

// HandleAnalyzeImpact handles POST /orgs/{slug}/impact. It computes the blast
// radius of the dependency identified by the request body's ecosystem and
// name, scoped to the org resolved from the slug path parameter.
func (h *Handler) HandleAnalyzeImpact(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var body analyzeImpactRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Ecosystem == "" || body.Name == "" {
		jsonError(w, "ecosystem and name are required", http.StatusBadRequest)
		return
	}

	orgID, found, err := h.orgResolver.GetOrgIDBySlug(r.Context(), slug)
	if err != nil {
		slog.Error("impact handler: failed to resolve org slug", "slug", slug, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		jsonError(w, "organization not found", http.StatusNotFound)
		return
	}

	repos, err := h.store.GetBlastRadius(r.Context(), orgID, body.Ecosystem, body.Name)
	if err != nil {
		slog.Error("impact handler: failed to compute blast radius",
			"org_id", orgID, "ecosystem", body.Ecosystem, "name", body.Name, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if repos == nil {
		repos = []AffectedRepo{}
	}

	score, level := computeRiskScore(repos)

	jsonOK(w, BlastRadius{
		Dependency:          DependencyRef(body),
		TotalRepos:          len(repos),
		RiskLevel:           level,
		RiskScore:           score,
		AffectedRepos:       repos,
		VersionDistribution: versionDistribution(repos),
	})
}

// versionDistribution groups affected repos by exact version string,
// returning one entry per distinct version with its repo count. The result
// is sorted by version for deterministic output and is never nil.
func versionDistribution(repos []AffectedRepo) []VersionDist {
	counts := make(map[string]int)
	for _, repo := range repos {
		counts[repo.Version]++
	}

	result := make([]VersionDist, 0, len(counts))
	for version, count := range counts {
		result = append(result, VersionDist{Version: version, RepoCount: count})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result
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
