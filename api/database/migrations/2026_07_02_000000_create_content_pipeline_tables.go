package migrations

import (
	_ "embed"

	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

const createContentPipelineTablesName = "2026_07_02_000000_create_content_pipeline_tables"

//go:embed content_pipeline.sql
var contentPipelineSQL string

func init() {
	register(createContentPipelineTablesName, &createContentPipelineTables{})
}

// createContentPipelineTables creates the AGI01 trend and article pipeline tables.
type createContentPipelineTables struct {
	migration.BaseMigration
}

// Up applies the migration.
func (m *createContentPipelineTables) Up(db *gorm.DB) error {
	if db.Name() != "postgres" {
		return nil
	}

	return db.Exec(contentPipelineSQL).Error
}

// Down reverts the migration.
func (m *createContentPipelineTables) Down(db *gorm.DB) error {
	if db.Name() != "postgres" {
		return nil
	}

	return db.Exec(`
		drop table if exists publication_packages;
		drop table if exists media_assets;
		drop table if exists article_reviews;
		drop table if exists article_artifacts;
		drop table if exists article_runs;
		drop table if exists article_projects;
		drop table if exists trend_evaluations;
		drop table if exists trend_items;
		drop table if exists content_sources;
		drop table if exists skill_snapshots;
		drop table if exists skill_repositories;
		drop type if exists job_status;
		drop type if exists artifact_kind;
		drop type if exists article_status;
		drop type if exists trend_status;
		drop type if exists content_source_status;
	`).Error
}
