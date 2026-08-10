package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"slate-backend/internal/auth"
	buildpkg "slate-backend/internal/build"
	"slate-backend/internal/envvar"
	"slate-backend/internal/framework"
	githubclient "slate-backend/internal/github"
	"slate-backend/internal/project"
	"slate-backend/internal/queue"
	"slate-backend/internal/user"
	"slate-backend/pkg/types"
	"strings"

	"github.com/google/uuid"
)

func (e *APIEngine) HandleGithubWebhook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20) // 5MB cap
	body, err := io.ReadAll(r.Body)
	if err != nil {
		apiLog().Error("failed to read request body", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}
	defer r.Body.Close()

	signature := r.Header.Get("X-Hub-Signature-256")
	if !verifyWebhookSignature(e.config.GithubWebhookSecret, body, signature) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	if eventType != "push" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var event types.GithubPushEvent
	if err := json.Unmarshal(body, &event); err != nil {
		apiLog().Error("failed to parse webhook payload", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	if event.Repository.ID == 0 || event.After == "" || strings.Trim(event.After, "0") == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	proj, err := project.GetProjectByRepoID(e.clients.DB, event.Repository.ID, r.Context())
	if err != nil {
		apiLog().Error("failed to look up project for repo", "repo_id", event.Repository.ID, "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}
	if proj == nil {
		apiLog().Info("no project found for repo, ignoring", "repo_id", event.Repository.ID)
		w.WriteHeader(http.StatusOK)
		return
	}

	branch := strings.TrimPrefix(event.Ref, "refs/heads/")
	if branch != proj.ProdBranch {
		apiLog().Info("branch does not match prod branch, ignoring", "branch", branch, "prod_branch", proj.ProdBranch)
		w.WriteHeader(http.StatusOK)
		return
	}

	existing, err := buildpkg.GetBuildByProjectAndCommit(e.clients.DB, proj.ID, event.After, r.Context())
	if err != nil {
		apiLog().Error("failed to check existing build", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}
	if existing != nil {
		apiLog().Info("build already exists for commit, skipping", "commit", event.After)
		w.WriteHeader(http.StatusOK)
		return
	}

	commitMsg := ""
	if event.HeadCommit != nil {
		commitMsg = event.HeadCommit.Message
	}

	usr, err := user.GetUserProfile(proj.OwnerID, e.clients.DB, r.Context())
	if err != nil || usr == nil {
		apiLog().Error("failed to get owner profile for project", "project_id", proj.ID, "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	installToken, err := auth.GetInstallationAccessToken(e.config, usr.GithubInstallationID, r.Context())
	if err != nil {
		apiLog().Error("failed to get installation token", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	buildID := uuid.New()
	newBuild := &types.Build{
		ID:        buildID,
		ProjectID: proj.ID,
		CommitSHA: event.After,
		CommitMsg: commitMsg,
		Status:    types.StatusQueued,
	}

	if err := buildpkg.CreateBuild(e.clients.DB, newBuild, r.Context()); err != nil {
		apiLog().Error("failed to create build", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	cfg := framework.Resolve(proj.Framework, proj.InstallCmd, proj.BuildCmd, proj.OutDir)

	buildEvent := types.BuildEvent{
		ProjectID:               proj.ID.String(),
		BuildID:                 buildID.String(),
		Slug:                    proj.Slug,
		RepoURL:                 proj.RepoURL,
		RepoName:                proj.RepoName,
		InstallationAccessToken: installToken,
		CommitSHA:               event.After,
		CommitMsg:               commitMsg,
		RootDir:                 proj.RootDir,
		InstallCmd:              cfg.InstallCmd,
		BuildCmd:                cfg.BuildCmd,
		OutDir:                  cfg.OutDir,
	}

	envVars, envErr := envvar.ResolveAll(e.clients.DB, []byte(e.config.EncryptionKey), proj.ID, r.Context())
	if envErr != nil {
		apiLog().Error("failed to resolve env vars", "error", envErr)
	} else {
		buildEvent.Env = envVars
	}

	if _, err := queue.PublishBuildRequest(r.Context(), e.clients.Redis, buildEvent); err != nil {
		apiLog().Error("failed to publish build event", "error", err)
	}

	apiLog().Info("queued build", "build_id", buildID, "repo", event.Repository.FullName, "commit", event.After)
	w.WriteHeader(http.StatusOK)
}

func (e *APIEngine) registerProjectWebhook(repoOwner, repoName, installToken string, ctx context.Context) {
	webhookURL := e.config.AppURL + "/api/webhooks/github"

	hooks, err := githubclient.ListRepoWebhooks(installToken, repoOwner, repoName, ctx)
	if err != nil {
		apiLog().Error("unable to check existing webhooks", "repo", repoOwner+"/"+repoName, "error", err)
		return
	}
	for _, h := range hooks {
		if h.Config.URL == webhookURL {
			apiLog().Info("webhook already registered", "repo", repoOwner+"/"+repoName)
			return
		}
	}

	if err := githubclient.CreateRepoWebhook(installToken, repoOwner, repoName, webhookURL, e.config.GithubWebhookSecret, ctx); err != nil {
		apiLog().Error("failed to register webhook", "repo", repoOwner+"/"+repoName, "error", err)
		return
	}

	apiLog().Info("registered push webhook", "repo", repoOwner+"/"+repoName, "url", webhookURL)
}

func verifyWebhookSignature(secret string, body []byte, signatureHeader string) bool {
	if !strings.HasPrefix(signatureHeader, "sha256=") {
		return false
	}

	expected := strings.TrimPrefix(signatureHeader, "sha256=")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	computed := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(computed), []byte(expected))
}
