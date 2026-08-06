package types

import (
	"time"

	"github.com/google/uuid"
)

type ProjectEnvVar struct {
	ID        uuid.UUID `gorm:"primaryKey"`
	ProjectID uuid.UUID `gorm:"not null;uniqueIndex:idx_project_key"`
	Key       string    `gorm:"not null;uniqueIndex:idx_project_key"`
	Value     string    `gorm:"not null;type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
