package org

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrgStore interface {
	UpsertOrg(ctx context.Context, o *Organization) (*Organization, error)
	GetOrgBySlug(ctx context.Context, slug string) (*Organization, error)
	GetOrgsByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]Organization, error)
	GetOrgByInstallationID(ctx context.Context, installationID int64) (*Organization, error)
	SetInstallationID(ctx context.Context, orgID uuid.UUID, installationID int64) error
	SetLastSyncedAt(ctx context.Context, orgID uuid.UUID, t time.Time) error
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) UpsertOrg(ctx context.Context, o *Organization) (*Organization, error) {
	var org Organization
	err := s.db.QueryRow(ctx, `
		INSERT INTO organizations (github_id, name, slug, github_installation_id, owner_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (github_id)
		DO UPDATE SET
			name                   = EXCLUDED.name,
			slug                   = EXCLUDED.slug,
			github_installation_id = EXCLUDED.github_installation_id,
			owner_id               = EXCLUDED.owner_id,
			updated_at             = NOW()
		RETURNING id, github_id, name, slug, github_installation_id, owner_id, last_synced_at, created_at, updated_at
	`, o.GitHubID, o.Name, o.Slug, o.GitHubInstallationID, o.OwnerID).
		Scan(&org.ID, &org.GitHubID, &org.Name, &org.Slug, &org.GitHubInstallationID, &org.OwnerID, &org.LastSyncedAt, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (s *Store) GetOrgBySlug(ctx context.Context, slug string) (*Organization, error) {
	var org Organization
	err := s.db.QueryRow(ctx, `
		SELECT id, github_id, name, slug, github_installation_id, owner_id, last_synced_at, created_at, updated_at
		FROM organizations WHERE slug = $1
	`, slug).
		Scan(&org.ID, &org.GitHubID, &org.Name, &org.Slug, &org.GitHubInstallationID, &org.OwnerID, &org.LastSyncedAt, &org.CreatedAt, &org.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (s *Store) GetOrgsByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]Organization, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, github_id, name, slug, github_installation_id, owner_id, last_synced_at, created_at, updated_at
		FROM organizations WHERE owner_id = $1
		ORDER BY created_at
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orgs := make([]Organization, 0)
	for rows.Next() {
		var o Organization
		if err := rows.Scan(&o.ID, &o.GitHubID, &o.Name, &o.Slug, &o.GitHubInstallationID, &o.OwnerID, &o.LastSyncedAt, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

func (s *Store) GetOrgByInstallationID(ctx context.Context, installationID int64) (*Organization, error) {
	var org Organization
	err := s.db.QueryRow(ctx, `
		SELECT id, github_id, name, slug, github_installation_id, owner_id, last_synced_at, created_at, updated_at
		FROM organizations WHERE github_installation_id = $1
	`, installationID).
		Scan(&org.ID, &org.GitHubID, &org.Name, &org.Slug, &org.GitHubInstallationID, &org.OwnerID, &org.LastSyncedAt, &org.CreatedAt, &org.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (s *Store) SetInstallationID(ctx context.Context, orgID uuid.UUID, installationID int64) error {
	_, err := s.db.Exec(ctx, `
		UPDATE organizations SET github_installation_id = $1, updated_at = NOW() WHERE id = $2
	`, installationID, orgID)
	return err
}

func (s *Store) SetLastSyncedAt(ctx context.Context, orgID uuid.UUID, t time.Time) error {
	_, err := s.db.Exec(ctx, `
		UPDATE organizations SET last_synced_at = $1, updated_at = NOW() WHERE id = $2
	`, t, orgID)
	return err
}
