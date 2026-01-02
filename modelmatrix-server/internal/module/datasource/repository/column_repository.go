package repository

import (
	"modelmatrix-server/internal/module/datasource/domain"
	"modelmatrix-server/internal/module/datasource/model"

	"gorm.io/gorm"
)

// ColumnRepositoryImpl implements ColumnRepository
type ColumnRepositoryImpl struct {
	db *gorm.DB
}

// NewColumnRepository creates a new column repository
func NewColumnRepository(db *gorm.DB) ColumnRepository {
	return &ColumnRepositoryImpl{db: db}
}

// Create creates a new column
func (r *ColumnRepositoryImpl) Create(column *domain.Column) error {
	m := r.toModel(column)
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	column.ID = m.ID
	column.CreatedAt = m.CreatedAt
	column.UpdatedAt = m.UpdatedAt
	return nil
}

// CreateBatch creates multiple columns
func (r *ColumnRepositoryImpl) CreateBatch(columns []domain.Column) error {
	if len(columns) == 0 {
		return nil
	}

	models := make([]model.ColumnModel, len(columns))
	for i, col := range columns {
		models[i] = *r.toModel(&col)
	}

	if err := r.db.Create(&models).Error; err != nil {
		return err
	}

	// Update IDs back to domain entities
	for i := range columns {
		columns[i].ID = models[i].ID
		columns[i].CreatedAt = models[i].CreatedAt
		columns[i].UpdatedAt = models[i].UpdatedAt
	}

	return nil
}

// Update updates an existing column
func (r *ColumnRepositoryImpl) Update(column *domain.Column) error {
	m := r.toModel(column)
	if err := r.db.Model(&model.ColumnModel{}).Where("id = ?", column.ID).Updates(m).Error; err != nil {
		return err
	}
	return nil
}

// Delete deletes a column
func (r *ColumnRepositoryImpl) Delete(id string) error {
	return r.db.Delete(&model.ColumnModel{}, "id = ?", id).Error
}

// GetByID retrieves a column by ID
func (r *ColumnRepositoryImpl) GetByID(id string) (*domain.Column, error) {
	var m model.ColumnModel
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrColumnNotFound
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

// GetByDatasourceID retrieves all columns for a datasource
func (r *ColumnRepositoryImpl) GetByDatasourceID(datasourceID string) ([]domain.Column, error) {
	var models []model.ColumnModel
	if err := r.db.Where("datasource_id = ?", datasourceID).Order("id ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	columns := make([]domain.Column, len(models))
	for i, m := range models {
		columns[i] = *r.toDomain(&m)
	}

	return columns, nil
}

// UpdateRole updates a column's role
func (r *ColumnRepositoryImpl) UpdateRole(id string, role domain.ColumnRole) error {
	return r.db.Model(&model.ColumnModel{}).Where("id = ?", id).Update("role", string(role)).Error
}

// DeleteByDatasourceID deletes all columns for a datasource
func (r *ColumnRepositoryImpl) DeleteByDatasourceID(datasourceID string) error {
	return r.db.Where("datasource_id = ?", datasourceID).Delete(&model.ColumnModel{}).Error
}

// toModel converts domain entity to GORM model
func (r *ColumnRepositoryImpl) toModel(column *domain.Column) *model.ColumnModel {
	return &model.ColumnModel{
		ID:           column.ID,
		DatasourceID: column.DatasourceID,
		Name:         column.Name,
		DataType:     column.DataType,
		Role:         string(column.Role),
		Description:  column.Description,
		CreatedAt:    column.CreatedAt,
		UpdatedAt:    column.UpdatedAt,
	}
}

// toDomain converts GORM model to domain entity
func (r *ColumnRepositoryImpl) toDomain(m *model.ColumnModel) *domain.Column {
	return &domain.Column{
		ID:           m.ID,
		DatasourceID: m.DatasourceID,
		Name:         m.Name,
		DataType:     m.DataType,
		Role:         domain.ColumnRole(m.Role),
		Description:  m.Description,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

