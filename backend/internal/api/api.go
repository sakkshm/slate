package api

import (
	"log/slog"
	"slate-backend/internal/clients"
	"slate-backend/pkg/config"
)

type APIEngine struct {
	config  *config.Config
	clients *clients.Clients
}

func NewAPIEngine(config *config.Config, c *clients.Clients) *APIEngine {
	return &APIEngine{
		config:  config,
		clients: c,
	}
}

// apiLog returns the configured structured logger scoped to the API component.
func apiLog() *slog.Logger {
	return slog.Default().With("component", "api")
}
