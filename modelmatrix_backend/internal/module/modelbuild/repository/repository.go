package repository

import (
	"modelmatrix_backend/internal/module/modelbuild/domain"
)

// BuildRepository defines the interface for model build data access
type BuildRepository interface {
	Create(build *domain.ModelBuild) error
	Update(build *domain.ModelBuild) error
	Delete(id string) error
	GetByID(id string) (*domain.ModelBuild, error)
	GetByName(name string) (*domain.ModelBuild, error)
	List(offset, limit int, search, status string) ([]domain.ModelBuild, int64, error)
	GetAllNames() ([]string, error)
	UpdateStatus(id string, status domain.BuildStatus, errorMsg string) error
}

