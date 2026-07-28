package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slate-backend/internal/auth"
	"slate-backend/internal/user"
	"slate-backend/pkg/utils"
)

func (e *APIEngine) HandleGetUserProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "BAD_REQ", "User ID not found")
		return
	}

	userProfile, err := user.GetUserProfile(userID, e.clients.DB, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "USR_NOT_FND", "User not found")
		return
	}
	if userProfile == nil {
		utils.WriteHTTPError(w, http.StatusNotFound, "USR_NOT_FND", "User not found")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userProfile)
}

func (e *APIEngine) HandleGetUserRepos(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "BAD_REQ", "User ID not found")
		return
	}

	userProfile, err := user.GetUserProfile(userID, e.clients.DB, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "USR_NOT_FND", "User not found")
		return
	}
	if userProfile == nil {
		utils.WriteHTTPError(w, http.StatusNotFound, "USR_NOT_FND", "User not found")
		return
	}

	userInstallAccessToken, err := auth.GetInstallationAccessToken(e.config, userProfile.GithubInstallationID, r.Context())
	if err != nil {
		fmt.Println(err.Error())
		utils.WriteHTTPError(w, http.StatusUnauthorized, "USR_INST_TKN_NOT_FND", "User Installation token not found")
		return
	}

	userRepos, err := user.GetUserInstalledRepos(userInstallAccessToken, r.Context())
	if err != nil {
		fmt.Println(err.Error())
		utils.WriteHTTPError(w, http.StatusUnauthorized, "GET_REPOS_FAILED", "Unable to get user Repos")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userRepos)
}
