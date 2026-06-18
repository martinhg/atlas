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

// VulnSyncer is a local interface for vulnerability sync. vuln.Service satisfies
// it. It runs once per org after the repo loop to batch-query OSV.dev for all
// dependencies. Using a local interface keeps the org package decoupled from the
// vuln package (Go structural typing).
type VulnSyncer interface {
	SyncOrgVulns(ctx context.Context, orgID uuid.UUID) error
}

func syncRepos(ghClient *gogithub.Client, orgStore OrgStore, catalogStore catalog.RepoStore, depSyncer DepSyncer, ownershipSyncer OwnershipSyncer, vulnSyncer VulnSyncer, orgID uuid.UUID) {
	ctx := context.Background()

	var allRepos []*gogithub.Repository
	opts := &gogithub.ListOptions{PerPage: 100, Page: 1}
	for {
		result, resp, err := ghClient.Apps.ListRepos(ctx, opts)
		if err != nil {
			slog.Error("sync: failed to list repos from GitHub", "org_id", orgID, "error", err)
			return
		}
		allRepos = append(allRepos, result.Repositories...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

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

		owner := r.GetOwner().GetLogin()

		if depSyncer != nil && upserted != nil {
			if err := depSyncer.SyncRepoDeps(ctx, ghClient, upserted.ID, owner, r.GetName(), r.GetDefaultBranch()); err != nil {
				slog.Error("sync: dep sync failed for repo", "repo", r.GetFullName(), "error", err)
			}
		}

		if ownershipSyncer != nil && upserted != nil {
			if err := ownershipSyncer.SyncRepoOwnership(ctx, ghClient, upserted.ID, owner, r.GetName()); err != nil {
				slog.Error("sync: ownership sync failed", "repo", r.GetFullName(), "error", err)
			}
		}
	}

	// Vulnerability sync runs once per org after all repos' deps are synced.
	// It is additive: a failure is logged but MUST NOT abort the org sync.
	if vulnSyncer != nil {
		if err := vulnSyncer.SyncOrgVulns(ctx, orgID); err != nil {
			slog.Error("sync: vuln sync failed", "org_id", orgID, "error", err)
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
