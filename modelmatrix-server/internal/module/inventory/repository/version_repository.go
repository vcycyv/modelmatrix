package repository

import (
	"modelmatrix-server/internal/module/inventory/domain"
)

// ModelVersionRepository defines the interface for model version snapshot data access
type ModelVersionRepository interface {
	Create(version *domain.ModelVersion) error
	ListByModelID(modelID string, limit, offset int) ([]domain.ModelVersion, int64, error)
	GetByID(versionID string) (*domain.ModelVersion, error)
	GetByModelIDAndNumber(modelID string, versionNumber int) (*domain.ModelVersion, error)
	GetNextVersionNumber(modelID string) (int, error)
}
