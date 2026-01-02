package migrations

import (
	"modelmatrix-server/internal/infrastructure/folderservice"
	dsModel "modelmatrix-server/internal/module/datasource/model"
	mbModel "modelmatrix-server/internal/module/build/model"
	mmModel "modelmatrix-server/internal/module/inventory/model"

	"gorm.io/gorm"
)

// Migrate runs all database migrations
func Migrate(db *gorm.DB) error {
	// Get folder service models
	folderModels := folderservice.GetModels()

	// Build migration list
	migrationModels := []interface{}{
		// Datasource module models
		&dsModel.CollectionModel{},
		&dsModel.DatasourceModel{},
		&dsModel.ColumnModel{},

		// Model Build module models
		&mbModel.ModelBuildModel{},

		// Model Manage module models
		&mmModel.ModelModel{},
		&mmModel.ModelVariableModel{},
		&mmModel.ModelFileModel{},
	}

	// Add folder service models
	migrationModels = append(migrationModels, folderModels...)

	// Auto-migrate all models
	return db.AutoMigrate(migrationModels...)
}

// CreateIndexes creates additional indexes not covered by GORM tags
func CreateIndexes(db *gorm.DB) error {
	// Create composite index for datasource uniqueness per collection
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_datasource_name_collection 
		ON datasources (collection_id, name)
	`).Error; err != nil {
		return err
	}

	// Create composite index for column uniqueness per datasource
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_column_unique_per_datasource 
		ON datasource_columns (datasource_id, name)
	`).Error; err != nil {
		return err
	}

	// Create composite index for variable uniqueness per model
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_variable_unique_per_model 
		ON model_variables (model_id, name)
	`).Error; err != nil {
		return err
	}

	// Create composite index for file type uniqueness per model
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_file_type_unique_per_model 
		ON model_files (model_id, file_type)
	`).Error; err != nil {
		return err
	}

	// Create composite index for folder uniqueness per parent
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_folder_name_unique_per_parent 
		ON folders (parent_id, name) WHERE parent_id IS NOT NULL
	`).Error; err != nil {
		return err
	}

	// Create unique index for root folders
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_folder_name_unique_root 
		ON folders (name) WHERE parent_id IS NULL
	`).Error; err != nil {
		return err
	}

	// Create index for folder path prefix queries (for descendant lookups)
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_folder_path_prefix 
		ON folders USING btree (path varchar_pattern_ops)
	`).Error; err != nil {
		return err
	}

	// Create composite index for project uniqueness per folder
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_project_name_unique_per_folder 
		ON projects (folder_id, name) WHERE folder_id IS NOT NULL
	`).Error; err != nil {
		return err
	}

	// Create unique index for root projects (projects not in any folder)
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_project_name_unique_root 
		ON projects (name) WHERE folder_id IS NULL
	`).Error; err != nil {
		return err
	}

	return nil
}

// DropAll drops all tables (use with caution!)
func DropAll(db *gorm.DB) error {
	return db.Migrator().DropTable(
		&mmModel.ModelFileModel{},
		&mmModel.ModelVariableModel{},
		&mmModel.ModelModel{},
		&mbModel.ModelBuildModel{},
		&dsModel.ColumnModel{},
		&dsModel.DatasourceModel{},
		&dsModel.CollectionModel{},
	)
}
