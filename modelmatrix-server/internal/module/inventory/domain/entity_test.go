package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Model.Activate / Deactivate / Archive
// ---------------------------------------------------------------------------

func TestModel_Activate_FromDraft(t *testing.T) {
	m := &Model{Status: ModelStatusDraft}
	err := m.Activate()
	require.NoError(t, err)
	assert.Equal(t, ModelStatusActive, m.Status)
}

func TestModel_Activate_FromInactive(t *testing.T) {
	m := &Model{Status: ModelStatusInactive}
	err := m.Activate()
	require.NoError(t, err)
	assert.Equal(t, ModelStatusActive, m.Status)
}

func TestModel_Activate_WhenAlreadyActive(t *testing.T) {
	m := &Model{Status: ModelStatusActive}
	err := m.Activate()
	require.Error(t, err)
	assert.Equal(t, ErrModelCannotActivate, err)
	assert.Equal(t, ModelStatusActive, m.Status) // unchanged
}

func TestModel_Deactivate_WhenActive(t *testing.T) {
	m := &Model{Status: ModelStatusActive}
	err := m.Deactivate()
	require.NoError(t, err)
	assert.Equal(t, ModelStatusInactive, m.Status)
}

func TestModel_Deactivate_WhenDraft(t *testing.T) {
	m := &Model{Status: ModelStatusDraft}
	err := m.Deactivate()
	require.Error(t, err)
	assert.Equal(t, ErrModelCannotDeactivate, err)
}

func TestModel_Archive(t *testing.T) {
	m := &Model{Status: ModelStatusInactive}
	m.Archive()
	assert.Equal(t, ModelStatusArchived, m.Status)
}

// ---------------------------------------------------------------------------
// CanBe* helpers
// ---------------------------------------------------------------------------

func TestModel_CanBeDeleted(t *testing.T) {
	assert.True(t, (&Model{Status: ModelStatusDraft}).CanBeDeleted())
	assert.True(t, (&Model{Status: ModelStatusInactive}).CanBeDeleted())
	assert.False(t, (&Model{Status: ModelStatusActive}).CanBeDeleted())
}

func TestModel_CanBeActivated(t *testing.T) {
	assert.True(t, (&Model{Status: ModelStatusDraft}).CanBeActivated())
	assert.True(t, (&Model{Status: ModelStatusInactive}).CanBeActivated())
	assert.False(t, (&Model{Status: ModelStatusActive}).CanBeActivated())
}

func TestModel_CanBeDeactivated(t *testing.T) {
	assert.True(t, (&Model{Status: ModelStatusActive}).CanBeDeactivated())
	assert.False(t, (&Model{Status: ModelStatusDraft}).CanBeDeactivated())
}

// ---------------------------------------------------------------------------
// GetInputVariables / GetTargetVariable / GetMainModelFile
// ---------------------------------------------------------------------------

func modelWithVariables() *Model {
	return &Model{
		Status: ModelStatusDraft,
		Variables: []ModelVariable{
			{Name: "age", Role: VariableRoleInput},
			{Name: "income", Role: VariableRoleInput},
			{Name: "churn", Role: VariableRoleTarget},
		},
	}
}

func TestModel_GetInputVariables(t *testing.T) {
	m := modelWithVariables()
	inputs := m.GetInputVariables()
	assert.Len(t, inputs, 2)
	names := []string{inputs[0].Name, inputs[1].Name}
	assert.Contains(t, names, "age")
	assert.Contains(t, names, "income")
}

func TestModel_GetInputVariables_Empty(t *testing.T) {
	m := &Model{}
	assert.Empty(t, m.GetInputVariables())
}

func TestModel_GetTargetVariable_Found(t *testing.T) {
	m := modelWithVariables()
	target := m.GetTargetVariable()
	require.NotNil(t, target)
	assert.Equal(t, "churn", target.Name)
}

func TestModel_GetTargetVariable_NotFound(t *testing.T) {
	m := &Model{
		Variables: []ModelVariable{{Name: "x", Role: VariableRoleInput}},
	}
	assert.Nil(t, m.GetTargetVariable())
}

func TestModel_GetMainModelFile_Found(t *testing.T) {
	m := &Model{
		Files: []ModelFile{
			{FileType: FileTypePreprocessor, FilePath: "pre.pkl"},
			{FileType: FileTypeModel, FilePath: "model.pkl"},
		},
	}
	f := m.GetMainModelFile()
	require.NotNil(t, f)
	assert.Equal(t, "model.pkl", f.FilePath)
}

func TestModel_GetMainModelFile_NotFound(t *testing.T) {
	m := &Model{
		Files: []ModelFile{{FileType: FileTypeMetadata, FilePath: "meta.json"}},
	}
	assert.Nil(t, m.GetMainModelFile())
}
