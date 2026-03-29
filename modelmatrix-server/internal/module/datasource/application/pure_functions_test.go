package application

// Tests for pure/IO functions in datasource_service.go.
// These have zero infrastructure coupling — they take plain data or io.Reader.
// inferColumnType is the core column type inference logic; bugs here silently corrupt schema.
// parseCSVPreview / extractColumnsFromCSV are called on every file upload — critical path.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// inferColumnType — pure function, no receiver needed
// ---------------------------------------------------------------------------

func TestInferColumnType_EmptyRows(t *testing.T) {
	assert.Equal(t, "string", inferColumnType(nil, 0))
	assert.Equal(t, "string", inferColumnType([][]string{}, 0))
}

func TestInferColumnType_IntegerColumn(t *testing.T) {
	rows := [][]string{{"1"}, {"42"}, {"100"}, {"-7"}}
	assert.Equal(t, "int64", inferColumnType(rows, 0))
}

func TestInferColumnType_FloatColumn(t *testing.T) {
	rows := [][]string{{"1.5"}, {"3.14"}, {"0.001"}}
	assert.Equal(t, "float64", inferColumnType(rows, 0))
}

func TestInferColumnType_MixedIntAndFloat(t *testing.T) {
	// An integer that looks like int but float appears → float wins
	rows := [][]string{{"1"}, {"2"}, {"3.14"}}
	assert.Equal(t, "float64", inferColumnType(rows, 0))
}

func TestInferColumnType_BooleanColumn(t *testing.T) {
	rows := [][]string{{"true"}, {"false"}, {"TRUE"}, {"False"}}
	assert.Equal(t, "boolean", inferColumnType(rows, 0))
}

func TestInferColumnType_StringColumn(t *testing.T) {
	rows := [][]string{{"alice"}, {"bob"}, {"charlie"}}
	assert.Equal(t, "string", inferColumnType(rows, 0))
}

func TestInferColumnType_MixedNumericAndString(t *testing.T) {
	// One non-numeric value → whole column is string
	rows := [][]string{{"1"}, {"2"}, {"three"}}
	assert.Equal(t, "string", inferColumnType(rows, 0))
}

func TestInferColumnType_AllEmpty(t *testing.T) {
	rows := [][]string{{""}, {""}, {""}}
	assert.Equal(t, "string", inferColumnType(rows, 0))
}

func TestInferColumnType_ColumnIndexOutOfBounds(t *testing.T) {
	// colIndex beyond row width → treated as empty → defaults to string
	rows := [][]string{{"a", "1"}, {"b", "2"}}
	assert.Equal(t, "string", inferColumnType(rows, 5))
}

// parquetTypeToDataType is tested indirectly via extractColumnsFromParquet
// in integration tests — parquet.Type values require a real schema to construct.

// ---------------------------------------------------------------------------
// extractColumnsFromCSV — method on DatasourceServiceImpl but uses no service state
// ---------------------------------------------------------------------------

func newBareDS() *DatasourceServiceImpl {
	return &DatasourceServiceImpl{}
}

func TestExtractColumnsFromCSV_BasicTypes(t *testing.T) {
	csv := "age,salary,name,is_active\n" +
		"25,50000.5,Alice,true\n" +
		"30,60000.0,Bob,false\n" +
		"22,45000.75,Charlie,true\n"

	svc := newBareDS()
	cols, err := svc.extractColumnsFromCSV(strings.NewReader(csv))
	require.NoError(t, err)
	require.Len(t, cols, 4)

	byName := make(map[string]string)
	for _, c := range cols {
		byName[c.Name] = c.DataType
	}
	assert.Equal(t, "int64", byName["age"])
	assert.Equal(t, "float64", byName["salary"])
	assert.Equal(t, "string", byName["name"])
	assert.Equal(t, "boolean", byName["is_active"])
}

func TestExtractColumnsFromCSV_HeaderOnly(t *testing.T) {
	// No data rows — all columns default to string
	csv := "col1,col2,col3\n"
	svc := newBareDS()
	cols, err := svc.extractColumnsFromCSV(strings.NewReader(csv))
	require.NoError(t, err)
	require.Len(t, cols, 3)
	for _, c := range cols {
		assert.Equal(t, "string", c.DataType)
	}
}

func TestExtractColumnsFromCSV_EmptyInput(t *testing.T) {
	svc := newBareDS()
	_, err := svc.extractColumnsFromCSV(strings.NewReader(""))
	assert.Error(t, err)
}

func TestExtractColumnsFromCSV_AllRolesDefaultToInput(t *testing.T) {
	csv := "a,b\n1,2\n"
	svc := newBareDS()
	cols, err := svc.extractColumnsFromCSV(strings.NewReader(csv))
	require.NoError(t, err)
	for _, c := range cols {
		assert.Equal(t, "input", string(c.Role))
	}
}

// ---------------------------------------------------------------------------
// parseCSVPreview — method on DatasourceServiceImpl but uses no service state
// ---------------------------------------------------------------------------

func TestParseCSVPreview_Success(t *testing.T) {
	csv := "name,age,score\n" +
		"Alice,25,95.5\n" +
		"Bob,30,88.0\n" +
		"Charlie,22,72.3\n"

	svc := newBareDS()
	result, err := svc.parseCSVPreview(strings.NewReader(csv), 10)
	require.NoError(t, err)

	assert.Equal(t, []string{"name", "age", "score"}, result.Columns)
	assert.Len(t, result.Rows, 3)
	assert.Equal(t, 3, result.TotalRows)
	assert.Equal(t, 10, result.PreviewMax)
}

func TestParseCSVPreview_LimitEnforced(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("id\n")
	for i := 0; i < 20; i++ {
		sb.WriteString("row\n")
	}
	svc := newBareDS()
	result, err := svc.parseCSVPreview(strings.NewReader(sb.String()), 5)
	require.NoError(t, err)
	assert.Len(t, result.Rows, 5, "rows in response should be capped at limit")
	assert.Equal(t, 20, result.TotalRows, "TotalRows should count all rows")
}

func TestParseCSVPreview_EmptyInput(t *testing.T) {
	svc := newBareDS()
	_, err := svc.parseCSVPreview(strings.NewReader(""), 10)
	assert.Error(t, err)
}
