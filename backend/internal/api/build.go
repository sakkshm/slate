package api

import (
	"encoding/json"
	"fmt"
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
	"slate-backend/pkg/utils"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (e *APIEngine) HandleTriggerBuild(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "BAD_REQ", "User ID not found")
		return
	}

	projectIDParam := chi.URLParam(r, "projectID")
	if projectIDParam == "" {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Project ID not found")
		return
	}

	projectID, err := uuid.Parse(projectIDParam)
	if err != nil {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Project ID is not valid")
		return
	}

	proj, err := project.GetProjectByID(projectID, e.clients.DB, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Unable to get project")
		return
	}
	if proj == nil {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Project not found")
		return
	}
	if proj.OwnerID != userID {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Project not owned by current user")
		return
	}

	usr, err := user.GetUserProfile(userID, e.clients.DB, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Unable to get user")
		return
	}
	if usr == nil {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "User not found")
		return
	}

	installToken, err := auth.GetInstallationAccessToken(e.config, usr.GithubInstallationID, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "INST_TKN_ERR", "Unable to get installation access token")
		return
	}

	parts := strings.SplitN(proj.RepoName, "/", 2)
	if len(parts) != 2 {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "BAD_REQ", "Invalid repo name format")
		return
	}

	lastCommit, err := githubclient.GetRepoLastCommit(installToken, parts[0], parts[1], proj.ProdBranch, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "GH_COMMIT_ERR", "Unable to fetch latest commit")
		return
	}

	buildID := uuid.New()
	newBuild := &types.Build{
		ID:        buildID,
		ProjectID: projectID,
		CommitSHA: lastCommit.SHA,
		CommitMsg: lastCommit.Commit.Message,
		Status:    types.StatusQueued,
	}

	if err := buildpkg.CreateBuild(e.clients.DB, newBuild, r.Context()); err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "DB_ERR", "Failed to create build")
		return
	}

	cfg := framework.Resolve(proj.Framework, proj.InstallCmd, proj.BuildCmd, proj.OutDir)

	_ = project.UpdateProject(e.clients.DB, projectID, userID, map[string]interface{}{"active_build_id": buildID}, r.Context())

	event := types.BuildEvent{
		ProjectID:               projectID.String(),
		BuildID:                 buildID.String(),
		RepoURL:                 proj.RepoURL,
		RepoName:                proj.RepoName,
		InstallationAccessToken: installToken,
		CommitSHA:               lastCommit.SHA,
		CommitMsg:               lastCommit.Commit.Message,
		RootDir:                 proj.RootDir,
		InstallCmd:              cfg.InstallCmd,
		BuildCmd:                cfg.BuildCmd,
		OutDir:                  cfg.OutDir,
	}

	envVars, envErr := envvar.ResolveAll(e.clients.DB, []byte(e.config.EncryptionKey), projectID, r.Context())
	if envErr != nil {
		fmt.Printf("[API] Failed to resolve env vars: %v\n", envErr)
	} else {
		event.Env = envVars
	}

	if _, err := queue.PublishBuildRequest(r.Context(), e.clients.Redis, event); err != nil {
		fmt.Printf("[API] Failed to publish build event: %v\n", err)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(types.TriggerBuildResponse{
		BuildID: buildID.String(),
		Status:  string(types.StatusQueued),
	})
}

func (e *APIEngine) HandleListBuilds(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "BAD_REQ", "User ID not found")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Invalid project ID")
		return
	}

	proj, err := project.GetProjectByID(projectID, e.clients.DB, r.Context())
	if err != nil || proj == nil {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Project not found")
		return
	}
	if proj.OwnerID != userID {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Project not owned by current user")
		return
	}

	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	builds, total, err := buildpkg.GetBuildByProject(e.clients.DB, projectID, limit, offset, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "DB_ERR", "Failed to fetch builds")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(types.ListBuildsResponse{
		Builds: builds,
		Total:  total,
	})
}

func (e *APIEngine) HandleGetBuild(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "BAD_REQ", "User ID not found")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Invalid project ID")
		return
	}

	proj, err := project.GetProjectByID(projectID, e.clients.DB, r.Context())
	if err != nil || proj == nil {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Project not found")
		return
	}
	if proj.OwnerID != userID {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Project not owned by current user")
		return
	}

	buildID, err := uuid.Parse(chi.URLParam(r, "buildID"))
	if err != nil {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Invalid build ID")
		return
	}

	b, err := buildpkg.GetBuildByID(e.clients.DB, buildID, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "DB_ERR", "Failed to fetch build")
		return
	}
	if b == nil || b.ProjectID != projectID {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Build not found")
		return
	}

	deploymentURL := fmt.Sprintf("https://%s.%s", proj.Slug, e.config.SiteBaseDomain)
	assetURL := fmt.Sprintf("http://%s/%s/projects/%s/builds/%s.tar.gz",
		e.config.MinIOEndpoint, e.config.MinIOBucket, projectID, b.AssetLocation)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(types.BuildDetailResponse{
		Build:         *b,
		DeploymentURL: deploymentURL,
		AssetURL:      assetURL,
	})
}

func (e *APIEngine) HandleCancelBuild(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "BAD_REQ", "User ID not found")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Invalid project ID")
		return
	}

	proj, err := project.GetProjectByID(projectID, e.clients.DB, r.Context())
	if err != nil || proj == nil {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Project not found")
		return
	}
	if proj.OwnerID != userID {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Project not owned by current user")
		return
	}

	buildID, err := uuid.Parse(chi.URLParam(r, "buildID"))
	if err != nil {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Invalid build ID")
		return
	}

	b, err := buildpkg.GetBuildByID(e.clients.DB, buildID, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "DB_ERR", "Failed to fetch build")
		return
	}
	if b == nil || b.ProjectID != projectID {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Build not found")
		return
	}

	switch b.Status {
	case types.StatusQueued:
		if err := buildpkg.UpdateBuildStatus(e.clients.DB, buildID, types.StatusCancelled, r.Context()); err != nil {
			utils.WriteHTTPError(w, http.StatusInternalServerError, "DB_ERR", "Failed to cancel build")
			return
		}
	case types.StatusBuilding:
		if err := queue.PublishCancel(r.Context(), e.clients.Redis, buildID.String()); err != nil {
			utils.WriteHTTPError(w, http.StatusInternalServerError, "CANCEL_ERR", "Failed to request build cancellation")
			return
		}
	default:
		utils.WriteHTTPError(w, http.StatusConflict, "CONFLICT", "Build cannot be cancelled")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(types.CancelBuildResponse{
		BuildID: buildID.String(),
		Status:  string(types.StatusCancelled),
	})
}
