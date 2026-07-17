package types

import "time"

type User struct {
	ID                   int64     `json:"id" gorm:"primaryKey;autoIncrement:false"` // Use explicit GitHub User ID
	GithubUsername       string    `json:"username" gorm:"not null"`
	GithubInstallationID int64     `json:"github_installation_id" gorm:"not null"`
	Name                 string    `json:"name" gorm:"not null"`
	Email                string    `json:"email" gorm:"not null"`
	AvatarURL            string    `json:"avatar_url" gorm:"not null"`
	EncryptedAccessToken string    `json:"-" gorm:"not null"`
	CreatedAt            time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt            time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type JWTClaim struct {
	ID             int64
	GithubUsername string
}
