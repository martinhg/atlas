package ownership

import (
	"time"

	"github.com/google/uuid"
)

// RepoOwner represents a single ownership rule parsed from a CODEOWNERS file.
type RepoOwner struct {
	ID         uuid.UUID `json:"id"`
	RepoID     uuid.UUID `json:"repo_id"`
	Pattern    string    `json:"pattern"`
	Owner      string    `json:"owner"`
	OwnerType  string    `json:"owner_type"`
	Source     string    `json:"source"`
	LineNumber *int      `json:"line_number,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// RepoOwnerSummary is returned by the list endpoint. It summarizes ownership
// for a single repository within an org.
type RepoOwnerSummary struct {
	RepoName   string   `json:"repo_name"`
	OwnerCount int      `json:"owner_count"`
	TeamCount  int      `json:"team_count"`
	Teams      []string `json:"teams"`
}

// OwnerRule is the detail-endpoint representation of a single CODEOWNERS line.
type OwnerRule struct {
	Pattern    string `json:"pattern"`
	Owner      string `json:"owner"`
	OwnerType  string `json:"owner_type"`
	LineNumber *int   `json:"line_number,omitempty"`
}
