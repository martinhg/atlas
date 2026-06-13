package org

import (
	"time"

	"github.com/google/uuid"
)

// Organization represents a GitHub organization or user account linked to Atlas.
type Organization struct {
	ID                   uuid.UUID  `json:"id"`
	GitHubID             int64      `json:"github_id"`
	Name                 string     `json:"name"`
	Slug                 string     `json:"slug"`
	GitHubInstallationID *int64     `json:"github_installation_id,omitempty"`
	OwnerID              uuid.UUID  `json:"owner_id"`
	LastSyncedAt         *time.Time `json:"last_synced_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}
