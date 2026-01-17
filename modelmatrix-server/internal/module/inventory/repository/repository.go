package repository

import (
	"modelmatrix-server/internal/module/inventory/domain"
)

// ModelRepository defines the interface for model data access
type ModelRepository interface {
	// Model CRUD
	Create(model *domain.Model) error
	Update(model *domain.Model) error
	Delete(id string) error
	GetByID(id string) (*domain.Model, error)
	GetByIDWithRelations(id string) (*domain.Model, error) // Includes variables and files
	GetByName(name string) (*domain.Model, error)
	GetByBuildID(buildID string) (*domain.Model, error)
	List(offset, limit int, search, status string) ([]domain.Model, int64, error)
	UpdateStatus(id string, status domain.ModelStatus) error

	// Folder/Project queries
	GetIDsByFolderID(folderID string) ([]string, error)
	GetIDsByProjectID(projectID string) ([]string, error)

	// Variable operations
	CreateVariable(variable *domain.ModelVariable) error
	CreateVariables(variables []domain.ModelVariable) error
	GetVariablesByModelID(modelID string) ([]domain.ModelVariable, error)
	DeleteVariablesByModelID(modelID string) error

	// File operations
	CreateFile(file *domain.ModelFile) error
	CreateFiles(files []domain.ModelFile) error
	GetFilesByModelID(modelID string) ([]domain.ModelFile, error)
	GetFileByModelIDAndType(modelID string, fileType domain.FileType) (*domain.ModelFile, error)
	DeleteFilesByModelID(modelID string) error
}
