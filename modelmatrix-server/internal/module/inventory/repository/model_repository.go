package repository

import (
	"encoding/json"

	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/model"

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
	if err := r.db.Save(dbModel).Error; err != nil {
		return err
	}
	return nil
}

// Delete deletes a model (cascades to variables and files)
func (r *ModelRepositoryImpl) Delete(id string) error {
	return r.db.Delete(&model.ModelModel{}, "id = ?", id).Error
}

// GetByID retrieves a model by ID (without relations)
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

// GetByIDWithRelations retrieves a model by ID with variables and files
func (r *ModelRepositoryImpl) GetByIDWithRelations(id string) (*domain.Model, error) {
	var dbModel model.ModelModel
	if err := r.db.Preload("Variables").Preload("Files").Where("id = ?", id).First(&dbModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrModelNotFound
		}
		return nil, err
	}
	return r.toDomainWithRelations(&dbModel), nil
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

// GetByBuildID retrieves a model by build ID
func (r *ModelRepositoryImpl) GetByBuildID(buildID string) (*domain.Model, error) {
	var dbModel model.ModelModel
	if err := r.db.Where("build_id = ?", buildID).First(&dbModel).Error; err != nil {
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

// UpdateStatus updates the status of a model
func (r *ModelRepositoryImpl) UpdateStatus(id string, status domain.ModelStatus) error {
	return r.db.Model(&model.ModelModel{}).Where("id = ?", id).Update("status", string(status)).Error
}

// CreateVariable creates a new model variable
func (r *ModelRepositoryImpl) CreateVariable(variable *domain.ModelVariable) error {
	dbVar := r.variableToModel(variable)
	if err := r.db.Create(dbVar).Error; err != nil {
		return err
	}
	variable.ID = dbVar.ID
	variable.CreatedAt = dbVar.CreatedAt
	return nil
}

// CreateVariables creates multiple variables in batch
func (r *ModelRepositoryImpl) CreateVariables(variables []domain.ModelVariable) error {
	if len(variables) == 0 {
		return nil
	}
	dbVars := make([]model.ModelVariableModel, len(variables))
	for i, v := range variables {
		dbVars[i] = *r.variableToModel(&v)
	}
	return r.db.Create(&dbVars).Error
}

// GetVariablesByModelID retrieves all variables for a model
func (r *ModelRepositoryImpl) GetVariablesByModelID(modelID string) ([]domain.ModelVariable, error) {
	var dbVars []model.ModelVariableModel
	if err := r.db.Where("model_id = ?", modelID).Order("ordinal ASC").Find(&dbVars).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ModelVariable, len(dbVars))
	for i, v := range dbVars {
		result[i] = *r.variableToDomain(&v)
	}
	return result, nil
}

// DeleteVariablesByModelID deletes all variables for a model
func (r *ModelRepositoryImpl) DeleteVariablesByModelID(modelID string) error {
	return r.db.Delete(&model.ModelVariableModel{}, "model_id = ?", modelID).Error
}

// CreateFile creates a new model file
func (r *ModelRepositoryImpl) CreateFile(file *domain.ModelFile) error {
	dbFile := r.fileToModel(file)
	if err := r.db.Create(dbFile).Error; err != nil {
		return err
	}
	file.ID = dbFile.ID
	file.CreatedAt = dbFile.CreatedAt
	return nil
}

// CreateFiles creates multiple files in batch
func (r *ModelRepositoryImpl) CreateFiles(files []domain.ModelFile) error {
	if len(files) == 0 {
		return nil
	}
	dbFiles := make([]model.ModelFileModel, len(files))
	for i, f := range files {
		dbFiles[i] = *r.fileToModel(&f)
	}
	return r.db.Create(&dbFiles).Error
}

// GetFilesByModelID retrieves all files for a model
func (r *ModelRepositoryImpl) GetFilesByModelID(modelID string) ([]domain.ModelFile, error) {
	var dbFiles []model.ModelFileModel
	if err := r.db.Where("model_id = ?", modelID).Find(&dbFiles).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ModelFile, len(dbFiles))
	for i, f := range dbFiles {
		result[i] = *r.fileToDomain(&f)
	}
	return result, nil
}

// GetFileByModelIDAndType retrieves a specific file type for a model
func (r *ModelRepositoryImpl) GetFileByModelIDAndType(modelID string, fileType domain.FileType) (*domain.ModelFile, error) {
	var dbFile model.ModelFileModel
	if err := r.db.Where("model_id = ? AND file_type = ?", modelID, string(fileType)).First(&dbFile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrFileNotFound
		}
		return nil, err
	}
	return r.fileToDomain(&dbFile), nil
}

// DeleteFilesByModelID deletes all files for a model
func (r *ModelRepositoryImpl) DeleteFilesByModelID(modelID string) error {
	return r.db.Delete(&model.ModelFileModel{}, "model_id = ?", modelID).Error
}

