package application

import (
	"fmt"
	"path/filepath"

	"modelmatrix-server/internal/infrastructure/fileservice"
	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"
	"modelmatrix-server/internal/module/inventory/repository"
	"modelmatrix-server/pkg/logger"
)

// ModelVersionService defines the interface for model version operations
type ModelVersionService interface {
	CreateVersion(modelID string, createdBy string) (*dto.VersionResponse, error)
	ListVersions(modelID string, params *dto.ListVersionsParams) (*dto.VersionListResponse, error)
	GetVersion(modelID string, versionID string) (*dto.VersionDetailResponse, error)
	RestoreVersion(modelID string, versionID string, restoredBy string) (*dto.ModelResponse, error)
}

// ModelVersionServiceImpl implements ModelVersionService
type ModelVersionServiceImpl struct {
	modelRepo   repository.ModelRepository
	versionRepo repository.ModelVersionRepository
	versionStore fileservice.VersionStore
}

// NewModelVersionService creates a new model version service
func NewModelVersionService(
	modelRepo repository.ModelRepository,
	versionRepo repository.ModelVersionRepository,
	versionStore fileservice.VersionStore,
) ModelVersionService {
	return &ModelVersionServiceImpl{
		modelRepo:   modelRepo,
		versionRepo: versionRepo,
		versionStore: versionStore,
	}
}

// CreateVersion creates an immutable snapshot of the current model (content-addressable file copy)
func (s *ModelVersionServiceImpl) CreateVersion(modelID string, createdBy string) (*dto.VersionResponse, error) {
	model, err := s.modelRepo.GetByIDWithRelations(modelID)
	if err != nil {
		return nil, err
	}

	nextNum, err := s.versionRepo.GetNextVersionNumber(modelID)
	if err != nil {
		return nil, fmt.Errorf("next version number: %w", err)
	}

	version := &domain.ModelVersion{
		ModelID:       modelID,
		VersionNumber: nextNum,
		Name:          model.Name,
		Description:   model.Description,
		BuildID:       model.BuildID,
		DatasourceID:  model.DatasourceID,
		ProjectID:     model.ProjectID,
		FolderID:      model.FolderID,
		Algorithm:     model.Algorithm,
		ModelType:     model.ModelType,
		TargetColumn:  model.TargetColumn,
		Status:        model.Status,
		Metrics:       model.Metrics,
		CreatedBy:     createdBy,
		Variables:     make([]domain.ModelVariable, len(model.Variables)),
		Files:         make([]domain.ModelFile, 0, len(model.Files)),
	}
	copy(version.Variables, model.Variables)

	for i := range model.Files {
		f := &model.Files[i]
		versionPath, err := s.versionStore.EnsureVersionedCopy(f.FilePath, f.Checksum, filepath.Ext(f.FileName))
		if err != nil {
			return nil, fmt.Errorf("version file %s: %w", f.FileName, err)
		}
		version.Files = append(version.Files, domain.ModelFile{
			FileType:    f.FileType,
			FilePath:    versionPath,
			FileName:    f.FileName,
			FileSize:    f.FileSize,
			Checksum:    f.Checksum,
			Description: f.Description,
		})
	}

	if err := s.versionRepo.Create(version); err != nil {
		return nil, err
	}
	logger.Audit(createdBy, "create_version", "model_version", version.ID, "success", nil)
	return toVersionResponse(version), nil
}

