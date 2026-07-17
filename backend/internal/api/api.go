package api

import (
	"slate-backend/pkg/config"

	"gorm.io/gorm"
)

type APIEngine struct {
	config *config.Config
	database     *gorm.DB
}

func NewAPIEngine(config *config.Config, DB *gorm.DB) *APIEngine {
	return &APIEngine{
		config: config,
		database: DB,
	}
}
