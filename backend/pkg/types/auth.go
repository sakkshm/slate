package types

type GitHubTokenResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

type GitHubAuthUserResponse struct {
	ID        int64  `json:"id"`         // GitHub global user ID
	Login     string `json:"login"`      // The user's handle/username
	Name      string `json:"name"`       // Display name (can be empty string)
	Email     string `json:"email"`      // Primary email
	AvatarURL string `json:"avatar_url"` // Profile image URL 
}

type GitHubEmailResponse struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}
