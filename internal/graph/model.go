// Package graph provides the dependency graph visualization domain for Atlas.
// It exposes a single read-only endpoint that returns the repo→dep→team topology
// for an org as a flat {nodes, edges, truncated} payload suitable for Sigma.js.
package graph

import "github.com/google/uuid"

// NodeType classifies a graph node.
type NodeType string

const (
	NodeTypeRepo NodeType = "repo"
	NodeTypeDep  NodeType = "dep"
	NodeTypeTeam NodeType = "team"
)

// Node represents a single vertex in the dependency graph.
// The ID format encodes the type: "repo:{uuid}", "dep:{uuid}", "team:{owner}".
type Node struct {
	ID        string    `json:"id"`
	Type      NodeType  `json:"type"`
	Label     string    `json:"label"`
	RiskLevel string    `json:"risk_level,omitempty"`
	Ecosystem string    `json:"ecosystem,omitempty"`
	Language  *string   `json:"language,omitempty"`
}

// Edge represents a directed relationship between two nodes.
type Edge struct {
	ID      string `json:"id"`
	Source  string `json:"source"`
	Target  string `json:"target"`
	DepType string `json:"dep_type,omitempty"`
	Label   string `json:"label,omitempty"`
}

// GraphResponse is the top-level response envelope for GET /orgs/{slug}/graph.
type GraphResponse struct {
	Nodes     []Node `json:"nodes"`
	Edges     []Edge `json:"edges"`
	Truncated bool   `json:"truncated"`
}

// GraphFilters holds optional server-side filter params parsed from query string.
type GraphFilters struct {
	Ecosystem string
	Risk      string
	Team      string
}

// depAggregate is the internal projection returned by the single SQL pass.
// One row per unique dependency; AffectedRepos and Teams are aggregated by SQL.
type depAggregate struct {
	DepID         uuid.UUID
	Ecosystem     string
	Name          string
	AffectedRepos []repoRef
}

// repoRef carries per-repo data that is part of a depAggregate row.
type repoRef struct {
	RepoID   uuid.UUID
	RepoName string
	Language *string
	DepType  string
	Teams    []string
}
