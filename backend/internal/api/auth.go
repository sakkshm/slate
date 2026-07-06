package api

import (
	"bytes"
	"context"
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

	// SameSite Lax works well if the client navigates directly.
	// However, if your frontend and backend run on completely different domains in prod,
	// you may need http.SameSiteNoneMode when Secure=true.
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

// GitHubTokenResponse represents the expected successful JSON response from GitHub
type GitHubTokenResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

func (e *APIEngine) HandleCallback(w http.ResponseWriter, r *http.Request) {
	var payload types.CallbackRequest // Maps to your CallbackPayload fields
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

	tokenURL := "https://github.com/login/oauth/access_token"
	reqData := map[string]string{
		"client_id":     e.config.GithubClientID,
		"client_secret": e.config.GithubClientSecret,
		"code":          payload.Code,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "SERVER_ERR", "Failed to process request data")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewBuffer(jsonData))
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "SERVER_ERR", "Failed to create token request")
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		utils.WriteHTTPError(w, http.StatusBadGateway, "GATEWAY_ERR", "Failed to reach GitHub")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		utils.WriteHTTPError(w, http.StatusBadGateway, "GATEWAY_ERR", "GitHub returned an invalid status")
		return
	}

	var tokenResp GitHubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "SERVER_ERR", "Failed to decode GitHub token")
		return
	}

	if tokenResp.AccessToken == "" {
		utils.WriteHTTPError(w, http.StatusBadRequest, "AUTH_ERR", "Invalid or expired authorization code")
		return
	}

	fmt.Println(tokenResp.AccessToken)
	fmt.Println(tokenResp.TokenType)
	fmt.Println(tokenResp.Scope)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "authenticated"})
}
