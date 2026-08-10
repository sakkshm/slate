package prune

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"slate-backend/pkg/types"
)

type recordingStore struct {
	deleted []string
}

func (s *recordingStore) Upload(context.Context, string, io.Reader, int64) error  { return nil }
func (s *recordingStore) Download(context.Context, string) (io.ReadCloser, error) { return nil, nil }
func (s *recordingStore) Exists(context.Context, string) (bool, error)            { return true, nil }
func (s *recordingStore) List(context.Context, string) ([]string, error)          { return nil, nil }
func (s *recordingStore) Delete(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return nil
}

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	if os.Getenv("SLATE_TEST_DB") == "" {
		t.Skip("SLATE_TEST_DB not set; skipping database integration test")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:secretpass@localhost:5432/postgres?sslmode=disable"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return db
}

func TestPruneArtifacts(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	ownerID := int64(43210805)
	projectWithNewer := uuid.New()
	projectSingle := uuid.New()
	projectTwoOld := uuid.New()

	for _, pid := range []uuid.UUID{projectWithNewer, projectSingle, projectTwoOld} {
		p := types.Project{ID: pid, OwnerID: ownerID, Name: "prune-test", Slug: "prune-test-" + pid.String(), RepoID: 999999, RepoURL: "https://github.com/x/y.git", RepoName: "x/y", ProdBranch: "main"}
		if err := db.Create(&p).Error; err != nil {
			t.Fatalf("create project: %v", err)
		}
		t.Cleanup(func() { db.Where("project_id = ?", pid).Delete(&types.Build{}); db.Delete(&types.Project{}, pid) })
	}

	old := time.Now().AddDate(0, 0, -30)
	newer := time.Now().AddDate(0, 0, -1)

	seed := func(pid uuid.UUID, status types.BuildEvents, createdAt time.Time, asset string) {
		b := types.Build{ID: uuid.New(), ProjectID: pid, Status: status, AssetLocation: asset, CreatedAt: createdAt}
		if err := db.Create(&b).Error; err != nil {
			t.Fatalf("create build: %v", err)
		}
	}

	// projectWithNewer: old build (prunable) + newer ready (protected)
	seed(projectWithNewer, types.StatusReady, old, "hash-old-a")
	seed(projectWithNewer, types.StatusReady, newer, "hash-new-b")

	// projectSingle: only build, old but latest -> protected
	seed(projectSingle, types.StatusReady, old, "hash-old-c")

	// projectTwoOld: two old ready builds, newer of the two is protected
	seed(projectTwoOld, types.StatusReady, old.Add(-time.Hour), "hash-old-d")
	seed(projectTwoOld, types.StatusReady, old, "hash-old-e")

	store := &recordingStore{}
	cutoff := time.Now().AddDate(0, 0, -15)
	deleted, err := PruneArtifacts(ctx, db, store, cutoff)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}

	want := map[string]bool{
		artifactKey(projectWithNewer, "hash-old-a"): true,
		artifactKey(projectTwoOld, "hash-old-d"):    true,
	}
	if deleted != len(want) {
		t.Fatalf("deleted %d artifacts, want %d (%v)", deleted, len(want), store.deleted)
	}

	for _, key := range store.deleted {
		if !want[key] {
			t.Fatalf("unexpected deletion of %q", key)
		}
		delete(want, key)
	}
	for key := range want {
		t.Fatalf("expected deletion of %q, not deleted", key)
	}
}
