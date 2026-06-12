package auth

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	GitHubID  int64     `json:"github_id"`
	Login     string    `json:"login"`
	Name      *string   `json:"name,omitempty"`
	Email     *string   `json:"email,omitempty"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
