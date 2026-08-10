package queue

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	DeployKeyPrefix = "deploy:"
	DeployTTL       = 7 * 24 * time.Hour
)

type DeployEntry struct {
	ProjectID string `json:"project_id"`
	AssetHash string `json:"asset_hash"`
	UpdatedAt string `json:"updated_at"`
}

func DeployKey(slug string) string {
	return DeployKeyPrefix + slug
}

func PublishDeployment(ctx context.Context, client *redis.Client, slug string, entry DeployEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return client.Set(ctx, DeployKey(slug), data, DeployTTL).Err()
}

func GetDeployment(ctx context.Context, client *redis.Client, slug string) (*DeployEntry, error) {
	raw, err := client.Get(ctx, DeployKey(slug)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var entry DeployEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func DeleteDeployment(ctx context.Context, client *redis.Client, slug string) error {
	return client.Del(ctx, DeployKey(slug)).Err()
}
