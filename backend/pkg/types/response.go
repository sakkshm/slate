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