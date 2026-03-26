package dto

import (
	"time"
)

// ===== Request DTOs =====

// CreateBaselineRequest represents request to create/update baseline metrics
type CreateBaselineRequest struct {
	Metrics     map[string]float64 `json:"metrics" binding:"required" example:"{\"accuracy\": 0.95, \"f1_score\": 0.93}"`
	SampleCount int                `json:"sample_count,omitempty" example:"10000"`
	Description string             `json:"description,omitempty" example:"Baseline from initial deployment"`
}

// EvaluatePerformanceRequest represents request to evaluate model performance
type EvaluatePerformanceRequest struct {
	DatasourceID     string `json:"datasource_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ActualColumn     string `json:"actual_column" binding:"required" example:"actual_target"`
	PredictionColumn string `json:"prediction_column,omitempty" example:"prediction"` // Optional, defaults to model's prediction column
}

// RecordPerformanceRequest represents request to manually record performance metrics
type RecordPerformanceRequest struct {
	DatasourceID string             `json:"datasource_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Metrics      map[string]float64 `json:"metrics" binding:"required" example:"{\"accuracy\": 0.92, \"f1_score\": 0.90}"`
	SampleCount  int                `json:"sample_count,omitempty" example:"5000"`
	WindowStart  *time.Time         `json:"window_start,omitempty"`
	WindowEnd    *time.Time         `json:"window_end,omitempty"`
}

// UpdateThresholdRequest represents request to update performance threshold
type UpdateThresholdRequest struct {
	MetricName          string   `json:"metric_name" binding:"required" example:"accuracy"`
	WarningThreshold    *float64 `json:"warning_threshold,omitempty" example:"10.0"`
	CriticalThreshold   *float64 `json:"critical_threshold,omitempty" example:"20.0"`
	Direction           *string  `json:"direction,omitempty" example:"lower"` // lower or higher
	Enabled             *bool    `json:"enabled,omitempty" example:"true"`
	ConsecutiveBreaches *int     `json:"consecutive_breaches,omitempty" example:"2"`
}

// UpdateAlertRequest represents request to update alert status
type UpdateAlertRequest struct {
	Status string `json:"status" binding:"required,oneof=acknowledged resolved" example:"acknowledged"`
}

// GetPerformanceHistoryParams represents query parameters for performance history
type GetPerformanceHistoryParams struct {
	MetricName string     `form:"metric_name" example:"accuracy"`
	Limit      int        `form:"limit" binding:"omitempty,min=1,max=1000" example:"100"`
	StartTime  *time.Time `form:"start_time"`
	EndTime    *time.Time `form:"end_time"`
}

// GetAlertsParams represents query parameters for alerts
type GetAlertsParams struct {
	Status string `form:"status" binding:"omitempty,oneof=active acknowledged resolved" example:"active"`
	Limit  int    `form:"limit" binding:"omitempty,min=1,max=100" example:"50"`
}

// ===== Response DTOs =====

// PerformanceBaselineResponse represents baseline metrics response
type PerformanceBaselineResponse struct {
	ID          string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ModelID     string    `json:"model_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	TaskType    string    `json:"task_type" example:"classification"`
	MetricName  string    `json:"metric_name" example:"accuracy"`
	MetricValue float64   `json:"metric_value" example:"0.95"`
	SampleCount int       `json:"sample_count" example:"10000"`
	Description string    `json:"description,omitempty" example:"Baseline from initial deployment"`
	CreatedBy   string    `json:"created_by" example:"admin"`
	CreatedAt   time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// PerformanceRecordResponse represents a performance measurement
type PerformanceRecordResponse struct {
	ID              string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ModelID         string    `json:"model_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	DatasourceID    string    `json:"datasource_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	MetricName      string    `json:"metric_name" example:"accuracy"`
	MetricValue     float64   `json:"metric_value" example:"0.92"`
	BaselineValue   *float64  `json:"baseline_value,omitempty" example:"0.95"`
	DriftPercentage *float64  `json:"drift_percentage,omitempty" example:"-3.16"`
	SampleCount     int       `json:"sample_count" example:"5000"`
	WindowStart     time.Time `json:"window_start" example:"2024-01-15T00:00:00Z"`
	WindowEnd       time.Time `json:"window_end" example:"2024-01-16T00:00:00Z"`
	CreatedBy       string    `json:"created_by" example:"admin"`
	CreatedAt       time.Time `json:"created_at" example:"2024-01-16T10:30:00Z"`
}

// PerformanceAlertResponse represents a performance alert
type PerformanceAlertResponse struct {
	ID                  string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ModelID             string     `json:"model_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	RecordID            *string    `json:"record_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	AlertType           string     `json:"alert_type" example:"performance_drift"`
	Severity            string     `json:"severity" example:"warning"`
	MetricName          string     `json:"metric_name" example:"accuracy"`
	BaselineValue       float64    `json:"baseline_value" example:"0.95"`
	CurrentValue        float64    `json:"current_value" example:"0.82"`
	ThresholdPercentage float64    `json:"threshold_percentage" example:"10.0"`
	DriftPercentage     float64    `json:"drift_percentage" example:"13.68"`
	Message             string     `json:"message" example:"Accuracy dropped 13.68% below baseline (threshold: 10%)"`
	Status              string     `json:"status" example:"active"`
	AcknowledgedBy      *string    `json:"acknowledged_by,omitempty"`
	AcknowledgedAt      *time.Time `json:"acknowledged_at,omitempty"`
	ResolvedAt          *time.Time `json:"resolved_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at" example:"2024-01-16T10:30:00Z"`
	UpdatedAt           time.Time  `json:"updated_at" example:"2024-01-16T10:30:00Z"`
}

// PerformanceThresholdResponse represents a threshold configuration
type PerformanceThresholdResponse struct {
	ID                  string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ModelID             string    `json:"model_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	MetricName          string    `json:"metric_name" example:"accuracy"`
	WarningThreshold    float64   `json:"warning_threshold" example:"10.0"`
	CriticalThreshold   float64   `json:"critical_threshold" example:"20.0"`
	Direction           string    `json:"direction" example:"lower"`
	Enabled             bool      `json:"enabled" example:"true"`
	ConsecutiveBreaches int       `json:"consecutive_breaches" example:"2"`
	CreatedAt           time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt           time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// PerformanceEvaluationResponse represents an evaluation job result
type PerformanceEvaluationResponse struct {
	ID           string                 `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ModelID      string                 `json:"model_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	DatasourceID string                 `json:"datasource_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	Status       string                 `json:"status" example:"completed"`
	TaskType     string                 `json:"task_type" example:"classification"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	SampleCount  int                    `json:"sample_count" example:"5000"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	CreatedBy    string                 `json:"created_by" example:"admin"`
	CreatedAt    time.Time              `json:"created_at" example:"2024-01-16T10:30:00Z"`
	UpdatedAt    time.Time              `json:"updated_at" example:"2024-01-16T10:35:00Z"`
}

// PerformanceSummaryResponse provides an overview of model performance status
type PerformanceSummaryResponse struct {
	ModelID             string              `json:"model_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	TaskType            string              `json:"task_type" example:"classification"`
	HasBaseline         bool                `json:"has_baseline" example:"true"`
	LastEvaluationAt    *time.Time          `json:"last_evaluation_at,omitempty"`
	ActiveAlerts        int                 `json:"active_alerts" example:"2"`
	WarningAlerts       int                 `json:"warning_alerts" example:"1"`
	CriticalAlerts      int                 `json:"critical_alerts" example:"1"`
	LatestMetrics       map[string]float64  `json:"latest_metrics,omitempty"`
	BaselineMetrics     map[string]float64  `json:"baseline_metrics,omitempty"`
	DriftPercentages    map[string]float64  `json:"drift_percentages,omitempty"`
	OverallHealthStatus string              `json:"overall_health_status" example:"warning"` // healthy, warning, critical
	RecordCount         int                 `json:"record_count" example:"45"`
}

// MetricTimeSeriesResponse represents time-series data for a single metric
type MetricTimeSeriesResponse struct {
	MetricName string                     `json:"metric_name" example:"accuracy"`
	Baseline   *float64                   `json:"baseline,omitempty" example:"0.95"`
	DataPoints []MetricDataPointResponse  `json:"data_points"`
}

// MetricDataPointResponse represents a single data point in the time series
type MetricDataPointResponse struct {
	Timestamp       time.Time `json:"timestamp" example:"2024-01-16T00:00:00Z"`
	Value           float64   `json:"value" example:"0.92"`
	DriftPercentage *float64  `json:"drift_percentage,omitempty" example:"-3.16"`
	SampleCount     int       `json:"sample_count" example:"5000"`
}

// PerformanceHistoryResponse contains historical performance data
type PerformanceHistoryResponse struct {
	ModelID    string                     `json:"model_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Records    []PerformanceRecordResponse `json:"records"`
	TotalCount int                        `json:"total_count" example:"45"`
}

// AlertsListResponse contains a list of alerts
type AlertsListResponse struct {
	Alerts     []PerformanceAlertResponse `json:"alerts"`
	TotalCount int                        `json:"total_count" example:"3"`
}

// EvaluationsListResponse contains a list of evaluations
type EvaluationsListResponse struct {
	Evaluations []PerformanceEvaluationResponse `json:"evaluations"`
	TotalCount  int                             `json:"total_count" example:"10"`
}

// ThresholdsListResponse contains a list of thresholds
type ThresholdsListResponse struct {
	Thresholds []PerformanceThresholdResponse `json:"thresholds"`
}

// UpdateThresholdDefaultRequest updates a global default threshold
type UpdateThresholdDefaultRequest struct {
	TaskType            string   `json:"task_type" binding:"required,oneof=classification regression" example:"classification"`
	MetricName          string   `json:"metric_name" binding:"required" example:"accuracy"`
	WarningThreshold    *float64 `json:"warning_threshold,omitempty" example:"5.0"`
	CriticalThreshold   *float64 `json:"critical_threshold,omitempty" example:"10.0"`
	Direction           *string  `json:"direction,omitempty" example:"lower"`
	Enabled             *bool    `json:"enabled,omitempty" example:"true"`
	ConsecutiveBreaches *int     `json:"consecutive_breaches,omitempty" example:"2"`
}

// PerformanceThresholdDefaultResponse is the API response for a global threshold default
type PerformanceThresholdDefaultResponse struct {
	ID                  string    `json:"id"`
	TaskType            string    `json:"task_type"`
	MetricName          string    `json:"metric_name"`
	WarningThreshold    float64   `json:"warning_threshold"`
	CriticalThreshold   float64   `json:"critical_threshold"`
	Direction           string    `json:"direction"`
	Enabled             bool      `json:"enabled"`
	ConsecutiveBreaches int       `json:"consecutive_breaches"`
	UpdatedBy           string    `json:"updated_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ThresholdDefaultsListResponse contains global defaults for a task type
type ThresholdDefaultsListResponse struct {
	TaskType  string                                `json:"task_type"`
	Defaults  []PerformanceThresholdDefaultResponse `json:"defaults"`
}

// BaselinesListResponse contains a list of baselines
type BaselinesListResponse struct {
	Baselines []PerformanceBaselineResponse `json:"baselines"`
}

// EvaluationCallbackRequest represents callback from compute service after evaluation
type EvaluationCallbackRequest struct {
	EvaluationID string                 `json:"evaluation_id"`
	ModelID      string                 `json:"model_id"`
	Status       string                 `json:"status"` // completed or failed
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	SampleCount  int                    `json:"sample_count,omitempty"`
	Error        string                 `json:"error,omitempty"`
}
