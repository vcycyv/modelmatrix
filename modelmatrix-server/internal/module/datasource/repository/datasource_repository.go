package repository

import (
	"modelmatrix-server/internal/module/datasource/domain"
	"modelmatrix-server/internal/module/datasource/model"

	"gorm.io/gorm"
)

// DatasourceRepositoryImpl implements DatasourceRepository
type DatasourceRepositoryImpl struct {
	db *gorm.DB
}

// NewDatasourceRepository creates a new datasource repository
func NewDatasourceRepository(db *gorm.DB) DatasourceRepository {
	return &DatasourceRepositoryImpl{db: db}
}

// Create creates a new datasource
func (r *DatasourceRepositoryImpl) Create(datasource *domain.Datasource) error {
	m := r.toModel(datasource)
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	datasource.ID = m.ID
	datasource.CreatedAt = m.CreatedAt
	datasource.UpdatedAt = m.UpdatedAt
	return nil
}

// Update updates an existing datasource
func (r *DatasourceRepositoryImpl) Update(datasource *domain.Datasource) error {
	m := r.toModel(datasource)
	if err := r.db.Model(&model.DatasourceModel{}).Where("id = ?", datasource.ID).Updates(map[string]interface{}{
		"name":        m.Name,
		"description": m.Description,
	}).Error; err != nil {
		return err
	}
	return nil
}

// Delete deletes a datasource
func (r *DatasourceRepositoryImpl) Delete(id string) error {
	return r.db.Delete(&model.DatasourceModel{}, "id = ?", id).Error
}

// GetByID retrieves a datasource by ID
func (r *DatasourceRepositoryImpl) GetByID(id string) (*domain.Datasource, error) {
	var m model.DatasourceModel
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrDatasourceNotFound
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

// GetByIDWithColumns retrieves a datasource with its columns
func (r *DatasourceRepositoryImpl) GetByIDWithColumns(id string) (*domain.Datasource, error) {
	var m model.DatasourceModel
	if err := r.db.Preload("Columns").Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrDatasourceNotFound
		}
		return nil, err
	}
	return r.toDomainWithColumns(&m), nil
}

// GetByName retrieves a datasource by name within a collection
func (r *DatasourceRepositoryImpl) GetByName(collectionID string, name string) (*domain.Datasource, error) {
	var m model.DatasourceModel
	if err := r.db.Where("collection_id = ? AND name = ?", collectionID, name).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

// List retrieves datasources with pagination and search
func (r *DatasourceRepositoryImpl) List(collectionID *string, offset, limit int, search string) ([]domain.Datasource, int64, error) {
	var models []model.DatasourceModel
	var total int64

	query := r.db.Model(&model.DatasourceModel{}).Preload("Collection")

	if collectionID != nil {
		query = query.Where("collection_id = ?", *collectionID)
	}

	if search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	datasources := make([]domain.Datasource, len(models))
	for i, m := range models {
		datasources[i] = *r.toDomain(&m)
	}

	return datasources, total, nil
}

// ListByCollection retrieves datasources in a collection
func (r *DatasourceRepositoryImpl) ListByCollection(collectionID string, offset, limit int) ([]domain.Datasource, int64, error) {
	return r.List(&collectionID, offset, limit, "")
}

// GetNamesInCollection retrieves all datasource names in a collection
func (r *DatasourceRepositoryImpl) GetNamesInCollection(collectionID string) ([]string, error) {
	var names []string
	if err := r.db.Model(&model.DatasourceModel{}).Where("collection_id = ?", collectionID).Pluck("name", &names).Error; err != nil {
		return nil, err
	}
	return names, nil
}

// UpdateFilePath updates the file path of a datasource
func (r *DatasourceRepositoryImpl) UpdateFilePath(id string, filePath string) error {
	return r.db.Model(&model.DatasourceModel{}).Where("id = ?", id).Update("file_path", filePath).Error
}

// toModel converts domain entity to GORM model
func (r *DatasourceRepositoryImpl) toModel(datasource *domain.Datasource) *model.DatasourceModel {
	m := &model.DatasourceModel{
		ID:           datasource.ID,
		CollectionID: datasource.CollectionID,
		Name:         datasource.Name,
		Description:  datasource.Description,
		Type:         string(datasource.Type),
		FilePath:     datasource.FilePath,
		CreatedBy:    datasource.CreatedBy,
		CreatedAt:    datasource.CreatedAt,
		UpdatedAt:    datasource.UpdatedAt,
	}

	if datasource.ConnectionConfig != nil {
		m.ConnectionConfig = model.JSONMap{
			"host":     datasource.ConnectionConfig.Host,
			"port":     datasource.ConnectionConfig.Port,
			"database": datasource.ConnectionConfig.Database,
			"username": datasource.ConnectionConfig.Username,
			"password": datasource.ConnectionConfig.Password,
			"schema":   datasource.ConnectionConfig.Schema,
			"table":    datasource.ConnectionConfig.Table,
			"sslmode":  datasource.ConnectionConfig.SSLMode,
		}
	}

	return m
}

// toDomain converts GORM model to domain entity
func (r *DatasourceRepositoryImpl) toDomain(m *model.DatasourceModel) *domain.Datasource {
	datasource := &domain.Datasource{
		ID:           m.ID,
		CollectionID: m.CollectionID,
		Name:         m.Name,
		Description:  m.Description,
		Type:         domain.DatasourceType(m.Type),
		FilePath:     m.FilePath,
		CreatedBy:    m.CreatedBy,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}

	if m.ConnectionConfig != nil {
		datasource.ConnectionConfig = &domain.ConnectionConfig{}
		if host, ok := m.ConnectionConfig["host"].(string); ok {
			datasource.ConnectionConfig.Host = host
		}
		if port, ok := m.ConnectionConfig["port"].(float64); ok {
			datasource.ConnectionConfig.Port = int(port)
		}
		if database, ok := m.ConnectionConfig["database"].(string); ok {
			datasource.ConnectionConfig.Database = database
		}
		if username, ok := m.ConnectionConfig["username"].(string); ok {
			datasource.ConnectionConfig.Username = username
		}
		if password, ok := m.ConnectionConfig["password"].(string); ok {
			datasource.ConnectionConfig.Password = password
		}
		if schema, ok := m.ConnectionConfig["schema"].(string); ok {
			datasource.ConnectionConfig.Schema = schema
		}
		if table, ok := m.ConnectionConfig["table"].(string); ok {
			datasource.ConnectionConfig.Table = table
		}
		if sslmode, ok := m.ConnectionConfig["sslmode"].(string); ok {
			datasource.ConnectionConfig.SSLMode = sslmode
		}
	}

	return datasource
}

// toDomainWithColumns converts GORM model with columns to domain entity
func (r *DatasourceRepositoryImpl) toDomainWithColumns(m *model.DatasourceModel) *domain.Datasource {
	datasource := r.toDomain(m)

	datasource.Columns = make([]domain.Column, len(m.Columns))
	for i, col := range m.Columns {
		datasource.Columns[i] = domain.Column{
			ID:           col.ID,
			DatasourceID: col.DatasourceID,
			Name:         col.Name,
			DataType:     col.DataType,
			Role:         domain.ColumnRole(col.Role),
			Description:  col.Description,
			CreatedAt:    col.CreatedAt,
			UpdatedAt:    col.UpdatedAt,
		}
	}

	return datasource
}
