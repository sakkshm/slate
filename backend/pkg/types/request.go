package types

type CallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type CreateProjectRequest struct {
	Name       string `json:"name"`
	RepoURL    string `json:"repo_url"`
	RepoID     int64  `json:"repo_id"`
	RepoName   string `json:"repo_name"`
	FullName   string `json:"full_name"`
	ProdBranch string `json:"prod_branch"`
	Framework  string `json:"framework"`
	RootDir    string `json:"root_dir"`
	BuildCmd   string `json:"build_cmd"`
	OutDir     string `json:"out_dir"`
}

type UpdateProjectRequest struct {
	Name       *string `json:"name,omitempty"`
	ProdBranch *string `json:"prod_branch,omitempty"`
	Framework  *string `json:"framework,omitempty"`
	RootDir    *string `json:"root_dir,omitempty"`
	BuildCmd   *string `json:"build_cmd,omitempty"`
	OutDir     *string `json:"out_dir,omitempty"`
}
