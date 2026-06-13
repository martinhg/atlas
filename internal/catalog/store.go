package catalog

import (
	"context"

	"github.com/google/uuid"
)

type RepoStore interface {
	UpsertRepository(ctx context.Context, repo *Repository) (*Repository, error)
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]Repository, error)
}
