package domain

import (
	"errors"
	"strings"
)

// Domain errors
var (
	ErrCollectionNameEmpty      = errors.New("collection name cannot be empty")
	ErrCollectionNameExists     = errors.New("collection name already exists")
	ErrDatasourceNameEmpty      = errors.New("datasource name cannot be empty")
	ErrDatasourceNameExists     = errors.New("datasource name already exists in this collection")
	ErrInvalidDatasourceType    = errors.New("invalid datasource type")
	ErrInvalidColumnRole        = errors.New("invalid column role")
	ErrMultipleTargetColumns    = errors.New("only one target column is allowed per datasource")
	ErrColumnNotFound           = errors.New("column not found")
	ErrDatasourceNotFound       = errors.New("datasource not found")
	ErrCollectionNotFound       = errors.New("collection not found")
	ErrCollectionHasDatasources = errors.New("cannot delete collection with existing datasources")
	ErrFilePathRequired         = errors.New("file path is required for file-based datasources")
	ErrConnectionConfigRequired = errors.New("connection config is required for database datasources")
)

// Service provides domain business logic for datasources (Domain Service)
// This layer contains ONLY business rules and operates ONLY on Domain Entities
// It has NO knowledge of DTOs, Repositories, or Infrastructure
type Service struct{}

// NewService creates a new domain service
func NewService() *Service {
	return &Service{}
}

// ValidateCollection validates a collection entity
func (s *Service) ValidateCollection(collection *Collection) error {
	if strings.TrimSpace(collection.Name) == "" {
		return ErrCollectionNameEmpty
	}
	return nil
}

// ValidateCollectionNameUnique checks if collection name is unique
// existingNames should be provided by the application layer
func (s *Service) ValidateCollectionNameUnique(name string, existingNames []string) error {
	for _, existing := range existingNames {
		if strings.EqualFold(existing, name) {
			return ErrCollectionNameExists
		}
	}
	return nil
}

// ValidateDatasource validates a datasource entity
func (s *Service) ValidateDatasource(datasource *Datasource) error {
	if strings.TrimSpace(datasource.Name) == "" {
		return ErrDatasourceNameEmpty
	}

	if !datasource.Type.IsValid() {
		return ErrInvalidDatasourceType
	}

	// Validate based on type
	if datasource.Type.IsFile() {
		if strings.TrimSpace(datasource.FilePath) == "" {
			return ErrFilePathRequired
		}
	} else {
		if datasource.ConnectionConfig == nil {
			return ErrConnectionConfigRequired
		}
	}

	return nil
}

// ValidateDatasourceNameUnique checks if datasource name is unique within collection
func (s *Service) ValidateDatasourceNameUnique(name string, existingNames []string) error {
	for _, existing := range existingNames {
		if strings.EqualFold(existing, name) {
			return ErrDatasourceNameExists
		}
	}
	return nil
}

// ValidateColumn validates a column entity
func (s *Service) ValidateColumn(column *Column) error {
	if !column.Role.IsValid() {
		return ErrInvalidColumnRole
	}
	return nil
}

// ValidateColumnRoles validates column roles for a datasource
// Ensures only one target column exists
func (s *Service) ValidateColumnRoles(datasource *Datasource) error {
	targetCount := datasource.CountTargetColumns()
	if targetCount > 1 {
		return ErrMultipleTargetColumns
	}
	return nil
}

// CanSetColumnAsTarget checks if a column can be set as target
func (s *Service) CanSetColumnAsTarget(datasource *Datasource, columnName string) error {
	// Check if there's already a target column (that's not the one being set)
	for _, col := range datasource.Columns {
		if col.Role == ColumnRoleTarget && col.Name != columnName {
			return ErrMultipleTargetColumns
		}
	}
	return nil
}

// SetColumnRole sets a column's role with validation
func (s *Service) SetColumnRole(datasource *Datasource, columnName string, role ColumnRole) error {
	if !role.IsValid() {
		return ErrInvalidColumnRole
	}

	// If setting as target, check for existing target
	if role == ColumnRoleTarget {
		if err := s.CanSetColumnAsTarget(datasource, columnName); err != nil {
			return err
		}
	}

	// Find and update column
	found := false
	for i := range datasource.Columns {
		if datasource.Columns[i].Name == columnName {
			datasource.Columns[i].Role = role
			found = true
			break
		}
	}

	if !found {
		return ErrColumnNotFound
	}

	return nil
}

// CanDeleteCollection checks if a collection can be deleted
func (s *Service) CanDeleteCollection(datasourceCount int) error {
	if datasourceCount > 0 {
		return ErrCollectionHasDatasources
	}
	return nil
}

// ValidateForTraining checks if datasource is ready for ML training
func (s *Service) ValidateForTraining(datasource *Datasource) error {
	// Must have at least one target column
	if datasource.GetTargetColumn() == nil {
		return errors.New("datasource must have a target column for training")
	}

	// Must have at least one input column
	if len(datasource.GetInputColumns()) == 0 {
		return errors.New("datasource must have at least one input column for training")
	}

	return nil
}

