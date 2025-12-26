package repository

import (
	"modelmatrix_backend/internal/module/datasource/domain"
	"modelmatrix_backend/internal/module/datasource/model"

	"gorm.io/gorm"
)

// CollectionRepositoryImpl implements CollectionRepository
type CollectionRepositoryImpl struct {
	db *gorm.DB
}

// NewCollectionRepository creates a new collection repository
func NewCollectionRepository(db *gorm.DB) CollectionRepository {
	return &CollectionRepositoryImpl{db: db}
}

// Create creates a new collection
func (r *CollectionRepositoryImpl) Create(collection *domain.Collection) error {
	m := r.toModel(collection)
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	collection.ID = m.ID
	collection.CreatedAt = m.CreatedAt
	collection.UpdatedAt = m.UpdatedAt
	return nil
}

// Update updates an existing collection
func (r *CollectionRepositoryImpl) Update(collection *domain.Collection) error {
	m := r.toModel(collection)
	if err := r.db.Model(&model.CollectionModel{}).Where("id = ?", collection.ID).Updates(m).Error; err != nil {
		return err
	}
	return nil
}

// Delete soft-deletes a collection
func (r *CollectionRepositoryImpl) Delete(id string) error {
	return r.db.Delete(&model.CollectionModel{}, "id = ?", id).Error
}

// GetByID retrieves a collection by ID
func (r *CollectionRepositoryImpl) GetByID(id string) (*domain.Collection, error) {
	var m model.CollectionModel
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrCollectionNotFound
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

// GetByName retrieves a collection by name
func (r *CollectionRepositoryImpl) GetByName(name string) (*domain.Collection, error) {
	var m model.CollectionModel
	if err := r.db.Where("name = ?", name).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

// List retrieves collections with pagination and search
func (r *CollectionRepositoryImpl) List(offset, limit int, search string) ([]domain.Collection, int64, error) {
	var models []model.CollectionModel
	var total int64

	query := r.db.Model(&model.CollectionModel{})

	if search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	collections := make([]domain.Collection, len(models))
	for i, m := range models {
		collections[i] = *r.toDomain(&m)
	}

	return collections, total, nil
}

// GetAllNames retrieves all collection names
func (r *CollectionRepositoryImpl) GetAllNames() ([]string, error) {
	var names []string
	if err := r.db.Model(&model.CollectionModel{}).Pluck("name", &names).Error; err != nil {
		return nil, err
	}
	return names, nil
}

// CountDatasources counts datasources in a collection
func (r *CollectionRepositoryImpl) CountDatasources(collectionID string) (int64, error) {
	var count int64
	if err := r.db.Model(&model.DatasourceModel{}).Where("collection_id = ?", collectionID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// toModel converts domain entity to GORM model
func (r *CollectionRepositoryImpl) toModel(collection *domain.Collection) *model.CollectionModel {
	return &model.CollectionModel{
		ID:          collection.ID,
		Name:        collection.Name,
		Description: collection.Description,
		CreatedBy:   collection.CreatedBy,
		CreatedAt:   collection.CreatedAt,
		UpdatedAt:   collection.UpdatedAt,
	}
}

// toDomain converts GORM model to domain entity
func (r *CollectionRepositoryImpl) toDomain(m *model.CollectionModel) *domain.Collection {
	return &domain.Collection{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		CreatedBy:   m.CreatedBy,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

