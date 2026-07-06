package types

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID      uuid.UUID `json:"id"`
	OwnerID uuid.UUID `json:"owner_id"`
	Owner   User      `json:"owner,omitempty"`
	Name    string    `json:"name"`
	Slug    string    `json:"slug"`

	RepoURL    string `json:"repo_url"`
	RepoID     string `json:"repo_id"`
	RepoName   string `json:"repo_name"`
	ProdBranch string `json:"prod_branch"`

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
	Status    string    `json:"status"` // e.g., "QUEUED", "BUILDING", "READY", "FAILED"

	LogLocation   string `json:"log_location"`
	AssetLocation string `json:"asset_location"`

	CreatedAt time.Time `json:"created_at"`
}
