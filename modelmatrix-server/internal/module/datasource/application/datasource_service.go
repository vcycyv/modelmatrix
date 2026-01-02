package application

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"modelmatrix-server/internal/infrastructure/fileservice"
	"modelmatrix-server/internal/module/datasource/domain"
	"modelmatrix-server/internal/module/datasource/dto"
	"modelmatrix-server/internal/module/datasource/repository"
	"modelmatrix-server/pkg/logger"

	"gorm.io/gorm"
)

// ExternalDBConnector interface for fetching data from external databases
type ExternalDBConnector interface {
	// FetchTableData connects to an external database and returns the data as CSV bytes
	FetchTableData(config *domain.ConnectionConfig) ([]byte, []domain.Column, error)
}

// DatasourceServiceImpl implements DatasourceService
type DatasourceServiceImpl struct {
	db             *gorm.DB
	datasourceRepo repository.DatasourceRepository
	collectionRepo repository.CollectionRepository
	columnRepo     repository.ColumnRepository
	domainService  *domain.Service
	fileService    fileservice.FileService
	dbConnector    ExternalDBConnector
}

// NewDatasourceService creates a new datasource service
func NewDatasourceService(
	db *gorm.DB,
	datasourceRepo repository.DatasourceRepository,
	collectionRepo repository.CollectionRepository,
	columnRepo repository.ColumnRepository,
	domainService *domain.Service,
	fileService fileservice.FileService,
	dbConnector ExternalDBConnector,
) DatasourceService {
	return &DatasourceServiceImpl{
		db:             db,
		datasourceRepo: datasourceRepo,
		collectionRepo: collectionRepo,
		columnRepo:     columnRepo,
		domainService:  domainService,
		fileService:    fileService,
		dbConnector:    dbConnector,
	}
}

// Create creates a new datasource.
// For file types (csv/parquet), filename and fileData are required.
// For database types (postgresql/mysql), connection_config is used to fetch data and save as parquet.
func (s *DatasourceServiceImpl) Create(req *dto.CreateDatasourceRequest, filename string, fileData []byte, createdBy string) (*dto.DatasourceResponse, error) {
	// Verify collection exists
	collection, err := s.collectionRepo.GetByID(req.CollectionID)
	if err != nil {
		return nil, err
	}

	dsType := domain.DatasourceType(req.Type)

	// Check name uniqueness within collection
	existingNames, err := s.datasourceRepo.GetNamesInCollection(req.CollectionID)
	if err != nil {
		return nil, err
	}
	if err := s.domainService.ValidateDatasourceNameUnique(req.Name, existingNames); err != nil {
		return nil, err
	}

	// Handle based on type
	if dsType.IsFile() {
		return s.createFromFile(req, filename, fileData, createdBy, collection)
	}
	return s.createFromDatabase(req, createdBy, collection)
}

