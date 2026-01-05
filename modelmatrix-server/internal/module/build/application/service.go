package application

import (
	"fmt"
	"strings"

	"modelmatrix-server/internal/infrastructure/compute"
	"modelmatrix-server/internal/infrastructure/folderservice"
	dsApp "modelmatrix-server/internal/module/datasource/application"
	dsDomain "modelmatrix-server/internal/module/datasource/domain"
	"modelmatrix-server/internal/module/build/domain"
	"modelmatrix-server/internal/module/build/dto"
	"modelmatrix-server/internal/module/build/repository"
	invApp "modelmatrix-server/internal/module/inventory/application"
	invDto "modelmatrix-server/internal/module/inventory/dto"
	"modelmatrix-server/pkg/config"
	"modelmatrix-server/pkg/logger"
)

// BuildService defines the interface for build application service
type BuildService interface {
	Create(req *dto.CreateBuildRequest, createdBy string) (*dto.BuildResponse, error)
	Update(id string, req *dto.UpdateBuildRequest) (*dto.BuildResponse, error)
	Delete(id string) error
	GetByID(id string) (*dto.BuildResponse, error)
	List(params *dto.ListParams) (*dto.BuildListResponse, error)
	Start(id string) (*dto.BuildResponse, error)
	Cancel(id string) (*dto.BuildResponse, error)
	HandleCallback(req *dto.BuildCallbackRequest) error
}

// BuildServiceImpl implements BuildService
type BuildServiceImpl struct {
	buildRepo         repository.BuildRepository
	domainService     *domain.Service
	computeClient     compute.Client
	datasourceService dsApp.DatasourceService
	modelService      invApp.ModelService
	folderService     folderservice.FolderService
	config            *config.Config
}

// NewBuildService creates a new build service
func NewBuildService(
	buildRepo repository.BuildRepository,
	domainService *domain.Service,
	computeClient compute.Client,
	datasourceService dsApp.DatasourceService,
	modelService invApp.ModelService,
	folderSvc folderservice.FolderService,
	cfg *config.Config,
) BuildService {
	return &BuildServiceImpl{
		buildRepo:         buildRepo,
		domainService:     domainService,
		computeClient:     computeClient,
		datasourceService: datasourceService,
		modelService:      modelService,
		folderService:     folderSvc,
		config:            cfg,
	}
}

// Create creates a new model build
func (s *BuildServiceImpl) Create(req *dto.CreateBuildRequest, createdBy string) (*dto.BuildResponse, error) {
	// Convert DTO to domain entity
	modelType := domain.ModelType(req.ModelType)
	params := s.domainService.GetDefaultParameters(modelType)

	// Override with request parameters if provided
	if req.Parameters != nil {
		if req.Parameters.Algorithm != "" {
			params.Algorithm = req.Parameters.Algorithm
		}
		if req.Parameters.Hyperparameters != nil {
			params.Hyperparameters = req.Parameters.Hyperparameters
		}
		if req.Parameters.TrainTestSplit > 0 {
			params.TrainTestSplit = req.Parameters.TrainTestSplit
		}
		if req.Parameters.RandomSeed > 0 {
			params.RandomSeed = req.Parameters.RandomSeed
		}
		if req.Parameters.MaxIterations > 0 {
			params.MaxIterations = req.Parameters.MaxIterations
		}
		if req.Parameters.EarlyStopRounds > 0 {
			params.EarlyStopRounds = req.Parameters.EarlyStopRounds
		}
	}

	build := &domain.ModelBuild{
		Name:         req.Name,
		Description:  req.Description,
		DatasourceID: req.DatasourceID,
		ProjectID:    req.ProjectID,
		FolderID:     req.FolderID,
		ModelType:    modelType,
		Status:       domain.BuildStatusPending,
		Parameters:   params,
		CreatedBy:    createdBy,
	}

	// Validate using domain service
	if err := s.domainService.ValidateBuild(build); err != nil {
		return nil, err
	}

	// Check name uniqueness
	existingNames, err := s.buildRepo.GetAllNames()
	if err != nil {
		logger.Error("Failed to get build names: %v", err)
		return nil, err
	}

	if err := s.domainService.ValidateBuildNameUnique(build.Name, existingNames); err != nil {
		return nil, err
	}

	// Validate parameters
	if err := s.domainService.ValidateParameters(&build.Parameters); err != nil {
		return nil, err
	}

	// Create via repository
	if err := s.buildRepo.Create(build); err != nil {
		logger.Error("Failed to create build: %v", err)
		return nil, err
	}

	logger.Audit(createdBy, "create", "model_build", "", "success", nil)

	return toBuildResponse(build), nil
}

