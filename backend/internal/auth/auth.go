package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slate-backend/pkg/config"
	"slate-backend/pkg/types"
	"slate-backend/pkg/utils"
	"time"
)

func GetOAuthURL(cfg *config.Config, stateToken string) string {
	return fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&scope=repo,user:email&state=%s",
		cfg.GithubClientID,
		url.QueryEscape(stateToken),
	)
}

func GetInstallURL(cfg *config.Config) string {
	return fmt.Sprintf("https://github.com/apps/%s/installations/new?setup_action=install",
		cfg.GithubAppSlug,
	)
}

func GetUserInstallations(accessToken string, githubUserID int64, ctx context.Context) (int64, error) {
	apiURL := "https://api.github.com/user/installations?per_page=100"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create installations request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to reach GitHub installations API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("github installations endpoint returned unexpected status: %d", resp.StatusCode)
	}

	var installationsResp types.GitHubInstallationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&installationsResp); err != nil {
		return 0, fmt.Errorf("failed to decode installations response: %w", err)
	}

	return selectOwnInstallation(installationsResp.Installations, githubUserID)
}

func selectOwnInstallation(installations []types.GitHubInstallation, githubUserID int64) (int64, error) {
	for _, inst := range installations {
		if inst.Account.ID == githubUserID {
			return inst.ID, nil
		}
	}

	if len(installations) == 0 {
		return 0, fmt.Errorf("no installations found for this user")
	}
	return 0, fmt.Errorf("no installation found on the authenticated user's own account")
}

func GetAccessToken(cfg *config.Config, payloadCode string, ctx context.Context) (types.GitHubTokenResponse, error) {
	tokenURL := "https://github.com/login/oauth/access_token"
	reqData := map[string]string{
		"client_id":     cfg.GithubClientID,
		"client_secret": cfg.GithubClientSecret,
		"code":          payloadCode,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return types.GitHubTokenResponse{}, fmt.Errorf("failed to process request data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return types.GitHubTokenResponse{}, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return types.GitHubTokenResponse{}, fmt.Errorf("failed to reach GitHub token endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return types.GitHubTokenResponse{}, fmt.Errorf("github token endpoint returned unexpected status: %d", resp.StatusCode)
	}

	var tokenResp types.GitHubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return types.GitHubTokenResponse{}, fmt.Errorf("failed to decode GitHub token payload: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return types.GitHubTokenResponse{}, fmt.Errorf("invalid or expired authorization code received from provider")
	}

	return tokenResp, nil
}

func GetUserProfile(cfg *config.Config, accessToken string, ctx context.Context) (types.GitHubAuthUserResponse, error) {
	apiURL := "https://api.github.com/user"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return types.GitHubAuthUserResponse{}, fmt.Errorf("failed to create profile request payload: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return types.GitHubAuthUserResponse{}, fmt.Errorf("failed to reach GitHub user profile API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return types.GitHubAuthUserResponse{}, fmt.Errorf("github profile endpoint returned unexpected status: %d", resp.StatusCode)
	}

	var userResp types.GitHubAuthUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return types.GitHubAuthUserResponse{}, fmt.Errorf("failed to decode GitHub profile response: %w", err)
	}

	// If public profile has email hidden, query the explicit sub-resource.
	// Resolving the email is best-effort: GitHub App tokens can be denied the
	// /user/emails resource (missing email permission), and a login must not
	// fail just because the user keeps their email private.
	if userResp.Email == "" {
		if email, err := getPrimaryEmail(accessToken, ctx); err == nil {
			userResp.Email = email
		} else {
			slog.Default().Warn("failed to resolve user primary email, continuing without it", "error", err)
		}
	}

	return userResp, nil
}

func getPrimaryEmail(accessToken string, ctx context.Context) (string, error) {
	apiURL := "https://api.github.com/user/emails"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to initialize email sub-resource request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("network failure requesting fallback email listing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github user/emails returned status code: %d", resp.StatusCode)
	}

	var emails []types.GitHubEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("failed parsing structured user email array: %w", err)
	}

	// Locate the verified primary address entry
	for _, entry := range emails {
		if entry.Primary && entry.Verified {
			return entry.Email, nil
		}
	}

	// Secondary Fallback if primary is unverified but exists
	for _, entry := range emails {
		if entry.Primary {
			return entry.Email, nil
		}
	}

	return "", fmt.Errorf("no primary email records found associated with this authorization scope")
}

func GetInstallationAccessToken(cfg *config.Config, installationID int64, ctx context.Context) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	githubJWT, err := utils.GenerateGithubJWT(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to generate Github JWT: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create profile request payload: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", githubJWT))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to reach GitHub Installation API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("github installation endpoint returned unexpected status: %d", resp.StatusCode)
	}

	var apiResponse types.InstallationAccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return "", fmt.Errorf("failed to decode GitHub Installation access token response: %w", err)
	}

	return apiResponse.InstallationAccessToken, nil
}
