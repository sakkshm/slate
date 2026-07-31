package api

const (
	AuthInitRoute     = "/api/auth/github/initiate-login"
	AuthCallbackRoute = "/api/auth/github/callback"
	AuthInstallRoute  = "/api/auth/github/install-url"
	AuthLogoutRoute   = "/api/auth/github/logout"

	WebhookRoute = "/api/webhooks/github"

	UserRoute     = "/api/user"
	UserRepoRoute = "/api/user/repos"

	RepoBranchesRoute = "/api/repos/branches"
	RepoContentsRoute = "/api/repos/contents"

	ProjectRoute     = "/api/projects"
	ProjectByIDRoute = "/api/projects/{projectID}"

	BuildsRoute      = "/api/projects/{projectID}/builds"
	BuildByIDRoute   = "/api/projects/{projectID}/builds/{buildID}"
	BuildLogsRoute   = "/api/projects/{projectID}/builds/{buildID}/logs"
	CancelBuildRoute = "/api/projects/{projectID}/builds/{buildID}/cancel"
)
