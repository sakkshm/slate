package api

import (
	"encoding/json"
	"net/http"
	"slate-backend/internal/auth"
	"slate-backend/internal/user"
	"slate-backend/pkg/types"
	"slate-backend/pkg/utils"
	"time"
)

func (e *APIEngine) HandleInitiateLogin(w http.ResponseWriter, r *http.Request) {
	stateToken, err := utils.GenerateRandomString(32)
	if err != nil {
		utils.WriteHTTPError(w, 500, "INTERNAL_ERROR", err.Error())
		return
	}

	isProd := e.config.Environment == "production"

	cookie := &http.Cookie{
		Name:     "slate-auth-state-token",
		Value:    stateToken,
		Path:     "/",
		Expires:  time.Now().Add(15 * time.Minute),
		HttpOnly: true,
		Secure:   isProd,
	}

	if isProd {
		cookie.SameSite = http.SameSiteNoneMode
	} else {
		cookie.SameSite = http.SameSiteLaxMode
	}

	http.SetCookie(w, cookie)

	oauthURL := auth.GetOAuthURL(e.config, stateToken)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.AuthRedirectResponse{
		URL: oauthURL,
	})
}

func (e *APIEngine) HandleCallback(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB cap
	var payload types.CallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQUEST", "Bad request")
		return
	}

	if payload.Code == "" || payload.State == "" {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing required fields")
		return
	}

	cookie, err := r.Cookie("slate-auth-state-token")
	if err != nil {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	if cookie.Value != payload.State {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "slate-auth-state-token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	tokenResp, err := auth.GetAccessToken(e.config, payload.Code, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "TOKEN_EXCHANGE_ERR", "Unable to authenticate with GitHub")
		return
	}

	userResp, err := auth.GetUserProfile(e.config, tokenResp.AccessToken, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "PROFILE_FETCH_ERR", "Unable to fetch GitHub profile")
		return
	}

	installationID, err := auth.GetUserInstallations(tokenResp.AccessToken, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusConflict, "NO_INSTALLATION", "GitHub App not installed. Please install the app first.")
		return
	}

	encryptedAccessToken, err := utils.EncryptAESString(tokenResp.AccessToken, []byte(e.config.EncryptionKey))
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "ENCRYPTION_ERR", "Internal server error")
		return
	}

	currentTimeStamp := time.Now().UTC()

	userProfile := &types.User{
		ID:                   userResp.ID,
		GithubUsername:       userResp.Login,
		GithubInstallationID: installationID,
		Name:                 userResp.Name,
		Email:                userResp.Email,
		AvatarURL:            userResp.AvatarURL,
		EncryptedAccessToken: encryptedAccessToken,
		CreatedAt:            currentTimeStamp,
		UpdatedAt:            currentTimeStamp,
	}

	err = user.UpsertUserProfile(e.clients.DB, userProfile, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "DB_ERR", "Failed to save user profile")
		return
	}

	TTLSec := 60 * 60 * 24 * 7
	userJWTClaims := &types.JWTClaim{
		ID:             userProfile.ID,
		GithubUsername: userProfile.GithubUsername,
	}

	jwtToken, err := utils.GenerateJWT(userJWTClaims, int64(TTLSec), e.config)
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "SESSION_ERR", "Failed to create session")
		return
	}

	isProd := e.config.Environment == "production"

	sessionCookie := &http.Cookie{
		Name:     "slate-session",
		Value:    jwtToken,
		Path:     "/",
		Expires:  time.Now().Add(time.Duration(TTLSec) * time.Second),
		HttpOnly: true,
		Secure:   isProd,
	}

	if isProd {
		sessionCookie.SameSite = http.SameSiteNoneMode
	} else {
		sessionCookie.SameSite = http.SameSiteLaxMode
	}

	http.SetCookie(w, sessionCookie)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"authenticated": "ok",
	})
}

func (e *APIEngine) HandleInstallURL(w http.ResponseWriter, r *http.Request) {
	installURL := auth.GetInstallURL(e.config)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.AuthRedirectResponse{
		URL: installURL,
	})
}

func (e *APIEngine) HandleAuthLogout(w http.ResponseWriter, r *http.Request) {
	isProd := e.config.Environment == "production"

	cookie := &http.Cookie{
		Name:     "slate-session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProd,
	}

	if isProd {
		cookie.SameSite = http.SameSiteNoneMode
	} else {
		cookie.SameSite = http.SameSiteLaxMode
	}

	http.SetCookie(w, cookie)

	w.WriteHeader(http.StatusOK)
}
