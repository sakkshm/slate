package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type HealthStatus struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// HandleHealth reports liveness of the API and its backing services. Returns
// 200 when all dependencies are reachable, 503 otherwise.
func (e *APIEngine) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	checks := map[string]string{}
	healthy := true

	sqlDB, err := e.clients.DB.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil {
		checks["database"] = "down"
		healthy = false
	} else {
		checks["database"] = "up"
	}

	if err := e.clients.Redis.Ping(ctx).Err(); err != nil {
		checks["redis"] = "down"
		healthy = false
	} else {
		checks["redis"] = "up"
	}

	if _, err := e.clients.MinIO.List(ctx, ""); err != nil {
		checks["storage"] = "down"
		healthy = false
	} else {
		checks["storage"] = "up"
	}

	status := "ok"
	code := http.StatusOK
	if !healthy {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(HealthStatus{Status: status, Checks: checks})
}
