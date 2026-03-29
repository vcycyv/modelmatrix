package application

import (
	"errors"
	"testing"

	"modelmatrix-server/internal/module/build/domain"
	"modelmatrix-server/internal/module/build/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestBuildServiceImpl_Update_Name(t *testing.T) {
	b := &domain.ModelBuild{ID: "b1", Name: "Old", Status: domain.BuildStatusPending,
		ModelType: domain.ModelTypeRegression}
	repo := newFakeBuildRepo(b)
	svc := buildSvcImpl(repo)

	newName := "New Name"
	resp, err := svc.Update("b1", &dto.UpdateBuildRequest{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "New Name", resp.Name)
}

func TestBuildServiceImpl_Update_NotPending_Rejected(t *testing.T) {
	b := &domain.ModelBuild{ID: "b1", Name: "Running", Status: domain.BuildStatusRunning,
		ModelType: domain.ModelTypeRegression}
	repo := newFakeBuildRepo(b)
	svc := buildSvcImpl(repo)

	newName := "Try"
	_, err := svc.Update("b1", &dto.UpdateBuildRequest{Name: &newName})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrBuildNotPending))
}

func TestBuildServiceImpl_Update_EmptyName_Rejected(t *testing.T) {
	b := &domain.ModelBuild{ID: "b1", Name: "Pending", Status: domain.BuildStatusPending,
		ModelType: domain.ModelTypeRegression}
	svc := buildSvcImpl(newFakeBuildRepo(b))

	empty := ""
	_, err := svc.Update("b1", &dto.UpdateBuildRequest{Name: &empty})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrBuildNameEmpty))
}

func TestBuildServiceImpl_Update_NotFound(t *testing.T) {
	svc := buildSvcImpl(newFakeBuildRepo())
	_, err := svc.Update("missing", &dto.UpdateBuildRequest{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrBuildNotFound))
}

// ---------------------------------------------------------------------------
// DeleteByFolderID / DeleteByProjectID
// ---------------------------------------------------------------------------

func TestBuildServiceImpl_DeleteByFolderID_DeletesContainedBuilds(t *testing.T) {
	folderID := "folder-1"
	b1 := &domain.ModelBuild{ID: "b1", Name: "B1", Status: domain.BuildStatusPending}
	b2 := &domain.ModelBuild{ID: "b2", Name: "B2", Status: domain.BuildStatusCompleted}

	repo := newFakeBuildRepo(b1, b2)
	repo.folderIDs = map[string][]string{folderID: {"b1", "b2"}}
	svc := buildSvcImpl(repo)

	err := svc.DeleteByFolderID(folderID)
	require.NoError(t, err)
	_, err1 := svc.GetByID("b1")
	_, err2 := svc.GetByID("b2")
	assert.ErrorIs(t, err1, domain.ErrBuildNotFound)
	assert.ErrorIs(t, err2, domain.ErrBuildNotFound)
}

func TestBuildServiceImpl_DeleteByFolderID_SkipsRunningBuilds(t *testing.T) {
	folderID := "folder-2"
	running := &domain.ModelBuild{ID: "r1", Name: "Running", Status: domain.BuildStatusRunning}
	repo := newFakeBuildRepo(running)
	repo.folderIDs = map[string][]string{folderID: {"r1"}}
	svc := buildSvcImpl(repo)

	// Should return no error but the running build should not be deleted
	err := svc.DeleteByFolderID(folderID)
	require.NoError(t, err)
	// Running builds are not deleted (Delete rejects them)
	_, getErr := svc.GetByID("r1")
	assert.NoError(t, getErr, "running build should still exist")
}

func TestBuildServiceImpl_DeleteByProjectID_DeletesContainedBuilds(t *testing.T) {
	projectID := "proj-1"
	b := &domain.ModelBuild{ID: "bp1", Name: "P Build", Status: domain.BuildStatusFailed}
	repo := newFakeBuildRepo(b)
	repo.projectIDs = map[string][]string{projectID: {"bp1"}}
	svc := buildSvcImpl(repo)

	err := svc.DeleteByProjectID(projectID)
	require.NoError(t, err)
	_, getErr := svc.GetByID("bp1")
	assert.ErrorIs(t, getErr, domain.ErrBuildNotFound)
}
