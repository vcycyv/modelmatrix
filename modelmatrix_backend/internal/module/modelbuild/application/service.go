package application

import (
	"modelmatrix_backend/internal/module/modelbuild/domain"
	"modelmatrix_backend/internal/module/modelbuild/dto"
	"modelmatrix_backend/internal/module/modelbuild/repository"
	"modelmatrix_backend/pkg/logger"
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
}

// BuildServiceImpl implements BuildService
type BuildServiceImpl struct {
	buildRepo     repository.BuildRepository
	domainService *domain.Service
}

// NewBuildService creates a new build service
func NewBuildService(
	buildRepo repository.BuildRepository,
	domainService *domain.Service,
) BuildService {
	return &BuildServiceImpl{
		buildRepo:     buildRepo,
		domainService: domainService,
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

// Start starts a model build
func (s *BuildServiceImpl) Start(id string) (*dto.BuildResponse, error) {
	build, err := s.buildRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Validate using domain service
	if err := s.domainService.CanStartBuild(build); err != nil {
		return nil, err
	}

	// Start the build
	build.Start()

	// Update via repository
	if err := s.buildRepo.Update(build); err != nil {
		logger.Error("Failed to start build: %v", err)
		return nil, err
	}

	// TODO: Implement actual model training orchestration here
	// This would typically involve:
	// 1. Loading the datasource
	// 2. Preparing the data
	// 3. Training the model
	// 4. Evaluating the model
	// 5. Saving the model artifacts

	return toBuildResponse(build), nil
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

// toBuildResponse converts domain entity to DTO
func toBuildResponse(build *domain.ModelBuild) *dto.BuildResponse {
	resp := &dto.BuildResponse{
		ID:           build.ID,
		Name:         build.Name,
		Description:  build.Description,
		DatasourceID: build.DatasourceID,
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

