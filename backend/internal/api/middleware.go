package api

import (
	"context"
	"net/http"
	"slate-backend/pkg/utils"
)

const (
	UserIDKey   string = "id"
	UsernameKey string = "username"
)

func (e *APIEngine) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("slate-session")
		if err != nil {
			utils.WriteHTTPError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
			return
		}

		profile, err := utils.ParseJWT(cookie.Value, e.config)
		if err != nil {
			utils.WriteHTTPError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired session")
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, profile.ID)
		ctx = context.WithValue(ctx, UsernameKey, profile.GithubUsername)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(ctx context.Context) (int64, bool) {
	val := ctx.Value(UserIDKey)
	if val == nil {
		return 0, false
	}

	id, ok := val.(int64)
	return id, ok
}

func GetUsername(ctx context.Context) (string, bool) {
	val := ctx.Value(UsernameKey)
	if val == nil {
		return "", false
	}
	
	username, ok := val.(string)
	return username, ok
}