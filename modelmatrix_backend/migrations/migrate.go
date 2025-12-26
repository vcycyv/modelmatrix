package migrations

import (
	dsModel "modelmatrix_backend/internal/module/datasource/model"
	mbModel "modelmatrix_backend/internal/module/modelbuild/model"
	mmModel "modelmatrix_backend/internal/module/modelmanage/model"

	"gorm.io/gorm"
)

// Migrate runs all database migrations
func Migrate(db *gorm.DB) error {
	// Auto-migrate all models
	return db.AutoMigrate(
		// Datasource module models
		&dsModel.CollectionModel{},
		&dsModel.DatasourceModel{},
		&dsModel.ColumnModel{},

		// Model Build module models
		&mbModel.ModelBuildModel{},

		// Model Manage module models
		&mmModel.ModelModel{},
		&mmModel.ModelVersionModel{},
	)
}

// CreateIndexes creates additional indexes not covered by GORM tags
func CreateIndexes(db *gorm.DB) error {
	// Create composite index for datasource uniqueness per collection
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_datasource_name_collection 
		ON datasources (collection_id, name) 
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}

	// Create composite index for column uniqueness per datasource
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_column_unique_per_datasource 
		ON datasource_columns (datasource_id, name) 
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}

	// Create composite index for version uniqueness per model
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_version_unique_per_model 
		ON model_versions (model_id, version) 
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}

	return nil
}

// DropAll drops all tables (use with caution!)
func DropAll(db *gorm.DB) error {
	return db.Migrator().DropTable(
		&mmModel.ModelVersionModel{},
		&mmModel.ModelModel{},
		&mbModel.ModelBuildModel{},
		&dsModel.ColumnModel{},
		&dsModel.DatasourceModel{},
		&dsModel.CollectionModel{},
	)
}

