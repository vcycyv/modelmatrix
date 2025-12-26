package repository

import (
	"modelmatrix_backend/internal/module/modelmanage/domain"
	"modelmatrix_backend/internal/module/modelmanage/model"

	"gorm.io/gorm"
)

// VersionRepositoryImpl implements VersionRepository
type VersionRepositoryImpl struct {
	db *gorm.DB
}

// NewVersionRepository creates a new version repository
func NewVersionRepository(db *gorm.DB) VersionRepository {
	return &VersionRepositoryImpl{db: db}
}

// Create creates a new model version
func (r *VersionRepositoryImpl) Create(v *domain.ModelVersion) error {
	dbModel := r.toModel(v)
	if err := r.db.Create(dbModel).Error; err != nil {
		return err
	}
	v.ID = dbModel.ID
	v.CreatedAt = dbModel.CreatedAt
	v.UpdatedAt = dbModel.UpdatedAt
	return nil
}

// Update updates an existing model version
func (r *VersionRepositoryImpl) Update(v *domain.ModelVersion) error {
	dbModel := r.toModel(v)
	if err := r.db.Model(&model.ModelVersionModel{}).Where("id = ?", v.ID).Updates(dbModel).Error; err != nil {
		return err
	}
	return nil
}

// Delete soft-deletes a model version
func (r *VersionRepositoryImpl) Delete(id string) error {
	return r.db.Delete(&model.ModelVersionModel{}, "id = ?", id).Error
}

// GetByID retrieves a model version by ID
func (r *VersionRepositoryImpl) GetByID(id string) (*domain.ModelVersion, error) {
	var dbModel model.ModelVersionModel
	if err := r.db.Where("id = ?", id).First(&dbModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrModelVersionNotFound
		}
		return nil, err
	}
	return r.toDomain(&dbModel), nil
}

// GetByModelIDAndVersion retrieves a version by model ID and version string
func (r *VersionRepositoryImpl) GetByModelIDAndVersion(modelID string, version string) (*domain.ModelVersion, error) {
	var dbModel model.ModelVersionModel
	if err := r.db.Where("model_id = ? AND version = ?", modelID, version).First(&dbModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&dbModel), nil
}

// ListByModelID retrieves all versions for a model
func (r *VersionRepositoryImpl) ListByModelID(modelID string) ([]domain.ModelVersion, error) {
	var models []model.ModelVersionModel
	if err := r.db.Where("model_id = ?", modelID).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}

	versions := make([]domain.ModelVersion, len(models))
	for i, m := range models {
		versions[i] = *r.toDomain(&m)
	}

	return versions, nil
}

// GetVersionStrings retrieves all version strings for a model
func (r *VersionRepositoryImpl) GetVersionStrings(modelID string) ([]string, error) {
	var versions []string
	if err := r.db.Model(&model.ModelVersionModel{}).Where("model_id = ?", modelID).Pluck("version", &versions).Error; err != nil {
		return nil, err
	}
	return versions, nil
}

// UpdateStatus updates the status of a model version
func (r *VersionRepositoryImpl) UpdateStatus(id string, status domain.ModelStatus) error {
	return r.db.Model(&model.ModelVersionModel{}).Where("id = ?", id).Update("status", string(status)).Error
}

// toModel converts domain entity to GORM model
func (r *VersionRepositoryImpl) toModel(v *domain.ModelVersion) *model.ModelVersionModel {
	dbModel := &model.ModelVersionModel{
		ID:           v.ID,
		ModelID:      v.ModelID,
		Version:      v.Version,
		BuildID:      v.BuildID,
		Status:       string(v.Status),
		ArtifactPath: v.ArtifactPath,
		Notes:        v.Notes,
		CreatedBy:    v.CreatedBy,
		CreatedAt:    v.CreatedAt,
		UpdatedAt:    v.UpdatedAt,
	}

	if v.Metrics != nil {
		metricsMap := make(model.JSONMap)
		for k, val := range v.Metrics {
			metricsMap[k] = val
		}
		dbModel.Metrics = metricsMap
	}

	return dbModel
}

// toDomain converts GORM model to domain entity
func (r *VersionRepositoryImpl) toDomain(m *model.ModelVersionModel) *domain.ModelVersion {
	version := &domain.ModelVersion{
		ID:           m.ID,
		ModelID:      m.ModelID,
		Version:      m.Version,
		BuildID:      m.BuildID,
		Status:       domain.ModelStatus(m.Status),
		ArtifactPath: m.ArtifactPath,
		Notes:        m.Notes,
		CreatedBy:    m.CreatedBy,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}

	if m.Metrics != nil {
		version.Metrics = make(map[string]float64)
		for k, v := range m.Metrics {
			if fv, ok := v.(float64); ok {
				version.Metrics[k] = fv
			}
		}
	}

	return version
}

