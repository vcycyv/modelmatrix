package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Type validity helpers
// ---------------------------------------------------------------------------

func TestTaskType_IsValid(t *testing.T) {
	assert.True(t, TaskTypeClassification.IsValid())
	assert.True(t, TaskTypeRegression.IsValid())
	assert.False(t, TaskType("clustering").IsValid())
}

func TestAlertSeverity_IsValid(t *testing.T) {
	assert.True(t, AlertSeverityInfo.IsValid())
	assert.True(t, AlertSeverityWarning.IsValid())
	assert.True(t, AlertSeverityCritical.IsValid())
	assert.False(t, AlertSeverity("fatal").IsValid())
}

func TestAlertStatus_IsValid(t *testing.T) {
	assert.True(t, AlertStatusActive.IsValid())
	assert.True(t, AlertStatusAcknowledged.IsValid())
	assert.True(t, AlertStatusResolved.IsValid())
	assert.False(t, AlertStatus("pending").IsValid())
}

// ---------------------------------------------------------------------------
// PerformanceRecord.CalculateDrift
// ---------------------------------------------------------------------------

func TestCalculateDrift_NoChange(t *testing.T) {
	r := &PerformanceRecord{MetricValue: 0.90}
	drift := r.CalculateDrift(0.90, ThresholdDirectionLower)
	assert.Equal(t, 0.0, drift)
}

func TestCalculateDrift_Lower_Degradation(t *testing.T) {
	// Metric dropped from 0.90 to 0.80 — should be positive drift (bad)
	r := &PerformanceRecord{MetricValue: 0.80}
	drift := r.CalculateDrift(0.90, ThresholdDirectionLower)
	assert.Greater(t, drift, 0.0, "degradation should be positive drift for Lower direction")
}

func TestCalculateDrift_Lower_Improvement(t *testing.T) {
	// Metric improved from 0.80 to 0.90 — negative drift (good)
	r := &PerformanceRecord{MetricValue: 0.90}
	drift := r.CalculateDrift(0.80, ThresholdDirectionLower)
	assert.Less(t, drift, 0.0)
}

func TestCalculateDrift_ZeroBaseline_NonzeroValue(t *testing.T) {
	r := &PerformanceRecord{MetricValue: 0.5}
	drift := r.CalculateDrift(0, ThresholdDirectionLower)
	assert.Equal(t, 100.0, drift)
}

func TestCalculateDrift_ZeroBaseline_ZeroValue(t *testing.T) {
	r := &PerformanceRecord{MetricValue: 0}
	drift := r.CalculateDrift(0, ThresholdDirectionLower)
	assert.Equal(t, 0.0, drift)
}

// ---------------------------------------------------------------------------
// PerformanceThreshold.CheckBreach
// ---------------------------------------------------------------------------

func threshold(warning, critical float64, enabled bool) *PerformanceThreshold {
	return &PerformanceThreshold{
		WarningThreshold:  warning,
		CriticalThreshold: critical,
		Enabled:           enabled,
	}
}

func TestCheckBreach_Disabled(t *testing.T) {
	breached, _ := threshold(5, 10, false).CheckBreach(20)
	assert.False(t, breached)
}

func TestCheckBreach_Improvement_NoBreach(t *testing.T) {
	breached, _ := threshold(5, 10, true).CheckBreach(-5)
	assert.False(t, breached)
}

func TestCheckBreach_BelowWarning_NoBreach(t *testing.T) {
	breached, _ := threshold(5, 10, true).CheckBreach(3)
	assert.False(t, breached)
}

func TestCheckBreach_Warning(t *testing.T) {
	breached, severity := threshold(5, 10, true).CheckBreach(7)
	require.True(t, breached)
	assert.Equal(t, AlertSeverityWarning, severity)
}

func TestCheckBreach_Critical(t *testing.T) {
	breached, severity := threshold(5, 10, true).CheckBreach(15)
	require.True(t, breached)
	assert.Equal(t, AlertSeverityCritical, severity)
}

func TestCheckBreach_ExactlyWarningThreshold(t *testing.T) {
	breached, severity := threshold(5, 10, true).CheckBreach(5)
	require.True(t, breached)
	assert.Equal(t, AlertSeverityWarning, severity)
}

// ---------------------------------------------------------------------------
// PerformanceEvaluation lifecycle
// ---------------------------------------------------------------------------

func TestEvaluation_Start(t *testing.T) {
	e := &PerformanceEvaluation{Status: EvaluationStatusPending}
	e.Start()
	assert.Equal(t, EvaluationStatusRunning, e.Status)
	assert.NotNil(t, e.StartedAt)
}

func TestEvaluation_Complete(t *testing.T) {
	e := &PerformanceEvaluation{Status: EvaluationStatusRunning}
	metrics := map[string]interface{}{"accuracy": 0.95}
	e.Complete(metrics, 500)
	assert.Equal(t, EvaluationStatusCompleted, e.Status)
	assert.Equal(t, 500, e.SampleCount)
	assert.NotNil(t, e.CompletedAt)
}

func TestEvaluation_Fail(t *testing.T) {
	e := &PerformanceEvaluation{Status: EvaluationStatusRunning}
	e.Fail("out of memory")
	assert.Equal(t, EvaluationStatusFailed, e.Status)
	assert.Equal(t, "out of memory", e.ErrorMessage)
	assert.NotNil(t, e.CompletedAt)
}

// ---------------------------------------------------------------------------
// PerformanceAlert.Acknowledge / Resolve
// ---------------------------------------------------------------------------

func TestAlert_Acknowledge(t *testing.T) {
	a := &PerformanceAlert{Status: AlertStatusActive}
	a.Acknowledge("alice")
	assert.Equal(t, AlertStatusAcknowledged, a.Status)
	require.NotNil(t, a.AcknowledgedBy)
	assert.Equal(t, "alice", *a.AcknowledgedBy)
	assert.NotNil(t, a.AcknowledgedAt)
}

func TestAlert_Resolve(t *testing.T) {
	a := &PerformanceAlert{Status: AlertStatusActive}
	a.Resolve()
	assert.Equal(t, AlertStatusResolved, a.Status)
	assert.NotNil(t, a.ResolvedAt)
}

// ---------------------------------------------------------------------------
// GetDefaultThresholds
// ---------------------------------------------------------------------------

func TestGetDefaultThresholds_Classification(t *testing.T) {
	thresholds := GetDefaultThresholds(TaskTypeClassification)
	assert.NotEmpty(t, thresholds)
	for _, th := range thresholds {
		assert.True(t, th.Enabled)
		assert.Greater(t, th.WarningThreshold, 0.0)
		assert.Greater(t, th.CriticalThreshold, th.WarningThreshold)
	}
}

func TestGetDefaultThresholds_Regression(t *testing.T) {
	thresholds := GetDefaultThresholds(TaskTypeRegression)
	assert.NotEmpty(t, thresholds)
}
