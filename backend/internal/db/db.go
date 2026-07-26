package db

import (
	"slate-backend/pkg/types"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func New(dbURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&types.User{},
		&types.Project{},
		&types.Build{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}