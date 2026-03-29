package application

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"modelmatrix-server/internal/infrastructure/fileservice"
	"modelmatrix-server/internal/module/datasource/domain"
	"modelmatrix-server/internal/module/datasource/dto"
	"modelmatrix-server/internal/module/datasource/repository"
	"modelmatrix-server/pkg/logger"

	"github.com/parquet-go/parquet-go"
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

	// Check name uniqueness if name changed (use original name to filter self out)
	if req.Name != nil {
		originalName := datasource.Name
		existingNames, err := s.datasourceRepo.GetNamesInCollection(datasource.CollectionID)
		if err != nil {
			return nil, err
		}
		// Filter out this datasource's original name so we don't conflict with ourselves
		var filteredNames []string
		for _, name := range existingNames {
			if name != originalName {
				filteredNames = append(filteredNames, name)
			}
		}
		if err := s.domainService.ValidateDatasourceNameUnique(*req.Name, filteredNames); err != nil {
			return nil, err
		}
	}

	// Apply updates
	if req.Name != nil {
		datasource.Name = *req.Name
	}
	if req.Description != nil {
		datasource.Description = *req.Description
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
	switch dsType {
	case domain.DatasourceTypeCSV:
		reader, _, err := s.fileService.Get(fileID)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return s.extractColumnsFromCSV(reader)
	case domain.DatasourceTypeParquet:
		// Parquet requires full file content to parse
		content, _, err := s.fileService.ReadFileContent(fileID)
		if err != nil {
			return nil, fmt.Errorf("failed to read parquet file: %w", err)
		}
		return s.extractColumnsFromParquet(content)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", dsType)
	}
}

// extractColumnsFromParquet extracts column names and data types from Parquet file
func (s *DatasourceServiceImpl) extractColumnsFromParquet(content []byte) ([]domain.Column, error) {
	reader := bytes.NewReader(content)
	file, err := parquet.OpenFile(reader, int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to open parquet file: %w", err)
	}

	schema := file.Schema()
	columns := make([]domain.Column, 0, len(schema.Fields()))

	for _, field := range schema.Fields() {
		dataType := parquetTypeToDataType(field.Type())
		columns = append(columns, domain.Column{
			Name:     field.Name(),
			DataType: dataType,
			Role:     domain.ColumnRoleInput,
		})
	}

	return columns, nil
}

// parquetTypeToDataType converts parquet type to our data type string
func parquetTypeToDataType(t parquet.Type) string {
	switch t.Kind() {
	case parquet.Boolean:
		return "boolean"
	case parquet.Int32, parquet.Int64:
		return "int64"
	case parquet.Float, parquet.Double:
		return "float64"
	case parquet.ByteArray, parquet.FixedLenByteArray:
		return "string"
	default:
		return "string"
	}
}

// extractColumnsFromCSV extracts column names and infers data types from CSV
func (s *DatasourceServiceImpl) extractColumnsFromCSV(reader io.Reader) ([]domain.Column, error) {
	csvReader := csv.NewReader(reader)
	headers, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Read sample rows to infer types (up to 100 rows)
	const maxSampleRows = 100
	sampleData := make([][]string, 0, maxSampleRows)
	for i := 0; i < maxSampleRows; i++ {
		row, err := csvReader.Read()
		if err != nil {
			break // EOF or error, stop reading
		}
		sampleData = append(sampleData, row)
	}

	columns := make([]domain.Column, len(headers))
	for i, header := range headers {
		dataType := inferColumnType(sampleData, i)
		columns[i] = domain.Column{
			Name:     strings.TrimSpace(header),
			DataType: dataType,
			Role:     domain.ColumnRoleInput,
		}
	}

	return columns, nil
}