// createFromFile handles creation from uploaded CSV/Parquet files
func (s *DatasourceServiceImpl) createFromFile(req *dto.CreateDatasourceRequest, filename string, fileData []byte, createdBy string, collection *domain.Collection) (*dto.DatasourceResponse, error) {
	dsType := domain.DatasourceType(req.Type)

	// Validate file data provided
	if len(fileData) == 0 {
		return nil, fmt.Errorf("file data is required for %s datasource type", req.Type)
	}

	// Create datasource entity
	datasource := &domain.Datasource{
		CollectionID: req.CollectionID,
		Name:         req.Name,
		Description:  req.Description,
		Type:         dsType,
		CreatedBy:    createdBy,
	}

	// Create datasource in database
	if err := s.datasourceRepo.Create(datasource); err != nil {
		logger.Error("Failed to create datasource: %v", err)
		return nil, err
	}

	// Save file to MinIO
	reader := bytes.NewReader(fileData)
	subPath := fmt.Sprintf("datasources/%s", datasource.ID)
	fileInfo, err := s.fileService.SaveWithPath(subPath, filename, reader, int64(len(fileData)))
	if err != nil {
		logger.Error("Failed to save file: %v", err)
		s.datasourceRepo.Delete(datasource.ID)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Validate file based on type
	if dsType == domain.DatasourceTypeParquet {
		if err := s.fileService.ValidateParquet(fileInfo.ID); err != nil {
			s.fileService.Delete(fileInfo.ID)
			s.datasourceRepo.Delete(datasource.ID)
			return nil, fmt.Errorf("invalid Parquet file: %w", err)
		}
	} else if dsType == domain.DatasourceTypeCSV {
		if err := s.fileService.ValidateCSV(fileInfo.ID); err != nil {
			s.fileService.Delete(fileInfo.ID)
			s.datasourceRepo.Delete(datasource.ID)
			return nil, fmt.Errorf("invalid CSV file: %w", err)
		}
	}

	// Update datasource file path
	if err := s.datasourceRepo.UpdateFilePath(datasource.ID, fileInfo.ID); err != nil {
		s.fileService.Delete(fileInfo.ID)
		s.datasourceRepo.Delete(datasource.ID)
		return nil, err
	}

	// Extract columns from file
	columns, err := s.extractColumnsFromFile(dsType, fileInfo.ID)
	if err != nil {
		logger.Warn("Failed to extract columns from file: %v", err)
		columns = []domain.Column{}
	}

	// Create columns if extracted
	if len(columns) > 0 {
		for i := range columns {
			columns[i].DatasourceID = datasource.ID
		}
		if err := s.columnRepo.CreateBatch(columns); err != nil {
			logger.Warn("Failed to create columns: %v", err)
		}
	}

	// Reload datasource with columns for response
	datasourceWithColumns, err := s.datasourceRepo.GetByIDWithColumns(datasource.ID)
	if err != nil {
		logger.Warn("Failed to reload datasource with columns: %v", err)
		datasourceWithColumns = datasource
	}

	logger.Audit(createdBy, "create", "datasource", datasource.ID, "success", nil)
	return toDatasourceResponse(datasourceWithColumns, collection.Name), nil
}

// createFromDatabase handles creation by fetching data from external database
func (s *DatasourceServiceImpl) createFromDatabase(req *dto.CreateDatasourceRequest, createdBy string, collection *domain.Collection) (*dto.DatasourceResponse, error) {
	if req.ConnectionConfig == nil {
		return nil, fmt.Errorf("connection_config is required for database datasource types")
	}

	// Convert DTO to domain config
	connConfig := &domain.ConnectionConfig{
		Host:     req.ConnectionConfig.Host,
		Port:     req.ConnectionConfig.Port,
		Database: req.ConnectionConfig.Database,
		Username: req.ConnectionConfig.Username,
		Password: req.ConnectionConfig.Password,
		Schema:   req.ConnectionConfig.Schema,
		Table:    req.ConnectionConfig.Table,
		SSLMode:  req.ConnectionConfig.SSLMode,
	}

	// Fetch data from external database
	csvData, columns, err := s.dbConnector.FetchTableData(connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data from database: %w", err)
	}

	// Create datasource entity (with connection config for reference)
	datasource := &domain.Datasource{
		CollectionID:     req.CollectionID,
		Name:             req.Name,
		Description:      req.Description,
		Type:             domain.DatasourceType(req.Type),
		ConnectionConfig: connConfig,
		CreatedBy:        createdBy,
	}

	// Create datasource in database
	if err := s.datasourceRepo.Create(datasource); err != nil {
		logger.Error("Failed to create datasource: %v", err)
		return nil, err
	}

	// Save CSV data to MinIO with table name as filename
	filename := fmt.Sprintf("%s.csv", connConfig.Table)
	reader := bytes.NewReader(csvData)
	subPath := fmt.Sprintf("datasources/%s", datasource.ID)
	fileInfo, err := s.fileService.SaveWithPath(subPath, filename, reader, int64(len(csvData)))
	if err != nil {
		logger.Error("Failed to save file: %v", err)
		s.datasourceRepo.Delete(datasource.ID)
		return nil, fmt.Errorf("failed to save data file: %w", err)
	}

	// Update datasource file path
	if err := s.datasourceRepo.UpdateFilePath(datasource.ID, fileInfo.ID); err != nil {
		s.fileService.Delete(fileInfo.ID)
		s.datasourceRepo.Delete(datasource.ID)
		return nil, err
	}

	// Create columns
	if len(columns) > 0 {
		for i := range columns {
			columns[i].DatasourceID = datasource.ID
		}
		if err := s.columnRepo.CreateBatch(columns); err != nil {
			logger.Warn("Failed to create columns: %v", err)
		}
	}

	// Reload datasource with columns for response
	datasourceWithColumns, err := s.datasourceRepo.GetByIDWithColumns(datasource.ID)
	if err != nil {
		logger.Warn("Failed to reload datasource with columns: %v", err)
		datasourceWithColumns = datasource
	}

	logger.Audit(createdBy, "create", "datasource", datasource.ID, "success", nil)
	return toDatasourceResponse(datasourceWithColumns, collection.Name), nil
}

// Update updates an existing datasource
func (s *DatasourceServiceImpl) Update(id string, req *dto.UpdateDatasourceRequest) (*dto.DatasourceResponse, error) {
	// Get existing datasource
	datasource, err := s.datasourceRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Get collection for response
	collection, err := s.collectionRepo.GetByID(datasource.CollectionID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		datasource.Name = *req.Name
	}
	if req.Description != nil {
		datasource.Description = *req.Description
	}

	// Check name uniqueness if name changed
	if req.Name != nil {
		existingNames, err := s.datasourceRepo.GetNamesInCollection(datasource.CollectionID)
		if err != nil {
			return nil, err
		}
		// Filter out current datasource's name
		var filteredNames []string
		for _, name := range existingNames {
			if name != datasource.Name {
				filteredNames = append(filteredNames, name)
			}
		}
		if err := s.domainService.ValidateDatasourceNameUnique(*req.Name, filteredNames); err != nil {
			return nil, err
		}
	}

	// Update via repository
	if err := s.datasourceRepo.Update(datasource); err != nil {
		logger.Error("Failed to update datasource: %v", err)
		return nil, err
	}

	return toDatasourceResponse(datasource, collection.Name), nil
}

// Delete deletes a datasource
func (s *DatasourceServiceImpl) Delete(id string) error {
	// Get datasource to check file
	datasource, err := s.datasourceRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Delete within transaction
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete columns first
		if err := s.columnRepo.DeleteByDatasourceID(id); err != nil {
			logger.Error("Failed to delete datasource columns: %v", err)
			return err
		}

		// Delete datasource
		if err := s.datasourceRepo.Delete(id); err != nil {
			logger.Error("Failed to delete datasource: %v", err)
			return err
		}

		// Delete file if exists
		if datasource.FilePath != "" {
			if err := s.fileService.Delete(datasource.FilePath); err != nil {
				logger.Warn("Failed to delete datasource file: %v", err)
				// Don't fail the deletion for file cleanup errors
			}
		}

		return nil
	})
}

