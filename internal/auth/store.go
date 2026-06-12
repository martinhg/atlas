package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) UpsertUser(ctx context.Context, u *User, accessToken string) (*User, error) {
	var user User
	err := s.db.QueryRow(ctx, `
		INSERT INTO users (github_id, login, name, email, avatar_url, access_token)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (github_id)
		DO UPDATE SET
			login       = EXCLUDED.login,
			name        = EXCLUDED.name,
			email       = EXCLUDED.email,
			avatar_url  = EXCLUDED.avatar_url,
			access_token = EXCLUDED.access_token,
			updated_at  = NOW()
		RETURNING id, github_id, login, name, email, avatar_url, created_at, updated_at
	`, u.GitHubID, u.Login, u.Name, u.Email, u.AvatarURL, accessToken).
		Scan(&user.ID, &user.GitHubID, &user.Login, &user.Name, &user.Email, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	err := s.db.QueryRow(ctx, `
		SELECT id, github_id, login, name, email, avatar_url, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.GitHubID, &user.Login, &user.Name, &user.Email, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