// CreateFromExistingFile creates a datasource pointing to an existing file in MinIO
// Used for scored output files that are already saved to MinIO by compute service
func (s *DatasourceServiceImpl) CreateFromExistingFile(collectionID, name, filePath string, rowCount int, createdBy string) (*dto.DatasourceResponse, error) {
	// Verify collection exists
	collection, err := s.collectionRepo.GetByID(collectionID)
	if err != nil {
		return nil, fmt.Errorf("collection not found: %w", err)
	}

	// Check name uniqueness within collection
	existingNames, err := s.datasourceRepo.GetNamesInCollection(collectionID)
	if err != nil {
		return nil, err
	}
	if err := s.domainService.ValidateDatasourceNameUnique(name, existingNames); err != nil {
		return nil, err
	}

	// Determine type from file extension
	dsType := domain.DatasourceTypeParquet
	if strings.HasSuffix(strings.ToLower(filePath), ".csv") {
		dsType = domain.DatasourceTypeCSV
	}

	// Create datasource entity
	datasource := &domain.Datasource{
		CollectionID: collectionID,
		Name:         name,
		Description:  fmt.Sprintf("Scored output from model (rows: %d)", rowCount),
		Type:         dsType,
		FilePath:     filePath,
		CreatedBy:    createdBy,
	}

	// Create datasource in database
	if err := s.datasourceRepo.Create(datasource); err != nil {
		logger.Error("Failed to create datasource from existing file: %v", err)
		return nil, err
	}

	// Extract columns if possible (for parquet files)
	if dsType == domain.DatasourceTypeParquet {
		columns, err := s.extractColumnsFromFile(dsType, filePath)
		if err != nil {
			logger.Warn("Failed to extract columns from scored parquet file: %v", err)
		} else if len(columns) > 0 {
			for i := range columns {
				columns[i].DatasourceID = datasource.ID
			}
			if err := s.columnRepo.CreateBatch(columns); err != nil {
				logger.Warn("Failed to create columns: %v", err)
			}
		}
	}

	logger.Audit(createdBy, "create", "datasource", datasource.ID, "success", nil)
	return toDatasourceResponse(datasource, collection.Name), nil
}

// inferColumnType infers the data type of a column from sample values
func inferColumnType(rows [][]string, colIndex int) string {
	if len(rows) == 0 {
		return "string"
	}

	var hasInt, hasFloat, hasBool, hasString bool
	nonEmptyCount := 0

	for _, row := range rows {
		if colIndex >= len(row) {
			continue
		}
		val := strings.TrimSpace(row[colIndex])
		if val == "" {
			continue // Skip empty values
		}
		nonEmptyCount++

		// Try to parse as different types
		if _, err := strconv.ParseInt(val, 10, 64); err == nil {
			hasInt = true
		} else if _, err := strconv.ParseFloat(val, 64); err == nil {
			hasFloat = true
		} else if strings.EqualFold(val, "true") || strings.EqualFold(val, "false") {
			hasBool = true
		} else {
			hasString = true
		}
	}

	// If any value is a non-numeric string, the column is string type
	if hasString {
		return "string"
	}
	// If we have booleans and nothing else, it's boolean
	if hasBool && !hasInt && !hasFloat {
		return "boolean"
	}
	// If we have any floats, the column is float (even if some look like ints)
	if hasFloat {
		return "float64"
	}
	// If all values are integers
	if hasInt {
		return "int64"
	}
	// Default to string if no data or all empty
	return "string"
}

// GetDataPreview returns a preview of the datasource data
func (s *DatasourceServiceImpl) GetDataPreview(id string, limit int) (*dto.DataPreviewResponse, error) {
	// Default limit to 100 rows
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	// Get datasource to find file path
	datasource, err := s.datasourceRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if datasource.FilePath == "" {
		return nil, fmt.Errorf("datasource has no associated file")
	}

	// Read file from MinIO
	reader, _, err := s.fileService.Get(datasource.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read datasource file: %w", err)
	}
	defer reader.Close()

	// Parse based on type
	switch datasource.Type {
	case domain.DatasourceTypeCSV:
		return s.parseCSVPreview(reader, limit)
	case domain.DatasourceTypeParquet:
		// For parquet, we need the full file content
		content, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read parquet file: %w", err)
		}
		return s.parseParquetPreview(content, limit)
	default:
		return nil, fmt.Errorf("unsupported file type for preview: %s", datasource.Type)
	}
}

