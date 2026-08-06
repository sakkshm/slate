package types

import "time"

type APIErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"msg"`
}

type AuthRedirectResponse struct {
	URL string `json:"url"`
}

type InstallationAccessTokenResponse struct {
	InstallationAccessToken string    `json:"token"`
	ExpiresAt               time.Time `json:"expires_at"`
}

type GithubInstallationReposResponse struct {
	TotalCount   int64                    `json:"total_count"`
	Repositories []GithubInstallationRepo `json:"repositories"`
}

type RepoBranchesResponse struct {
	Branches []GithubRepoBranch `json:"branches"`
}

type RepoContentsResponse struct {
	Entries []GithubRepoContentEntry `json:"entries"`
}

type TriggerBuildResponse struct {
	BuildID string `json:"build_id"`
	Status  string `json:"status"`
}

type ListBuildsResponse struct {
	Builds []Build `json:"builds"`
	Total  int64   `json:"total"`
}

type CancelBuildResponse struct {
	BuildID string `json:"build_id"`
	Status  string `json:"status"`
}

type EnvVarResponse struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}
