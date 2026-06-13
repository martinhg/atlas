package org

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	gogithub "github.com/google/go-github/v69/github"
	"github.com/nesbite/atlas/internal/catalog"
)

func syncRepos(ghClient *gogithub.Client, orgStore OrgStore, catalogStore catalog.RepoStore, orgID uuid.UUID, orgSlug string) {
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

		if _, err := catalogStore.UpsertRepository(ctx, repo); err != nil {
			slog.Error("sync: failed to upsert repository", "repo", r.GetFullName(), "error", err)
			syncErrors++
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
