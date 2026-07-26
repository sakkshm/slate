package queue

import (
	"context"
	"slate-backend/pkg/config"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(cfg *config.Config) (*redis.Client, error) {
	cli := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0, // use default DB
	})

	err := cli.Ping(context.Background()).Err()
	if err != nil {
		return nil, err
	}

	err = ensureRedisStream(cli)
	if err != nil {
		return nil, err
	}

	return cli, err
}

func ensureRedisStream(client *redis.Client) error {
	err := client.Do(
		context.Background(),
		"XGROUP", "CREATE", BuildStream, ConsumerGroup, "$", "MKSTREAM",
	).Err()

	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}
