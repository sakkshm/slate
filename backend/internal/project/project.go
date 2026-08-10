package project

import (
	"context"
	"errors"
	"slate-backend/pkg/types"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrProjectNotFound = errors.New("project not found or not owned by user")

func CreateProject(database *gorm.DB, project *types.Project, ctx context.Context) error {
	result := database.WithContext(ctx).Create(project)
	return result.Error
}

func GetProjectByID(projectID uuid.UUID, database *gorm.DB, ctx context.Context) (*types.Project, error) {
	var project types.Project
	result := database.WithContext(ctx).First(&project, projectID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &project, nil
}

func GetProjectBySlug(slug string, database *gorm.DB, ctx context.Context) (*types.Project, error) {
	var project types.Project
	result := database.WithContext(ctx).
	Where("slug = ?", slug).
	First(&project)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &project, nil
}

func GetProjectByRepoID(database *gorm.DB, repoID int64, ctx context.Context) (*types.Project, error) {
	var project types.Project
	result := database.WithContext(ctx).Where("repo_id = ?", repoID).First(&project)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &project, nil
}

func GetProjectsByOwner(ownerID int64, database *gorm.DB, ctx context.Context) ([]types.Project, error) {
	var projects []types.Project
	result := database.WithContext(ctx).Where("owner_id = ?", ownerID).Order("created_at DESC").Find(&projects)
	if result.Error != nil {
		return nil, result.Error
	}
	return projects, nil
}

func GetLatestReadyBuild(projectID uuid.UUID, database *gorm.DB, ctx context.Context) (*types.Build, error) {
	var build types.Build
	result := database.WithContext(ctx).
	Where("project_id = ? AND status = ?", projectID, string(types.StatusReady)).
	Order("created_at DESC").
	First(&build)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &build, nil
}

func UpdateProject(database *gorm.DB, projectID uuid.UUID, ownerID int64, updates map[string]interface{}, ctx context.Context) error {
	result := database.WithContext(ctx).
		Model(&types.Project{}).
		Where("id = ? AND owner_id = ?", projectID, ownerID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func DeleteProject(projectID uuid.UUID, ownerID int64, database *gorm.DB, ctx context.Context) error {
	return database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", projectID).Delete(&types.Build{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", projectID).Delete(&types.ProjectEnvVar{}).Error; err != nil {
			return err
		}
		result := tx.Where("id = ? AND owner_id = ?", projectID, ownerID).Delete(&types.Project{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrProjectNotFound
		}
		return nil
	})
}
