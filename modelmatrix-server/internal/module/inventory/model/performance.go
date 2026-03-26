package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PerformanceBaseline stores baseline metrics when model is first deployed
type PerformanceBaseline struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ModelID     string    `gorm:"type:uuid;not null;index:idx_baseline_model"`
	TaskType    string    `gorm:"type:varchar(50);not null"` // classification, regression
	MetricName  string    `gorm:"type:varchar(100);not null"`
	MetricValue float64   `gorm:"type:decimal(18,8);not null"`
	SampleCount int       `gorm:"not null;default:0"`
	Description string    `gorm:"type:text"`
	CreatedBy   string    `gorm:"type:varchar(255);not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for PerformanceBaseline
func (PerformanceBaseline) TableName() string {
	return "model_performance_baselines"
}

// BeforeCreate generates UUID before creating record
func (b *PerformanceBaseline) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

// PerformanceRecord stores time-series performance metrics
type PerformanceRecord struct {
	ID              string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ModelID         string    `gorm:"type:uuid;not null;index:idx_record_model"`
	DatasourceID    string    `gorm:"type:uuid;not null"` // The evaluation datasource
	MetricName      string    `gorm:"type:varchar(100);not null;index:idx_record_metric"`
	MetricValue     float64   `gorm:"type:decimal(18,8);not null"`
	BaselineValue   *float64  `gorm:"type:decimal(18,8)"` // Cached baseline for comparison
	DriftPercentage *float64  `gorm:"type:decimal(10,4)"` // Percentage drift from baseline
	SampleCount     int       `gorm:"not null;default:0"`
	WindowStart     time.Time `gorm:"not null;index:idx_record_window"`
	WindowEnd       time.Time `gorm:"not null"`
	CreatedBy       string    `gorm:"type:varchar(255);not null"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
}

// TableName returns the table name for PerformanceRecord
func (PerformanceRecord) TableName() string {
	return "model_performance_records"
}

// BeforeCreate generates UUID before creating record
func (r *PerformanceRecord) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

// PerformanceAlert stores alerts when metrics cross thresholds
type PerformanceAlert struct {
	ID                  string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ModelID             string    `gorm:"type:uuid;not null;index:idx_alert_model"`
	RecordID            *string   `gorm:"type:uuid;index:idx_alert_record"` // Reference to the triggering record
	AlertType           string    `gorm:"type:varchar(50);not null"`        // performance_drift, data_drift, threshold_breach
	Severity            string    `gorm:"type:varchar(20);not null"`        // info, warning, critical
	MetricName          string    `gorm:"type:varchar(100);not null"`
	BaselineValue       float64   `gorm:"type:decimal(18,8);not null"`
	CurrentValue        float64   `gorm:"type:decimal(18,8);not null"`
	ThresholdPercentage float64   `gorm:"type:decimal(10,4);not null"`
	DriftPercentage     float64   `gorm:"type:decimal(10,4);not null"`
	Message             string    `gorm:"type:text"`
	Status              string    `gorm:"type:varchar(20);not null;default:'active';index:idx_alert_status"` // active, acknowledged, resolved
	AcknowledgedBy      *string   `gorm:"type:varchar(255)"`
	AcknowledgedAt      *time.Time
	ResolvedAt          *time.Time
	CreatedAt           time.Time `gorm:"autoCreateTime"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for PerformanceAlert
func (PerformanceAlert) TableName() string {
	return "model_performance_alerts"
}

// BeforeCreate generates UUID before creating record
func (a *PerformanceAlert) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// PerformanceThreshold stores configurable thresholds for each model
type PerformanceThreshold struct {
	ID                    string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ModelID               string    `gorm:"type:uuid;not null;uniqueIndex:idx_threshold_model_metric"`
	MetricName            string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_threshold_model_metric"`
	WarningThreshold      float64   `gorm:"type:decimal(10,4);not null;default:10.0"`  // % drop triggers warning
	CriticalThreshold     float64   `gorm:"type:decimal(10,4);not null;default:20.0"`  // % drop triggers critical
	Direction             string    `gorm:"type:varchar(10);not null;default:'lower'"` // lower = bad when decreases, higher = bad when increases
	Enabled               bool      `gorm:"not null;default:true"`
	ConsecutiveBreaches   int       `gorm:"not null;default:1"` // Number of consecutive breaches before alerting
	CreatedAt             time.Time `gorm:"autoCreateTime"`
	UpdatedAt             time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for PerformanceThreshold
func (PerformanceThreshold) TableName() string {
	return "model_performance_thresholds"
}

// BeforeCreate generates UUID before creating record
func (t *PerformanceThreshold) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

// PerformanceThresholdDefault stores org-wide default thresholds used when a new model baseline is created
type PerformanceThresholdDefault struct {
	ID                  string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TaskType            string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_threshold_default_task_metric"`
	MetricName          string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_threshold_default_task_metric"`
	WarningThreshold    float64   `gorm:"type:decimal(10,4);not null"`
	CriticalThreshold   float64   `gorm:"type:decimal(10,4);not null"`
	Direction           string    `gorm:"type:varchar(10);not null;default:'lower'"`
	Enabled             bool      `gorm:"not null;default:true"`
	ConsecutiveBreaches int       `gorm:"not null;default:2"`
	UpdatedBy           string    `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt           time.Time `gorm:"autoCreateTime"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for PerformanceThresholdDefault
func (PerformanceThresholdDefault) TableName() string {
	return "performance_threshold_defaults"
}

// BeforeCreate generates UUID before creating record
func (t *PerformanceThresholdDefault) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

// PerformanceEvaluation stores the full evaluation job results
type PerformanceEvaluation struct {
	ID           string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ModelID      string    `gorm:"type:uuid;not null;index:idx_evaluation_model"`
	DatasourceID string    `gorm:"type:uuid;not null"` // Evaluation data with actuals
	Status       string    `gorm:"type:varchar(20);not null;default:'pending';index:idx_evaluation_status"` // pending, running, completed, failed
	TaskType     string    `gorm:"type:varchar(50);not null"`  // classification, regression
	Metrics      JSONMap   `gorm:"type:jsonb"`                 // All computed metrics
	SampleCount  int       `gorm:"not null;default:0"`
	ErrorMessage string    `gorm:"type:text"`
	StartedAt    *time.Time
	CompletedAt  *time.Time
	CreatedBy    string    `gorm:"type:varchar(255);not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for PerformanceEvaluation
func (PerformanceEvaluation) TableName() string {
	return "model_performance_evaluations"
}

// BeforeCreate generates UUID before creating record
func (e *PerformanceEvaluation) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	return nil
}