// Update updates an existing model build
func (s *BuildServiceImpl) Update(id string, req *dto.UpdateBuildRequest) (*dto.BuildResponse, error) {
	// Get existing build
	build, err := s.buildRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Only allow updates for pending builds
	if build.Status != domain.BuildStatusPending {
		return nil, domain.ErrBuildNotPending
	}

	// Apply updates
	if req.Name != nil {
		build.Name = *req.Name
	}
	if req.Description != nil {
		build.Description = *req.Description
	}

	// Validate using domain service
	if err := s.domainService.ValidateBuild(build); err != nil {
		return nil, err
	}

	// Update via repository
	if err := s.buildRepo.Update(build); err != nil {
		logger.Error("Failed to update build: %v", err)
		return nil, err
	}

	return toBuildResponse(build), nil
}

// Delete deletes a model build
func (s *BuildServiceImpl) Delete(id string) error {
	// Check if build exists
	build, err := s.buildRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Don't allow deletion of running builds
	if build.Status == domain.BuildStatusRunning {
		return domain.ErrBuildAlreadyRunning
	}

	if err := s.buildRepo.Delete(id); err != nil {
		logger.Error("Failed to delete build: %v", err)
		return err
	}

	return nil
}

// GetByID retrieves a model build by ID
func (s *BuildServiceImpl) GetByID(id string) (*dto.BuildResponse, error) {
	build, err := s.buildRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	return toBuildResponse(build), nil
}

// List retrieves model builds with pagination
func (s *BuildServiceImpl) List(params *dto.ListParams) (*dto.BuildListResponse, error) {
	params.SetDefaults()

	builds, total, err := s.buildRepo.List(params.Offset(), params.PageSize, params.Search, params.Status)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.BuildResponse, len(builds))
	for i, build := range builds {
		responses[i] = *toBuildResponse(&build)
	}

	return &dto.BuildListResponse{
		Builds: responses,
		Total:  total,
	}, nil
}

// Start starts a model build by calling the compute service
func (s *BuildServiceImpl) Start(id string) (*dto.BuildResponse, error) {
	// Get the build
	build, err := s.buildRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Validate using domain service
	if err := s.domainService.CanStartBuild(build); err != nil {
		return nil, err
	}

	// Get datasource information
	datasourceDetail, err := s.datasourceService.GetByID(build.DatasourceID)
	if err != nil {
		logger.Error("Failed to get datasource: %v", err)
		return nil, fmt.Errorf("failed to get datasource: %w", err)
	}

	// Validate datasource has file path
	if datasourceDetail.FilePath == "" {
		return nil, fmt.Errorf("datasource does not have a file path")
	}

	// Find target and input columns
	var targetColumn string
	var inputColumns []string

	for _, col := range datasourceDetail.Columns {
		switch col.Role {
		case string(dsDomain.ColumnRoleTarget):
			if targetColumn != "" {
				return nil, fmt.Errorf("multiple target columns found, expected exactly one")
			}
			targetColumn = col.Name
		case string(dsDomain.ColumnRoleInput):
			inputColumns = append(inputColumns, col.Name)
		}
	}

	if targetColumn == "" {
		return nil, fmt.Errorf("no target column found in datasource, please set one column as target")
	}

	if len(inputColumns) == 0 {
		return nil, fmt.Errorf("no input columns found in datasource, please set at least one column as input")
	}

	// Convert file path to MinIO format (minio://bucket/path)
	filePath := s.convertToMinIOPath(datasourceDetail.FilePath)

	// Build callback URL for compute service to notify when done
	callbackURL := fmt.Sprintf("%s/api/builds/callback", s.config.Server.BaseURL)

	// Prepare compute service request
	trainReq := &compute.TrainRequest{
		DatasourceID:    build.DatasourceID,
		BuildID:         build.ID,
		FilePath:        filePath,
		Algorithm:       build.Parameters.Algorithm,
		Hyperparameters: build.Parameters.Hyperparameters,
		TargetColumn:    targetColumn,
		InputColumns:    inputColumns,
		CallbackURL:     callbackURL,
	}

	// Call compute service to start training
	trainResp, err := s.computeClient.TrainModel(trainReq)
	if err != nil {
		logger.Error("Failed to start training job: %v", err)
		// Mark build as failed
		build.Fail(fmt.Sprintf("Failed to start training: %v", err))
		if updateErr := s.buildRepo.Update(build); updateErr != nil {
			logger.Error("Failed to update build status: %v", updateErr)
		}
		return nil, fmt.Errorf("failed to start training job: %w", err)
	}

	// Update build status to running
	build.Start()

	// Store job ID in parameters (we'll use it for status polling)
	if build.Parameters.Hyperparameters == nil {
		build.Parameters.Hyperparameters = make(map[string]interface{})
	}
	build.Parameters.Hyperparameters["_job_id"] = trainResp.JobID

	// Update via repository
	if err := s.buildRepo.Update(build); err != nil {
		logger.Error("Failed to update build status: %v", err)
		return nil, err
	}

	logger.Info("Training job started: build_id=%s, job_id=%s", build.ID, trainResp.JobID)
	logger.Audit("system", "start", "model_build", build.ID, "success", nil)

	return toBuildResponse(build), nil
}

