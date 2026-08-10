package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

// clientIPKey returns the canonical client IP stored by the ClientIPFrom*
// middleware installed upstream in main.go. IPv6 addresses are bucketed by
// their /64 prefix.
func clientIPKey(r *http.Request) (string, error) {
	return httprate.CanonicalizeIP(middleware.GetClientIP(r.Context())), nil
}

// RateLimitGlobal guards the whole API per client IP. It is registered after
// the gateway short-circuit, so deployed sites are never throttled.
func RateLimitGlobal(rps int) func(http.Handler) http.Handler {
	return httprate.LimitBy(rps, time.Second, clientIPKey)
}

// RateLimitAuth protects OAuth initiate/callback endpoints from abuse.
func RateLimitAuth(rpm int) func(http.Handler) http.Handler {
	return httprate.LimitBy(rpm, time.Minute, clientIPKey)
}

// RateLimitWebhook guards the GitHub webhook receiver. The limit is generous so
// GitHub's own retries are not cut off.
func RateLimitWebhook(rpm int) func(http.Handler) http.Handler {
	return httprate.LimitBy(rpm, time.Minute, clientIPKey)
}

// RateLimitBuild caps build triggers per user. The worker is single, so this
// prevents queue flooding.
func RateLimitBuild(rph int) func(http.Handler) http.Handler {
	return httprate.LimitBy(rph, time.Hour, func(r *http.Request) (string, error) {
		uid, ok := GetUserID(r.Context())
		if !ok {
			return "anonymous", nil
		}
		return strconv.FormatInt(uid, 10), nil
	})
}
