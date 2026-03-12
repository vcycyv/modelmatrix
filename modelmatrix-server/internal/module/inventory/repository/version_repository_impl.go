package repository

import (
	"encoding/json"

	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/model"

	"gorm.io/gorm"
)

// ModelVersionRepositoryImpl implements ModelVersionRepository
type ModelVersionRepositoryImpl struct {
	db *gorm.DB
}

// NewModelVersionRepository creates a new model version repository
func NewModelVersionRepository(db *gorm.DB) ModelVersionRepository {
	return &ModelVersionRepositoryImpl{db: db}
}

// Create creates a new model version snapshot (with variables and files)
func (r *ModelVersionRepositoryImpl) Create(version *domain.ModelVersion) error {
	vModel := r.versionToModel(version)
	// Create version row first so we have ID for variables and files
	if err := r.db.Omit("Variables", "Files").Create(vModel).Error; err != nil {
		return err
	}
	version.ID = vModel.ID
	version.CreatedAt = vModel.CreatedAt

	for i := range vModel.Variables {
		vModel.Variables[i].ModelVersionID = vModel.ID
	}
	for i := range vModel.Files {
		vModel.Files[i].ModelVersionID = vModel.ID
	}
	if len(vModel.Variables) > 0 {
		if err := r.db.Create(&vModel.Variables).Error; err != nil {
			return err
		}
		for i := range version.Variables {
			version.Variables[i].ID = vModel.Variables[i].ID
			version.Variables[i].CreatedAt = vModel.Variables[i].CreatedAt
		}
	}
	if len(vModel.Files) > 0 {
		if err := r.db.Create(&vModel.Files).Error; err != nil {
			return err
		}
		for i := range version.Files {
			version.Files[i].ID = vModel.Files[i].ID
			version.Files[i].CreatedAt = vModel.Files[i].CreatedAt
		}
	}
	return nil
}

// ListByModelID returns versions for a model (newest first)
func (r *ModelVersionRepositoryImpl) ListByModelID(modelID string, limit, offset int) ([]domain.ModelVersion, int64, error) {
	var total int64
	if err := r.db.Model(&model.ModelVersionModel{}).Where("model_id = ?", modelID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.ModelVersionModel
	if err := r.db.Preload("Variables").Preload("Files").Where("model_id = ?", modelID).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	result := make([]domain.ModelVersion, len(list))
	for i := range list {
		result[i] = *r.versionToDomain(&list[i])
	}
	return result, total, nil
}

// GetByID returns a version by ID
func (r *ModelVersionRepositoryImpl) GetByID(versionID string) (*domain.ModelVersion, error) {
	var v model.ModelVersionModel
	if err := r.db.Preload("Variables").Preload("Files").Where("id = ?", versionID).First(&v).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrVersionNotFound
		}
		return nil, err
	}
	return r.versionToDomain(&v), nil
}

// GetByModelIDAndNumber returns a version by model ID and version number
func (r *ModelVersionRepositoryImpl) GetByModelIDAndNumber(modelID string, versionNumber int) (*domain.ModelVersion, error) {
	var v model.ModelVersionModel
	if err := r.db.Preload("Variables").Preload("Files").
		Where("model_id = ? AND version_number = ?", modelID, versionNumber).First(&v).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrVersionNotFound
		}
		return nil, err
	}
	return r.versionToDomain(&v), nil
}

// GetNextVersionNumber returns the next version number for a model (max(version_number)+1 or 1)
func (r *ModelVersionRepositoryImpl) GetNextVersionNumber(modelID string) (int, error) {
	var max *int
	err := r.db.Model(&model.ModelVersionModel{}).Where("model_id = ?", modelID).Select("MAX(version_number)").Scan(&max).Error
	if err != nil {
		return 0, err
	}
	if max == nil || *max == 0 {
		return 1, nil
	}
	return *max + 1, nil
}

func (r *ModelVersionRepositoryImpl) versionToModel(v *domain.ModelVersion) *model.ModelVersionModel {
	m := &model.ModelVersionModel{
		ID:            v.ID,
		ModelID:       v.ModelID,
		VersionNumber: v.VersionNumber,
		Name:          v.Name,
		Description:   v.Description,
		BuildID:       v.BuildID,
		DatasourceID:  v.DatasourceID,
		ProjectID:     v.ProjectID,
		FolderID:      v.FolderID,
		Algorithm:     v.Algorithm,
		ModelType:     v.ModelType,
		TargetColumn:  v.TargetColumn,
		Status:        string(v.Status),
		CreatedBy:     v.CreatedBy,
		CreatedAt:     v.CreatedAt,
	}
	if v.Metrics != nil {
		metricsJSON, _ := json.Marshal(v.Metrics)
		var metricsMap model.JSONMap
		json.Unmarshal(metricsJSON, &metricsMap)
		m.Metrics = metricsMap
	}
	m.Variables = make([]model.ModelVersionVariableModel, len(v.Variables))
	for i := range v.Variables {
		m.Variables[i] = *r.versionVariableToModel(&v.Variables[i], v.ID)
	}
	m.Files = make([]model.ModelVersionFileModel, len(v.Files))
	for i := range v.Files {
		m.Files[i] = *r.versionFileToModel(&v.Files[i], v.ID)
	}
	return m
}

