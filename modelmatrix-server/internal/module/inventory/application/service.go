package application

import (
	"fmt"
	"path/filepath"

	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"
	"modelmatrix-server/internal/module/inventory/repository"
	"modelmatrix-server/pkg/logger"
)

// ModelService defines the interface for model application service
type ModelService interface {
	// CRUD operations
	Create(req *dto.CreateModelRequest, createdBy string) (*dto.ModelResponse, error)
	Update(id string, req *dto.UpdateModelRequest) (*dto.ModelResponse, error)
	Delete(id string) error
	GetByID(id string) (*dto.ModelDetailResponse, error)
	List(params *dto.ListParams) (*dto.ModelListResponse, error)

	// Lifecycle operations
	Activate(id string) (*dto.ModelResponse, error)
	Deactivate(id string) (*dto.ModelResponse, error)

	// Create from build (called when build completes)
	CreateFromBuild(req *dto.CreateModelFromBuildRequest) (*dto.ModelResponse, error)
}

// ModelServiceImpl implements ModelService
type ModelServiceImpl struct {
	modelRepo     repository.ModelRepository
	domainService *domain.Service
}

// NewModelService creates a new model service
func NewModelService(
	modelRepo repository.ModelRepository,
	domainService *domain.Service,
) ModelService {
	return &ModelServiceImpl{
		modelRepo:     modelRepo,
		domainService: domainService,
	}
}

// Create creates a new model
func (s *ModelServiceImpl) Create(req *dto.CreateModelRequest, createdBy string) (*dto.ModelResponse, error) {
	// Check name uniqueness
	existing, _ := s.modelRepo.GetByName(req.Name)
	if existing != nil {
		return nil, domain.ErrModelNameExists
	}

	// Convert DTO to domain entity
	model := &domain.Model{
		Name:         req.Name,
		Description:  req.Description,
		BuildID:      req.BuildID,
		DatasourceID: req.DatasourceID,
		Algorithm:    req.Algorithm,
		ModelType:    req.ModelType,
		TargetColumn: req.TargetColumn,
		Status:       domain.ModelStatusDraft,
		Version:      1,
		CreatedBy:    createdBy,
	}

	// Convert metrics if provided
	if req.Metrics != nil {
		model.Metrics = convertMetrics(req.Metrics)
	}

	// Validate using domain service
	if err := s.domainService.ValidateModel(model); err != nil {
		return nil, err
	}

	// Create model
	if err := s.modelRepo.Create(model); err != nil {
		logger.Error("Failed to create model: %v", err)
		return nil, err
	}

	// Create variables
	if len(req.Variables) > 0 {
		variables := make([]domain.ModelVariable, len(req.Variables))
		for i, v := range req.Variables {
			variables[i] = domain.ModelVariable{
				ModelID:      model.ID,
				Name:         v.Name,
				DataType:     domain.VariableDataType(v.DataType),
				Role:         domain.VariableRole(v.Role),
				Importance:   v.Importance,
				Statistics:   v.Statistics,
				EncodingInfo: v.EncodingInfo,
				Ordinal:      v.Ordinal,
			}
		}
		if err := s.modelRepo.CreateVariables(variables); err != nil {
			logger.Error("Failed to create variables: %v", err)
			// Don't fail the whole operation, but log the error
		}
	}

	// Create files
	if len(req.Files) > 0 {
		files := make([]domain.ModelFile, len(req.Files))
		for i, f := range req.Files {
			files[i] = domain.ModelFile{
				ModelID:     model.ID,
				FileType:    domain.FileType(f.FileType),
				FilePath:    f.FilePath,
				FileName:    f.FileName,
				FileSize:    f.FileSize,
				Checksum:    f.Checksum,
				Description: f.Description,
			}
		}
		if err := s.modelRepo.CreateFiles(files); err != nil {
			logger.Error("Failed to create files: %v", err)
			// Don't fail the whole operation, but log the error
		}
	}

	logger.Audit(createdBy, "create", "model", model.ID, "success", nil)

	return toModelResponse(model), nil
}

