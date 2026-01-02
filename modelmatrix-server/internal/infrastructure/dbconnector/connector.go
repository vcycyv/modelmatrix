package dbconnector

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"

	"modelmatrix-server/internal/module/datasource/domain"
	"modelmatrix-server/pkg/logger"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// ExternalDBConnectorImpl implements ExternalDBConnector
type ExternalDBConnectorImpl struct{}

// NewExternalDBConnector creates a new external database connector
func NewExternalDBConnector() *ExternalDBConnectorImpl {
	return &ExternalDBConnectorImpl{}
}

// FetchTableData connects to an external database and returns the data as CSV bytes
func (c *ExternalDBConnectorImpl) FetchTableData(config *domain.ConnectionConfig) ([]byte, []domain.Column, error) {
	// Determine driver based on port or explicit type detection
	driver := c.detectDriver(config)

	// Build DSN
	dsn := c.buildDSN(driver, config)

	// Connect to database
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Connected to external database: %s:%d/%s", config.Host, config.Port, config.Database)

	// Build query
	tableName := config.Table
	if config.Schema != "" && driver == "postgres" {
		tableName = fmt.Sprintf("%s.%s", config.Schema, config.Table)
	}
	query := fmt.Sprintf("SELECT * FROM %s", tableName)

	// Execute query
	rows, err := db.Query(query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column info
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get columns: %w", err)
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get column types: %w", err)
	}

	// Build domain columns
	columns := make([]domain.Column, len(columnNames))
	for i, name := range columnNames {
		dataType := "string"
		if i < len(columnTypes) && columnTypes[i] != nil {
			dataType = c.mapDBType(columnTypes[i].DatabaseTypeName())
		}
		columns[i] = domain.Column{
			Name:     name,
			DataType: dataType,
			Role:     domain.ColumnRoleInput,
		}
	}

	// Write to CSV buffer
	var buf bytes.Buffer
	csvWriter := csv.NewWriter(&buf)

	// Write header
	if err := csvWriter.Write(columnNames); err != nil {
		return nil, nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Prepare value holders
	values := make([]interface{}, len(columnNames))
	valuePtrs := make([]interface{}, len(columnNames))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Write data rows
	rowCount := 0
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make([]string, len(columnNames))
		for i, val := range values {
			row[i] = c.formatValue(val)
		}

		if err := csvWriter.Write(row); err != nil {
			return nil, nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error during row iteration: %w", err)
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return nil, nil, fmt.Errorf("CSV writer error: %w", err)
	}

	logger.Info("Fetched %d rows with %d columns from table %s", rowCount, len(columnNames), config.Table)

	return buf.Bytes(), columns, nil
}

// detectDriver determines the database driver based on port
func (c *ExternalDBConnectorImpl) detectDriver(config *domain.ConnectionConfig) string {
	switch config.Port {
	case 3306:
		return "mysql"
	case 5432:
		return "postgres"
	default:
		// Default to postgres
		return "postgres"
	}
}

// buildDSN builds the connection string
func (c *ExternalDBConnectorImpl) buildDSN(driver string, config *domain.ConnectionConfig) string {
	switch driver {
	case "mysql":
		// MySQL DSN format: user:password@tcp(host:port)/database
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			config.Username, config.Password, config.Host, config.Port, config.Database)
	case "postgres":
		// PostgreSQL DSN format
		sslMode := config.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			config.Host, config.Port, config.Username, config.Password, config.Database, sslMode)
	default:
		return ""
	}
}

// mapDBType maps database column types to simple type names
func (c *ExternalDBConnectorImpl) mapDBType(dbType string) string {
	switch dbType {
	case "INT", "INT4", "INT8", "BIGINT", "SMALLINT", "TINYINT", "INTEGER":
		return "integer"
	case "FLOAT", "FLOAT4", "FLOAT8", "DOUBLE", "REAL", "NUMERIC", "DECIMAL":
		return "float"
	case "BOOL", "BOOLEAN":
		return "boolean"
	case "DATE", "TIME", "TIMESTAMP", "TIMESTAMPTZ", "DATETIME":
		return "datetime"
	case "VARCHAR", "TEXT", "CHAR", "BPCHAR", "NAME":
		return "string"
	case "JSON", "JSONB":
		return "json"
	default:
		return "string"
	}
}

// formatValue converts a database value to string for CSV
func (c *ExternalDBConnectorImpl) formatValue(val interface{}) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case []byte:
		return string(v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

