package application

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"modelmatrix-server/internal/infrastructure/compute"
	"modelmatrix-server/internal/infrastructure/fileservice"
	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"
	"modelmatrix-server/internal/module/inventory/repository"
	"modelmatrix-server/pkg/config"
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

	// Scoring
	Score(modelID string, req *dto.ScoreRequest, scoredBy string) (*dto.ScoreResponse, error)
	HandleScoreCallback(req *dto.ScoreCallbackRequest) error
	ConfigureScoring(computeClient compute.Client, datasourceGetter DatasourceGetter, datasourceCreator DatasourceCreator, cfg *config.Config)
}

// DatasourceGetter interface for getting datasource details (to avoid circular imports)
type DatasourceGetter interface {
	GetFilePath(datasourceID string) (string, error)
}

// DatasourceCreator interface for creating scored output datasource
type DatasourceCreator interface {
	CreateScoredOutput(collectionID, name, filePath string, rowCount int, createdBy string) (string, error)
}

// ModelServiceImpl implements ModelService
type ModelServiceImpl struct {
	modelRepo         repository.ModelRepository
	domainService     *domain.Service
	fileService       fileservice.FileService
	computeClient     compute.Client
	datasourceGetter  DatasourceGetter
	datasourceCreator DatasourceCreator
	config            *config.Config
}

// NewModelService creates a new model service
func NewModelService(
	modelRepo repository.ModelRepository,
	domainService *domain.Service,
	fileService fileservice.FileService,
) ModelService {
	return &ModelServiceImpl{
		modelRepo:     modelRepo,
		domainService: domainService,
		fileService:   fileService,
	}
}

// NewModelServiceWithScoring creates a model service with scoring capabilities
func NewModelServiceWithScoring(
	modelRepo repository.ModelRepository,
	domainService *domain.Service,
	fileService fileservice.FileService,
	computeClient compute.Client,
	datasourceGetter DatasourceGetter,
	datasourceCreator DatasourceCreator,
	cfg *config.Config,
) ModelService {
	return &ModelServiceImpl{
		modelRepo:         modelRepo,
		domainService:     domainService,
		fileService:       fileService,
		computeClient:     computeClient,
		datasourceGetter:  datasourceGetter,
		datasourceCreator: datasourceCreator,
		config:            cfg,
	}
}

