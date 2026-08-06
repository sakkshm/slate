package api

import (
	"encoding/json"
	"net/http"
	"slate-backend/internal/auth"
	"slate-backend/internal/framework"
	githubclient "slate-backend/internal/github"
	"slate-backend/internal/project"
	"slate-backend/internal/user"
	"slate-backend/pkg/types"
	"slate-backend/pkg/utils"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (e *APIEngine) HandleCreateProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "BAD_REQ", "User ID not found")
		return
	}

	username, _ := GetUsername(r.Context())

	var req types.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Invalid request body")
		return
	}

	if req.Name == "" || req.RepoURL == "" || req.RepoName == "" || req.FullName == "" || req.ProdBranch == "" {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Missing required fields")
		return
	}

	parts := strings.SplitN(req.FullName, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "fullName must be in the format owner/repo")
		return
	}
	repoOwner, repoName := parts[0], parts[1]

	userProfile, err := user.GetUserProfile(userID, e.clients.DB, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "USR_ERR", "Unable to fetch user profile")
		return
	}
	if userProfile == nil {
		utils.WriteHTTPError(w, http.StatusNotFound, "USR_NOT_FND", "User not found")
		return
	}

	installToken, err := auth.GetInstallationAccessToken(e.config, userProfile.GithubInstallationID, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "INST_TKN_ERR", "Unable to get installation access token")
		return
	}

	installCmd, buildCmd, outDir := req.InstallCmd, req.BuildCmd, req.OutDir
	detectedFramework := req.Framework

	if detectedFramework == "" {
		packagePath := "package.json"
		if req.RootDir != "" {
			packagePath = strings.TrimPrefix(req.RootDir, "/") + "/package.json"
		}
		if content, err := githubclient.GetRepoFileContent(installToken, repoOwner, repoName, packagePath, req.ProdBranch, r.Context()); err == nil {
			if fw, _ := framework.DetectFromPackageJSON(content); fw != "" {
				detectedFramework = fw
			}
		}
	}

	cfg := framework.Resolve(detectedFramework, installCmd, buildCmd, outDir)

	branches, err := githubclient.GetRepoBranches(installToken, repoOwner, repoName, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "GH_BRANCHES_ERR", "Unable to fetch repository branches")
		return
	}

	branchFound := false
	for _, b := range branches {
		if b.Name == req.ProdBranch {
			branchFound = true
			break
		}
	}
	if !branchFound {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BRANCH_NOT_FND", "Branch not found in repository")
		return
	}

	if req.RootDir != "" {
		_, err := githubclient.GetRepoContents(installToken, repoOwner, repoName, req.RootDir, req.ProdBranch, r.Context())
		if err != nil {
			utils.WriteHTTPError(w, http.StatusBadRequest, "ROOT_DIR_ERR", "Root directory not found in repository")
			return
		}
	}

	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	suffix, err := utils.GenerateRandomString(4)
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "SLUG_ERR", "Failed to generate project slug")
		return
	}
	slug = username + "-" + repoName + "-" + suffix

	proj := &types.Project{
		ID:         uuid.New(),
		OwnerID:    userID,
		Name:       req.Name,
		Slug:       slug,
		RepoID:     req.RepoID,
		RepoURL:    req.RepoURL,
		RepoName:   req.RepoName,
		ProdBranch: req.ProdBranch,
		Framework:  detectedFramework,
		RootDir:    req.RootDir,
		InstallCmd: cfg.InstallCmd,
		BuildCmd:   cfg.BuildCmd,
		OutDir:     cfg.OutDir,
	}

	if err := project.CreateProject(e.clients.DB, proj, r.Context()); err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "DB_ERR", "Failed to create project")
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(proj)
}

func (e *APIEngine) HandleListProjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "BAD_REQ", "User ID not found")
		return
	}

	projects, err := project.GetProjectsByOwner(userID, e.clients.DB, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "DB_ERR", "Failed to fetch projects")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(projects)
}

func (e *APIEngine) HandleGetProject(w http.ResponseWriter, r *http.Request) {
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
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "DB_ERR", "Failed to fetch project")
		return
	}
	if proj == nil {
		utils.WriteHTTPError(w, http.StatusNotFound, "PRJ_NOT_FND", "Project not found")
		return
	}
	if proj.OwnerID != userID {
		utils.WriteHTTPError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this project")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(proj)
}

func (e *APIEngine) HandleUpdateProject(w http.ResponseWriter, r *http.Request) {
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

	var req types.UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.ProdBranch != nil {
		updates["prod_branch"] = *req.ProdBranch
	}
	if req.Framework != nil {
		updates["framework"] = *req.Framework
	}
	if req.RootDir != nil {
		updates["root_dir"] = *req.RootDir
	}
	if req.InstallCmd != nil {
		updates["install_cmd"] = *req.InstallCmd
	}
	if req.BuildCmd != nil {
		updates["build_cmd"] = *req.BuildCmd
	}
	if req.OutDir != nil {
		updates["out_dir"] = *req.OutDir
	}

	if len(updates) == 0 {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "No fields to update")
		return
	}

	if err := project.UpdateProject(e.clients.DB, projectID, userID, updates, r.Context()); err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "DB_ERR", "Failed to update project")
		return
	}

	updated, err := project.GetProjectByID(projectID, e.clients.DB, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "DB_ERR", "Failed to fetch updated project")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updated)
}

func (e *APIEngine) HandleDeleteProject(w http.ResponseWriter, r *http.Request) {
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

	if err := project.DeleteProject(projectID, userID, e.clients.DB, r.Context()); err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "DB_ERR", "Failed to delete project")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
