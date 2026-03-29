package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCollection_EmptyName(t *testing.T) {
	svc := NewService()
	err := svc.ValidateCollection(&Collection{Name: ""})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCollectionNameEmpty)
}

func TestValidateCollection_ValidName(t *testing.T) {
	svc := NewService()
	err := svc.ValidateCollection(&Collection{Name: "My Collection"})
	assert.NoError(t, err)
}

func TestValidateCollectionNameUnique(t *testing.T) {
	svc := NewService()
	existing := []string{"Alpha", "Beta"}

	assert.NoError(t, svc.ValidateCollectionNameUnique("Gamma", existing))
	assert.ErrorIs(t, svc.ValidateCollectionNameUnique("Alpha", existing), ErrCollectionNameExists)
	// Case-insensitive
	assert.ErrorIs(t, svc.ValidateCollectionNameUnique("alpha", existing), ErrCollectionNameExists)
}

func TestValidateDatasource_EmptyName(t *testing.T) {
	svc := NewService()
	ds := &Datasource{Name: "", Type: DatasourceTypeCSV, FilePath: "path/to/file.csv"}
	err := svc.ValidateDatasource(ds)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDatasourceNameEmpty)
}

func TestValidateDatasource_InvalidType(t *testing.T) {
	svc := NewService()
	ds := &Datasource{Name: "DS", Type: "invalid_type", FilePath: "f.csv"}
	err := svc.ValidateDatasource(ds)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidDatasourceType)
}

func TestValidateDatasource_CSV_MissingFilePath(t *testing.T) {
	svc := NewService()
	ds := &Datasource{Name: "DS", Type: DatasourceTypeCSV, FilePath: ""}
	err := svc.ValidateDatasource(ds)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFilePathRequired)
}

func TestValidateDatasource_PostgreSQL_MissingConnectionConfig(t *testing.T) {
	svc := NewService()
	ds := &Datasource{Name: "DS", Type: DatasourceTypePostgreSQL, ConnectionConfig: nil}
	err := svc.ValidateDatasource(ds)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrConnectionConfigRequired)
}

func TestValidateDatasource_PostgreSQL_WithConfig(t *testing.T) {
	svc := NewService()
	ds := &Datasource{
		Name: "DS",
		Type: DatasourceTypePostgreSQL,
		ConnectionConfig: &ConnectionConfig{
			Host: "localhost", Port: 5432, Database: "db", Username: "user",
		},
	}
	err := svc.ValidateDatasource(ds)
	assert.NoError(t, err)
}

func TestValidateDatasourceNameUnique(t *testing.T) {
	svc := NewService()
	existing := []string{"Sales", "HR"}
	assert.NoError(t, svc.ValidateDatasourceNameUnique("Finance", existing))
	assert.ErrorIs(t, svc.ValidateDatasourceNameUnique("Sales", existing), ErrDatasourceNameExists)
}

func TestValidateColumn_ValidRole(t *testing.T) {
	svc := NewService()
	col := &Column{Role: ColumnRoleInput}
	assert.NoError(t, svc.ValidateColumn(col))
}

func TestValidateColumn_InvalidRole(t *testing.T) {
	svc := NewService()
	col := &Column{Role: "garbage"}
	err := svc.ValidateColumn(col)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidColumnRole)
}

func TestValidateColumnRoles_MultipleTargets(t *testing.T) {
	svc := NewService()
	ds := &Datasource{
		Columns: []Column{
			{Name: "a", Role: ColumnRoleTarget},
			{Name: "b", Role: ColumnRoleTarget},
		},
	}
	err := svc.ValidateColumnRoles(ds)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMultipleTargetColumns)
}

func TestValidateColumnRoles_SingleTarget(t *testing.T) {
	svc := NewService()
	ds := &Datasource{
		Columns: []Column{
			{Name: "a", Role: ColumnRoleInput},
			{Name: "b", Role: ColumnRoleTarget},
		},
	}
	assert.NoError(t, svc.ValidateColumnRoles(ds))
}

func TestCanDeleteCollection_WithDatasources(t *testing.T) {
	svc := NewService()
	err := svc.CanDeleteCollection(3)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCollectionHasDatasources)
}

func TestCanDeleteCollection_NoDatasources(t *testing.T) {
	svc := NewService()
	assert.NoError(t, svc.CanDeleteCollection(0))
}

func TestSetColumnRole_NotFound(t *testing.T) {
	svc := NewService()
	ds := &Datasource{
		Columns: []Column{{Name: "a", Role: ColumnRoleInput}},
	}
	err := svc.SetColumnRole(ds, "nonexistent", ColumnRoleTarget)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrColumnNotFound)
}

func TestSetColumnRole_TargetWhenAlreadyExists(t *testing.T) {
	svc := NewService()
	ds := &Datasource{
		Columns: []Column{
			{Name: "target_col", Role: ColumnRoleTarget},
			{Name: "other_col", Role: ColumnRoleInput},
		},
	}
	// Setting another column as target should fail because one already exists
	err := svc.SetColumnRole(ds, "other_col", ColumnRoleTarget)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMultipleTargetColumns)
}

// ---------------------------------------------------------------------------
// Datasource entity methods
// ---------------------------------------------------------------------------

func datasourceWithColumns(cols ...Column) *Datasource {
	return &Datasource{Columns: cols}
}

func col(name string, role ColumnRole) Column {
	return Column{Name: name, Role: role}
}

func TestDatasource_GetTargetColumn_Found(t *testing.T) {
	ds := datasourceWithColumns(col("age", ColumnRoleInput), col("churn", ColumnRoleTarget))
	target := ds.GetTargetColumn()
	require.NotNil(t, target)
	assert.Equal(t, "churn", target.Name)
}

func TestDatasource_GetTargetColumn_NotFound(t *testing.T) {
	ds := datasourceWithColumns(col("age", ColumnRoleInput))
	assert.Nil(t, ds.GetTargetColumn())
}

func TestDatasource_GetInputColumns(t *testing.T) {
	ds := datasourceWithColumns(col("a", ColumnRoleInput), col("b", ColumnRoleTarget), col("c", ColumnRoleInput))
	inputs := ds.GetInputColumns()
	assert.Len(t, inputs, 2)
}

func TestDatasource_GetInputColumns_Empty(t *testing.T) {
	ds := datasourceWithColumns(col("b", ColumnRoleTarget))
	assert.Empty(t, ds.GetInputColumns())
}

func TestDatasource_HasColumn_True(t *testing.T) {
	ds := datasourceWithColumns(col("revenue", ColumnRoleInput))
	assert.True(t, ds.HasColumn("revenue"))
}

func TestDatasource_HasColumn_False(t *testing.T) {
	ds := datasourceWithColumns(col("revenue", ColumnRoleInput))
	assert.False(t, ds.HasColumn("unknown"))
}

// ---------------------------------------------------------------------------
// ValidateForTraining
// ---------------------------------------------------------------------------

func TestValidateForTraining_Valid(t *testing.T) {
	svc := NewService()
	ds := datasourceWithColumns(col("feature", ColumnRoleInput), col("label", ColumnRoleTarget))
	require.NoError(t, svc.ValidateForTraining(ds))
}

func TestValidateForTraining_NoTarget(t *testing.T) {
	svc := NewService()
	ds := datasourceWithColumns(col("feature", ColumnRoleInput))
	err := svc.ValidateForTraining(ds)
	require.Error(t, err)
}

func TestValidateForTraining_NoInput(t *testing.T) {
	svc := NewService()
	ds := datasourceWithColumns(col("label", ColumnRoleTarget))
	err := svc.ValidateForTraining(ds)
	require.Error(t, err)
}
