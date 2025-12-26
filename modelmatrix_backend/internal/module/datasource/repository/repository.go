package repository

import (
	"modelmatrix_backend/internal/module/datasource/domain"
)

// CollectionRepository defines the interface for collection data access
type CollectionRepository interface {
	Create(collection *domain.Collection) error
	Update(collection *domain.Collection) error
	Delete(id string) error
	GetByID(id string) (*domain.Collection, error)
	GetByName(name string) (*domain.Collection, error)
	List(offset, limit int, search string) ([]domain.Collection, int64, error)
	GetAllNames() ([]string, error)
	CountDatasources(collectionID string) (int64, error)
}

// DatasourceRepository defines the interface for datasource data access
type DatasourceRepository interface {
	Create(datasource *domain.Datasource) error
	Update(datasource *domain.Datasource) error
	Delete(id string) error
	GetByID(id string) (*domain.Datasource, error)
	GetByIDWithColumns(id string) (*domain.Datasource, error)
	GetByName(collectionID string, name string) (*domain.Datasource, error)
	List(collectionID *string, offset, limit int, search string) ([]domain.Datasource, int64, error)
	ListByCollection(collectionID string, offset, limit int) ([]domain.Datasource, int64, error)
	GetNamesInCollection(collectionID string) ([]string, error)
	UpdateFilePath(id string, filePath string) error
}

// ColumnRepository defines the interface for column data access
type ColumnRepository interface {
	Create(column *domain.Column) error
	CreateBatch(columns []domain.Column) error
	Update(column *domain.Column) error
	Delete(id string) error
	GetByID(id string) (*domain.Column, error)
	GetByDatasourceID(datasourceID string) ([]domain.Column, error)
	UpdateRole(id string, role domain.ColumnRole) error
	DeleteByDatasourceID(datasourceID string) error
}

