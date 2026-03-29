package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSvc() *Service { return NewService() }

// ---------------------------------------------------------------------------
// ValidateModel
// ---------------------------------------------------------------------------

func TestValidateModel_Valid(t *testing.T) {
	svc := newSvc()
	m := &Model{Name: "My Model", Status: ModelStatusDraft}
	assert.NoError(t, svc.ValidateModel(m))
}

func TestValidateModel_EmptyName(t *testing.T) {
	svc := newSvc()
	m := &Model{Name: "  ", Status: ModelStatusDraft}
	err := svc.ValidateModel(m)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrModelNameEmpty))
}

func TestValidateModel_InvalidStatus(t *testing.T) {
	svc := newSvc()
	m := &Model{Name: "OK", Status: "unknown"}
	err := svc.ValidateModel(m)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidModelStatus))
}

// ---------------------------------------------------------------------------
// CanDeleteModel
// ---------------------------------------------------------------------------

func TestCanDeleteModel_Draft(t *testing.T) {
	assert.NoError(t, newSvc().CanDeleteModel(&Model{Status: ModelStatusDraft}))
}

func TestCanDeleteModel_Inactive(t *testing.T) {
	assert.NoError(t, newSvc().CanDeleteModel(&Model{Status: ModelStatusInactive}))
}

func TestCanDeleteModel_Active(t *testing.T) {
	err := newSvc().CanDeleteModel(&Model{Status: ModelStatusActive})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrModelCannotDelete))
}

// ---------------------------------------------------------------------------
// CanActivateModel
// ---------------------------------------------------------------------------

func TestCanActivateModel_Draft(t *testing.T) {
	assert.NoError(t, newSvc().CanActivateModel(&Model{Status: ModelStatusDraft}))
}

func TestCanActivateModel_Inactive(t *testing.T) {
	assert.NoError(t, newSvc().CanActivateModel(&Model{Status: ModelStatusInactive}))
}

func TestCanActivateModel_AlreadyActive(t *testing.T) {
	err := newSvc().CanActivateModel(&Model{Status: ModelStatusActive})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrModelCannotActivate))
}

// ---------------------------------------------------------------------------
// CanDeactivateModel
// ---------------------------------------------------------------------------

func TestCanDeactivateModel_Active(t *testing.T) {
	assert.NoError(t, newSvc().CanDeactivateModel(&Model{Status: ModelStatusActive}))
}

func TestCanDeactivateModel_Draft(t *testing.T) {
	err := newSvc().CanDeactivateModel(&Model{Status: ModelStatusDraft})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrModelCannotDeactivate))
}

// ---------------------------------------------------------------------------
// ModelStatus helper methods
// ---------------------------------------------------------------------------

func TestModelStatus_IsValid(t *testing.T) {
	valid := []ModelStatus{ModelStatusDraft, ModelStatusActive, ModelStatusInactive, ModelStatusArchived}
	for _, s := range valid {
		assert.True(t, s.IsValid(), "expected %q to be valid", s)
	}
	assert.False(t, ModelStatus("bogus").IsValid())
}

func TestModelStatus_CanDelete(t *testing.T) {
	assert.True(t, ModelStatusDraft.CanDelete())
	assert.True(t, ModelStatusInactive.CanDelete())
	assert.False(t, ModelStatusActive.CanDelete())
}

func TestModelStatus_CanActivate(t *testing.T) {
	assert.True(t, ModelStatusDraft.CanActivate())
	assert.True(t, ModelStatusInactive.CanActivate())
	assert.False(t, ModelStatusActive.CanActivate())
}

func TestModelStatus_CanDeactivate(t *testing.T) {
	assert.True(t, ModelStatusActive.CanDeactivate())
	assert.False(t, ModelStatusDraft.CanDeactivate())
	assert.False(t, ModelStatusInactive.CanDeactivate())
}
