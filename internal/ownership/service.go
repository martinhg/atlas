package ownership

import (
	"context"
	"log/slog"

	gogithub "github.com/google/go-github/v69/github"
	"github.com/google/uuid"
	ownerparser "github.com/nesbite/atlas/internal/ownership/parser"
)

// Service orchestrates CODEOWNERS discovery, parsing, and storage for a single
// repository. It satisfies the OwnershipSyncer interface defined in org/sync.go.
type Service struct {
	store OwnershipStore
}

// NewService constructs a Service backed by the given OwnershipStore.
func NewService(store OwnershipStore) *Service {
	return &Service{store: store}
}

// candidatePaths are the three well-known locations where GitHub looks for a
// CODEOWNERS file. They are tried in order; the first successful response wins.
var candidatePaths = []string{"CODEOWNERS", ".github/CODEOWNERS", "docs/CODEOWNERS"}

// SyncRepoOwnership fetches the CODEOWNERS file for a repository (trying 3 candidate
// paths in order), parses it, and persists the result via the store.
//
// Error policy: this function NEVER returns a non-nil error. All errors are logged
// with slog.Error and the function returns nil so that sync of other repos continues
// uninterrupted.
func (s *Service) SyncRepoOwnership(
	ctx context.Context,
	ghClient *gogithub.Client,
	repoID uuid.UUID,
	owner, repo string,
) error {
	var content string
	found := false

	for _, path := range candidatePaths {
		fileContent, _, resp, err := ghClient.Repositories.GetContents(ctx, owner, repo, path, nil)
		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				// File not at this path — try next candidate.
				continue
			}
			// Non-404 error: log and return nil (error isolation).
			slog.Error("ownership sync: failed to fetch CODEOWNERS",
				"repo", owner+"/"+repo,
				"path", path,
				"error", err,
			)
			return nil
		}

		// Decode base64 content from the GitHub API response.
		decoded, err := fileContent.GetContent()
		if err != nil {
			slog.Error("ownership sync: failed to decode CODEOWNERS content",
				"repo", owner+"/"+repo,
				"path", path,
				"error", err,
			)
			return nil
		}

		content = decoded
		found = true
		break
	}

	// Parse the content (or use empty slice if no CODEOWNERS found).
	var owners []ownerparser.ParsedOwner
	if found {
		owners = ownerparser.ParseCODEOWNERS([]byte(content))
	} else {
		owners = []ownerparser.ParsedOwner{}
	}

	// Persist (DELETE + INSERT in transaction). Even an empty slice clears stale data.
	if err := s.store.SyncRepoOwners(ctx, repoID, owners); err != nil {
		slog.Error("ownership sync: failed to sync repo owners",
			"repo", owner+"/"+repo,
			"error", err,
		)
		return nil
	}

	return nil
}
