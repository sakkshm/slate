package build

import (
	"context"
	"errors"
	"slate-backend/pkg/types"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func CreateBuild(database *gorm.DB, build *types.Build, ctx context.Context) error {
	result := database.WithContext(ctx).Create(build)
	return result.Error
}

func GetBuildByProject(database *gorm.DB, projectID uuid.UUID, limit, offset int, ctx context.Context) ([]types.Build, int64, error) {
	var builds []types.Build
	var total int64

	database.WithContext(ctx).Model(&types.Build{}).
		Where("project_id = ?", projectID).
		Count(&total)

	result := database.WithContext(ctx).Model(&types.Build{}).
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&builds)

	if result.Error != nil {
		return nil, 0, result.Error
	}

	return builds, total, nil
}

func GetBuildByID(database *gorm.DB, buildID uuid.UUID, ctx context.Context) (*types.Build, error) {
	var build types.Build

	result := database.WithContext(ctx).First(&build, buildID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &build, nil

}

func UpdateBuildStatus(database *gorm.DB, buildID uuid.UUID, status types.BuildEvents, ctx context.Context) error {
	result := database.WithContext(ctx).
		Model(&types.Build{}).
		Where("id = ?", buildID).
		Update("status", status)
	return result.Error
}

func UpdateBuildDuration(database *gorm.DB, buildID uuid.UUID, durationMs int64, ctx context.Context) error {
	result := database.WithContext(ctx).
		Model(&types.Build{}).
		Where("id = ?", buildID).
		Update("duration", durationMs)
	return result.Error
}

func UpdateBuildAssetLocation(database *gorm.DB, buildID uuid.UUID, assetLocation string, ctx context.Context) error {
	result := database.WithContext(ctx).
		Model(&types.Build{}).
		Where("id = ?", buildID).
		Update("asset_location", assetLocation)
	return result.Error
}
