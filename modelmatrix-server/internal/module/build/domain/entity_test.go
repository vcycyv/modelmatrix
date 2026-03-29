package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// BuildStatus helper methods
// ---------------------------------------------------------------------------

func TestBuildStatus_IsValid(t *testing.T) {
	valid := []BuildStatus{
		BuildStatusPending, BuildStatusRunning,
		BuildStatusCompleted, BuildStatusFailed, BuildStatusCancelled,
	}
	for _, s := range valid {
		assert.True(t, s.IsValid(), "expected %q to be valid", s)
	}
	assert.False(t, BuildStatus("unknown").IsValid())
}

func TestBuildStatus_IsTerminal(t *testing.T) {
	assert.True(t, BuildStatusCompleted.IsTerminal())
	assert.True(t, BuildStatusFailed.IsTerminal())
	assert.True(t, BuildStatusCancelled.IsTerminal())
	assert.False(t, BuildStatusPending.IsTerminal())
	assert.False(t, BuildStatusRunning.IsTerminal())
}

// ---------------------------------------------------------------------------
// ModelType helper methods
// ---------------------------------------------------------------------------

func TestModelType_IsValid(t *testing.T) {
	assert.True(t, ModelTypeClassification.IsValid())
	assert.True(t, ModelTypeRegression.IsValid())
	assert.True(t, ModelTypeClustering.IsValid())
	assert.False(t, ModelType("svm").IsValid())
}

// ---------------------------------------------------------------------------
// ModelBuild entity methods
// ---------------------------------------------------------------------------

func TestModelBuild_CanStart_OnlyPending(t *testing.T) {
	cases := map[BuildStatus]bool{
		BuildStatusPending:   true,
		BuildStatusRunning:   false,
		BuildStatusCompleted: false,
		BuildStatusFailed:    false,
		BuildStatusCancelled: false,
	}
	for status, want := range cases {
		b := &ModelBuild{Status: status}
		assert.Equal(t, want, b.CanStart(), "CanStart for status=%q", status)
	}
}

func TestModelBuild_CanCancel_PendingAndRunning(t *testing.T) {
	assert.True(t, (&ModelBuild{Status: BuildStatusPending}).CanCancel())
	assert.True(t, (&ModelBuild{Status: BuildStatusRunning}).CanCancel())
	assert.False(t, (&ModelBuild{Status: BuildStatusCompleted}).CanCancel())
	assert.False(t, (&ModelBuild{Status: BuildStatusFailed}).CanCancel())
	assert.False(t, (&ModelBuild{Status: BuildStatusCancelled}).CanCancel())
}

func TestModelBuild_Start_SetsRunningAndStartedAt(t *testing.T) {
	b := &ModelBuild{Status: BuildStatusPending}
	b.Start()
	assert.Equal(t, BuildStatusRunning, b.Status)
	assert.NotNil(t, b.StartedAt)
}

func TestModelBuild_Complete_SetsCompletedAndMetrics(t *testing.T) {
	b := &ModelBuild{Status: BuildStatusRunning}
	metrics := &BuildMetrics{Accuracy: 0.95, R2: 0.89}
	b.Complete(metrics)
	assert.Equal(t, BuildStatusCompleted, b.Status)
	assert.NotNil(t, b.CompletedAt)
	assert.Equal(t, 0.95, b.Metrics.Accuracy)
	assert.True(t, b.Status.IsTerminal())
}

func TestModelBuild_Complete_NilMetrics(t *testing.T) {
	b := &ModelBuild{Status: BuildStatusRunning}
	b.Complete(nil)
	assert.Equal(t, BuildStatusCompleted, b.Status)
	assert.Nil(t, b.Metrics)
}

func TestModelBuild_Fail_SetsErrorMessage(t *testing.T) {
	b := &ModelBuild{Status: BuildStatusRunning}
	b.Fail("out of memory")
	assert.Equal(t, BuildStatusFailed, b.Status)
	assert.Equal(t, "out of memory", b.ErrorMessage)
	assert.NotNil(t, b.CompletedAt)
	assert.True(t, b.Status.IsTerminal())
}

func TestModelBuild_Cancel_SetsCancelledAndCompletedAt(t *testing.T) {
	b := &ModelBuild{Status: BuildStatusPending}
	b.Cancel()
	assert.Equal(t, BuildStatusCancelled, b.Status)
	assert.NotNil(t, b.CompletedAt)
	assert.True(t, b.Status.IsTerminal())
}