// ConfigureScoring adds scoring capabilities to an existing model service
func (s *ModelServiceImpl) ConfigureScoring(
	computeClient compute.Client,
	datasourceGetter DatasourceGetter,
	datasourceCreator DatasourceCreator,
	cfg *config.Config,
) {
	s.computeClient = computeClient
	s.datasourceGetter = datasourceGetter
	s.datasourceCreator = datasourceCreator
	s.config = cfg
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
		ProjectID:    req.ProjectID,
		FolderID:     req.FolderID,
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
	// Model input variables = original columns (preprocessing like one-hot encoding is part of scoring logic)
	logger.Info("Creating model with %d input variables", len(req.InputColumns))

	variables := make([]domain.ModelVariable, 0, len(req.InputColumns)+1)

	// Add input variables with importance scores
	for i, colName := range req.InputColumns {
		variable := domain.ModelVariable{
			ModelID:  model.ID,
			Name:     colName,
			DataType: domain.VariableDataTypeNumeric, // Default, can be refined later
			Role:     domain.VariableRoleInput,
			Ordinal:  i,
		}
		// Set importance if available from training
		if req.FeatureImportances != nil {
			if imp, ok := req.FeatureImportances[colName]; ok {
				variable.Importance = &imp
			}
		}
		variables = append(variables, variable)
	}

	// Add target variable (only for supervised learning)
	if req.TargetColumn != "" {
		variables = append(variables, domain.ModelVariable{
			ModelID:  model.ID,
			Name:     req.TargetColumn,
			DataType: domain.VariableDataTypeNumeric, // Default
			Role:     domain.VariableRoleTarget,
			Ordinal:  len(req.InputColumns),
		})
	}

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

	// Create training code file
	if req.CodeFilePath != "" {
		codeFile := domain.ModelFile{
			ModelID:     model.ID,
			FileType:    domain.FileTypeTrainingCode,
			FilePath:    req.CodeFilePath,
			FileName:    filepath.Base(req.CodeFilePath),
			Description: "Python code used to train this model",
		}
		if err := s.modelRepo.CreateFile(&codeFile); err != nil {
			logger.Error("Failed to create training code file: %v", err)
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

// Delete deletes a model and its files from storage
func (s *ModelServiceImpl) Delete(id string) error {
	// Get model with files to know what to delete from storage
	model, err := s.modelRepo.GetByIDWithRelations(id)
	if err != nil {
		return err
	}

	// Validate using domain service
	if err := s.domainService.CanDeleteModel(model); err != nil {
		return err
	}

	// Delete model files from MinIO storage
	for _, file := range model.Files {
		if file.FilePath != "" {
			// Strip minio://bucket/ prefix if present to get the object key
			objectKey := file.FilePath
			if strings.HasPrefix(objectKey, "minio://") {
				// Format: minio://bucket/path -> extract path after bucket
				parts := strings.SplitN(objectKey, "/", 4) // ["minio:", "", "bucket", "path"]
				if len(parts) >= 4 {
					objectKey = parts[3]
				}
			}

			// Delete the model file
			if err := s.fileService.Delete(objectKey); err != nil {
				// Log error but continue - don't fail delete if file is already gone
				logger.Warn("Failed to delete model file from storage: %s (key: %s), error: %v", file.FilePath, objectKey, err)
			} else {
				logger.Info("Deleted model file from storage: %s", objectKey)
			}
		}
	}

	// Delete model from database (cascades to variables and files tables)
	if err := s.modelRepo.Delete(id); err != nil {
		logger.Error("Failed to delete model: %v", err)
		return err
	}

	logger.Info("Deleted model %s with %d files", id, len(model.Files))
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
		ProjectID:    model.ProjectID,
		FolderID:     model.FolderID,
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

// Score scores data using a trained model
func (s *ModelServiceImpl) Score(modelID string, req *dto.ScoreRequest, scoredBy string) (*dto.ScoreResponse, error) {
	// Check if scoring dependencies are available
	if s.computeClient == nil || s.datasourceGetter == nil {
		return nil, fmt.Errorf("scoring not configured: missing dependencies")
	}

	// Get model with files and variables
	model, err := s.modelRepo.GetByIDWithRelations(modelID)
	if err != nil {
		return nil, err
	}

	// Get model file path
	var modelFilePath string
	for _, f := range model.Files {
		if f.FileType == domain.FileTypeModel {
			modelFilePath = f.FilePath
			break
		}
	}
	if modelFilePath == "" {
		return nil, fmt.Errorf("model file not found")
	}

	// Get input feature columns
	var inputColumns []string
	for _, v := range model.Variables {
		if v.Role == domain.VariableRoleInput {
			inputColumns = append(inputColumns, v.Name)
		}
	}

	// Get input datasource file path
	inputFilePath, err := s.datasourceGetter.GetFilePath(req.DatasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get input datasource: %w", err)
	}

	// Generate output table name if not provided
	outputTableName := ""
	if req.OutputTableName != nil && *req.OutputTableName != "" {
		outputTableName = *req.OutputTableName
	} else {
		outputTableName = fmt.Sprintf("scored_%s_%s", model.Name, time.Now().Format("20060102_150405"))
	}

	// Generate output path in MinIO
	outputPath := fmt.Sprintf("scored/%s/%s.parquet", modelID, outputTableName)

	// Build callback URL with query params for creating output datasource
	callbackURL := ""
	if s.config != nil {
		callbackURL = fmt.Sprintf("%s/api/models/%s/score/callback?collection_id=%s&table_name=%s&created_by=%s",
			s.config.Server.BaseURL, modelID, req.OutputCollectionID, outputTableName, scoredBy)
	}

	// Call compute service
	scoreReq := &compute.ScoreRequest{
		ModelID:       modelID,
		ModelFilePath: modelFilePath,
		InputFilePath: inputFilePath,
		OutputPath:    outputPath,
		InputColumns:  inputColumns,
		ModelType:     model.ModelType,
		Algorithm:     model.Algorithm,
		CallbackURL:   callbackURL,
	}

	scoreResp, err := s.computeClient.ScoreModel(scoreReq)
	if err != nil {
		return nil, fmt.Errorf("failed to start scoring: %w", err)
	}

	logger.Info("Scoring started for model %s, job_id: %s", modelID, scoreResp.JobID)

	return &dto.ScoreResponse{
		JobID:   scoreResp.JobID,
		Status:  scoreResp.Status,
		Message: scoreResp.Message,
	}, nil
}

// HandleScoreCallback processes the callback from compute service after scoring completes
func (s *ModelServiceImpl) HandleScoreCallback(req *dto.ScoreCallbackRequest) error {
	logger.Info("Received scoring callback for model %s, job_id: %s, status: %s", req.ModelID, req.JobID, req.Status)

	if req.Status == "failed" {
		logger.Error("Scoring failed for model %s: %s", req.ModelID, req.Error)
		return fmt.Errorf("scoring failed: %s", req.Error)
	}

	if req.Status == "completed" {
		logger.Info("Scoring completed for model %s, output: %s, rows: %d", req.ModelID, req.OutputFilePath, req.RowCount)

		// Clean file path - remove minio://bucket/ prefix if present
		filePath := req.OutputFilePath
		if strings.HasPrefix(filePath, "minio://") {
			// Extract path after bucket name: minio://bucket/path -> path
			parts := strings.SplitN(filePath, "/", 4) // ["minio:", "", "bucket", "path/to/file"]
			if len(parts) >= 4 {
				filePath = parts[3]
			}
		}

		// Create output datasource if we have all required info
		if s.datasourceCreator != nil && req.CollectionID != "" && req.TableName != "" {
			dsID, err := s.datasourceCreator.CreateScoredOutput(
				req.CollectionID,
				req.TableName,
				filePath,
				int(req.RowCount),
				req.CreatedBy,
			)
			if err != nil {
				logger.Error("Failed to create output datasource for model %s: %v", req.ModelID, err)
				return fmt.Errorf("failed to create output datasource: %w", err)
			}
			logger.Info("Created output datasource %s for model %s", dsID, req.ModelID)
		} else {
			logger.Warn("Cannot create output datasource: missing creator or params (collection_id=%s, table_name=%s)",
				req.CollectionID, req.TableName)
		}
	}

	return nil
}
