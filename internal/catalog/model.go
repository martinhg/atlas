package catalog

import (
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	ID            uuid.UUID  `json:"id"`
	OrgID         uuid.UUID  `json:"org_id"`
	GitHubID      int64      `json:"github_id"`
	Name          string     `json:"name"`
	FullName      string     `json:"full_name"`
	Description   *string    `json:"description,omitempty"`
	DefaultBranch string     `json:"default_branch"`
	Language      *string    `json:"language,omitempty"`
	Private       bool       `json:"private"`
	Fork          bool       `json:"fork"`
	Stars         int        `json:"stars"`
	LastSyncedAt  *time.Time `json:"last_synced_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
