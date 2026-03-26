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
	// UpdateFromBuild updates an existing model from a completed build (retrain callback)
	UpdateFromBuild(modelID string, req *dto.CreateModelFromBuildRequest) (*dto.ModelResponse, error)

	// Scoring
	Score(modelID string, req *dto.ScoreRequest, scoredBy string) (*dto.ScoreResponse, error)
	HandleScoreCallback(req *dto.ScoreCallbackRequest) error
	ConfigureScoring(computeClient compute.Client, datasourceGetter DatasourceGetter, datasourceCreator DatasourceCreator, cfg *config.Config)

	// File operations
	GetFileContent(modelID string, fileID string) (*dto.FileContentResponse, error)

	// Folder/Project cascade operations
	DeleteByFolderID(folderID string) error
	DeleteByProjectID(projectID string) error
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

// UpdateFromBuild updates an existing model from a completed build (used for retrain callback)
func (s *ModelServiceImpl) UpdateFromBuild(modelID string, req *dto.CreateModelFromBuildRequest) (*dto.ModelResponse, error) {
	model, err := s.modelRepo.GetByIDWithRelations(modelID)
	if err != nil {
		return nil, err
	}

	model.BuildID = req.BuildID
	model.DatasourceID = req.DatasourceID
	model.Metrics = convertMetrics(req.Metrics)
	model.Version++

	if err := s.modelRepo.Update(model); err != nil {
		return nil, err
	}

	if err := s.modelRepo.DeleteVariablesByModelID(modelID); err != nil {
		return nil, err
	}
	if err := s.modelRepo.DeleteFilesByModelID(modelID); err != nil {
		return nil, err
	}

	variables := make([]domain.ModelVariable, 0, len(req.InputColumns)+1)
	for i, colName := range req.InputColumns {
		variable := domain.ModelVariable{
			ModelID:  modelID,
			Name:     colName,
			DataType: domain.VariableDataTypeNumeric,
			Role:     domain.VariableRoleInput,
			Ordinal:  i,
		}
		if req.FeatureImportances != nil {
			if imp, ok := req.FeatureImportances[colName]; ok {
				variable.Importance = &imp
			}
		}
		variables = append(variables, variable)
	}
	if req.TargetColumn != "" {
		variables = append(variables, domain.ModelVariable{
			ModelID:  modelID,
			Name:     req.TargetColumn,
			DataType: domain.VariableDataTypeNumeric,
			Role:     domain.VariableRoleTarget,
			Ordinal:  len(req.InputColumns),
		})
	}
	if err := s.modelRepo.CreateVariables(variables); err != nil {
		return nil, err
	}

	if req.ModelFilePath != "" {
		file := domain.ModelFile{
			ModelID:     modelID,
			FileType:    domain.FileTypeModel,
			FilePath:    req.ModelFilePath,
			FileName:    filepath.Base(req.ModelFilePath),
			Description: fmt.Sprintf("Trained %s model", req.Algorithm),
		}
		if err := s.modelRepo.CreateFile(&file); err != nil {
			return nil, err
		}
	}
	if req.CodeFilePath != "" {
		codeFile := domain.ModelFile{
			ModelID:     modelID,
			FileType:    domain.FileTypeTrainingCode,
			FilePath:    req.CodeFilePath,
			FileName:    filepath.Base(req.CodeFilePath),
			Description: "Python code used to train this model",
		}
		if err := s.modelRepo.CreateFile(&codeFile); err != nil {
			return nil, err
		}
	}

	logger.Info("Updated model %s from build %s (version %d)", modelID, req.BuildID, model.Version)
	updated, _ := s.modelRepo.GetByID(modelID)
	return toModelResponse(updated), nil
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
	// Apply description: both nil (UI cleared field) and non-nil update the field
	if req.Description != nil {
		model.Description = *req.Description
	} else {
		model.Description = ""
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

// DeleteByFolderID deletes all models directly in a folder
func (s *ModelServiceImpl) DeleteByFolderID(folderID string) error {
	ids, err := s.modelRepo.GetIDsByFolderID(folderID)
	if err != nil {
		return fmt.Errorf("failed to get models in folder: %w", err)
	}

	for _, id := range ids {
		if err := s.Delete(id); err != nil {
			logger.Warn("Failed to delete model %s: %v", id, err)
			// Continue deleting others
		}
	}
	return nil
}

// DeleteByProjectID deletes all models in a project
func (s *ModelServiceImpl) DeleteByProjectID(projectID string) error {
	ids, err := s.modelRepo.GetIDsByProjectID(projectID)
	if err != nil {
		return fmt.Errorf("failed to get models in project: %w", err)
	}

	for _, id := range ids {
		if err := s.Delete(id); err != nil {
			logger.Warn("Failed to delete model %s: %v", id, err)
			// Continue deleting others
		}
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

// GetFileContent retrieves the content of a model file (for text files only)
func (s *ModelServiceImpl) GetFileContent(modelID string, fileID string) (*dto.FileContentResponse, error) {
	// Get the model to ensure it exists and verify file belongs to it
	model, err := s.modelRepo.GetByIDWithRelations(modelID)
	if err != nil {
		return nil, err
	}

	// Find the file in the model's files
	var targetFile *domain.ModelFile
	for i := range model.Files {
		if model.Files[i].ID == fileID {
			targetFile = &model.Files[i]
			break
		}
	}

	if targetFile == nil {
		return nil, domain.ErrFileNotFound
	}

	// Check if it's a text-based file type
	isTextFile := isTextFileType(targetFile.FileName, string(targetFile.FileType))
	if !isTextFile {
		return &dto.FileContentResponse{
			FileID:      targetFile.ID,
			FileName:    targetFile.FileName,
			FileType:    string(targetFile.FileType),
			ContentType: "application/octet-stream",
			Content:     "",
			Size:        getFileSize(targetFile.FileSize),
			IsText:      false,
		}, nil
	}

	// Extract object key from file path (strip minio://bucket/ prefix if present)
	objectKey := targetFile.FilePath
	if strings.HasPrefix(objectKey, "minio://") {
		parts := strings.SplitN(objectKey, "/", 4) // ["minio:", "", "bucket", "path"]
		if len(parts) >= 4 {
			objectKey = parts[3]
		}
	}

	// Read file content
	content, fileInfo, err := s.fileService.ReadFileContent(objectKey)
	if err != nil {
		logger.Error("Failed to read file content: %v", err)
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	contentType := getContentTypeFromFileName(targetFile.FileName)

	return &dto.FileContentResponse{
		FileID:      targetFile.ID,
		FileName:    targetFile.FileName,
		FileType:    string(targetFile.FileType),
		ContentType: contentType,
		Content:     string(content),
		Size:        fileInfo.Size,
		IsText:      true,
	}, nil
}

// isTextFileType checks if a file is a text-based file that can be displayed
func isTextFileType(fileName string, fileType string) bool {
	// Check by file type
	if fileType == "training_code" || fileType == "metadata" || fileType == "feature_names" {
		return true
	}

	// Check by file extension
	ext := strings.ToLower(filepath.Ext(fileName))
	textExtensions := map[string]bool{
		".py":   true,
		".txt":  true,
		".json": true,
		".yaml": true,
		".yml":  true,
		".md":   true,
		".csv":  true,
		".log":  true,
		".xml":  true,
		".html": true,
		".css":  true,
		".js":   true,
		".ts":   true,
		".sql":  true,
		".sh":   true,
		".r":    true,
		".ipynb": true,
	}

	return textExtensions[ext]
}

// getContentTypeFromFileName returns the content type based on file extension
func getContentTypeFromFileName(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".py":
		return "text/x-python"
	case ".json":
		return "application/json"
	case ".yaml", ".yml":
		return "text/yaml"
	case ".md":
		return "text/markdown"
	case ".csv":
		return "text/csv"
	case ".txt", ".log":
		return "text/plain"
	case ".xml":
		return "text/xml"
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "text/javascript"
	case ".ts":
		return "text/typescript"
	case ".sql":
		return "text/sql"
	case ".sh":
		return "text/x-shellscript"
	case ".r":
		return "text/x-r"
	default:
		return "text/plain"
	}
}

// getFileSize safely gets file size from pointer
func getFileSize(size *int64) int64 {
	if size == nil {
		return 0
	}
	return *size
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
