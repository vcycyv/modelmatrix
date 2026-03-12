package repository

import (
	"encoding/json"

	"modelmatrix-server/internal/module/build/domain"
	"modelmatrix-server/internal/module/build/model"

	"gorm.io/gorm"
)

// BuildRepositoryImpl implements BuildRepository
type BuildRepositoryImpl struct {
	db *gorm.DB
}

// NewBuildRepository creates a new build repository
func NewBuildRepository(db *gorm.DB) BuildRepository {
	return &BuildRepositoryImpl{db: db}
}

// Create creates a new model build
func (r *BuildRepositoryImpl) Create(build *domain.ModelBuild) error {
	m := r.toModel(build)
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	build.ID = m.ID
	build.CreatedAt = m.CreatedAt
	build.UpdatedAt = m.UpdatedAt
	return nil
}

// Update updates an existing model build
func (r *BuildRepositoryImpl) Update(build *domain.ModelBuild) error {
	m := r.toModel(build)
	// Use Save to update all fields, including zero values
	// This ensures Status, CompletedAt, and Metrics are always updated
	if err := r.db.Save(m).Error; err != nil {
		return err
	}
	return nil
}

// Delete deletes a model build
func (r *BuildRepositoryImpl) Delete(id string) error {
	return r.db.Delete(&model.ModelBuildModel{}, "id = ?", id).Error
}

// GetByID retrieves a model build by ID
func (r *BuildRepositoryImpl) GetByID(id string) (*domain.ModelBuild, error) {
	var m model.ModelBuildModel
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrBuildNotFound
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

// GetByName retrieves a model build by name
func (r *BuildRepositoryImpl) GetByName(name string) (*domain.ModelBuild, error) {
	var m model.ModelBuildModel
	if err := r.db.Where("name = ?", name).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

// List retrieves model builds with pagination and search
func (r *BuildRepositoryImpl) List(offset, limit int, search, status string) ([]domain.ModelBuild, int64, error) {
	var models []model.ModelBuildModel
	var total int64

	query := r.db.Model(&model.ModelBuildModel{})

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

	builds := make([]domain.ModelBuild, len(models))
	for i, m := range models {
		builds[i] = *r.toDomain(&m)
	}

	return builds, total, nil
}

// GetAllNames retrieves all build names
func (r *BuildRepositoryImpl) GetAllNames() ([]string, error) {
	var names []string
	if err := r.db.Model(&model.ModelBuildModel{}).Pluck("name", &names).Error; err != nil {
		return nil, err
	}
	return names, nil
}

// UpdateStatus updates the status of a model build
func (r *BuildRepositoryImpl) UpdateStatus(id string, status domain.BuildStatus, errorMsg string) error {
	updates := map[string]interface{}{
		"status": string(status),
	}
	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}
	return r.db.Model(&model.ModelBuildModel{}).Where("id = ?", id).Updates(updates).Error
}

// toModel converts domain entity to GORM model
func (r *BuildRepositoryImpl) toModel(build *domain.ModelBuild) *model.ModelBuildModel {
	m := &model.ModelBuildModel{
		ID:             build.ID,
		Name:           build.Name,
		Description:    build.Description,
		DatasourceID:   build.DatasourceID,
		ProjectID:      build.ProjectID,
		FolderID:       build.FolderID,
		SourceModelID:  build.SourceModelID,
		ModelType:      string(build.ModelType),
		Algorithm:      build.Algorithm,
		Status:         string(build.Status),
		ErrorMessage:   build.ErrorMessage,
		StartedAt:      build.StartedAt,
		CompletedAt:    build.CompletedAt,
		CreatedBy:      build.CreatedBy,
		CreatedAt:      build.CreatedAt,
		UpdatedAt:      build.UpdatedAt,
	}

	// Convert parameters
	paramsJSON, _ := json.Marshal(build.Parameters)
	var paramsMap model.JSONMap
	json.Unmarshal(paramsJSON, &paramsMap)
	m.Parameters = paramsMap

	// Convert metrics
	if build.Metrics != nil {
		metricsJSON, _ := json.Marshal(build.Metrics)
		var metricsMap model.JSONMap
		json.Unmarshal(metricsJSON, &metricsMap)
		m.Metrics = metricsMap
	}

	return m
}

// toDomain converts GORM model to domain entity
func (r *BuildRepositoryImpl) toDomain(m *model.ModelBuildModel) *domain.ModelBuild {
	build := &domain.ModelBuild{
		ID:             m.ID,
		Name:           m.Name,
		Description:    m.Description,
		DatasourceID:   m.DatasourceID,
		ProjectID:      m.ProjectID,
		FolderID:       m.FolderID,
		SourceModelID:  m.SourceModelID,
		ModelType:      domain.ModelType(m.ModelType),
		Algorithm:      m.Algorithm,
		Status:         domain.BuildStatus(m.Status),
		ErrorMessage:   m.ErrorMessage,
		StartedAt:      m.StartedAt,
		CompletedAt:    m.CompletedAt,
		CreatedBy:      m.CreatedBy,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}

	// Convert parameters
	if m.Parameters != nil {
		paramsJSON, _ := json.Marshal(m.Parameters)
		json.Unmarshal(paramsJSON, &build.Parameters)
	}

	// Convert metrics
	if m.Metrics != nil {
		build.Metrics = &domain.BuildMetrics{}
		metricsJSON, _ := json.Marshal(m.Metrics)
		json.Unmarshal(metricsJSON, build.Metrics)
	}

	return build
}

// GetIDsByFolderID returns build IDs directly in a folder
func (r *BuildRepositoryImpl) GetIDsByFolderID(folderID string) ([]string, error) {
	var ids []string
	err := r.db.Model(&model.ModelBuildModel{}).Where("folder_id = ?", folderID).Pluck("id", &ids).Error
	return ids, err
}

// GetIDsByProjectID returns build IDs in a project
func (r *BuildRepositoryImpl) GetIDsByProjectID(projectID string) ([]string, error) {
	var ids []string
	err := r.db.Model(&model.ModelBuildModel{}).Where("project_id = ?", projectID).Pluck("id", &ids).Error
	return ids, err
}
