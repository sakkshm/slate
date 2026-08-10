package queue

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"slate-backend/pkg/types"
)

func newTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	return mr, client
}

func TestPublishAndClaimBuild(t *testing.T) {
	_, client := newTestRedis(t)
	ctx := context.Background()

	if err := ensureRedisStream(client); err != nil {
		t.Fatalf("ensure stream: %v", err)
	}

	event := types.BuildEvent{
		ProjectID: uuid.NewString(),
		BuildID:   uuid.NewString(),
		Slug:      "testsite",
		RepoURL:   "https://github.com/owner/repo.git",
		RootDir:   "frontend",
		OutDir:    "dist",
	}

	msgID, err := PublishBuildRequest(ctx, client, event)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if msgID == "" {
		t.Fatal("expected non-empty message ID")
	}

	got, claimID, err := ClaimBuildRequest(ctx, client, "test-consumer")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if got == nil {
		t.Fatal("expected a claimed event")
	}
	if got.BuildID != event.BuildID || got.Slug != event.Slug {
		t.Fatalf("claimed event mismatch: got %+v", got)
	}
	if claimID != msgID {
		t.Fatalf("claim ID mismatch: got %s want %s", claimID, msgID)
	}

	if err := AckBuild(ctx, client, claimID); err != nil {
		t.Fatalf("ack: %v", err)
	}
}

func TestClaimBuildNoPendingMessages(t *testing.T) {
	mr, client := newTestRedis(t)
	ctx := context.Background()

	if err := ensureRedisStream(client); err != nil {
		t.Fatalf("ensure stream: %v", err)
	}

	go func() {
		<-ctx.Done()
	}()
	_ = mr
	got, _, err := ClaimBuildRequest(ctx, client, "test-consumer")
	if err != nil {
		t.Fatalf("claim on empty stream should not error, got: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil event, got %+v", got)
	}
}

func TestDeployEntryLifecycle(t *testing.T) {
	_, client := newTestRedis(t)
	ctx := context.Background()

	entry := DeployEntry{
		ProjectID: uuid.NewString(),
		AssetHash: "abc123def456",
		UpdatedAt: "2026-08-08T06:44:10Z",
	}

	if err := PublishDeployment(ctx, client, "testsite", entry); err != nil {
		t.Fatalf("publish deployment: %v", err)
	}

	got, err := GetDeployment(ctx, client, "testsite")
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if got == nil {
		t.Fatal("expected deployment entry")
	}
	if got.ProjectID != entry.ProjectID || got.AssetHash != entry.AssetHash {
		t.Fatalf("deployment mismatch: got %+v", got)
	}

	missing, err := GetDeployment(ctx, client, "missing")
	if err != nil {
		t.Fatalf("get missing deployment: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil for missing deployment, got %+v", missing)
	}

	if err := DeleteDeployment(ctx, client, "testsite"); err != nil {
		t.Fatalf("delete deployment: %v", err)
	}

	after, err := GetDeployment(ctx, client, "testsite")
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if after != nil {
		t.Fatalf("expected nil after delete, got %+v", after)
	}
}

func TestLogLineStorage(t *testing.T) {
	_, client := newTestRedis(t)
	ctx := context.Background()

	buildID := uuid.NewString()
	lines := []string{"line one", "line two", "line three"}
	for _, l := range lines {
		if err := WriteLogLine(ctx, client, buildID, l); err != nil {
			t.Fatalf("write log line: %v", err)
		}
	}

	got, err := GetLogLines(ctx, client, buildID)
	if err != nil {
		t.Fatalf("get log lines: %v", err)
	}
	if len(got) != len(lines) {
		t.Fatalf("expected %d lines, got %d", len(lines), len(got))
	}
	for i, l := range lines {
		if got[i] != l {
			t.Fatalf("line %d mismatch: got %q want %q", i, got[i], l)
		}
	}

	missing, err := GetLogLines(ctx, client, uuid.NewString())
	if err != nil {
		t.Fatalf("get missing log lines: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("expected no lines for missing build, got %v", missing)
	}
}
