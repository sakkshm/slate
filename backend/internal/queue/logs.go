package queue

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// list -> durable logs for late-joining SSE clients
// pub/sub channel -> real-time fan-out for connected SSE clients

const logTTL = 24 * time.Hour

func LogListKey(buidID string) string     { return "slate:logs:" + buidID }
func LogChanKey(buidID string) string     { return "slate:logs:" + buidID }
func DoneChanKey(buidID string) string    { return "slate:build-done:" + buidID }
func CancelChannelKey(buildID string) string { return "slate:cancel:" + buildID }

func WriteLogLine(ctx context.Context, cli *redis.Client, buildID, line string) error {
	key := LogListKey(buildID)

	if err := cli.RPush(ctx, key, line).Err(); err != nil {
		return err
	}

	return cli.Expire(ctx, key, logTTL).Err()
}

func PublishLogLine(ctx context.Context, cli *redis.Client, buildID, line string) error {
	return cli.Publish(ctx, LogChanKey(buildID), line).Err()
}

func PublishBuildDone(ctx context.Context, cli *redis.Client, buildID string) error {
	return cli.Publish(ctx, DoneChanKey(buildID), "1").Err()
}

func GetLogLines(ctx context.Context, cli *redis.Client, buildID string) ([]string, error) {
	lines, err := cli.LRange(ctx, LogListKey(buildID), 0, -1).Result()
	if err == redis.Nil {
		return nil, nil
	}

	return lines, err
}

func PublishCancel(ctx context.Context, cli *redis.Client, buildID string) error {
	return cli.Publish(ctx, CancelChannelKey(buildID), "1").Err()
}
