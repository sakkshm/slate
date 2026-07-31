package types

type BuildEvents string

const (
	StatusQueued    BuildEvents = "queued"
	StatusBuilding  BuildEvents = "building"
	StatusReady     BuildEvents = "ready"
	StatusFailed    BuildEvents = "failed"
	StatusCancelled BuildEvents = "cancelled"
)

type BuildEvent struct {
	ProjectID               string   `json:"project_id"`
	BuildID                 string   `json:"build_id"`
	RepoName                string   `json:"repo_name"`
	RepoURL                 string   `json:"repo_url"`
	InstallationAccessToken string   `json:"installation_access_token"`
	CommitSHA               string   `json:"commit_sha"`
	CommitMsg               string   `json:"commit_message"`
	RootDir                 string   `json:"root_dir"`
	InstallCmd              string   `json:"install_cmd"`
	BuildCmd                string   `json:"build_cmd"`
	OutDir                  string   `json:"out_dir"`
	Env                     []string `json:"env"`
}
