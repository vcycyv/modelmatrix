package application

import (
	"modelmatrix_backend/internal/module/modelmanage/domain"
	"modelmatrix_backend/internal/module/modelmanage/dto"
	"modelmatrix_backend/internal/module/modelmanage/repository"
	"modelmatrix_backend/pkg/logger"
)

// ModelService defines the interface for model application service
type ModelService interface {
	Create(req *dto.CreateModelRequest, createdBy string) (*dto.ModelResponse, error)
	Update(id string, req *dto.UpdateModelRequest) (*dto.ModelResponse, error)
	Delete(id string) error
	GetByID(id string) (*dto.ModelDetailResponse, error)
	List(params *dto.ListParams) (*dto.ModelListResponse, error)
	Activate(id string) (*dto.ModelResponse, error)
	Deactivate(id string) (*dto.ModelResponse, error)
	CreateVersion(modelID string, req *dto.CreateVersionRequest, createdBy string) (*dto.VersionResponse, error)
	GetVersions(modelID string) ([]dto.VersionResponse, error)
}

// ModelServiceImpl implements ModelService
type ModelServiceImpl struct {
	modelRepo     repository.ModelRepository
	versionRepo   repository.VersionRepository
	domainService *domain.Service
}

// NewModelService creates a new model service
func NewModelService(
	modelRepo repository.ModelRepository,
	versionRepo repository.VersionRepository,
	domainService *domain.Service,
) ModelService {
	return &ModelServiceImpl{
		modelRepo:     modelRepo,
		versionRepo:   versionRepo,
		domainService: domainService,
	}
}

// Create creates a new model
func (s *ModelServiceImpl) Create(req *dto.CreateModelRequest, createdBy string) (*dto.ModelResponse, error) {
	// Convert DTO to domain entity
	model := &domain.Model{
		Name:        req.Name,
		Description: req.Description,
		BuildID:     req.BuildID,
		Status:      domain.ModelStatusDraft,
		CreatedBy:   createdBy,
	}

	// Set metadata if provided
	if req.Metadata != nil {
		model.Metadata = &domain.ModelMetadata{
			Algorithm:     req.Metadata.Algorithm,
			ModelType:     req.Metadata.ModelType,
			InputFeatures: req.Metadata.InputFeatures,
			TargetFeature: req.Metadata.TargetFeature,
			Framework:     req.Metadata.Framework,
			Version:       req.Metadata.Version,
			Custom:        req.Metadata.Custom,
		}
	}

	// Validate using domain service
	if err := s.domainService.ValidateModel(model); err != nil {
		return nil, err
	}

	// Check name uniqueness
	existingNames, err := s.modelRepo.GetAllNames()
	if err != nil {
		logger.Error("Failed to get model names: %v", err)
		return nil, err
	}

	if err := s.domainService.ValidateModelNameUnique(model.Name, existingNames); err != nil {
		return nil, err
	}

	// Create via repository
	if err := s.modelRepo.Create(model); err != nil {
		logger.Error("Failed to create model: %v", err)
		return nil, err
	}

	logger.Audit(createdBy, "create", "model", "", "success", nil)

	return toModelResponse(model, 0), nil
}

