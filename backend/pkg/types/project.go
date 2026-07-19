package types

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID      uuid.UUID `json:"id" gorm:"primaryKey"`
	OwnerID int64     `json:"owner_id" gorm:"index;not null"`
	Owner   User      `json:"owner,omitempty"`
	Name    string    `json:"name" gorm:"not null"`
	Slug    string    `json:"slug" gorm:"not null;uniqueIndex:idx_owner_slug"`

	RepoID    int64  `json:"repo_id" gorm:"not null"`
	RepoURL   string `json:"repo_url" gorm:"not null"`
	RepoName  string `json:"repo_name" gorm:"not null"`
	ProdBranch string `json:"prod_branch" gorm:"not null"`

	Framework string `json:"framework"`
	RootDir   string `json:"root_dir"`
	BuildCmd  string `json:"build_cmd"`
	OutDir    string `json:"out_dir"`

	ActiveBuildID *uuid.UUID `json:"active_build_id,omitempty"`
	ActiveBuild   *Build     `json:"active_build,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Build struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	CommitSHA string    `json:"commit_sha"`
	CommitMsg string    `json:"commit_message"`
	Status    string    `json:"status"`

	LogLocation   string `json:"log_location"`
	AssetLocation string `json:"asset_location"`

	CreatedAt time.Time `json:"created_at"`
}