// CreateFromBuild creates a model from a completed build
func (s *ModelServiceImpl) CreateFromBuild(req *dto.CreateModelFromBuildRequest) (*dto.ModelResponse, error) {
	// Check if model already exists for this build
	existing, _ := s.modelRepo.GetByBuildID(req.BuildID)
	if existing != nil {
		logger.Warn("Model already exists for build %s", req.BuildID)
		return toModelResponse(existing), nil
	}

	// Check name uniqueness, generate unique name if needed
	name := req.Name
	existing, _ = s.modelRepo.GetByName(name)
	if existing != nil {
		name = fmt.Sprintf("%s_%s", req.Name, req.BuildID[:8])
	}

	// Create domain entity
	model := &domain.Model{
		Name:         name,
		Description:  req.Description,
		BuildID:      req.BuildID,
		DatasourceID: req.DatasourceID,
		Algorithm:    req.Algorithm,
		ModelType:    req.ModelType,
		TargetColumn: req.TargetColumn,
		Status:       domain.ModelStatusDraft,
		Version:      1,
		CreatedBy:    req.CreatedBy,
	}

	// Convert metrics
	if req.Metrics != nil {
		model.Metrics = convertMetrics(req.Metrics)
	}

	// Create model
	if err := s.modelRepo.Create(model); err != nil {
		logger.Error("Failed to create model from build: %v", err)
		return nil, err
	}

	// Create variables from input columns
	variables := make([]domain.ModelVariable, 0, len(req.InputColumns)+1)

	// Add input variables
	for i, colName := range req.InputColumns {
		variables = append(variables, domain.ModelVariable{
			ModelID:  model.ID,
			Name:     colName,
			DataType: domain.VariableDataTypeNumeric, // Default, can be refined later
			Role:     domain.VariableRoleInput,
			Ordinal:  i,
		})
	}

	// Add target variable
	variables = append(variables, domain.ModelVariable{
		ModelID:  model.ID,
		Name:     req.TargetColumn,
		DataType: domain.VariableDataTypeNumeric, // Default
		Role:     domain.VariableRoleTarget,
		Ordinal:  len(req.InputColumns),
	})

	if err := s.modelRepo.CreateVariables(variables); err != nil {
		logger.Error("Failed to create variables from build: %v", err)
	}

	// Create model file
	if req.ModelFilePath != "" {
		file := domain.ModelFile{
			ModelID:     model.ID,
			FileType:    domain.FileTypeModel,
			FilePath:    req.ModelFilePath,
			FileName:    filepath.Base(req.ModelFilePath),
			Description: fmt.Sprintf("Trained %s model", req.Algorithm),
		}
		if err := s.modelRepo.CreateFile(&file); err != nil {
			logger.Error("Failed to create model file: %v", err)
		}
	}

	logger.Audit(req.CreatedBy, "create_from_build", "model", model.ID, "success", nil)
	logger.Info("Created model %s from build %s", model.ID, req.BuildID)

	return toModelResponse(model), nil
}

// Update updates an existing model
func (s *ModelServiceImpl) Update(id string, req *dto.UpdateModelRequest) (*dto.ModelResponse, error) {
	model, err := s.modelRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		// Check name uniqueness
		existing, _ := s.modelRepo.GetByName(*req.Name)
		if existing != nil && existing.ID != id {
			return nil, domain.ErrModelNameExists
		}
		model.Name = *req.Name
	}
	if req.Description != nil {
		model.Description = *req.Description
	}

	// Validate using domain service
	if err := s.domainService.ValidateModel(model); err != nil {
		return nil, err
	}

	// Update via repository
	if err := s.modelRepo.Update(model); err != nil {
		logger.Error("Failed to update model: %v", err)
		return nil, err
	}

	return toModelResponse(model), nil
}

// Delete deletes a model
func (s *ModelServiceImpl) Delete(id string) error {
	model, err := s.modelRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Validate using domain service
	if err := s.domainService.CanDeleteModel(model); err != nil {
		return err
	}

	if err := s.modelRepo.Delete(id); err != nil {
		logger.Error("Failed to delete model: %v", err)
		return err
	}

	return nil
}

// GetByID retrieves a model by ID with variables and files
func (s *ModelServiceImpl) GetByID(id string) (*dto.ModelDetailResponse, error) {
	model, err := s.modelRepo.GetByIDWithRelations(id)
	if err != nil {
		return nil, err
	}

	response := &dto.ModelDetailResponse{
		ModelResponse: *toModelResponse(model),
		Variables:     toVariableResponseList(model.Variables),
		Files:         toFileResponseList(model.Files),
	}

	return response, nil
}

// List retrieves models with pagination
func (s *ModelServiceImpl) List(params *dto.ListParams) (*dto.ModelListResponse, error) {
	params.SetDefaults()

	models, total, err := s.modelRepo.List(params.Offset(), params.PageSize, params.Search, params.Status)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.ModelResponse, len(models))
	for i, model := range models {
		responses[i] = *toModelResponse(&model)
	}

	return &dto.ModelListResponse{
		Models: responses,
		Total:  total,
	}, nil
}

