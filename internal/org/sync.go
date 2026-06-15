package org

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	gogithub "github.com/google/go-github/v69/github"
	"github.com/nesbite/atlas/internal/catalog"
)

// DepSyncer is a local interface that the org package uses to trigger
// dependency sync for a repository. dependency.Service satisfies it.
// Using a local interface keeps the org package decoupled from the
// dependency package (Go structural typing).
type DepSyncer interface {
	SyncRepoDeps(ctx context.Context, ghClient *gogithub.Client, repoID uuid.UUID, owner, repo, branch string) error
}

// OwnershipSyncer is a local interface for ownership sync. ownership.Service
// satisfies it. Using a local interface keeps the org package decoupled from
// the ownership package (Go structural typing).
type OwnershipSyncer interface {
	SyncRepoOwnership(ctx context.Context, ghClient *gogithub.Client, repoID uuid.UUID, owner, repo string) error
}

func syncRepos(ghClient *gogithub.Client, orgStore OrgStore, catalogStore catalog.RepoStore, depSyncer DepSyncer, ownershipSyncer OwnershipSyncer, orgID uuid.UUID, orgSlug string) {
	ctx := context.Background()

	repos, _, err := ghClient.Repositories.ListByOrg(ctx, orgSlug, &gogithub.RepositoryListByOrgOptions{
		ListOptions: gogithub.ListOptions{PerPage: 100},
	})
	if err != nil {
		slog.Error("sync: failed to list repos from GitHub", "org_id", orgID, "error", err)
		return
	}

	allRepos := repos

	var syncErrors int
	for _, r := range allRepos {
		repo := &catalog.Repository{
			OrgID:         orgID,
			GitHubID:      r.GetID(),
			Name:          r.GetName(),
			FullName:      r.GetFullName(),
			DefaultBranch: r.GetDefaultBranch(),
			Private:       r.GetPrivate(),
			Fork:          r.GetFork(),
			Stars:         r.GetStargazersCount(),
		}
		if r.Description != nil {
			repo.Description = r.Description
		}
		if r.Language != nil {
			repo.Language = r.Language
		}

		upserted, err := catalogStore.UpsertRepository(ctx, repo)
		if err != nil {
			slog.Error("sync: failed to upsert repository", "repo", r.GetFullName(), "error", err)
			syncErrors++
			continue
		}

		if depSyncer != nil && upserted != nil {
			if err := depSyncer.SyncRepoDeps(ctx, ghClient, upserted.ID, orgSlug, r.GetName(), r.GetDefaultBranch()); err != nil {
				slog.Error("sync: dep sync failed for repo", "repo", r.GetFullName(), "error", err)
				// error isolation — continue processing remaining repos
			}
		}

		if ownershipSyncer != nil && upserted != nil {
			if err := ownershipSyncer.SyncRepoOwnership(ctx, ghClient, upserted.ID, orgSlug, r.GetName()); err != nil {
				slog.Error("sync: ownership sync failed", "repo", r.GetFullName(), "error", err)
				// error isolation — continue processing remaining repos
			}
		}
	}

	if syncErrors > 0 {
		slog.Error("sync: completed with errors", "org_id", orgID, "errors", syncErrors, "total", len(allRepos))
		return
	}

	if err := orgStore.SetLastSyncedAt(ctx, orgID, time.Now().UTC()); err != nil {
		slog.Error("sync: failed to update last_synced_at", "org_id", orgID, "error", err)
	}
}
