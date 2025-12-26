package repository

import (
	"modelmatrix_backend/internal/module/modelmanage/domain"
)

// ModelRepository defines the interface for model data access
type ModelRepository interface {
	Create(model *domain.Model) error
	Update(model *domain.Model) error
	Delete(id string) error
	GetByID(id string) (*domain.Model, error)
	GetByName(name string) (*domain.Model, error)
	List(offset, limit int, search, status string) ([]domain.Model, int64, error)
	GetAllNames() ([]string, error)
	UpdateStatus(id string, status domain.ModelStatus) error
	CountVersions(modelID string) (int64, error)
}

// VersionRepository defines the interface for model version data access
type VersionRepository interface {
	Create(version *domain.ModelVersion) error
	Update(version *domain.ModelVersion) error
	Delete(id string) error
	GetByID(id string) (*domain.ModelVersion, error)
	GetByModelIDAndVersion(modelID string, version string) (*domain.ModelVersion, error)
	ListByModelID(modelID string) ([]domain.ModelVersion, error)
	GetVersionStrings(modelID string) ([]string, error)
	UpdateStatus(id string, status domain.ModelStatus) error
}

