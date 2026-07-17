package api

import (
	"encoding/json"
	"net/http"
	"slate-backend/internal/auth"
	"slate-backend/internal/github"
	"slate-backend/internal/user"
	"slate-backend/pkg/types"
	"slate-backend/pkg/utils"
	"strings"
)

func (e *APIEngine) HandleGetRepoBranches(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "BAD_REQ", "User ID not found")
		return
	}

	fullName := r.URL.Query().Get("fullName")
	if fullName == "" {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "fullName query parameter is required")
		return
	}

	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "fullName must be in the format owner/repo")
		return
	}
	repoOwner, repoName := parts[0], parts[1]

	userProfile, err := user.GetUserProfile(userID, e.database, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "USR_NOT_FND", "Unable to fetch user profile")
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

	branches, err := github.GetRepoBranches(installToken, repoOwner, repoName, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "GH_BRANCHES_ERR", "Unable to fetch repository branches")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(types.RepoBranchesResponse{Branches: branches})
}

func (e *APIEngine) HandleGetRepoContents(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "BAD_REQ", "User ID not found")
		return
	}

	fullName := r.URL.Query().Get("fullName")
	if fullName == "" {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "fullName query parameter is required")
		return
	}

	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "fullName must be in the format owner/repo")
		return
	}
	repoOwner, repoName := parts[0], parts[1]

	ref := r.URL.Query().Get("ref")
	path := r.URL.Query().Get("path")

	userProfile, err := user.GetUserProfile(userID, e.database, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "USR_NOT_FND", "Unable to fetch user profile")
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

	entries, err := github.GetRepoContents(installToken, repoOwner, repoName, path, ref, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "GH_CONTENTS_ERR", "Unable to fetch repository contents")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(types.RepoContentsResponse{Entries: entries})
}
