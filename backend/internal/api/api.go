package api

import "slate-backend/pkg/config"

type APIEngine struct {
	config *config.Config
}

func NewAPIEngine(config *config.Config) *APIEngine {
	return &APIEngine{
		config: config,
	}
}