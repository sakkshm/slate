package build

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"slate-backend/pkg/types"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("gorm open: %v", err)
	}
	return gormDB, mock
}

func TestUpdateBuildStatusIfQueued(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	ctx := context.Background()
	buildID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "builds" SET "status"=$1 WHERE id = $2 AND status = $3`)).
		WithArgs(string(types.StatusBuilding), buildID.String(), string(types.StatusQueued)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	claimed, err := UpdateBuildStatusIfQueued(gormDB, buildID, ctx)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !claimed {
		t.Fatal("expected build to be claimed")
	}
}

func TestUpdateBuildStatusIfQueuedNotClaimed(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	ctx := context.Background()
	buildID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "builds" SET "status"=$1 WHERE id = $2 AND status = $3`)).
		WithArgs(string(types.StatusBuilding), buildID.String(), string(types.StatusQueued)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	claimed, err := UpdateBuildStatusIfQueued(gormDB, buildID, ctx)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if claimed {
		t.Fatal("expected build NOT to be claimed when no rows affected")
	}
}

func TestUpdateBuildStatus(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	ctx := context.Background()
	buildID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "builds" SET "status"=$1 WHERE id = $2`)).
		WithArgs(string(types.StatusReady), buildID.String()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := UpdateBuildStatus(gormDB, buildID, types.StatusReady, ctx); err != nil {
		t.Fatalf("update: %v", err)
	}
}

func TestUpdateBuildAssetLocation(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	ctx := context.Background()
	buildID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "builds" SET "asset_location"=$1 WHERE id = $2`)).
		WithArgs("cafebabecafebabe", buildID.String()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := UpdateBuildAssetLocation(gormDB, buildID, "cafebabecafebabe", ctx); err != nil {
		t.Fatalf("update: %v", err)
	}
}
