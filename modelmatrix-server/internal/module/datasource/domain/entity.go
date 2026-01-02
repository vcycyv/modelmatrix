package domain

import (
	"time"
)

// ColumnRole represents the role of a column in a datasource
type ColumnRole string

const (
	ColumnRoleTarget ColumnRole = "target"
	ColumnRoleInput  ColumnRole = "input"
	ColumnRoleOutput ColumnRole = "output"
	ColumnRoleIgnore ColumnRole = "ignore"
)

// IsValid checks if the column role is valid
func (r ColumnRole) IsValid() bool {
	switch r {
	case ColumnRoleTarget, ColumnRoleInput, ColumnRoleOutput, ColumnRoleIgnore:
		return true
	default:
		return false
	}
}

// DatasourceType represents the type of datasource
type DatasourceType string

const (
	DatasourceTypePostgreSQL DatasourceType = "postgresql"
	DatasourceTypeMySQL      DatasourceType = "mysql"
	DatasourceTypeCSV        DatasourceType = "csv"
	DatasourceTypeParquet    DatasourceType = "parquet"
)

// IsValid checks if the datasource type is valid
func (t DatasourceType) IsValid() bool {
	switch t {
	case DatasourceTypePostgreSQL, DatasourceTypeMySQL, DatasourceTypeCSV, DatasourceTypeParquet:
		return true
	default:
		return false
	}
}

// IsFile returns true if datasource type is file-based
func (t DatasourceType) IsFile() bool {
	return t == DatasourceTypeCSV || t == DatasourceTypeParquet
}

// Collection represents a logical group of datasources (Domain Entity)
type Collection struct {
	ID          string
	Name        string
	Description string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Datasource represents a data source (Domain Entity)
type Datasource struct {
	ID               string
	CollectionID     string
	Name             string
	Description      string
	Type             DatasourceType
	FilePath         string            // For file-based datasources
	ConnectionConfig *ConnectionConfig // For database datasources
	Columns          []Column
	CreatedBy        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ConnectionConfig holds database connection details
type ConnectionConfig struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	Schema   string
	Table    string
	SSLMode  string
}

// Column represents a column in a datasource (Domain Entity)
type Column struct {
	ID           string
	DatasourceID string
	Name         string
	DataType     string
	Role         ColumnRole
	Description  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// GetTargetColumn returns the target column if exists
func (d *Datasource) GetTargetColumn() *Column {
	for i := range d.Columns {
		if d.Columns[i].Role == ColumnRoleTarget {
			return &d.Columns[i]
		}
	}
	return nil
}

// GetInputColumns returns all input columns
func (d *Datasource) GetInputColumns() []Column {
	var inputs []Column
	for _, col := range d.Columns {
		if col.Role == ColumnRoleInput {
			inputs = append(inputs, col)
		}
	}
	return inputs
}

// CountTargetColumns counts the number of target columns
func (d *Datasource) CountTargetColumns() int {
	count := 0
	for _, col := range d.Columns {
		if col.Role == ColumnRoleTarget {
			count++
		}
	}
	return count
}

// HasColumn checks if datasource has a column with given name
func (d *Datasource) HasColumn(name string) bool {
	for _, col := range d.Columns {
		if col.Name == name {
			return true
		}
	}
	return false
}

