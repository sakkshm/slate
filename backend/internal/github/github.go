package github

import (
	"context"
	"encoding/json"
	"fmt"
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