// convertToMinIOPath converts a file path to MinIO format
// Input: "datasources/{id}/filename.csv" or file ID
// Output: "minio://{bucket}/{path}"
func (s *BuildServiceImpl) convertToMinIOPath(filePath string) string {
	// If already in minio:// format, return as is
	if strings.HasPrefix(filePath, "minio://") {
		return filePath
	}

	// Get bucket name from config
	bucket := s.config.FileService.MinioBucket
	if bucket == "" {
		bucket = "modelmatrix" // default
	}

	// Convert to minio:// format
	return fmt.Sprintf("minio://%s/%s", bucket, filePath)
}

// Cancel cancels a model build
func (s *BuildServiceImpl) Cancel(id string) (*dto.BuildResponse, error) {
	build, err := s.buildRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Validate using domain service
	if err := s.domainService.CanCancelBuild(build); err != nil {
		return nil, err
	}

	// Cancel the build
	build.Cancel()

	// Update via repository
	if err := s.buildRepo.Update(build); err != nil {
		logger.Error("Failed to cancel build: %v", err)
		return nil, err
	}

	return toBuildResponse(build), nil
}

// HandleCallback processes callback from compute service when training completes
func (s *BuildServiceImpl) HandleCallback(req *dto.BuildCallbackRequest) error {
	logger.Info("Received callback for build %s, status: %s, job_id: %s", req.BuildID, req.Status, req.JobID)
	logger.Debug("Callback payload: model_path=%v, metrics=%v, error=%v", req.ModelPath, req.Metrics, req.Error)

	// Get the build
	build, err := s.buildRepo.GetByID(req.BuildID)
	if err != nil {
		logger.Error("Failed to get build for callback: %v", err)
		return err
	}

	logger.Debug("Current build status: %s", build.Status)

	// Only process if build is still running
	if build.Status != domain.BuildStatusRunning {
		logger.Warn("Ignoring callback for build %s: not in running status (current: %s)", req.BuildID, build.Status)
		return nil
	}

	// Update build based on status
	if req.Status == "completed" {
		// Convert metrics from map to domain struct
		metrics := convertMetrics(req.Metrics)
		build.Complete(metrics)

		// Store model path in hyperparameters
		if req.ModelPath != nil {
			if build.Parameters.Hyperparameters == nil {
				build.Parameters.Hyperparameters = make(map[string]interface{})
			}
			build.Parameters.Hyperparameters["_model_path"] = *req.ModelPath
		}

		logger.Info("Build %s completed successfully", req.BuildID)
	} else {
		// Failed
		errorMsg := "Training failed"
		if req.Error != nil {
			errorMsg = *req.Error
		}
		build.Fail(errorMsg)
		logger.Info("Build %s failed: %s", req.BuildID, errorMsg)
	}

	// Update via repository
	logger.Debug("Updating build %s to status %s", build.ID, build.Status)
	if err := s.buildRepo.Update(build); err != nil {
		logger.Error("Failed to update build after callback: %v", err)
		return err
	}

	logger.Info("Build %s successfully updated to status %s", build.ID, build.Status)

	// Create model if build completed successfully
	if req.Status == "completed" {
		if err := s.createModelFromBuild(build, req); err != nil {
			logger.Error("Failed to create model from build: %v", err)
			// Don't return error - build was successful, model creation can be retried
		}
	}

	logger.Audit("system", "callback", "model_build", build.ID, "success", nil)
	return nil
}