// ListVersions returns versions for a model (newest first)
func (s *ModelVersionServiceImpl) ListVersions(modelID string, params *dto.ListVersionsParams) (*dto.VersionListResponse, error) {
	if _, err := s.modelRepo.GetByID(modelID); err != nil {
		return nil, err
	}
	page, pageSize := 1, 20
	if params != nil {
		if params.Page > 0 {
			page = params.Page
		}
		if params.PageSize > 0 {
			pageSize = params.PageSize
		}
	}
	offset := (page - 1) * pageSize
	list, total, err := s.versionRepo.ListByModelID(modelID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	resp := &dto.VersionListResponse{
		Versions: make([]dto.VersionResponse, len(list)),
		Total:    total,
	}
	for i := range list {
		resp.Versions[i] = *toVersionResponse(&list[i])
	}
	return resp, nil
}

// GetVersion returns a version by ID (and verifies modelID)
func (s *ModelVersionServiceImpl) GetVersion(modelID string, versionID string) (*dto.VersionDetailResponse, error) {
	version, err := s.versionRepo.GetByID(versionID)
	if err != nil {
		return nil, err
	}
	if version.ModelID != modelID {
		return nil, domain.ErrVersionNotFound
	}
	return toVersionDetailResponse(version), nil
}

// RestoreVersion restores the current model from a version snapshot (metadata + variables + files)
func (s *ModelVersionServiceImpl) RestoreVersion(modelID string, versionID string, restoredBy string) (*dto.ModelResponse, error) {
	version, err := s.versionRepo.GetByID(versionID)
	if err != nil {
		return nil, err
	}
	if version.ModelID != modelID {
		return nil, domain.ErrVersionNotFound
	}

	current, err := s.modelRepo.GetByIDWithRelations(modelID)
	if err != nil {
		return nil, err
	}

	// Update model row from version (keep ID, UpdatedAt)
	current.Name = version.Name
	current.Description = version.Description
	current.BuildID = version.BuildID
	current.DatasourceID = version.DatasourceID
	current.ProjectID = version.ProjectID
	current.FolderID = version.FolderID
	current.Algorithm = version.Algorithm
	current.ModelType = version.ModelType
	current.TargetColumn = version.TargetColumn
	current.Status = version.Status
	current.Metrics = version.Metrics
	current.Version = version.VersionNumber
	if err := s.modelRepo.Update(current); err != nil {
		return nil, err
	}

	if err := s.modelRepo.DeleteVariablesByModelID(modelID); err != nil {
		return nil, err
	}
	if err := s.modelRepo.DeleteFilesByModelID(modelID); err != nil {
		return nil, err
	}

	for i := range version.Variables {
		v := &version.Variables[i]
		v.ModelID = modelID
		v.ID = ""
		if err := s.modelRepo.CreateVariable(v); err != nil {
			return nil, err
		}
	}
	for i := range version.Files {
		f := &version.Files[i]
		f.ModelID = modelID
		f.ID = ""
		if err := s.modelRepo.CreateFile(f); err != nil {
			return nil, err
		}
	}

	logger.Audit(restoredBy, "restore_version", "model", modelID, "success", nil)
	// Reload to get new variables/files for response
	updated, _ := s.modelRepo.GetByID(modelID)
	return modelToResponse(updated), nil
}

func toVersionResponse(v *domain.ModelVersion) *dto.VersionResponse {
	return &dto.VersionResponse{
		ID:            v.ID,
		ModelID:       v.ModelID,
		VersionNumber: v.VersionNumber,
		Name:          v.Name,
		Description:   v.Description,
		CreatedBy:     v.CreatedBy,
		CreatedAt:     v.CreatedAt,
	}
}

func toVersionDetailResponse(v *domain.ModelVersion) *dto.VersionDetailResponse {
	resp := &dto.VersionDetailResponse{
		VersionResponse: *toVersionResponse(v),
		BuildID:         v.BuildID,
		DatasourceID:    v.DatasourceID,
		ProjectID:       v.ProjectID,
		FolderID:        v.FolderID,
		Algorithm:       v.Algorithm,
		ModelType:       v.ModelType,
		TargetColumn:    v.TargetColumn,
		Status:          string(v.Status),
		Variables:       make([]dto.VariableResponse, len(v.Variables)),
		Files:           make([]dto.FileResponse, len(v.Files)),
	}
	if v.Metrics != nil {
		resp.Metrics = &dto.MetricsResponse{
			Accuracy:  v.Metrics.Accuracy,
			Precision: v.Metrics.Precision,
			Recall:    v.Metrics.Recall,
			F1Score:   v.Metrics.F1Score,
			MSE:       v.Metrics.MSE,
			RMSE:      v.Metrics.RMSE,
			MAE:       v.Metrics.MAE,
			R2:        v.Metrics.R2,
		}
	}
	for i := range v.Variables {
		resp.Variables[i] = variableToResponse(&v.Variables[i])
	}
	for i := range v.Files {
		resp.Files[i] = fileToResponse(&v.Files[i])
	}
	return resp
}

func variableToResponse(v *domain.ModelVariable) dto.VariableResponse {
	return dto.VariableResponse{
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

func fileToResponse(f *domain.ModelFile) dto.FileResponse {
	return dto.FileResponse{
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

func modelToResponse(m *domain.Model) *dto.ModelResponse {
	if m == nil {
		return nil
	}
	resp := &dto.ModelResponse{
		ID:           m.ID,
		Name:         m.Name,
		Description:  m.Description,
		BuildID:      m.BuildID,
		DatasourceID: m.DatasourceID,
		ProjectID:    m.ProjectID,
		FolderID:     m.FolderID,
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
		resp.Metrics = &dto.MetricsResponse{
			Accuracy:  m.Metrics.Accuracy,
			Precision: m.Metrics.Precision,
			Recall:    m.Metrics.Recall,
			F1Score:   m.Metrics.F1Score,
			MSE:       m.Metrics.MSE,
			RMSE:      m.Metrics.RMSE,
			MAE:       m.Metrics.MAE,
			R2:        m.Metrics.R2,
		}
	}
	return resp
}