// GetByID retrieves a datasource by ID with columns
func (s *DatasourceServiceImpl) GetByID(id string) (*dto.DatasourceDetailResponse, error) {
	datasource, err := s.datasourceRepo.GetByIDWithColumns(id)
	if err != nil {
		return nil, err
	}

	collection, err := s.collectionRepo.GetByID(datasource.CollectionID)
	if err != nil {
		return nil, err
	}

	response := &dto.DatasourceDetailResponse{
		DatasourceResponse: *toDatasourceResponse(datasource, collection.Name),
		Columns:            toColumnResponseList(datasource.Columns),
	}

	return response, nil
}

// List retrieves datasources with pagination
func (s *DatasourceServiceImpl) List(collectionID *string, params *dto.ListParams) (*dto.DatasourceListResponse, error) {
	params.SetDefaults()

	datasources, total, err := s.datasourceRepo.List(collectionID, params.Offset(), params.PageSize, params.Search)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.DatasourceResponse, len(datasources))
	for i, ds := range datasources {
		collection, _ := s.collectionRepo.GetByID(ds.CollectionID)
		collectionName := ""
		if collection != nil {
			collectionName = collection.Name
		}
		responses[i] = *toDatasourceResponse(&ds, collectionName)
	}

	return &dto.DatasourceListResponse{
		Datasources: responses,
		Total:       total,
	}, nil
}

// extractColumnsFromFile extracts column metadata from a file
func (s *DatasourceServiceImpl) extractColumnsFromFile(dsType domain.DatasourceType, fileID string) ([]domain.Column, error) {
	reader, _, err := s.fileService.Get(fileID)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	switch dsType {
	case domain.DatasourceTypeCSV:
		return s.extractColumnsFromCSV(reader)
	case domain.DatasourceTypeParquet:
		// Parquet column extraction requires more complex handling
		// For now, return empty - columns can be added manually
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", dsType)
	}
}

// extractColumnsFromCSV extracts column names from CSV header
func (s *DatasourceServiceImpl) extractColumnsFromCSV(reader io.Reader) ([]domain.Column, error) {
	csvReader := csv.NewReader(reader)
	headers, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	columns := make([]domain.Column, len(headers))
	for i, header := range headers {
		columns[i] = domain.Column{
			Name:     strings.TrimSpace(header),
			DataType: "string", // Default to string, can be updated later
			Role:     domain.ColumnRoleInput,
		}
	}

	return columns, nil
}
