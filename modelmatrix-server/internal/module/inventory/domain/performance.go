package domain

import (
	"errors"
	"math"
	"time"
)

// Performance monitoring domain errors
var (
	ErrBaselineNotFound       = errors.New("performance baseline not found")
	ErrRecordNotFound         = errors.New("performance record not found")
	ErrAlertNotFound          = errors.New("performance alert not found")
	ErrEvaluationNotFound     = errors.New("performance evaluation not found")
	ErrThresholdNotFound      = errors.New("performance threshold not found")
	ErrInvalidTaskType        = errors.New("invalid task type")
	ErrInvalidAlertSeverity   = errors.New("invalid alert severity")
	ErrInvalidAlertStatus     = errors.New("invalid alert status")
	ErrEvaluationRunning      = errors.New("evaluation is already running")
	ErrNoActualTargetColumn   = errors.New("no actual target column in evaluation data")
	ErrInvalidThresholdValues = errors.New("invalid threshold values")
)

// TaskType represents the ML task type
type TaskType string

const (
	TaskTypeClassification TaskType = "classification"
	TaskTypeRegression     TaskType = "regression"
)

// IsValid checks if task type is valid
func (t TaskType) IsValid() bool {
	return t == TaskTypeClassification || t == TaskTypeRegression
}

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// IsValid checks if alert severity is valid
func (s AlertSeverity) IsValid() bool {
	switch s {
	case AlertSeverityInfo, AlertSeverityWarning, AlertSeverityCritical:
		return true
	default:
		return false
	}
}

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusActive       AlertStatus = "active"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
)

// IsValid checks if alert status is valid
func (s AlertStatus) IsValid() bool {
	switch s {
	case AlertStatusActive, AlertStatusAcknowledged, AlertStatusResolved:
		return true
	default:
		return false
	}
}

// AlertType represents the type of performance alert
type AlertType string

const (
	AlertTypePerformanceDrift AlertType = "performance_drift"
	AlertTypeDataDrift        AlertType = "data_drift"
	AlertTypeThresholdBreach  AlertType = "threshold_breach"
)

// EvaluationStatus represents the status of a performance evaluation
type EvaluationStatus string

const (
	EvaluationStatusPending   EvaluationStatus = "pending"
	EvaluationStatusRunning   EvaluationStatus = "running"
	EvaluationStatusCompleted EvaluationStatus = "completed"
	EvaluationStatusFailed    EvaluationStatus = "failed"
)

// ThresholdDirection indicates when metric change is bad
type ThresholdDirection string

const (
	ThresholdDirectionLower  ThresholdDirection = "lower"  // Lower is bad (accuracy, F1, etc.)
	ThresholdDirectionHigher ThresholdDirection = "higher" // Higher is bad (error metrics like MSE, MAE)
)

// MetricName constants for classification
const (
	MetricAccuracy  = "accuracy"
	MetricPrecision = "precision"
	MetricRecall    = "recall"
	MetricF1Score   = "f1_score"
	MetricAUCROC    = "auc_roc"
	MetricAUCPR     = "auc_pr"
	MetricPSI       = "psi" // Population Stability Index
)

// MetricName constants for regression
const (
	MetricMAE  = "mae"  // Mean Absolute Error
	MetricMSE  = "mse"  // Mean Squared Error
	MetricRMSE = "rmse" // Root Mean Squared Error
	MetricR2   = "r2"   // R-squared
	MetricMAPE = "mape" // Mean Absolute Percentage Error
)

