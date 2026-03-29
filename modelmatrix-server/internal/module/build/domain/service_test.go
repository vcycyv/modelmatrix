package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBuildSvc() *Service { return NewService() }

// ---------------------------------------------------------------------------
// ValidateBuild
// ---------------------------------------------------------------------------

func TestValidateBuild_Valid(t *testing.T) {
	svc := newBuildSvc()
	err := svc.ValidateBuild(&ModelBuild{Name: "My Build", ModelType: ModelTypeClassification})
	require.NoError(t, err)
}

func TestValidateBuild_EmptyName(t *testing.T) {
	svc := newBuildSvc()
	err := svc.ValidateBuild(&ModelBuild{Name: "  ", ModelType: ModelTypeRegression})
	require.Error(t, err)
	assert.Equal(t, ErrBuildNameEmpty, err)
}

func TestValidateBuild_InvalidModelType(t *testing.T) {
	svc := newBuildSvc()
	err := svc.ValidateBuild(&ModelBuild{Name: "B", ModelType: "deep_learning"})
	require.Error(t, err)
	assert.Equal(t, ErrInvalidModelType, err)
}

// ---------------------------------------------------------------------------
// ValidateBuildNameUnique
// ---------------------------------------------------------------------------

func TestValidateBuildNameUnique_Unique(t *testing.T) {
	err := newBuildSvc().ValidateBuildNameUnique("Alpha", []string{"Beta", "Gamma"})
	require.NoError(t, err)
}

func TestValidateBuildNameUnique_Duplicate_CaseInsensitive(t *testing.T) {
	err := newBuildSvc().ValidateBuildNameUnique("alpha", []string{"Alpha", "Beta"})
	require.Error(t, err)
	assert.Equal(t, ErrBuildNameExists, err)
}

// ---------------------------------------------------------------------------
// CanStartBuild
// ---------------------------------------------------------------------------

func TestCanStartBuild_Pending(t *testing.T) {
	b := &ModelBuild{Status: BuildStatusPending}
	require.NoError(t, newBuildSvc().CanStartBuild(b))
}

func TestCanStartBuild_Running_Error(t *testing.T) {
	b := &ModelBuild{Status: BuildStatusRunning}
	err := newBuildSvc().CanStartBuild(b)
	require.Error(t, err)
	assert.Equal(t, ErrBuildNotPending, err)
}

func TestCanStartBuild_Completed_Error(t *testing.T) {
	b := &ModelBuild{Status: BuildStatusCompleted}
	err := newBuildSvc().CanStartBuild(b)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// CanCancelBuild
// ---------------------------------------------------------------------------

func TestCanCancelBuild_Pending(t *testing.T) {
	require.NoError(t, newBuildSvc().CanCancelBuild(&ModelBuild{Status: BuildStatusPending}))
}

func TestCanCancelBuild_Running(t *testing.T) {
	require.NoError(t, newBuildSvc().CanCancelBuild(&ModelBuild{Status: BuildStatusRunning}))
}

func TestCanCancelBuild_Completed_Error(t *testing.T) {
	err := newBuildSvc().CanCancelBuild(&ModelBuild{Status: BuildStatusCompleted})
	require.Error(t, err)
	assert.Equal(t, ErrBuildCannotBeCancelled, err)
}

// ---------------------------------------------------------------------------
// ValidateParameters
// ---------------------------------------------------------------------------

func TestValidateParameters_DefaultsApplied(t *testing.T) {
	params := &TrainingParameters{TrainTestSplit: 0, MaxIterations: 0}
	require.NoError(t, newBuildSvc().ValidateParameters(params))
	assert.Equal(t, 0.8, params.TrainTestSplit)
	assert.Equal(t, 100, params.MaxIterations)
}

func TestValidateParameters_ValidValues_Unchanged(t *testing.T) {
	params := &TrainingParameters{TrainTestSplit: 0.7, MaxIterations: 200}
	require.NoError(t, newBuildSvc().ValidateParameters(params))
	assert.Equal(t, 0.7, params.TrainTestSplit)
	assert.Equal(t, 200, params.MaxIterations)
}

// ---------------------------------------------------------------------------
// GetDefaultParameters
// ---------------------------------------------------------------------------

func TestGetDefaultParameters_Classification(t *testing.T) {
	p := newBuildSvc().GetDefaultParameters(ModelTypeClassification)
	assert.Equal(t, 0.8, p.TrainTestSplit)
	assert.Equal(t, 42, p.RandomSeed)
	assert.Equal(t, 100, p.MaxIterations)
	_, ok := p.Hyperparameters["n_estimators"]
	assert.True(t, ok)
}

func TestGetDefaultParameters_Regression(t *testing.T) {
	p := newBuildSvc().GetDefaultParameters(ModelTypeRegression)
	_, ok := p.Hyperparameters["learning_rate"]
	assert.True(t, ok)
}

func TestGetDefaultParameters_Clustering(t *testing.T) {
	p := newBuildSvc().GetDefaultParameters(ModelTypeClustering)
	_, ok := p.Hyperparameters["n_clusters"]
	assert.True(t, ok)
}
