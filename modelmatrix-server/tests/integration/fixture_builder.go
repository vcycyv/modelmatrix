package integration

import (
	"testing"

	dsmodel "modelmatrix-server/internal/module/datasource/model"

	"github.com/stretchr/testify/require"
)

// integrationTestUser matches the LDAP test user used in tests/testdata/test-users.ldif.
const integrationTestUser = "michael.jordan"

// CollectionBuilder builds a CollectionModel for direct DB inserts (faster than HTTP for test setup).
type CollectionBuilder struct {
	t           *testing.T
	name        string
	description string
	createdBy   string
}

// NewCollectionBuilder starts a fluent builder. Name defaults if empty.
func NewCollectionBuilder(t *testing.T) *CollectionBuilder {
	t.Helper()
	return &CollectionBuilder{
		t:           t,
		name:        "fixture-collection",
		description: "seeded via fixture builder",
		createdBy:   integrationTestUser,
	}
}

// WithName sets the collection name (must be unique when not truncating between tests).
func (b *CollectionBuilder) WithName(name string) *CollectionBuilder {
	b.name = name
	return b
}

// WithDescription sets the description.
func (b *CollectionBuilder) WithDescription(desc string) *CollectionBuilder {
	b.description = desc
	return b
}

// WithCreatedBy sets created_by (defaults to integrationTestUser).
func (b *CollectionBuilder) WithCreatedBy(user string) *CollectionBuilder {
	b.createdBy = user
	return b
}

// Build inserts the collection and returns it with ID populated.
func (b *CollectionBuilder) Build() *dsmodel.CollectionModel {
	b.t.Helper()
	c := &dsmodel.CollectionModel{
		Name:        b.name,
		Description: b.description,
		CreatedBy:   b.createdBy,
	}
	dbInsert(b.t, c)
	require.NotEmpty(b.t, c.ID)
	return c
}

// DatasourceBuilder builds a DatasourceModel under a collection.
type DatasourceBuilder struct {
	t            *testing.T
	collectionID string
	name         string
	description  string
	dsType       string
	filePath     string
	createdBy    string
}

// NewDatasourceBuilder creates a builder for a datasource belonging to collectionID.
func NewDatasourceBuilder(t *testing.T, collectionID string) *DatasourceBuilder {
	t.Helper()
	return &DatasourceBuilder{
		t:            t,
		collectionID: collectionID,
		name:         "fixture-datasource",
		description:  "seeded via fixture builder",
		dsType:       "csv",
		filePath:     "fixtures/placeholder.parquet",
		createdBy:    integrationTestUser,
	}
}

func (b *DatasourceBuilder) WithName(name string) *DatasourceBuilder {
	b.name = name
	return b
}

func (b *DatasourceBuilder) WithDescription(desc string) *DatasourceBuilder {
	b.description = desc
	return b
}

func (b *DatasourceBuilder) WithType(dsType string) *DatasourceBuilder {
	b.dsType = dsType
	return b
}

func (b *DatasourceBuilder) WithFilePath(path string) *DatasourceBuilder {
	b.filePath = path
	return b
}

// Build inserts the datasource and returns it with ID populated.
func (b *DatasourceBuilder) Build() *dsmodel.DatasourceModel {
	b.t.Helper()
	d := &dsmodel.DatasourceModel{
		CollectionID: b.collectionID,
		Name:         b.name,
		Description:  b.description,
		Type:         b.dsType,
		FilePath:     b.filePath,
		CreatedBy:    b.createdBy,
	}
	dbInsert(b.t, d)
	require.NotEmpty(b.t, d.ID)
	return d
}

// ColumnBuilder builds a ColumnModel for a datasource.
type ColumnBuilder struct {
	t            *testing.T
	datasourceID string
	name         string
	dataType     string
	role         string
}

// NewColumnBuilder creates a column row for datasourceID.
func NewColumnBuilder(t *testing.T, datasourceID string) *ColumnBuilder {
	t.Helper()
	return &ColumnBuilder{
		t:            t,
		datasourceID: datasourceID,
		name:         "feature_a",
		dataType:     "numeric",
		role:         "input",
	}
}

func (b *ColumnBuilder) WithName(name string) *ColumnBuilder {
	b.name = name
	return b
}

func (b *ColumnBuilder) WithDataType(dt string) *ColumnBuilder {
	b.dataType = dt
	return b
}

func (b *ColumnBuilder) WithRole(role string) *ColumnBuilder {
	b.role = role
	return b
}

// Build inserts the column.
func (b *ColumnBuilder) Build() *dsmodel.ColumnModel {
	b.t.Helper()
	col := &dsmodel.ColumnModel{
		DatasourceID: b.datasourceID,
		Name:         b.name,
		DataType:     b.dataType,
		Role:         b.role,
	}
	dbInsert(b.t, col)
	require.NotEmpty(b.t, col.ID)
	return col
}
