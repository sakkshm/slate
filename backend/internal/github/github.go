package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slate-backend/pkg/types"
	"time"
)

func GetRepoBranches(userInstallAccessToken string, repoOwner string, repoName string, ctx context.Context) ([]types.GithubRepoBranch, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches", repoOwner, repoName)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create repo request payload: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", userInstallAccessToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Github repos endpoint returned unexpected status: %d", resp.StatusCode)
	}

	var apiResponse []types.GithubRepoBranch
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub branches response: %w", err)
	}

	return apiResponse, nil
}

func GetRepoContents(userInstallAccessToken string, repoOwner string, repoName string, path string, ref string, ctx context.Context) ([]types.GithubRepoContentEntry, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", repoOwner, repoName, path)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create repo contents request payload: %w", err)
	}

	q := req.URL.Query()
	if ref != "" {
		q.Set("ref", ref)
	}
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", userInstallAccessToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub contents endpoint returned unexpected status: %d", resp.StatusCode)
	}

	var apiResponse []types.GithubRepoContentEntry
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub contents response: %w", err)
	}

	return apiResponse, nil
}

func GetRepoLastCommit(userInstallAccessToken string, repoOwner string, repoName string, branchName string, ctx context.Context) (types.GithubRepoLastCommit, error) {

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s?per_page=1", repoOwner, repoName, branchName)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return types.GithubRepoLastCommit{}, fmt.Errorf("failed to create last commit request payload: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", userInstallAccessToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return types.GithubRepoLastCommit{}, fmt.Errorf("failed to reach GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return types.GithubRepoLastCommit{}, fmt.Errorf("Github repos endpoint returned unexpected status: %d", resp.StatusCode)
	}

	var apiResponse types.GithubRepoLastCommit
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return types.GithubRepoLastCommit{}, fmt.Errorf("failed to decode GitHub branches response: %w", err)
	}

	return apiResponse, nil
}

func CreateCommitStatus(userInstallAccessToken, repoOwner, repoName, commitSHA, state, description, contextName, targetURL string, ctx context.Context) error {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/statuses/%s", repoOwner, repoName, commitSHA)

	payload := types.GithubStatusPayload{
		State:       state,
		TargetURL:   targetURL,
		Description: description,
		Context:     contextName,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to generate GitHub status update payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to status update request payload: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+userInstallAccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 12 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("github statuses endpoint returned unexpected status: %d", resp.StatusCode)
	}
	return nil
}

func GetRepoFileContent(userInstallAccessToken string, repoOwner string, repoName string, path string, ref string, ctx context.Context) ([]byte, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", repoOwner, repoName, path)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create repo contents request payload: %w", err)
	}

	q := req.URL.Query()
	if ref != "" {
		q.Set("ref", ref)
	}
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", userInstallAccessToken))
	req.Header.Set("Accept", "application/vnd.github.raw")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub contents endpoint returned unexpected status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
