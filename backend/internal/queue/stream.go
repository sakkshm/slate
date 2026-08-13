package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"slate-backend/pkg/types"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	BuildStream   = "slate:builds"
	ConsumerGroup = "workers"
)

func PublishBuildRequest(ctx context.Context, client *redis.Client, event types.BuildEvent) (string, error) {
	jsonData, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("unable to marshal build events")
	}

	return client.XAdd(ctx, &redis.XAddArgs{
		Stream: BuildStream,
		MaxLen: 1000,
		Approx: true,
		Values: map[string]interface{}{
			"payload": jsonData,
		},
	}).Result()
}

func ClaimBuildRequest(ctx context.Context, client *redis.Client, consumerName string) (*types.BuildEvent, string, error) {
	streams, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    ConsumerGroup,
		Consumer: consumerName,
		Streams:  []string{BuildStream, ">"},
		Count:    1,
		Block:    5 * time.Second,
	}).Result()
	if err == redis.Nil || len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, "", nil
	}

	msg := streams[0].Messages[0]
	payload, ok := msg.Values["payload"].(string)
	if !ok {
		return nil, "", fmt.Errorf("missing or invalid payload field in message %s", msg.ID)
	}

	var event types.BuildEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal build event: %w", err)

	}

	return &event, msg.ID, nil
}

func AckBuild(ctx context.Context, client *redis.Client, msgID string) error {
	_, err := client.XAck(ctx, BuildStream, ConsumerGroup, msgID).Result()
	return err
}

func ReclaimStuckBuilds(ctx context.Context, client *redis.Client, consumerName string, minIdle time.Duration) ([]*types.BuildEvent, []string, error) {
	pending, err := client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: BuildStream,
		Group:  ConsumerGroup,
		Idle:   minIdle,
		Start:  "-",
		End:    "+",
		Count:  100,
	}).Result()
	if err != nil {
		return nil, nil, err
	}
	if len(pending) == 0 {
		return nil, nil, nil
	}

	claimed, _, err := client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   BuildStream,
		Group:    ConsumerGroup,
		Consumer: consumerName,
		MinIdle:  minIdle,
		Start:    "0",
		Count:    100,
	}).Result()
	if err != nil {
		return nil, nil, err
	}

	var events []*types.BuildEvent
	var ids []string
	for _, msg := range claimed {
		payload, ok := msg.Values["payload"].(string)
		if !ok {
			continue
		}

		var event types.BuildEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}

		events = append(events, &event)
		ids = append(ids, msg.ID)
	}

	return events, ids, nil
}
