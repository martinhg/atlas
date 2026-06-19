package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/nesbite/atlas/internal/risk"
)

// edgeLimit is the maximum number of edges returned in a single graph response.
// When exceeded, the response sets truncated=true.
const edgeLimit = 5000

// validRiskLevels holds the accepted ?risk= values for validation.
var validRiskLevels = map[string]bool{
	string(risk.RiskLow):    true,
	string(risk.RiskMedium): true,
	string(risk.RiskHigh):   true,
}

// OrgResolver resolves an org slug to its UUID. The graph handler depends on
// this narrow interface rather than the full org.OrgStore to stay decoupled.
// The same orgStoreResolver adapter used by other domains in main.go satisfies
// this interface.
type OrgResolver interface {
	GetOrgIDBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error)
}

// Handler exposes HTTP endpoints for the graph domain.
type Handler struct {
	store       GraphStore
	orgResolver OrgResolver
}

// NewHandler constructs a graph Handler.
func NewHandler(store GraphStore, orgResolver OrgResolver) *Handler {
	return &Handler{store: store, orgResolver: orgResolver}
}

// Routes registers the graph endpoints on the provided chi router.
func (h *Handler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/orgs/{slug}/graph", h.HandleGetGraph)
	}
}

// HandleGetGraph handles GET /orgs/{slug}/graph.
// Query params: ecosystem (string), risk (low|medium|high), team (string).
func (h *Handler) HandleGetGraph(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	// Parse and validate query params.
	q := r.URL.Query()
	riskParam := q.Get("risk")
	if riskParam != "" && !validRiskLevels[riskParam] {
		jsonError(w, fmt.Sprintf("invalid risk value %q: must be low, medium, or high", riskParam), http.StatusBadRequest)
		return
	}

	filters := GraphFilters{
		Ecosystem: q.Get("ecosystem"),
		Risk:      riskParam,
		Team:      q.Get("team"),
	}

	// Resolve org slug → orgID.
	orgID, found, err := h.orgResolver.GetOrgIDBySlug(r.Context(), slug)
	if err != nil {
		slog.Error("graph handler: failed to resolve org slug", "slug", slug, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		jsonError(w, "organization not found", http.StatusNotFound)
		return
	}

	// Fetch raw aggregates from the store (one SQL pass).
	aggregates, err := h.store.GetGraph(r.Context(), orgID, filters)
	if err != nil {
		slog.Error("graph handler: failed to fetch graph", "org_id", orgID, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Build the response from aggregates.
	resp := buildGraphResponse(aggregates, filters)
	jsonOK(w, resp)
}

// buildGraphResponse converts raw depAggregates into a GraphResponse,
// computing risk per dep, deriving repo risk (= max dep risk), applying
// server-side risk/team/ecosystem filtering, and enforcing the edge limit.
func buildGraphResponse(aggregates []depAggregate, filters GraphFilters) GraphResponse {
	type repoState struct {
		node     Node
		maxRisk  risk.RiskLevel
		depEdges []Edge
		teamsSeen map[string]struct{}
	}

	repoStates := make(map[uuid.UUID]*repoState)
	depNodes := make(map[uuid.UUID]Node)
	teamNodes := make(map[string]Node)
	var teamEdges []Edge

	for _, agg := range aggregates {
		// Compute dep risk from all affected repos.
		affected := make([]risk.Affected, len(agg.AffectedRepos))
		for i, r := range agg.AffectedRepos {
			affected[i] = risk.Affected{
				DepType: r.DepType,
				Teams:   r.Teams,
			}
		}
		_, depRiskLevel := risk.ComputeRiskScore(affected)

		// Apply risk filter (server-side, after computation).
		if filters.Risk != "" && string(depRiskLevel) != filters.Risk {
			continue
		}

		depNodeID := fmt.Sprintf("dep:%s", agg.DepID)
		depNodes[agg.DepID] = Node{
			ID:        depNodeID,
			Type:      NodeTypeDep,
			Label:     agg.Name,
			Ecosystem: agg.Ecosystem,
			RiskLevel: string(depRiskLevel),
		}

		for _, repo := range agg.AffectedRepos {
			repoNodeID := fmt.Sprintf("repo:%s", repo.RepoID)

			rs, exists := repoStates[repo.RepoID]
			if !exists {
				rs = &repoState{
					node: Node{
						ID:       repoNodeID,
						Type:     NodeTypeRepo,
						Label:    repo.RepoName,
						Language: repo.Language,
						RiskLevel: string(risk.RiskLow),
					},
					maxRisk:  risk.RiskLow,
					teamsSeen: make(map[string]struct{}),
				}
				repoStates[repo.RepoID] = rs
			}

			// Update repo risk = max(dep risks).
			rs.maxRisk = maxRiskLevel(rs.maxRisk, depRiskLevel)
			rs.node.RiskLevel = string(rs.maxRisk)

			// repo→dep edge
			edgeID := fmt.Sprintf("e:repo:%s:dep:%s", repo.RepoID, agg.DepID)
			rs.depEdges = append(rs.depEdges, Edge{
				ID:      edgeID,
				Source:  repoNodeID,
				Target:  depNodeID,
				DepType: repo.DepType,
			})

			// repo→team edges (deduplicated per repo)
			for _, team := range repo.Teams {
				if _, seen := rs.teamsSeen[team]; seen {
					continue
				}
				rs.teamsSeen[team] = struct{}{}

				teamNodeID := fmt.Sprintf("team:%s", team)
				teamNodes[team] = Node{
					ID:    teamNodeID,
					Type:  NodeTypeTeam,
					Label: team,
				}
				teamEdge := Edge{
					ID:     fmt.Sprintf("e:repo:%s:team:%s", repo.RepoID, team),
					Source: repoNodeID,
					Target: teamNodeID,
					Label:  "owns",
				}
				teamEdges = append(teamEdges, teamEdge)
			}
		}
	}

	// Assemble final node and edge slices.
	nodes := make([]Node, 0)
	edges := make([]Edge, 0)

	for _, rs := range repoStates {
		nodes = append(nodes, rs.node)
		edges = append(edges, rs.depEdges...)
	}
	for _, n := range depNodes {
		nodes = append(nodes, n)
	}
	for _, n := range teamNodes {
		nodes = append(nodes, n)
	}
	edges = append(edges, teamEdges...)

	// Enforce edge limit.
	truncated := false
	if len(edges) > edgeLimit {
		edges = edges[:edgeLimit]
		truncated = true
	}

	// Ensure slices are non-nil for clean JSON serialization.
	if nodes == nil {
		nodes = []Node{}
	}
	if edges == nil {
		edges = []Edge{}
	}

	return GraphResponse{
		Nodes:     nodes,
		Edges:     edges,
		Truncated: truncated,
	}
}

// maxRiskLevel returns the higher of two risk levels.
func maxRiskLevel(a, b risk.RiskLevel) risk.RiskLevel {
	order := map[risk.RiskLevel]int{
		risk.RiskLow:    0,
		risk.RiskMedium: 1,
		risk.RiskHigh:   2,
	}
	if order[b] > order[a] {
		return b
	}
	return a
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
