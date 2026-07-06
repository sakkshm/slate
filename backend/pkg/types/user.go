package types

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID      `json:"id"`
	Name           string         `json:"name"`
	Email          string         `json:"email"`
	CreatedAt      time.Time      `json:"created_at"`
	GithubIdentity GitHubIdentity `json:"github_username"`
}

type GitHubIdentity struct {
	ID             uint      `json:"id"`
	UserID         uint      `json:"user_id"`
	GitHubUserID   int64     `json:"github_user_id"`
	InstallationID int64     `json:"installation_id"` // Used for App-level actions
	AccessToken    string    `json:"access_token"`    // Encrypt this field!
	RefreshToken   string    `json:"refresh_token"`   // Encrypt this field!
	ExpiresAt      time.Time `json:"expires_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
