package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/nesbite/atlas/internal/risk"
)

// edgeLimit is the maximum number of edges returned in a single graph response.
// When exceeded, the response sets truncated=true. It is a package-level var
// (not a const) so tests can inject a small limit to exercise truncation.
var edgeLimit = 5000

// depTypePrecedence ranks dependency types so that when the same (repo, dep)
// pair appears under multiple dep_types (different source files), we pick a
// single deterministic representative: the highest-precedence type wins.
// Order mirrors the risk weighting: direct/dep > peer > optional > dev.
var depTypePrecedence = map[string]int{
	"direct":   5,
	"dep":      5,
	"peer":     4,
	"optional": 3,
	"dev":      2,
	"devDep":   2,
}

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
		node      Node
		maxRisk   risk.RiskLevel
		depEdges  []Edge
		depSeen   map[uuid.UUID]int // dep_id → index into depEdges (dedup per (repo,dep))
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
						ID:        repoNodeID,
						Type:      NodeTypeRepo,
						Label:     repo.RepoName,
						Language:  repo.Language,
						RiskLevel: string(risk.RiskLow),
					},
					maxRisk:   risk.RiskLow,
					depSeen:   make(map[uuid.UUID]int),
					teamsSeen: make(map[string]struct{}),
				}
				repoStates[repo.RepoID] = rs
			}

			// Update repo risk = max(dep risks).
			rs.maxRisk = maxRiskLevel(rs.maxRisk, depRiskLevel)
			rs.node.RiskLevel = string(rs.maxRisk)

			// repo→dep edge. A repo can declare the same dep across multiple
			// source files / dep_types (UNIQUE is on repo_id, dep_id,
			// source_file), so emit exactly ONE edge per (repo, dep). When the
			// pair already exists, keep the higher-precedence dep_type as the
			// deterministic representative.
			if idx, seen := rs.depSeen[agg.DepID]; seen {
				if depTypePrecedence[repo.DepType] > depTypePrecedence[rs.depEdges[idx].DepType] {
					rs.depEdges[idx].DepType = repo.DepType
				}
			} else {
				rs.depSeen[agg.DepID] = len(rs.depEdges)
				rs.depEdges = append(rs.depEdges, Edge{
					ID:      fmt.Sprintf("e:repo:%s:dep:%s", repo.RepoID, agg.DepID),
					Source:  repoNodeID,
					Target:  depNodeID,
					DepType: repo.DepType,
				})
			}

			// repo→team edges (deduplicated per repo)
			for _, team := range repo.Teams {
				if _, seen := rs.teamsSeen[team]; seen {
					continue
				}
				rs.teamsSeen[team] = struct{}{}

				tnID := teamNodeID(team)
				teamNodes[team] = Node{
					ID:    tnID,
					Type:  NodeTypeTeam,
					Label: team,
				}
				teamEdge := Edge{
					ID:     fmt.Sprintf("e:repo:%s:team:%s", repo.RepoID, encodeOwner(team)),
					Source: repoNodeID,
					Target: tnID,
					Label:  "owns",
				}
				teamEdges = append(teamEdges, teamEdge)
			}
		}
	}

	// Assemble the full edge slice first so truncation can run on the final
	// post-filter set. repo→dep edges come from each repoState; repo→team edges
	// are appended after.
	edges := make([]Edge, 0)
	for _, rs := range repoStates {
		edges = append(edges, rs.depEdges...)
	}
	edges = append(edges, teamEdges...)

	// Deterministic truncation: sort by a stable key (source, target, id) so
	// the surviving subset is reproducible across requests, then apply the
	// limit. Sorting BEFORE the limit guarantees the same edges survive.
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Source != edges[j].Source {
			return edges[i].Source < edges[j].Source
		}
		if edges[i].Target != edges[j].Target {
			return edges[i].Target < edges[j].Target
		}
		return edges[i].ID < edges[j].ID
	})

	truncated := false
	if len(edges) > edgeLimit {
		edges = edges[:edgeLimit]
		truncated = true
	}

	// Prune orphan nodes: keep only nodes referenced by a surviving edge. This
	// removes dangling repo/dep/team nodes left behind by truncation or by the
	// risk filter dropping a dep but keeping a repo with no other edges.
	referenced := make(map[string]struct{}, len(edges)*2)
	for _, e := range edges {
		referenced[e.Source] = struct{}{}
		referenced[e.Target] = struct{}{}
	}

	nodes := make([]Node, 0)
	for _, rs := range repoStates {
		if _, ok := referenced[rs.node.ID]; ok {
			nodes = append(nodes, rs.node)
		}
	}
	for _, n := range depNodes {
		if _, ok := referenced[n.ID]; ok {
			nodes = append(nodes, n)
		}
	}
	for _, n := range teamNodes {
		if _, ok := referenced[n.ID]; ok {
			nodes = append(nodes, n)
		}
	}

	return GraphResponse{
		Nodes:     nodes,
		Edges:     edges,
		Truncated: truncated,
	}
}

// teamNodeID builds the stable node ID for a team owner. The owner is
// percent-encoded so that characters used as ID separators (':') or whitespace
// cannot introduce a second separator that would collide with the
// repo:/dep:/team: ID scheme. url.PathEscape leaves ':' untouched, so it is
// escaped explicitly. The raw owner is preserved in the node label.
func teamNodeID(owner string) string {
	return "team:" + encodeOwner(owner)
}

// encodeOwner percent-encodes an owner string, additionally escaping ':' which
// url.PathEscape leaves intact, so the encoded segment never contains a raw
// separator that could collide with the node ID scheme.
func encodeOwner(owner string) string {
	return strings.ReplaceAll(url.PathEscape(owner), ":", "%3A")
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