// Activate activates a model
func (s *ModelServiceImpl) Activate(id string) (*dto.ModelResponse, error) {
	model, err := s.modelRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Validate using domain service
	if err := s.domainService.CanActivateModel(model); err != nil {
		return nil, err
	}

	// Activate the model
	if err := model.Activate(); err != nil {
		return nil, err
	}

	// Update via repository
	if err := s.modelRepo.UpdateStatus(id, model.Status); err != nil {
		logger.Error("Failed to activate model: %v", err)
		return nil, err
	}

	return toModelResponse(model), nil
}

// Deactivate deactivates a model
func (s *ModelServiceImpl) Deactivate(id string) (*dto.ModelResponse, error) {
	model, err := s.modelRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Validate using domain service
	if err := s.domainService.CanDeactivateModel(model); err != nil {
		return nil, err
	}

	// Deactivate the model
	if err := model.Deactivate(); err != nil {
		return nil, err
	}

	// Update via repository
	if err := s.modelRepo.UpdateStatus(id, model.Status); err != nil {
		logger.Error("Failed to deactivate model: %v", err)
		return nil, err
	}

	return toModelResponse(model), nil
}

// convertMetrics converts map to domain metrics struct
func convertMetrics(metricsMap map[string]interface{}) *domain.ModelMetrics {
	if metricsMap == nil {
		return nil
	}

	metrics := &domain.ModelMetrics{}

	if v, ok := metricsMap["accuracy"].(float64); ok {
		metrics.Accuracy = v
	}
	if v, ok := metricsMap["precision"].(float64); ok {
		metrics.Precision = v
	}
	if v, ok := metricsMap["recall"].(float64); ok {
		metrics.Recall = v
	}
	if v, ok := metricsMap["f1_score"].(float64); ok {
		metrics.F1Score = v
	}
	if v, ok := metricsMap["mse"].(float64); ok {
		metrics.MSE = v
	}
	if v, ok := metricsMap["rmse"].(float64); ok {
		metrics.RMSE = v
	}
	if v, ok := metricsMap["mae"].(float64); ok {
		metrics.MAE = v
	}
	if v, ok := metricsMap["r2"].(float64); ok {
		metrics.R2 = v
	}

	return metrics
}

// toModelResponse converts domain entity to DTO
func toModelResponse(model *domain.Model) *dto.ModelResponse {
	resp := &dto.ModelResponse{
		ID:           model.ID,
		Name:         model.Name,
		Description:  model.Description,
		BuildID:      model.BuildID,
		DatasourceID: model.DatasourceID,
		Algorithm:    model.Algorithm,
		ModelType:    model.ModelType,
		TargetColumn: model.TargetColumn,
		Status:       string(model.Status),
		Version:      model.Version,
		CreatedBy:    model.CreatedBy,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}

	if model.Metrics != nil {
		resp.Metrics = &dto.MetricsResponse{
			Accuracy:  model.Metrics.Accuracy,
			Precision: model.Metrics.Precision,
			Recall:    model.Metrics.Recall,
			F1Score:   model.Metrics.F1Score,
			MSE:       model.Metrics.MSE,
			RMSE:      model.Metrics.RMSE,
			MAE:       model.Metrics.MAE,
			R2:        model.Metrics.R2,
		}
	}

	return resp
}

// toVariableResponseList converts domain variables to DTOs
func toVariableResponseList(variables []domain.ModelVariable) []dto.VariableResponse {
	result := make([]dto.VariableResponse, len(variables))
	for i, v := range variables {
		result[i] = dto.VariableResponse{
			ID:           v.ID,
			ModelID:      v.ModelID,
			Name:         v.Name,
			DataType:     string(v.DataType),
			Role:         string(v.Role),
			Importance:   v.Importance,
			Statistics:   v.Statistics,
			EncodingInfo: v.EncodingInfo,
			Ordinal:      v.Ordinal,
			CreatedAt:    v.CreatedAt,
		}
	}
	return result
}

// toFileResponseList converts domain files to DTOs
func toFileResponseList(files []domain.ModelFile) []dto.FileResponse {
	result := make([]dto.FileResponse, len(files))
	for i, f := range files {
		result[i] = dto.FileResponse{
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
	return result
}
