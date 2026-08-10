package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slate-backend/pkg/types"
	"strings"

	"github.com/google/uuid"
)

func GenerateRandomString(length int) (string, error) {
	if length > 36 {
		return "", fmt.Errorf("UUID max size is 36")
	}
	
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	return uuid.String()[:length], nil
}

const MaxSlugLen = 63

var slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

// Slugify converts an arbitrary string into a deployment-safe slug label:
// lowercase alphanumerics joined by single hyphens, never leading or trailing.
func Slugify(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "site"
	}
	return slug
}

// ValidSlug reports whether s is a valid deployment slug (lowercase
// alphanumerics and hyphens, matching the gateway's host matcher).
func ValidSlug(s string) bool {
	return slugPattern.MatchString(s)
}

// ProjectSlug composes "<username>-<repo>-<suffix>" where the repo portion is
// slugified and the result is capped to MaxSlugLen so it stays a valid DNS
// label. suffix is appended verbatim and must be short.
func ProjectSlug(username, repoName, suffix string) string {
	repoSlug := Slugify(repoName)
	// leave room for "<username>-" and "-<suffix>"
	maxRepo := MaxSlugLen - len(username) - len(suffix) - 2
	if maxRepo < 1 {
		maxRepo = 1
	}
	if len(repoSlug) > maxRepo {
		repoSlug = strings.TrimRight(repoSlug[:maxRepo], "-")
	}
	return username + "-" + repoSlug + "-" + suffix
}

func WriteHTTPError(w http.ResponseWriter, statusCode int, errorCode string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(types.APIErrorResponse{
		Code:    errorCode,
		Message: message,
	})
}