// parseCSVPreview parses CSV file and returns preview data
func (s *DatasourceServiceImpl) parseCSVPreview(reader io.Reader, limit int) (*dto.DataPreviewResponse, error) {
	csvReader := csv.NewReader(reader)
	
	// Read header
	headers, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Read rows up to limit
	rows := make([]map[string]interface{}, 0, limit)
	totalRows := 0
	for {
		record, err := csvReader.Read()
		if err != nil {
			break // EOF or error
		}
		totalRows++
		
		if len(rows) < limit {
			row := make(map[string]interface{})
			for i, header := range headers {
				if i < len(record) {
					row[header] = record[i]
				}
			}
			rows = append(rows, row)
		}
	}

	return &dto.DataPreviewResponse{
		Columns:    headers,
		Rows:       rows,
		TotalRows:  totalRows,
		PreviewMax: limit,
	}, nil
}

// parseParquetPreview parses Parquet file and returns preview data
func (s *DatasourceServiceImpl) parseParquetPreview(content []byte, limit int) (*dto.DataPreviewResponse, error) {
	// Create reader from bytes
	reader := bytes.NewReader(content)
	file, err := parquet.OpenFile(reader, int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to open parquet file: %w", err)
	}

	// Get schema to extract column names
	schema := file.Schema()
	columns := make([]string, 0, len(schema.Fields()))
	for _, field := range schema.Fields() {
		columns = append(columns, field.Name())
	}

	totalRows := int(file.NumRows())
	rowsToRead := limit
	if totalRows < limit {
		rowsToRead = totalRows
	}

	// Initialize rows with empty maps
	rows := make([]map[string]interface{}, rowsToRead)
	for i := range rows {
		rows[i] = make(map[string]interface{})
	}

	// Read data column by column - this handles optional/nullable fields better
	for colIdx, colName := range columns {
		rowIdx := 0

		for _, rowGroup := range file.RowGroups() {
			if rowIdx >= rowsToRead {
				break
			}

			chunks := rowGroup.ColumnChunks()
			if colIdx >= len(chunks) {
				continue
			}

			chunk := chunks[colIdx]
			pages := chunk.Pages()

			for rowIdx < rowsToRead {
				page, err := pages.ReadPage()
				if err == io.EOF {
					break
				}
				if err != nil {
					break // Skip problematic pages
				}

				values := page.Values()
				valueBuf := make([]parquet.Value, 100)

				for rowIdx < rowsToRead {
					n, err := values.ReadValues(valueBuf)
					if n == 0 || (err != nil && err != io.EOF) {
						break
					}

					for i := 0; i < n && rowIdx < rowsToRead; i++ {
						rows[rowIdx][colName] = parquetValueToGo(valueBuf[i])
						rowIdx++
					}
				}
			}
			pages.Close()
		}
	}

	return &dto.DataPreviewResponse{
		Columns:    columns,
		Rows:       rows,
		TotalRows:  totalRows,
		PreviewMax: limit,
	}, nil
}

// parquetValueToGo converts a parquet.Value to a native Go value
func parquetValueToGo(v parquet.Value) interface{} {
	if v.IsNull() {
		return nil
	}

	switch v.Kind() {
	case parquet.Boolean:
		return v.Boolean()
	case parquet.Int32:
		return v.Int32()
	case parquet.Int64:
		return v.Int64()
	case parquet.Float:
		return v.Float()
	case parquet.Double:
		return v.Double()
	case parquet.ByteArray, parquet.FixedLenByteArray:
		return string(v.ByteArray())
	default:
		return v.String()
	}
}

