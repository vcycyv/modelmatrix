package repository

import (
	"encoding/json"

	"modelmatrix_backend/internal/module/modelmanage/domain"
	"modelmatrix_backend/internal/module/modelmanage/model"

	"gorm.io/gorm"
)

// ModelRepositoryImpl implements ModelRepository
type ModelRepositoryImpl struct {
	db *gorm.DB
}

// NewModelRepository creates a new model repository
func NewModelRepository(db *gorm.DB) ModelRepository {
	return &ModelRepositoryImpl{db: db}
}

// Create creates a new model
func (r *ModelRepositoryImpl) Create(m *domain.Model) error {
	dbModel := r.toModel(m)
	if err := r.db.Create(dbModel).Error; err != nil {
		return err
	}
	m.ID = dbModel.ID
	m.CreatedAt = dbModel.CreatedAt
	m.UpdatedAt = dbModel.UpdatedAt
	return nil
}

// Update updates an existing model
func (r *ModelRepositoryImpl) Update(m *domain.Model) error {
	dbModel := r.toModel(m)
	if err := r.db.Model(&model.ModelModel{}).Where("id = ?", m.ID).Updates(dbModel).Error; err != nil {
		return err
	}
	return nil
}

// Delete soft-deletes a model
func (r *ModelRepositoryImpl) Delete(id string) error {
	return r.db.Delete(&model.ModelModel{}, "id = ?", id).Error
}

// GetByID retrieves a model by ID
func (r *ModelRepositoryImpl) GetByID(id string) (*domain.Model, error) {
	var dbModel model.ModelModel
	if err := r.db.Where("id = ?", id).First(&dbModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrModelNotFound
		}
		return nil, err
	}
	return r.toDomain(&dbModel), nil
}

// GetByName retrieves a model by name
func (r *ModelRepositoryImpl) GetByName(name string) (*domain.Model, error) {
	var dbModel model.ModelModel
	if err := r.db.Where("name = ?", name).First(&dbModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&dbModel), nil
}

// List retrieves models with pagination and search
func (r *ModelRepositoryImpl) List(offset, limit int, search, status string) ([]domain.Model, int64, error) {
	var models []model.ModelModel
	var total int64

	query := r.db.Model(&model.ModelModel{})

	if search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	result := make([]domain.Model, len(models))
	for i, m := range models {
		result[i] = *r.toDomain(&m)
	}

	return result, total, nil
}

// GetAllNames retrieves all model names
func (r *ModelRepositoryImpl) GetAllNames() ([]string, error) {
	var names []string
	if err := r.db.Model(&model.ModelModel{}).Pluck("name", &names).Error; err != nil {
		return nil, err
	}
	return names, nil
}

// UpdateStatus updates the status of a model
func (r *ModelRepositoryImpl) UpdateStatus(id string, status domain.ModelStatus) error {
	return r.db.Model(&model.ModelModel{}).Where("id = ?", id).Update("status", string(status)).Error
}

// CountVersions counts versions for a model
func (r *ModelRepositoryImpl) CountVersions(modelID string) (int64, error) {
	var count int64
	if err := r.db.Model(&model.ModelVersionModel{}).Where("model_id = ?", modelID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// toModel converts domain entity to GORM model
func (r *ModelRepositoryImpl) toModel(m *domain.Model) *model.ModelModel {
	dbModel := &model.ModelModel{
		ID:           m.ID,
		Name:         m.Name,
		Description:  m.Description,
		BuildID:      m.BuildID,
		Status:       string(m.Status),
		ArtifactPath: m.ArtifactPath,
		CreatedBy:    m.CreatedBy,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}

	if m.Metadata != nil {
		metadataJSON, _ := json.Marshal(m.Metadata)
		var metadataMap model.JSONMap
		json.Unmarshal(metadataJSON, &metadataMap)
		dbModel.Metadata = metadataMap
	}

	return dbModel
}

// toDomain converts GORM model to domain entity
func (r *ModelRepositoryImpl) toDomain(m *model.ModelModel) *domain.Model {
	domainModel := &domain.Model{
		ID:           m.ID,
		Name:         m.Name,
		Description:  m.Description,
		BuildID:      m.BuildID,
		Status:       domain.ModelStatus(m.Status),
		ArtifactPath: m.ArtifactPath,
		CreatedBy:    m.CreatedBy,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}

	if m.Metadata != nil {
		domainModel.Metadata = &domain.ModelMetadata{}
		metadataJSON, _ := json.Marshal(m.Metadata)
		json.Unmarshal(metadataJSON, domainModel.Metadata)
	}

	return domainModel
}

