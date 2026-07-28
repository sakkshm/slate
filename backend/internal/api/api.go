package api

import (
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
