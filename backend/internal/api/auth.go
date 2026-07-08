package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slate-backend/internal/auth"
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

	fmt.Println(isProd)

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

	installURL := auth.GetInstallURL(e.config, stateToken)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.GetInstallURLResponse{
		URL: installURL,
	})
}

func (e *APIEngine) HandleCallback(w http.ResponseWriter, r *http.Request) {
	var payload types.CallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Bad request")
		return
	}

	cookie, err := r.Cookie("slate-auth-state-token")
	if err != nil {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "BAD_REQ", "Cookies not found")
		return
	}

	if cookie.Value != payload.State {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "AUTH_ERR", "State mismatch error")
		return
	}

	tokenResp, err := auth.GetAccessToken(e.config, payload.Code, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "SERVER_ERR", err.Error())
		return
	}

	userResp, err := auth.GetUserProfile(e.config, tokenResp.AccessToken, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "SERVER_ERR", err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userResp)
}
