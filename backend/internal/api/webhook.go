package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"slate-backend/internal/auth"
	buildpkg "slate-backend/internal/build"
	"slate-backend/internal/framework"
	"slate-backend/internal/project"
	"slate-backend/internal/queue"
	"slate-backend/internal/user"
	"slate-backend/pkg/types"
	"strings"

	"github.com/google/uuid"
)

func (e *APIEngine) HandleGithubWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[WEBHOOK] Failed to read request body: %v", err)
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
		log.Printf("[WEBHOOK] Failed to parse payload: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	if event.Repository.ID == 0 || event.After == "" || strings.Trim(event.After, "0") == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	proj, err := project.GetProjectByRepoID(e.clients.DB, event.Repository.ID, r.Context())
	if err != nil {
		log.Printf("[WEBHOOK] Failed to look up project for repo %d: %v", event.Repository.ID, err)
		w.WriteHeader(http.StatusOK)
		return
	}
	if proj == nil {
		log.Printf("[WEBHOOK] No project found for repo %d, ignoring", event.Repository.ID)
		w.WriteHeader(http.StatusOK)
		return
	}

	branch := strings.TrimPrefix(event.Ref, "refs/heads/")
	if branch != proj.ProdBranch {
		log.Printf("[WEBHOOK] Branch %s does not match prod branch %s, ignoring", branch, proj.ProdBranch)
		w.WriteHeader(http.StatusOK)
		return
	}

	existing, err := buildpkg.GetBuildByProjectAndCommit(e.clients.DB, proj.ID, event.After, r.Context())
	if err != nil {
		log.Printf("[WEBHOOK] Failed to check existing build: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}
	if existing != nil {
		log.Printf("[WEBHOOK] Build already exists for commit %s, skipping", event.After)
		w.WriteHeader(http.StatusOK)
		return
	}

	commitMsg := ""
	if event.HeadCommit != nil {
		commitMsg = event.HeadCommit.Message
	}

	usr, err := user.GetUserProfile(proj.OwnerID, e.clients.DB, r.Context())
	if err != nil || usr == nil {
		log.Printf("[WEBHOOK] Failed to get owner profile for project %s: %v", proj.ID, err)
		w.WriteHeader(http.StatusOK)
		return
	}

	installToken, err := auth.GetInstallationAccessToken(e.config, usr.GithubInstallationID, r.Context())
	if err != nil {
		log.Printf("[WEBHOOK] Failed to get installation token: %v", err)
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
		log.Printf("[WEBHOOK] Failed to create build: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	cfg := framework.Resolve(proj.Framework, proj.InstallCmd, proj.BuildCmd, proj.OutDir)

	_ = project.UpdateProject(e.clients.DB, proj.ID, proj.OwnerID, map[string]interface{}{"active_build_id": buildID}, r.Context())

	buildEvent := types.BuildEvent{
		ProjectID:               proj.ID.String(),
		BuildID:                 buildID.String(),
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

	if _, err := queue.PublishBuildRequest(r.Context(), e.clients.Redis, buildEvent); err != nil {
		log.Printf("[WEBHOOK] Failed to publish build event: %v", err)
	}

	log.Printf("[WEBHOOK] Queued build %s for repo %s commit %s", buildID, event.Repository.FullName, event.After)
	w.WriteHeader(http.StatusOK)
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
