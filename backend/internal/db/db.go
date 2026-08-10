package db

import (
	"log/slog"
	"slate-backend/pkg/types"
	"slate-backend/pkg/utils"
	"strings"

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
		&types.ProjectEnvVar{},
	)
	if err != nil {
		return nil, err
	}

	dropStaleColumns(db)
	sanitizeProjectSlugs(db)

	return db, nil
}

// dropStaleColumns removes columns and constraints left behind by retired
// schema versions. AutoMigrate only adds columns, never drops them.
func dropStaleColumns(db *gorm.DB) {
	db.Exec(`ALTER TABLE projects DROP COLUMN IF EXISTS active_build_id`)
}

// sanitizeProjectSlugs rewrites slugs the gateway's host matcher would reject
// (e.g. repos with uppercase letters or dots), so projects created before the
// slug rules were tightened keep working.
func sanitizeProjectSlugs(db *gorm.DB) {
	var projects []types.Project
	if err := db.Find(&projects).Error; err != nil {
		slog.Error("failed to load projects for slug sanitization", "error", err)
		return
	}

	for i := range projects {
		p := &projects[i]
		if utils.ValidSlug(p.Slug) {
			continue
		}

		var owner types.User
		if err := db.Where("id = ?", p.OwnerID).First(&owner).Error; err != nil || owner.GithubUsername == "" {
			slog.Error("failed to resolve owner for slug sanitization", "project_id", p.ID, "error", err)
			continue
		}

		repoName := p.RepoName
		if idx := strings.LastIndex(repoName, "/"); idx != -1 {
			repoName = repoName[idx+1:]
		}

		base := utils.Slugify(repoName)
		newSlug := ""
		for attempt := 0; attempt < 10 && newSlug == ""; attempt++ {
			suffix, err := utils.GenerateRandomString(4)
			if err != nil {
				continue
			}
			candidate := utils.ProjectSlug(owner.GithubUsername, base, suffix)
			var count int64
			db.Model(&types.Project{}).
				Where("owner_id = ? AND slug = ? AND id <> ?", p.OwnerID, candidate, p.ID).
				Count(&count)
			if count == 0 {
				newSlug = candidate
			}
		}
		if newSlug == "" {
			slog.Error("failed to generate a unique slug", "project_id", p.ID)
			continue
		}

		slog.Info("sanitized project slug", "project_id", p.ID, "old_slug", p.Slug, "new_slug", newSlug)
		if err := db.Model(&types.Project{}).Where("id = ?", p.ID).Update("slug", newSlug).Error; err != nil {
			slog.Error("failed to update project slug", "project_id", p.ID, "error", err)
		}
	}
}