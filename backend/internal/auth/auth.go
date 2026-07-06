package auth

import (
	"fmt"
	"net/url"
	"slate-backend/pkg/config"
)

func GetInstallURL(config *config.Config, stateToken string) string {
	appSlug := config.GithubAppSlug
	targetURL := fmt.Sprintf("https://github.com/apps/%s/installations/new?state=%s",
		appSlug,
		url.QueryEscape(stateToken),
	)

	return targetURL
}
