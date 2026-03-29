package application

import (
	"errors"
	"testing"

	"modelmatrix-server/internal/module/inventory/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Activate
// ---------------------------------------------------------------------------

func TestModelService_Activate_FromDraft(t *testing.T) {
	m := &domain.Model{ID: "m1", Name: "M", Status: domain.ModelStatusDraft}
	repo := newCRUDRepo(m)
	svc := buildCRUDSvc(repo)

	resp, err := svc.Activate("m1")
	require.NoError(t, err)
	assert.Equal(t, "active", resp.Status)
}

func TestModelService_Activate_FromInactive(t *testing.T) {
	m := &domain.Model{ID: "m1", Name: "M", Status: domain.ModelStatusInactive}
	svc := buildCRUDSvc(newCRUDRepo(m))
	resp, err := svc.Activate("m1")
	require.NoError(t, err)
	assert.Equal(t, "active", resp.Status)
}

func TestModelService_Activate_AlreadyActive_Error(t *testing.T) {
	m := &domain.Model{ID: "m1", Name: "M", Status: domain.ModelStatusActive}
	svc := buildCRUDSvc(newCRUDRepo(m))
	_, err := svc.Activate("m1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrModelCannotActivate))
}

func TestModelService_Activate_NotFound(t *testing.T) {
	svc := buildCRUDSvc(newCRUDRepo())
	_, err := svc.Activate("missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrModelNotFound))
}

// ---------------------------------------------------------------------------
// Deactivate
// ---------------------------------------------------------------------------

func TestModelService_Deactivate_WhenActive(t *testing.T) {
	m := &domain.Model{ID: "m1", Name: "M", Status: domain.ModelStatusActive}
	svc := buildCRUDSvc(newCRUDRepo(m))
	resp, err := svc.Deactivate("m1")
	require.NoError(t, err)
	assert.Equal(t, "inactive", resp.Status)
}

func TestModelService_Deactivate_WhenDraft_Error(t *testing.T) {
	m := &domain.Model{ID: "m1", Name: "M", Status: domain.ModelStatusDraft}
	svc := buildCRUDSvc(newCRUDRepo(m))
	_, err := svc.Deactivate("m1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrModelCannotDeactivate))
}

func TestModelService_Deactivate_NotFound(t *testing.T) {
	svc := buildCRUDSvc(newCRUDRepo())
	_, err := svc.Deactivate("missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrModelNotFound))
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestModelService_Delete_DraftModel(t *testing.T) {
	m := &domain.Model{
		ID: "m1", Name: "M", Status: domain.ModelStatusDraft,
		Files: []domain.ModelFile{
			{FileType: domain.FileTypeModel, FilePath: "models/m1/model.pkl"},
		},
	}
	repo := newCRUDRepo(m)
	svc := buildCRUDSvc(repo)

	err := svc.Delete("m1")
	require.NoError(t, err)
	// Confirm removed from repo
	_, getErr := svc.GetByID("m1")
	assert.ErrorIs(t, getErr, domain.ErrModelNotFound)
}

func TestModelService_Delete_ActiveModel_Rejected(t *testing.T) {
	m := &domain.Model{ID: "m1", Name: "M", Status: domain.ModelStatusActive}
	svc := buildCRUDSvc(newCRUDRepo(m))
	err := svc.Delete("m1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrModelCannotDelete))
}

func TestModelService_Delete_NotFound(t *testing.T) {
	svc := buildCRUDSvc(newCRUDRepo())
	err := svc.Delete("missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrModelNotFound))
}

// ---------------------------------------------------------------------------
// DeleteByFolderID / DeleteByProjectID
// ---------------------------------------------------------------------------

func TestModelService_DeleteByFolderID_DeletesInactiveModels(t *testing.T) {
	folderID := "folder-1"
	m1 := &domain.Model{ID: "m1", Name: "M1", Status: domain.ModelStatusInactive}
	m2 := &domain.Model{ID: "m2", Name: "M2", Status: domain.ModelStatusDraft}
	repo := newCRUDRepo(m1, m2)
	repo.getIDsByFolderFn = func(id string) []string {
		if id == folderID {
			return []string{"m1", "m2"}
		}
		return nil
	}
	svc := buildCRUDSvc(repo)

	err := svc.DeleteByFolderID(folderID)
	require.NoError(t, err)
	// Both should be deleted
	_, e1 := svc.GetByID("m1")
	_, e2 := svc.GetByID("m2")
	assert.ErrorIs(t, e1, domain.ErrModelNotFound)
	assert.ErrorIs(t, e2, domain.ErrModelNotFound)
}

func TestModelService_DeleteByProjectID_DeletesModels(t *testing.T) {
	projectID := "proj-1"
	m := &domain.Model{ID: "m1", Name: "M", Status: domain.ModelStatusDraft}
	repo := newCRUDRepo(m)
	repo.getIDsByProjectFn = func(id string) []string {
		if id == projectID {
			return []string{"m1"}
		}
		return nil
	}
	svc := buildCRUDSvc(repo)

	err := svc.DeleteByProjectID(projectID)
	require.NoError(t, err)
	_, e := svc.GetByID("m1")
	assert.ErrorIs(t, e, domain.ErrModelNotFound)
}
