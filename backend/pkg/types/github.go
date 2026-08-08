package types

type GithubInstallationRepo struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Private       bool   `json:"private"`
	HTMLURL       string `json:"html_url"`
	DefaultBranch string `json:"default_branch"`
}

type GithubRepoContentEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int64  `json:"size"`
	Type        string `json:"type"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url"`
}

type GithubRepoBranch struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
	Protected bool `json:"protected"`
}

type GithubRepoLastCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
	} `json:"commit"`
}

type GithubStatusPayload struct {
	State string `json:"state"`
	TargetURL string `json:"target_url,omitempty"`
	Description string `json:"description,omitempty"`
	Context string `json:"context,omitempty"`
}

type GithubHookConfig struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Secret      string `json:"secret"`
}

type GithubWebhookPayload struct {
	Name   string           `json:"name"`
	Active bool             `json:"active"`
	Events []string         `json:"events"`
	Config GithubHookConfig `json:"config"`
}

type GithubRepoWebhook struct {
	ID     int              `json:"id"`
	Name   string           `json:"name"`
	Active bool             `json:"active"`
	Events []string         `json:"events"`
	Config GithubHookConfig `json:"config"`
}