// createModelFromBuild creates a model from a completed build
func (s *BuildServiceImpl) createModelFromBuild(build *domain.ModelBuild, callback *dto.BuildCallbackRequest) error {
	// Get datasource info
	datasourceDetail, err := s.datasourceService.GetByID(build.DatasourceID)
	if err != nil {
		return fmt.Errorf("failed to get datasource: %w", err)
	}

	// Find input columns
	var inputColumns []string
	var targetColumn string
	for _, col := range datasourceDetail.Columns {
		if col.Role == string(dsDomain.ColumnRoleInput) {
			inputColumns = append(inputColumns, col.Name)
		} else if col.Role == string(dsDomain.ColumnRoleTarget) {
			targetColumn = col.Name
		}
	}

	// Get model file path
	var modelFilePath string
	if callback.ModelPath != nil {
		modelFilePath = *callback.ModelPath
	}

	// Create model with the same project/folder as the build
	createReq := &invDto.CreateModelFromBuildRequest{
		BuildID:       build.ID,
		Name:          build.Name,
		Description:   build.Description,
		DatasourceID:  build.DatasourceID,
		ProjectID:     build.ProjectID,
		FolderID:      build.FolderID,
		Algorithm:     build.Parameters.Algorithm,
		ModelType:     string(build.ModelType),
		TargetColumn:  targetColumn,
		InputColumns:  inputColumns,
		ModelFilePath: modelFilePath,
		Metrics:       callback.Metrics,
		CreatedBy:     build.CreatedBy,
	}

	_, err = s.modelService.CreateFromBuild(createReq)
	if err != nil {
		return fmt.Errorf("failed to create model: %w", err)
	}

	logger.Info("Created model from build %s", build.ID)

	return nil
}

// convertMetrics converts metrics map to domain struct
func convertMetrics(metricsMap map[string]interface{}) *domain.BuildMetrics {
	if metricsMap == nil {
		return nil
	}

	metrics := &domain.BuildMetrics{}

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

// toBuildResponse converts domain entity to DTO
func toBuildResponse(build *domain.ModelBuild) *dto.BuildResponse {
	resp := &dto.BuildResponse{
		ID:           build.ID,
		Name:         build.Name,
		Description:  build.Description,
		DatasourceID: build.DatasourceID,
		ProjectID:    build.ProjectID,
		FolderID:     build.FolderID,
		ModelType:    string(build.ModelType),
		Status:       string(build.Status),
		ErrorMessage: build.ErrorMessage,
		StartedAt:    build.StartedAt,
		CompletedAt:  build.CompletedAt,
		CreatedBy:    build.CreatedBy,
		CreatedAt:    build.CreatedAt,
		UpdatedAt:    build.UpdatedAt,
	}

	resp.Parameters = &dto.TrainingParametersResponse{
		Algorithm:       build.Parameters.Algorithm,
		Hyperparameters: build.Parameters.Hyperparameters,
		TrainTestSplit:  build.Parameters.TrainTestSplit,
		RandomSeed:      build.Parameters.RandomSeed,
		MaxIterations:   build.Parameters.MaxIterations,
		EarlyStopRounds: build.Parameters.EarlyStopRounds,
	}

	if build.Metrics != nil {
		resp.Metrics = &dto.MetricsResponse{
			Accuracy:  build.Metrics.Accuracy,
			Precision: build.Metrics.Precision,
			Recall:    build.Metrics.Recall,
			F1Score:   build.Metrics.F1Score,
			MSE:       build.Metrics.MSE,
			RMSE:      build.Metrics.RMSE,
			MAE:       build.Metrics.MAE,
			R2:        build.Metrics.R2,
		}
	}

	return resp
}