// Update updates an existing model
func (s *ModelServiceImpl) Update(id string, req *dto.UpdateModelRequest) (*dto.ModelResponse, error) {
	// Get existing model
	model, err := s.modelRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
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

	versionCount, _ := s.modelRepo.CountVersions(id)

	return toModelResponse(model, int(versionCount)), nil
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

// GetByID retrieves a model by ID with versions
func (s *ModelServiceImpl) GetByID(id string) (*dto.ModelDetailResponse, error) {
	model, err := s.modelRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	versions, err := s.versionRepo.ListByModelID(id)
	if err != nil {
		return nil, err
	}

	response := &dto.ModelDetailResponse{
		ModelResponse: *toModelResponse(model, len(versions)),
		Versions:      toVersionResponseList(versions),
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
		versionCount, _ := s.modelRepo.CountVersions(model.ID)
		responses[i] = *toModelResponse(&model, int(versionCount))
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
	model.Activate()

	// Update via repository
	if err := s.modelRepo.UpdateStatus(id, model.Status); err != nil {
		logger.Error("Failed to activate model: %v", err)
		return nil, err
	}

	versionCount, _ := s.modelRepo.CountVersions(id)

	return toModelResponse(model, int(versionCount)), nil
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
	model.Deactivate()

	// Update via repository
	if err := s.modelRepo.UpdateStatus(id, model.Status); err != nil {
		logger.Error("Failed to deactivate model: %v", err)
		return nil, err
	}

	versionCount, _ := s.modelRepo.CountVersions(id)

	return toModelResponse(model, int(versionCount)), nil
}

// CreateVersion creates a new model version
func (s *ModelServiceImpl) CreateVersion(modelID string, req *dto.CreateVersionRequest, createdBy string) (*dto.VersionResponse, error) {
	// Verify model exists
	if _, err := s.modelRepo.GetByID(modelID); err != nil {
		return nil, err
	}

	// Convert DTO to domain entity
	version := &domain.ModelVersion{
		ModelID:      modelID,
		Version:      req.Version,
		BuildID:      req.BuildID,
		Status:       domain.ModelStatusDraft,
		ArtifactPath: req.ArtifactPath,
		Notes:        req.Notes,
		Metrics:      req.Metrics,
		CreatedBy:    createdBy,
	}

	// Validate using domain service
	if err := s.domainService.ValidateVersion(version); err != nil {
		return nil, err
	}

	// Check version uniqueness
	existingVersions, err := s.versionRepo.GetVersionStrings(modelID)
	if err != nil {
		return nil, err
	}

	if err := s.domainService.ValidateVersionUnique(version.Version, existingVersions); err != nil {
		return nil, err
	}

	// Create via repository
	if err := s.versionRepo.Create(version); err != nil {
		logger.Error("Failed to create version: %v", err)
		return nil, err
	}

	return toVersionResponse(version), nil
}

// GetVersions retrieves all versions for a model
func (s *ModelServiceImpl) GetVersions(modelID string) ([]dto.VersionResponse, error) {
	// Verify model exists
	if _, err := s.modelRepo.GetByID(modelID); err != nil {
		return nil, err
	}

	versions, err := s.versionRepo.ListByModelID(modelID)
	if err != nil {
		return nil, err
	}

	return toVersionResponseList(versions), nil
}

// toModelResponse converts domain entity to DTO
func toModelResponse(model *domain.Model, versionCount int) *dto.ModelResponse {
	resp := &dto.ModelResponse{
		ID:           model.ID,
		Name:         model.Name,
		Description:  model.Description,
		BuildID:      model.BuildID,
		Status:       string(model.Status),
		ArtifactPath: model.ArtifactPath,
		VersionCount: versionCount,
		CreatedBy:    model.CreatedBy,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}

	if model.Metadata != nil {
		resp.Metadata = &dto.ModelMetadataResponse{
			Algorithm:     model.Metadata.Algorithm,
			ModelType:     model.Metadata.ModelType,
			InputFeatures: model.Metadata.InputFeatures,
			TargetFeature: model.Metadata.TargetFeature,
			Framework:     model.Metadata.Framework,
			Version:       model.Metadata.Version,
			Metrics:       model.Metadata.Metrics,
			Custom:        model.Metadata.Custom,
		}
	}

	return resp
}

// toVersionResponse converts domain entity to DTO
func toVersionResponse(version *domain.ModelVersion) *dto.VersionResponse {
	return &dto.VersionResponse{
		ID:           version.ID,
		ModelID:      version.ModelID,
		Version:      version.Version,
		BuildID:      version.BuildID,
		Status:       string(version.Status),
		ArtifactPath: version.ArtifactPath,
		Metrics:      version.Metrics,
		Notes:        version.Notes,
		CreatedBy:    version.CreatedBy,
		CreatedAt:    version.CreatedAt,
		UpdatedAt:    version.UpdatedAt,
	}
}

// toVersionResponseList converts domain entities to DTOs
func toVersionResponseList(versions []domain.ModelVersion) []dto.VersionResponse {
	result := make([]dto.VersionResponse, len(versions))
	for i, v := range versions {
		result[i] = *toVersionResponse(&v)
	}
	return result
}