// PerformanceBaseline represents baseline metrics for a model
type PerformanceBaseline struct {
	ID          string
	ModelID     string
	TaskType    TaskType
	MetricName  string
	MetricValue float64
	SampleCount int
	Description string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PerformanceRecord represents a single performance measurement
type PerformanceRecord struct {
	ID              string
	ModelID         string
	DatasourceID    string
	MetricName      string
	MetricValue     float64
	BaselineValue   *float64
	DriftPercentage *float64
	SampleCount     int
	WindowStart     time.Time
	WindowEnd       time.Time
	CreatedBy       string
	CreatedAt       time.Time
}

// CalculateDrift calculates the percentage drift from baseline
func (r *PerformanceRecord) CalculateDrift(baseline float64, direction ThresholdDirection) float64 {
	if baseline == 0 {
		if r.MetricValue == 0 {
			return 0
		}
		return 100.0 // 100% drift if baseline is 0 but current is not
	}

	drift := ((r.MetricValue - baseline) / math.Abs(baseline)) * 100

	// For "lower" direction, negative drift is bad (metric decreased)
	// For "higher" direction, positive drift is bad (metric increased)
	if direction == ThresholdDirectionLower {
		return -drift // Invert so positive = bad (drop in performance)
	}
	return drift
}

// PerformanceAlert represents an alert triggered by performance degradation
type PerformanceAlert struct {
	ID                  string
	ModelID             string
	RecordID            *string
	AlertType           AlertType
	Severity            AlertSeverity
	MetricName          string
	BaselineValue       float64
	CurrentValue        float64
	ThresholdPercentage float64
	DriftPercentage     float64
	Message             string
	Status              AlertStatus
	AcknowledgedBy      *string
	AcknowledgedAt      *time.Time
	ResolvedAt          *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Acknowledge marks the alert as acknowledged
func (a *PerformanceAlert) Acknowledge(username string) {
	a.Status = AlertStatusAcknowledged
	a.AcknowledgedBy = &username
	now := time.Now()
	a.AcknowledgedAt = &now
}

// Resolve marks the alert as resolved
func (a *PerformanceAlert) Resolve() {
	a.Status = AlertStatusResolved
	now := time.Now()
	a.ResolvedAt = &now
}

// PerformanceThreshold represents configurable thresholds
type PerformanceThreshold struct {
	ID                  string
	ModelID             string
	MetricName          string
	WarningThreshold    float64 // Percentage drop to trigger warning
	CriticalThreshold   float64 // Percentage drop to trigger critical
	Direction           ThresholdDirection
	Enabled             bool
	ConsecutiveBreaches int // Number of consecutive breaches before alerting
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// CheckBreach checks if a drift value breaches any threshold.
// driftPercentage must follow CalculateDrift semantics: positive = degradation, negative = improvement.
func (t *PerformanceThreshold) CheckBreach(driftPercentage float64) (breached bool, severity AlertSeverity) {
	if !t.Enabled {
		return false, ""
	}

	// Improvements (and no change) never breach; only compare magnitude of bad drift.
	if driftPercentage <= 0 {
		return false, ""
	}

	if driftPercentage >= t.CriticalThreshold {
		return true, AlertSeverityCritical
	}
	if driftPercentage >= t.WarningThreshold {
		return true, AlertSeverityWarning
	}
	return false, ""
}

// PerformanceEvaluation represents a full evaluation job
type PerformanceEvaluation struct {
	ID           string
	ModelID      string
	DatasourceID string
	Status       EvaluationStatus
	TaskType     TaskType
	Metrics      map[string]interface{}
	SampleCount  int
	ErrorMessage string
	StartedAt    *time.Time
	CompletedAt  *time.Time
	CreatedBy    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Start marks evaluation as running
func (e *PerformanceEvaluation) Start() {
	e.Status = EvaluationStatusRunning
	now := time.Now()
	e.StartedAt = &now
}

// Complete marks evaluation as completed
func (e *PerformanceEvaluation) Complete(metrics map[string]interface{}, sampleCount int) {
	e.Status = EvaluationStatusCompleted
	e.Metrics = metrics
	e.SampleCount = sampleCount
	now := time.Now()
	e.CompletedAt = &now
}

// Fail marks evaluation as failed
func (e *PerformanceEvaluation) Fail(errorMessage string) {
	e.Status = EvaluationStatusFailed
	e.ErrorMessage = errorMessage
	now := time.Now()
	e.CompletedAt = &now
}

// PerformanceSummary provides an overview of model performance status
type PerformanceSummary struct {
	ModelID             string
	TaskType            TaskType
	HasBaseline         bool
	LastEvaluationAt    *time.Time
	ActiveAlerts        int
	WarningAlerts       int
	CriticalAlerts      int
	LatestMetrics       map[string]float64
	BaselineMetrics     map[string]float64
	DriftPercentages    map[string]float64
	OverallHealthStatus string // healthy, warning, critical
	RecordCount         int
}

// PerformanceThresholdDefault represents org-wide default thresholds stored in the database
type PerformanceThresholdDefault struct {
	ID                  string
	TaskType            TaskType
	MetricName          string
	WarningThreshold    float64
	CriticalThreshold   float64
	Direction           ThresholdDirection
	Enabled             bool
	ConsecutiveBreaches int
	UpdatedBy           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// GetDefaultThresholds returns default threshold settings for a task type
func GetDefaultThresholds(taskType TaskType) []PerformanceThreshold {
	if taskType == TaskTypeClassification {
		return []PerformanceThreshold{
			{MetricName: MetricAccuracy, WarningThreshold: 5, CriticalThreshold: 10, Direction: ThresholdDirectionLower, Enabled: true, ConsecutiveBreaches: 2},
			{MetricName: MetricPrecision, WarningThreshold: 10, CriticalThreshold: 20, Direction: ThresholdDirectionLower, Enabled: true, ConsecutiveBreaches: 2},
			{MetricName: MetricRecall, WarningThreshold: 10, CriticalThreshold: 20, Direction: ThresholdDirectionLower, Enabled: true, ConsecutiveBreaches: 2},
			{MetricName: MetricF1Score, WarningThreshold: 10, CriticalThreshold: 20, Direction: ThresholdDirectionLower, Enabled: true, ConsecutiveBreaches: 2},
			{MetricName: MetricPSI, WarningThreshold: 10, CriticalThreshold: 25, Direction: ThresholdDirectionHigher, Enabled: true, ConsecutiveBreaches: 1},
		}
	}

	// Regression thresholds
	return []PerformanceThreshold{
		{MetricName: MetricMAE, WarningThreshold: 15, CriticalThreshold: 30, Direction: ThresholdDirectionHigher, Enabled: true, ConsecutiveBreaches: 2},
		{MetricName: MetricRMSE, WarningThreshold: 15, CriticalThreshold: 30, Direction: ThresholdDirectionHigher, Enabled: true, ConsecutiveBreaches: 2},
		{MetricName: MetricR2, WarningThreshold: 10, CriticalThreshold: 20, Direction: ThresholdDirectionLower, Enabled: true, ConsecutiveBreaches: 2},
		{MetricName: MetricMAPE, WarningThreshold: 15, CriticalThreshold: 30, Direction: ThresholdDirectionHigher, Enabled: true, ConsecutiveBreaches: 2},
	}
}
