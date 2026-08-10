package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"slate-backend/pkg/config"
)

// Integration tests against a live MinIO. Set SLATE_TEST_MINIO=1 to enable;
// otherwise they skip. Uses a unique prefix per run and cleans up after itself.
func newTestStore(t *testing.T) Store {
	t.Helper()
	if os.Getenv("SLATE_TEST_MINIO") == "" {
		t.Skip("SLATE_TEST_MINIO not set; skipping MinIO integration test")
	}

	cfg := &config.Config{
		MinIOEndpoint:  getenv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey: getenv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey: getenv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:    getenv("MINIO_BUCKET", "slate-assets"),
	}

	store, err := NewMinIOStore(cfg)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return store
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestMinIOUploadDownload(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := "test/integration-" + strings.ReplaceAll(t.Name(), "/", "-") + "/artifact.tar.gz"

	content := []byte("fake artifact bytes")
	if err := store.Upload(ctx, key, bytes.NewReader(content), int64(len(content))); err != nil {
		t.Fatalf("upload: %v", err)
	}
	t.Cleanup(func() { store.Delete(ctx, key) })

	rc, err := store.Download(ctx, key)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read download: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("downloaded bytes differ: got %d bytes want %d", len(got), len(content))
	}

	exists, err := store.Exists(ctx, key)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !exists {
		t.Fatal("object should exist after upload")
	}

	missing, err := store.Exists(ctx, "test/does-not-exist.tar.gz")
	if err != nil {
		t.Fatalf("exists missing: %v", err)
	}
	if missing {
		t.Fatal("missing object reported as existing")
	}
}

func TestMinIOListAndDelete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	prefix := "test/list-" + strings.ReplaceAll(t.Name(), "/", "-") + "/"

	keys := []string{
		prefix + "a/build.tar.gz",
		prefix + "b/build.tar.gz",
		prefix + "b/other.bin",
	}
	for _, k := range keys {
		if err := store.Upload(ctx, k, strings.NewReader("x"), 1); err != nil {
			t.Fatalf("upload %s: %v", k, err)
		}
	}
	t.Cleanup(func() {
		if listed, err := store.List(ctx, prefix); err == nil {
			for _, k := range listed {
				store.Delete(ctx, k)
			}
		}
	})

	listed, err := store.List(ctx, prefix)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != len(keys) {
		t.Fatalf("expected %d objects, got %d: %v", len(keys), len(listed), listed)
	}
	for _, want := range keys {
		found := false
		for _, k := range listed {
			if k == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected key %q in list", want)
		}
	}

	if err := store.Delete(ctx, keys[0]); err != nil {
		t.Fatalf("delete: %v", err)
	}
	exists, err := store.Exists(ctx, keys[0])
	if err != nil {
		t.Fatalf("exists after delete: %v", err)
	}
	if exists {
		t.Fatal("deleted object should not exist")
	}
}
