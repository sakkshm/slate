package prune

import (
	"context"
	"fmt"
	"time"

	"slate-backend/internal/storage"
	"slate-backend/pkg/types"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PruneArtifacts deletes MinIO build artifacts older than cutoff, skipping the
// latest READY build of each project (the currently served deployment).
//
// Deploy keys in Redis always point at the latest READY build, which is
// protected here, so they never reference a pruned artifact and need no
// separate cleanup. Returns the number of artifacts deleted.
func PruneArtifacts(ctx context.Context, database *gorm.DB, store storage.Store, cutoff time.Time) (int, error) {
	latest, err := latestReadyPerProject(ctx, database)
	if err != nil {
		return 0, err
	}

	var old []types.Build
	if err := database.WithContext(ctx).
		Where("status = ? AND asset_location <> '' AND created_at < ?", string(types.StatusReady), cutoff).
		Find(&old).Error; err != nil {
		return 0, err
	}

	deleted := 0
	for _, b := range old {
		key := artifactKey(b.ProjectID, b.AssetLocation)
		if latest[key] {
			continue
		}
		if err := store.Delete(ctx, key); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

// latestReadyPerProject returns the content-addressable artifact keys of the
// newest READY build per project.
func latestReadyPerProject(ctx context.Context, database *gorm.DB) (map[string]bool, error) {
	var rows []struct {
		ProjectID     uuid.UUID
		AssetLocation string
	}
	err := database.WithContext(ctx).Raw(`
		SELECT DISTINCT ON (project_id) project_id, asset_location
		FROM builds
		WHERE status = ? AND asset_location <> ''
		ORDER BY project_id, created_at DESC`,
		string(types.StatusReady),
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	keys := make(map[string]bool, len(rows))
	for _, r := range rows {
		keys[artifactKey(r.ProjectID, r.AssetLocation)] = true
	}
	return keys, nil
}

func artifactKey(projectID uuid.UUID, assetLocation string) string {
	return fmt.Sprintf("projects/%s/builds/%s.tar.gz", projectID, assetLocation)
}