// toModel converts domain entity to GORM model
func (r *ModelRepositoryImpl) toModel(m *domain.Model) *model.ModelModel {
	dbModel := &model.ModelModel{
		ID:           m.ID,
		Name:         m.Name,
		Description:  m.Description,
		BuildID:      m.BuildID,
		DatasourceID: m.DatasourceID,
		Algorithm:    m.Algorithm,
		ModelType:    m.ModelType,
		TargetColumn: m.TargetColumn,
		Status:       string(m.Status),
		Version:      m.Version,
		CreatedBy:    m.CreatedBy,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}

	if m.Metrics != nil {
		metricsJSON, _ := json.Marshal(m.Metrics)
		var metricsMap model.JSONMap
		json.Unmarshal(metricsJSON, &metricsMap)
		dbModel.Metrics = metricsMap
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
		DatasourceID: m.DatasourceID,
		Algorithm:    m.Algorithm,
		ModelType:    m.ModelType,
		TargetColumn: m.TargetColumn,
		Status:       domain.ModelStatus(m.Status),
		Version:      m.Version,
		CreatedBy:    m.CreatedBy,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}

	if m.Metrics != nil {
		domainModel.Metrics = &domain.ModelMetrics{}
		metricsJSON, _ := json.Marshal(m.Metrics)
		json.Unmarshal(metricsJSON, domainModel.Metrics)
	}

	return domainModel
}

// toDomainWithRelations converts GORM model to domain entity with variables and files
func (r *ModelRepositoryImpl) toDomainWithRelations(m *model.ModelModel) *domain.Model {
	domainModel := r.toDomain(m)

	// Convert variables
	domainModel.Variables = make([]domain.ModelVariable, len(m.Variables))
	for i, v := range m.Variables {
		domainModel.Variables[i] = *r.variableToDomain(&v)
	}

	// Convert files
	domainModel.Files = make([]domain.ModelFile, len(m.Files))
	for i, f := range m.Files {
		domainModel.Files[i] = *r.fileToDomain(&f)
	}

	return domainModel
}

// variableToModel converts domain variable to GORM model
func (r *ModelRepositoryImpl) variableToModel(v *domain.ModelVariable) *model.ModelVariableModel {
	dbVar := &model.ModelVariableModel{
		ID:         v.ID,
		ModelID:    v.ModelID,
		Name:       v.Name,
		DataType:   string(v.DataType),
		Role:       string(v.Role),
		Importance: v.Importance,
		Ordinal:    v.Ordinal,
		CreatedAt:  v.CreatedAt,
	}

	if v.Statistics != nil {
		statsJSON, _ := json.Marshal(v.Statistics)
		var statsMap model.JSONMap
		json.Unmarshal(statsJSON, &statsMap)
		dbVar.Statistics = statsMap
	}

	if v.EncodingInfo != nil {
		encJSON, _ := json.Marshal(v.EncodingInfo)
		var encMap model.JSONMap
		json.Unmarshal(encJSON, &encMap)
		dbVar.EncodingInfo = encMap
	}

	return dbVar
}

// variableToDomain converts GORM variable to domain entity
func (r *ModelRepositoryImpl) variableToDomain(v *model.ModelVariableModel) *domain.ModelVariable {
	domainVar := &domain.ModelVariable{
		ID:         v.ID,
		ModelID:    v.ModelID,
		Name:       v.Name,
		DataType:   domain.VariableDataType(v.DataType),
		Role:       domain.VariableRole(v.Role),
		Importance: v.Importance,
		Ordinal:    v.Ordinal,
		CreatedAt:  v.CreatedAt,
	}

	if v.Statistics != nil {
		domainVar.Statistics = make(map[string]interface{})
		statsJSON, _ := json.Marshal(v.Statistics)
		json.Unmarshal(statsJSON, &domainVar.Statistics)
	}

	if v.EncodingInfo != nil {
		domainVar.EncodingInfo = make(map[string]interface{})
		encJSON, _ := json.Marshal(v.EncodingInfo)
		json.Unmarshal(encJSON, &domainVar.EncodingInfo)
	}

	return domainVar
}

// fileToModel converts domain file to GORM model
func (r *ModelRepositoryImpl) fileToModel(f *domain.ModelFile) *model.ModelFileModel {
	return &model.ModelFileModel{
		ID:          f.ID,
		ModelID:     f.ModelID,
		FileType:    string(f.FileType),
		FilePath:    f.FilePath,
		FileName:    f.FileName,
		FileSize:    f.FileSize,
		Checksum:    f.Checksum,
		Description: f.Description,
		CreatedAt:   f.CreatedAt,
	}
}

// fileToDomain converts GORM file to domain entity
func (r *ModelRepositoryImpl) fileToDomain(f *model.ModelFileModel) *domain.ModelFile {
	return &domain.ModelFile{
		ID:          f.ID,
		ModelID:     f.ModelID,
		FileType:    domain.FileType(f.FileType),
		FilePath:    f.FilePath,
		FileName:    f.FileName,
		FileSize:    f.FileSize,
		Checksum:    f.Checksum,
		Description: f.Description,
		CreatedAt:   f.CreatedAt,
	}
}
