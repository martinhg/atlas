package dependency

import (
	"time"

	"github.com/google/uuid"
)

// Dependency represents a unique (ecosystem, name) pair across the platform.
type Dependency struct {
	ID        uuid.UUID `json:"id"`
	Ecosystem string    `json:"ecosystem"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// RepoDependency records the association between a repository and a dependency,
// including the version, type, and source file where it was found.
type RepoDependency struct {
	ID         uuid.UUID `json:"id"`
	RepoID     uuid.UUID `json:"repo_id"`
	DepID      uuid.UUID `json:"dep_id"`
	Version    string    `json:"version"`
	DepType    string    `json:"dep_type"`
	SourceFile string    `json:"source_file"`
	CreatedAt  time.Time `json:"created_at"`
}

// DependencyWithCount is returned by the list endpoint and includes
// the number of repos in the org that use this dependency.
type DependencyWithCount struct {
	ID        uuid.UUID `json:"id"`
	Ecosystem string    `json:"ecosystem"`
	Name      string    `json:"name"`
	RepoCount int       `json:"repo_count"`
}

// DepDetail is returned by the detail endpoint and describes one repo's
// usage of a specific dependency.
type DepDetail struct {
	RepoName   string `json:"repo_name"`
	Version    string `json:"version"`
	DepType    string `json:"dep_type"`
	SourceFile string `json:"source_file"`
}
