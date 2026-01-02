package application

import (
	"fmt"

	"modelmatrix-server/internal/module/datasource/domain"
	"modelmatrix-server/internal/module/datasource/dto"
	"modelmatrix-server/internal/module/datasource/repository"
	"modelmatrix-server/pkg/logger"

	"gorm.io/gorm"
)

// ColumnServiceImpl implements ColumnService
type ColumnServiceImpl struct {
	db             *gorm.DB
	columnRepo     repository.ColumnRepository
	datasourceRepo repository.DatasourceRepository
	domainService  *domain.Service
}

// NewColumnService creates a new column service
func NewColumnService(
	db *gorm.DB,
	columnRepo repository.ColumnRepository,
	datasourceRepo repository.DatasourceRepository,
	domainService *domain.Service,
) ColumnService {
	return &ColumnServiceImpl{
		db:             db,
		columnRepo:     columnRepo,
		datasourceRepo: datasourceRepo,
		domainService:  domainService,
	}
}

// GetByDatasourceID retrieves all columns for a datasource
func (s *ColumnServiceImpl) GetByDatasourceID(datasourceID string) ([]dto.ColumnResponse, error) {
	// Verify datasource exists
	if _, err := s.datasourceRepo.GetByID(datasourceID); err != nil {
		return nil, err
	}

	columns, err := s.columnRepo.GetByDatasourceID(datasourceID)
	if err != nil {
		return nil, err
	}

	return toColumnResponseList(columns), nil
}

// UpdateRole updates a column's role
func (s *ColumnServiceImpl) UpdateRole(datasourceID, columnID string, role string) (*dto.ColumnResponse, error) {
	// Verify datasource exists
	datasource, err := s.datasourceRepo.GetByIDWithColumns(datasourceID)
	if err != nil {
		return nil, err
	}

	// Verify column belongs to datasource
	column, err := s.columnRepo.GetByID(columnID)
	if err != nil {
		return nil, err
	}

	if column.DatasourceID != datasourceID {
		return nil, fmt.Errorf("column does not belong to this datasource")
	}

	// Convert role string to domain type
	columnRole := domain.ColumnRole(role)
	if !columnRole.IsValid() {
		return nil, domain.ErrInvalidColumnRole
	}

	// Validate using domain service
	if err := s.domainService.SetColumnRole(datasource, column.Name, columnRole); err != nil {
		return nil, err
	}

	// Update via repository
	if err := s.columnRepo.UpdateRole(columnID, columnRole); err != nil {
		logger.Error("Failed to update column role: %v", err)
		return nil, err
	}

	// Refresh column data
	column, err = s.columnRepo.GetByID(columnID)
	if err != nil {
		return nil, err
	}

	return toColumnResponse(column), nil
}

// BulkUpdateRoles updates multiple column roles
func (s *ColumnServiceImpl) BulkUpdateRoles(datasourceID string, req *dto.BulkUpdateColumnRolesRequest) ([]dto.ColumnResponse, error) {
	// Verify datasource exists
	datasource, err := s.datasourceRepo.GetByIDWithColumns(datasourceID)
	if err != nil {
		return nil, err
	}

	// Build a map of column ID to column
	columnMap := make(map[string]*domain.Column)
	for i := range datasource.Columns {
		columnMap[datasource.Columns[i].ID] = &datasource.Columns[i]
	}

	// Validate all updates first
	for _, update := range req.Columns {
		col, exists := columnMap[update.ColumnID]
		if !exists {
			return nil, fmt.Errorf("column %s not found in datasource", update.ColumnID)
		}

		columnRole := domain.ColumnRole(update.Role)
		if !columnRole.IsValid() {
			return nil, domain.ErrInvalidColumnRole
		}

		// Apply role change to domain entity for validation
		col.Role = columnRole
	}

	// Validate column roles (e.g., only one target)
	if err := s.domainService.ValidateColumnRoles(datasource); err != nil {
		return nil, err
	}

	// Apply updates in transaction
	err = s.db.Transaction(func(tx *gorm.DB) error {
		for _, update := range req.Columns {
			if err := s.columnRepo.UpdateRole(update.ColumnID, domain.ColumnRole(update.Role)); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		logger.Error("Failed to bulk update column roles: %v", err)
		return nil, err
	}

	// Refresh columns
	columns, err := s.columnRepo.GetByDatasourceID(datasourceID)
	if err != nil {
		return nil, err
	}

	return toColumnResponseList(columns), nil
}

// CreateColumns creates multiple columns for a datasource
func (s *ColumnServiceImpl) CreateColumns(datasourceID string, req *dto.CreateColumnsRequest) ([]dto.ColumnResponse, error) {
	// Verify datasource exists
	datasource, err := s.datasourceRepo.GetByIDWithColumns(datasourceID)
	if err != nil {
		return nil, err
	}

	// Convert DTOs to domain entities
	columns := make([]domain.Column, len(req.Columns))
	for i, col := range req.Columns {
		columnRole := domain.ColumnRole(col.Role)
		if !columnRole.IsValid() {
			return nil, domain.ErrInvalidColumnRole
		}

		// Check for duplicate names
		if datasource.HasColumn(col.Name) {
			return nil, fmt.Errorf("column %s already exists in datasource", col.Name)
		}

		columns[i] = domain.Column{
			DatasourceID: datasourceID,
			Name:         col.Name,
			DataType:     col.DataType,
			Role:         columnRole,
			Description:  col.Description,
		}
	}

	// Add columns to datasource for validation
	datasource.Columns = append(datasource.Columns, columns...)

	// Validate column roles
	if err := s.domainService.ValidateColumnRoles(datasource); err != nil {
		return nil, err
	}

	// Create via repository
	if err := s.columnRepo.CreateBatch(columns); err != nil {
		logger.Error("Failed to create columns: %v", err)
		return nil, err
	}

	return toColumnResponseList(columns), nil
}

