package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slate-backend/pkg/types"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func UpsertUserProfile(database *gorm.DB, userProfile *types.User, ctx context.Context) error {
	result := database.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"encrypted_access_token", "avatar_url", "name", "email", "github_installation_id", "github_username", "updated_at",
		}),
	}).Create(userProfile)

	return result.Error
}

func GetUserProfile(userID int64, database *gorm.DB, ctx context.Context) (*types.User, error) {

	var user types.User

	result := database.WithContext(ctx).First(&user, userID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &user, nil
}

func GetUserInstalledRepos(userInstallAccessToken string, ctx context.Context) (types.GithubInstallationReposResponse, error) {

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 12 * time.Second}

	result := types.GithubInstallationReposResponse{
		Repositories: []types.GithubInstallationRepo{},
	}

	const perPage = 100

	for page := 1; ; page++ {
		apiURL := fmt.Sprintf("https://api.github.com/installation/repositories?per_page=%d&page=%d", perPage, page)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			return types.GithubInstallationReposResponse{}, fmt.Errorf("failed to create repo request payload: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", userInstallAccessToken))
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := client.Do(req)
		if err != nil {
			return types.GithubInstallationReposResponse{}, fmt.Errorf("failed to reach GitHub API: %w", err)
		}

		var apiResponse types.GithubInstallationReposResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&apiResponse)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return types.GithubInstallationReposResponse{}, fmt.Errorf("Github repos endpoint returned unexpected status: %d", resp.StatusCode)
		}
		if decodeErr != nil {
			return types.GithubInstallationReposResponse{}, fmt.Errorf("failed to decode GitHub Installation access token response: %w", decodeErr)
		}

		result.TotalCount = apiResponse.TotalCount
		result.Repositories = append(result.Repositories, apiResponse.Repositories...)

		if int64(len(result.Repositories)) >= apiResponse.TotalCount || len(apiResponse.Repositories) == 0 {
			break
		}
	}

	return result, nil

}
