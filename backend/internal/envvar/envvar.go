package envvar

import (
	"context"
	"errors"
	"fmt"
	"slate-backend/pkg/types"
	"slate-backend/pkg/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func ListByProject(db *gorm.DB, projectID uuid.UUID, ctx context.Context) ([]types.ProjectEnvVar, error) {
	var vars []types.ProjectEnvVar

	result := db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("key ASC").
		Find(&vars)

	if result.Error != nil {
		return nil, result.Error
	}

	return vars, nil
}

func UpsertEnvVars(db *gorm.DB, projectID uuid.UUID, key, encryptedValue string, ctx context.Context) error {
	var existing types.ProjectEnvVar
	err := db.WithContext(ctx).
		Where("project_id = ? AND key = ?", projectID, key).
		First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.WithContext(ctx).Create(&types.ProjectEnvVar{
			ID:        uuid.New(),
			ProjectID: projectID,
			Key:       key,
			Value:     encryptedValue,
		}).Error
	}
	if err != nil {
		return err
	}

	return db.WithContext(ctx).
		Model(&existing).
		Update("value", encryptedValue).Error

}

func Delete(db *gorm.DB, projectID uuid.UUID, key string, ctx context.Context) error {
	result := db.WithContext(ctx).
		Where("project_id = ? AND key = ?", projectID, key).
		Delete(&types.ProjectEnvVar{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("env var not found")
	}
	return nil
}

func ResolveAll(db *gorm.DB, encryptionKey []byte, projectID uuid.UUID, ctx context.Context) ([]string, error) {
	vars, err := ListByProject(db, projectID, ctx)
	if err != nil {
		return nil, err
	}

	env := make([]string, 0, len(vars))
	for _, v := range vars {
		plain, err := utils.DecryptAESString(v.Value, encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt env var %q: %w", v.Key, err)
		}
		env = append(env, v.Key+"="+plain)
	}
	return env, nil
}