func (r *ModelVersionRepositoryImpl) versionVariableToModel(v *domain.ModelVariable, versionID string) *model.ModelVersionVariableModel {
	m := &model.ModelVersionVariableModel{
		ModelVersionID: versionID,
		Name:           v.Name,
		DataType:       string(v.DataType),
		Role:           string(v.Role),
		Importance:     v.Importance,
		Ordinal:        v.Ordinal,
		CreatedAt:      v.CreatedAt,
	}
	if v.Statistics != nil {
		statsJSON, _ := json.Marshal(v.Statistics)
		var statsMap model.JSONMap
		json.Unmarshal(statsJSON, &statsMap)
		m.Statistics = statsMap
	}
	if v.EncodingInfo != nil {
		encJSON, _ := json.Marshal(v.EncodingInfo)
		var encMap model.JSONMap
		json.Unmarshal(encJSON, &encMap)
		m.EncodingInfo = encMap
	}
	return m
}

func (r *ModelVersionRepositoryImpl) versionFileToModel(f *domain.ModelFile, versionID string) *model.ModelVersionFileModel {
	return &model.ModelVersionFileModel{
		ModelVersionID: versionID,
		FileType:       string(f.FileType),
		FilePath:       f.FilePath,
		FileName:       f.FileName,
		FileSize:       f.FileSize,
		Checksum:       f.Checksum,
		Description:    f.Description,
		CreatedAt:      f.CreatedAt,
	}
}

func (r *ModelVersionRepositoryImpl) versionToDomain(m *model.ModelVersionModel) *domain.ModelVersion {
	v := &domain.ModelVersion{
		ID:            m.ID,
		ModelID:       m.ModelID,
		VersionNumber: m.VersionNumber,
		Name:          m.Name,
		Description:   m.Description,
		BuildID:       m.BuildID,
		DatasourceID:  m.DatasourceID,
		ProjectID:     m.ProjectID,
		FolderID:      m.FolderID,
		Algorithm:     m.Algorithm,
		ModelType:     m.ModelType,
		TargetColumn:  m.TargetColumn,
		Status:        domain.ModelStatus(m.Status),
		CreatedBy:     m.CreatedBy,
		CreatedAt:     m.CreatedAt,
	}
	if m.Metrics != nil {
		v.Metrics = &domain.ModelMetrics{}
		metricsJSON, _ := json.Marshal(m.Metrics)
		json.Unmarshal(metricsJSON, v.Metrics)
	}
	v.Variables = make([]domain.ModelVariable, len(m.Variables))
	for i := range m.Variables {
		v.Variables[i] = *r.versionVariableToDomain(&m.Variables[i], m.ModelID)
	}
	v.Files = make([]domain.ModelFile, len(m.Files))
	for i := range m.Files {
		v.Files[i] = *r.versionFileToDomain(&m.Files[i], m.ModelID)
	}
	return v
}

func (r *ModelVersionRepositoryImpl) versionVariableToDomain(m *model.ModelVersionVariableModel, modelID string) *domain.ModelVariable {
	v := &domain.ModelVariable{
		ID:         m.ID,
		ModelID:    modelID,
		Name:       m.Name,
		DataType:   domain.VariableDataType(m.DataType),
		Role:       domain.VariableRole(m.Role),
		Importance: m.Importance,
		Ordinal:    m.Ordinal,
		CreatedAt:  m.CreatedAt,
	}
	if m.Statistics != nil {
		v.Statistics = make(map[string]interface{})
		statsJSON, _ := json.Marshal(m.Statistics)
		json.Unmarshal(statsJSON, &v.Statistics)
	}
	if m.EncodingInfo != nil {
		v.EncodingInfo = make(map[string]interface{})
		encJSON, _ := json.Marshal(m.EncodingInfo)
		json.Unmarshal(encJSON, &v.EncodingInfo)
	}
	return v
}

func (r *ModelVersionRepositoryImpl) versionFileToDomain(m *model.ModelVersionFileModel, modelID string) *domain.ModelFile {
	return &domain.ModelFile{
		ID:          m.ID,
		ModelID:     modelID,
		FileType:    domain.FileType(m.FileType),
		FilePath:    m.FilePath,
		FileName:    m.FileName,
		FileSize:    m.FileSize,
		Checksum:    m.Checksum,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
	}
}
