package clients

import (
	"slate-backend/internal/db"
	"slate-backend/internal/queue"
	"slate-backend/internal/storage"
	"slate-backend/pkg/config"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Clients struct {
	DB    *gorm.DB
	Redis *redis.Client
	MinIO storage.Store
}

func New(cfg *config.Config) (*Clients, error) {
	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	redisClient, err := queue.NewRedisClient(cfg)
	if err != nil {
		return nil, err
	}

	minioStore, err := storage.NewMinIOStore(cfg)
	if err != nil {
		return nil, err
	}

	return &Clients{
		DB:    database,
		Redis: redisClient,
		MinIO: minioStore,
	}, nil
}
