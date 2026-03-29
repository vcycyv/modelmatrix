package application

// Unit tests for PerformanceServiceImpl.
// The service uses repository.PerformanceRepository and repository.ModelRepository — both interfaces —
// so all core methods are fully testable without any infrastructure dependency.
// Only StartEvaluation/HandleEvaluationCallback (computeClient) are excluded here.

import (
	"testing"
	"time"

	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"
	"modelmatrix-server/internal/module/inventory/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// fakePerformanceRepo — in-memory implementation of repository.PerformanceRepository
// ---------------------------------------------------------------------------

type fakePerformanceRepo struct {
	baselines  []*domain.PerformanceBaseline
	records    []*domain.PerformanceRecord
	alerts     []*domain.PerformanceAlert
	thresholds []*domain.PerformanceThreshold
	defaults   []*domain.PerformanceThresholdDefault
	evals      []*domain.PerformanceEvaluation

	createAlertErr error
}

var _ repository.PerformanceRepository = (*fakePerformanceRepo)(nil)

func newFakePerfRepo() *fakePerformanceRepo { return &fakePerformanceRepo{} }

// baseline
func (r *fakePerformanceRepo) CreateBaseline(b *domain.PerformanceBaseline) error {
	r.baselines = append(r.baselines, b)
	return nil
}
func (r *fakePerformanceRepo) UpdateBaseline(b *domain.PerformanceBaseline) error {
	for i, bl := range r.baselines {
		if bl.ID == b.ID {
			r.baselines[i] = b
		}
	}
	return nil
}
func (r *fakePerformanceRepo) GetBaselineByModelAndMetric(modelID, metricName string) (*domain.PerformanceBaseline, error) {
	for _, bl := range r.baselines {
		if bl.ModelID == modelID && bl.MetricName == metricName {
			return bl, nil
		}
	}
	return nil, domain.ErrBaselineNotFound
}
func (r *fakePerformanceRepo) GetBaselinesByModelID(modelID string) ([]domain.PerformanceBaseline, error) {
	var out []domain.PerformanceBaseline
	for _, bl := range r.baselines {
		if bl.ModelID == modelID {
			out = append(out, *bl)
		}
	}
	return out, nil
}
func (r *fakePerformanceRepo) DeleteBaselinesByModelID(modelID string) error { return nil }

// record
func (r *fakePerformanceRepo) CreateRecord(rec *domain.PerformanceRecord) error {
	r.records = append(r.records, rec)
	return nil
}
func (r *fakePerformanceRepo) CreateRecords(recs []domain.PerformanceRecord) error {
	for i := range recs {
		cp := recs[i]
		r.records = append(r.records, &cp)
	}
	return nil
}
func (r *fakePerformanceRepo) GetRecordsByModelID(modelID string, limit int, start, end *time.Time) ([]domain.PerformanceRecord, error) {
	var out []domain.PerformanceRecord
	for _, rec := range r.records {
		if rec.ModelID == modelID {
			out = append(out, *rec)
		}
	}
	return out, nil
}
func (r *fakePerformanceRepo) GetRecordsByMetric(modelID, metricName string, limit int) ([]domain.PerformanceRecord, error) {
	var out []domain.PerformanceRecord
	for _, rec := range r.records {
		if rec.ModelID == modelID && rec.MetricName == metricName {
			out = append(out, *rec)
		}
	}
	return out, nil
}
func (r *fakePerformanceRepo) GetLatestRecordsByModelID(modelID string) ([]domain.PerformanceRecord, error) {
	return r.GetRecordsByModelID(modelID, 1, nil, nil)
}
func (r *fakePerformanceRepo) DeleteRecordsByModelID(modelID string) error { return nil }

// alert
func (r *fakePerformanceRepo) CreateAlert(alert *domain.PerformanceAlert) error {
	if r.createAlertErr != nil {
		return r.createAlertErr
	}
	r.alerts = append(r.alerts, alert)
	return nil
}
func (r *fakePerformanceRepo) UpdateAlert(alert *domain.PerformanceAlert) error {
	for i, a := range r.alerts {
		if a.ID == alert.ID {
			r.alerts[i] = alert
		}
	}
	return nil
}
func (r *fakePerformanceRepo) GetAlertByID(id string) (*domain.PerformanceAlert, error) {
	for _, a := range r.alerts {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, domain.ErrAlertNotFound
}
func (r *fakePerformanceRepo) GetActiveAlertsByModelID(modelID string) ([]domain.PerformanceAlert, error) {
	return nil, nil
}
func (r *fakePerformanceRepo) GetAlertsByModelID(modelID string, status string, limit int) ([]domain.PerformanceAlert, error) {
	var out []domain.PerformanceAlert
	for _, a := range r.alerts {
		if a.ModelID == modelID {
			out = append(out, *a)
		}
	}
	return out, nil
}
func (r *fakePerformanceRepo) CountActiveAlertsByModelID(modelID string) (int, int, int, error) {
	return 0, 0, 0, nil
}
func (r *fakePerformanceRepo) DeleteAlertsByModelID(modelID string) error { return nil }

// threshold
func (r *fakePerformanceRepo) CreateThreshold(t *domain.PerformanceThreshold) error {
	r.thresholds = append(r.thresholds, t)
	return nil
}
func (r *fakePerformanceRepo) UpdateThreshold(t *domain.PerformanceThreshold) error {
	for i, th := range r.thresholds {
		if th.ID == t.ID {
			r.thresholds[i] = t
		}
	}
	return nil
}
func (r *fakePerformanceRepo) GetThresholdByModelAndMetric(modelID, metricName string) (*domain.PerformanceThreshold, error) {
	for _, th := range r.thresholds {
		if th.ModelID == modelID && th.MetricName == metricName {
			return th, nil
		}
	}
	return nil, domain.ErrThresholdNotFound
}
func (r *fakePerformanceRepo) GetThresholdsByModelID(modelID string) ([]domain.PerformanceThreshold, error) {
	var out []domain.PerformanceThreshold
	for _, th := range r.thresholds {
		if th.ModelID == modelID {
			out = append(out, *th)
		}
	}
	return out, nil
}
func (r *fakePerformanceRepo) DeleteThresholdsByModelID(modelID string) error { return nil }

// threshold defaults
func (r *fakePerformanceRepo) GetThresholdDefaultsByTaskType(taskType string) ([]domain.PerformanceThresholdDefault, error) {
	var out []domain.PerformanceThresholdDefault
	for _, d := range r.defaults {
		if string(d.TaskType) == taskType {
			out = append(out, *d)
		}
	}
	return out, nil
}
func (r *fakePerformanceRepo) UpsertThresholdDefault(d *domain.PerformanceThresholdDefault) error {
	r.defaults = append(r.defaults, d)
	return nil
}

// evaluations
func (r *fakePerformanceRepo) CreateEvaluation(e *domain.PerformanceEvaluation) error {
	r.evals = append(r.evals, e)
	return nil
}
func (r *fakePerformanceRepo) UpdateEvaluation(e *domain.PerformanceEvaluation) error { return nil }
func (r *fakePerformanceRepo) GetEvaluationByID(id string) (*domain.PerformanceEvaluation, error) {
	for _, ev := range r.evals {
		if ev.ID == id {
			return ev, nil
		}
	}
	return nil, domain.ErrEvaluationNotFound
}
func (r *fakePerformanceRepo) GetEvaluationsByModelID(modelID string, limit int) ([]domain.PerformanceEvaluation, error) {
	var out []domain.PerformanceEvaluation
	for _, ev := range r.evals {
		if ev.ModelID == modelID {
			out = append(out, *ev)
		}
	}
	return out, nil
}
func (r *fakePerformanceRepo) GetPendingEvaluations() ([]domain.PerformanceEvaluation, error) {
	return nil, nil
}
func (r *fakePerformanceRepo) DeleteEvaluationsByModelID(modelID string) error { return nil }

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func buildPerfSvc(modelRepo *mockModelRepo, perfRepo *fakePerformanceRepo) PerformanceService {
	return NewPerformanceService(perfRepo, modelRepo)
}

func modelWithType(id string, mType string) *domain.Model {
	return &domain.Model{ID: id, Name: "TestModel", ModelType: mType, Status: domain.ModelStatusActive}
}

const testTaskType = "classification" // matches domain.TaskTypeClassification

// ---------------------------------------------------------------------------
// countRecordsWithDrift — pure function, no service state
// ---------------------------------------------------------------------------

func TestCountRecordsWithDrift_None(t *testing.T) {
	records := []domain.PerformanceRecord{
		{MetricName: "accuracy"},
		{MetricName: "f1"},
	}
	assert.Equal(t, 0, countRecordsWithDrift(records))
}

func TestCountRecordsWithDrift_Some(t *testing.T) {
	drift := 5.0
	records := []domain.PerformanceRecord{
		{MetricName: "accuracy", DriftPercentage: &drift},
		{MetricName: "f1"},
		{MetricName: "precision", DriftPercentage: &drift},
	}
	assert.Equal(t, 2, countRecordsWithDrift(records))
}

// ---------------------------------------------------------------------------
// CreateBaseline
// ---------------------------------------------------------------------------

func TestPerformanceService_CreateBaseline_Success(t *testing.T) {
	perfRepo := newFakePerfRepo()
	modelRepo := &mockModelRepo{
		getByID: func(id string) (*domain.Model, error) {
			return modelWithType(id, testTaskType), nil
		},
	}
	svc := buildPerfSvc(modelRepo, perfRepo)

	result, err := svc.CreateBaseline("m1", &dto.CreateBaselineRequest{
		Metrics:     map[string]float64{"accuracy": 0.92, "f1": 0.88},
		SampleCount: 1000,
	}, "alice")
	require.NoError(t, err)
	assert.Len(t, perfRepo.baselines, 2)
	assert.NotNil(t, result)
}

func TestPerformanceService_CreateBaseline_ModelNotFound(t *testing.T) {
	perfRepo := newFakePerfRepo()
	modelRepo := &mockModelRepo{
		getByID: func(id string) (*domain.Model, error) {
			return nil, domain.ErrModelNotFound
		},
	}
	svc := buildPerfSvc(modelRepo, perfRepo)
	_, err := svc.CreateBaseline("missing", &dto.CreateBaselineRequest{
		Metrics: map[string]float64{"accuracy": 0.9},
	}, "alice")
	require.Error(t, err)
	assert.Equal(t, domain.ErrModelNotFound, err)
}

func TestPerformanceService_CreateBaseline_UpdatesExisting(t *testing.T) {
	perfRepo := newFakePerfRepo()
	// Pre-seed an existing baseline
	perfRepo.baselines = append(perfRepo.baselines, &domain.PerformanceBaseline{
		ID: "bl1", ModelID: "m1", MetricName: "accuracy", MetricValue: 0.85,
	})
	modelRepo := &mockModelRepo{
		getByID: func(id string) (*domain.Model, error) {
			return modelWithType(id, testTaskType), nil
		},
	}
	svc := buildPerfSvc(modelRepo, perfRepo)
	_, err := svc.CreateBaseline("m1", &dto.CreateBaselineRequest{
		Metrics: map[string]float64{"accuracy": 0.95}, // Update existing
	}, "alice")
	require.NoError(t, err)
	// Should update, not create a new one
	count := 0
	for _, bl := range perfRepo.baselines {
		if bl.MetricName == "accuracy" {
			count++
		}
	}
	assert.Equal(t, 1, count, "should not duplicate existing baseline")
}

// ---------------------------------------------------------------------------
// GetBaselines
// ---------------------------------------------------------------------------

func TestPerformanceService_GetBaselines_Success(t *testing.T) {
	perfRepo := newFakePerfRepo()
	perfRepo.baselines = []*domain.PerformanceBaseline{
		{ID: "bl1", ModelID: "m1", MetricName: "accuracy", MetricValue: 0.92},
		{ID: "bl2", ModelID: "m1", MetricName: "f1", MetricValue: 0.88},
	}
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)
	result, err := svc.GetBaselines("m1")
	require.NoError(t, err)
	assert.Len(t, result.Baselines, 2)
}

func TestPerformanceService_GetBaselines_Empty(t *testing.T) {
	svc := buildPerfSvc(&mockModelRepo{}, newFakePerfRepo())
	result, err := svc.GetBaselines("m1")
	require.NoError(t, err)
	assert.Empty(t, result.Baselines)
}

// ---------------------------------------------------------------------------
// GetThresholds
// ---------------------------------------------------------------------------

func TestPerformanceService_GetThresholds_Success(t *testing.T) {
	perfRepo := newFakePerfRepo()
	perfRepo.thresholds = []*domain.PerformanceThreshold{
		{ModelID: "m1", MetricName: "accuracy", WarningThreshold: 5.0, CriticalThreshold: 10.0},
	}
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)
	result, err := svc.GetThresholds("m1")
	require.NoError(t, err)
	assert.Len(t, result.Thresholds, 1)
}

// ---------------------------------------------------------------------------
// InitializeDefaultThresholds
// ---------------------------------------------------------------------------

func TestPerformanceService_InitializeDefaultThresholds_CreatesDefaults(t *testing.T) {
	perfRepo := newFakePerfRepo()
	// Seed global defaults for classification
	perfRepo.defaults = []*domain.PerformanceThresholdDefault{
		{TaskType: domain.TaskTypeClassification, MetricName: "accuracy", WarningThreshold: 5.0, CriticalThreshold: 10.0, Enabled: true},
		{TaskType: domain.TaskTypeClassification, MetricName: "f1", WarningThreshold: 7.0, CriticalThreshold: 15.0, Enabled: true},
	}
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	err := svc.InitializeDefaultThresholds("m1", domain.TaskTypeClassification)
	require.NoError(t, err)
	// Should create 2 thresholds for the model
	assert.Len(t, perfRepo.thresholds, 2)
}

func TestPerformanceService_InitializeDefaultThresholds_SkipsExisting(t *testing.T) {
	perfRepo := newFakePerfRepo()
	perfRepo.defaults = []*domain.PerformanceThresholdDefault{
		{TaskType: domain.TaskTypeClassification, MetricName: "accuracy", WarningThreshold: 5.0, CriticalThreshold: 10.0, Enabled: true},
	}
	// Pre-existing threshold for accuracy
	perfRepo.thresholds = []*domain.PerformanceThreshold{
		{ModelID: "m1", MetricName: "accuracy", WarningThreshold: 3.0},
	}
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	err := svc.InitializeDefaultThresholds("m1", domain.TaskTypeClassification)
	require.NoError(t, err)
	// Should NOT add a duplicate — still 1 threshold
	assert.Len(t, perfRepo.thresholds, 1)
}

// ---------------------------------------------------------------------------
// UpdateAlert
// ---------------------------------------------------------------------------

func TestPerformanceService_UpdateAlert_Success(t *testing.T) {
	perfRepo := newFakePerfRepo()
	perfRepo.alerts = []*domain.PerformanceAlert{
		{ID: "a1", ModelID: "m1", Status: domain.AlertStatusActive, Severity: domain.AlertSeverityWarning},
	}
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	result, err := svc.UpdateAlert("a1", &dto.UpdateAlertRequest{
		Status: string(domain.AlertStatusAcknowledged),
	}, "alice")
	require.NoError(t, err)
	assert.Equal(t, string(domain.AlertStatusAcknowledged), result.Status)
}

func TestPerformanceService_UpdateAlert_NotFound(t *testing.T) {
	svc := buildPerfSvc(&mockModelRepo{}, newFakePerfRepo())
	_, err := svc.UpdateAlert("missing", &dto.UpdateAlertRequest{
		Status: "acknowledged",
	}, "alice")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// RecordPerformance — drift calculation and alert generation
// ---------------------------------------------------------------------------

func TestPerformanceService_RecordPerformance_NoDrift_NoBaseline(t *testing.T) {
	// No baselines → records created, but no drift calculated
	perfRepo := newFakePerfRepo()
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	_, err := svc.RecordPerformance("m1", &dto.RecordPerformanceRequest{
		Metrics:     map[string]float64{"accuracy": 0.88},
		SampleCount: 500,
	}, "alice")
	require.NoError(t, err)
	require.Len(t, perfRepo.records, 1)
	assert.Nil(t, perfRepo.records[0].DriftPercentage, "no drift without baseline")
}

func TestPerformanceService_RecordPerformance_DriftCalculated(t *testing.T) {
	perfRepo := newFakePerfRepo()
	// Baseline: accuracy = 0.92
	perfRepo.baselines = []*domain.PerformanceBaseline{
		{ModelID: "m1", MetricName: "accuracy", MetricValue: 0.92},
	}
	// Threshold: lower is worse → drift = how much it dropped
	perfRepo.thresholds = []*domain.PerformanceThreshold{
		{ModelID: "m1", MetricName: "accuracy", WarningThreshold: 5.0, CriticalThreshold: 10.0,
			Direction: domain.ThresholdDirectionLower, Enabled: true},
	}
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	_, err := svc.RecordPerformance("m1", &dto.RecordPerformanceRequest{
		Metrics:     map[string]float64{"accuracy": 0.80}, // dropped from 0.92
		SampleCount: 500,
	}, "alice")
	require.NoError(t, err)
	require.Len(t, perfRepo.records, 1)
	assert.NotNil(t, perfRepo.records[0].DriftPercentage, "drift should be calculated with baseline")
}

func TestPerformanceService_RecordPerformance_AlertCreatedOnThresholdBreach(t *testing.T) {
	perfRepo := newFakePerfRepo()
	// Baseline: accuracy = 0.92
	perfRepo.baselines = []*domain.PerformanceBaseline{
		{ModelID: "m1", MetricName: "accuracy", MetricValue: 0.92},
	}
	// Threshold: warning at 5%, critical at 10% drop
	perfRepo.thresholds = []*domain.PerformanceThreshold{
		{ModelID: "m1", MetricName: "accuracy",
			WarningThreshold: 5.0, CriticalThreshold: 10.0,
			Direction: domain.ThresholdDirectionLower, Enabled: true},
	}
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	// Record accuracy at 0.80 — ~13% drop from 0.92, above critical threshold
	_, err := svc.RecordPerformance("m1", &dto.RecordPerformanceRequest{
		Metrics:     map[string]float64{"accuracy": 0.80},
		SampleCount: 500,
	}, "alice")
	require.NoError(t, err)
	assert.NotEmpty(t, perfRepo.alerts, "alert should be created when threshold is breached")
}

func TestPerformanceService_RecordPerformance_NoAlertWhenWithinThreshold(t *testing.T) {
	perfRepo := newFakePerfRepo()
	perfRepo.baselines = []*domain.PerformanceBaseline{
		{ModelID: "m1", MetricName: "accuracy", MetricValue: 0.92},
	}
	// Threshold: warning at 5% — current value is 0.91, only ~1% drop
	perfRepo.thresholds = []*domain.PerformanceThreshold{
		{ModelID: "m1", MetricName: "accuracy",
			WarningThreshold: 5.0, CriticalThreshold: 10.0,
			Direction: domain.ThresholdDirectionLower, Enabled: true},
	}
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	_, err := svc.RecordPerformance("m1", &dto.RecordPerformanceRequest{
		Metrics:     map[string]float64{"accuracy": 0.91}, // ~1% drop, within warning threshold
		SampleCount: 500,
	}, "alice")
	require.NoError(t, err)
	assert.Empty(t, perfRepo.alerts, "no alert when within threshold")
}

// ---------------------------------------------------------------------------
// GetAlerts
// ---------------------------------------------------------------------------

func TestPerformanceService_GetAlerts_Success(t *testing.T) {
	perfRepo := newFakePerfRepo()
	perfRepo.alerts = []*domain.PerformanceAlert{
		{ID: "a1", ModelID: "m1", Status: domain.AlertStatusActive},
		{ID: "a2", ModelID: "m1", Status: domain.AlertStatusAcknowledged},
	}
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)
	result, err := svc.GetAlerts("m1", &dto.GetAlertsParams{})
	require.NoError(t, err)
	assert.Len(t, result.Alerts, 2)
}

// ---------------------------------------------------------------------------
// GetMetricTimeSeries
// ---------------------------------------------------------------------------

func TestPerformanceService_GetMetricTimeSeries_ReturnsDataPointsAndBaseline(t *testing.T) {
	perfRepo := newFakePerfRepo()
	baseline := 0.92
	perfRepo.baselines = []*domain.PerformanceBaseline{
		{ModelID: "m1", MetricName: "accuracy", MetricValue: baseline},
	}
	drift := 3.5
	perfRepo.records = []*domain.PerformanceRecord{
		{ModelID: "m1", MetricName: "accuracy", MetricValue: 0.89, DriftPercentage: &drift, SampleCount: 100},
	}
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	result, err := svc.GetMetricTimeSeries("m1", "accuracy", 10)
	require.NoError(t, err)
	assert.Equal(t, "accuracy", result.MetricName)
	require.NotNil(t, result.Baseline)
	assert.Equal(t, baseline, *result.Baseline)
	require.Len(t, result.DataPoints, 1)
	assert.Equal(t, 0.89, result.DataPoints[0].Value)
	require.NotNil(t, result.DataPoints[0].DriftPercentage)
	assert.Equal(t, drift, *result.DataPoints[0].DriftPercentage)
}

func TestPerformanceService_GetMetricTimeSeries_DefaultLimit(t *testing.T) {
	perfRepo := newFakePerfRepo()
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)
	// limit <= 0 should default to 100 (no panic)
	result, err := svc.GetMetricTimeSeries("m1", "accuracy", 0)
	require.NoError(t, err)
	assert.Empty(t, result.DataPoints)
}

func TestPerformanceService_GetMetricTimeSeries_NoBaseline_NilBaselinePointer(t *testing.T) {
	perfRepo := newFakePerfRepo() // no baselines
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)
	result, err := svc.GetMetricTimeSeries("m1", "f1_score", 10)
	require.NoError(t, err)
	assert.Nil(t, result.Baseline, "no baseline pointer when none recorded")
}

// ---------------------------------------------------------------------------
// GetEvaluations and GetEvaluation
// ---------------------------------------------------------------------------

func TestPerformanceService_GetEvaluations_ReturnsMostRecent(t *testing.T) {
	perfRepo := newFakePerfRepo()
	perfRepo.evals = []*domain.PerformanceEvaluation{
		{ID: "ev1", ModelID: "m1", Status: domain.EvaluationStatusCompleted},
		{ID: "ev2", ModelID: "m1", Status: domain.EvaluationStatusFailed},
	}
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	result, err := svc.GetEvaluations("m1", 10)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}

func TestPerformanceService_GetEvaluations_DefaultLimit(t *testing.T) {
	perfRepo := newFakePerfRepo()
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)
	result, err := svc.GetEvaluations("m1", 0) // 0 triggers default
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPerformanceService_GetEvaluation_Found(t *testing.T) {
	perfRepo := newFakePerfRepo()
	perfRepo.evals = []*domain.PerformanceEvaluation{
		{ID: "ev1", ModelID: "m1", Status: domain.EvaluationStatusCompleted},
	}
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	result, err := svc.GetEvaluation("ev1")
	require.NoError(t, err)
	assert.Equal(t, "ev1", result.ID)
}

func TestPerformanceService_GetEvaluation_NotFound(t *testing.T) {
	svc := buildPerfSvc(&mockModelRepo{}, newFakePerfRepo())
	_, err := svc.GetEvaluation("missing")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// UpdateThreshold — create new vs update existing
// ---------------------------------------------------------------------------

func TestPerformanceService_UpdateThreshold_CreatesNewWhenNotExists(t *testing.T) {
	perfRepo := newFakePerfRepo() // no thresholds
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	warn, crit := 8.0, 15.0
	result, err := svc.UpdateThreshold("m1", &dto.UpdateThresholdRequest{
		MetricName:        "accuracy",
		WarningThreshold:  &warn,
		CriticalThreshold: &crit,
	})
	require.NoError(t, err)
	assert.Equal(t, "accuracy", result.MetricName)
	assert.Equal(t, warn, result.WarningThreshold)
	assert.Equal(t, crit, result.CriticalThreshold)
	assert.Len(t, perfRepo.thresholds, 1, "new threshold should be persisted")
}

func TestPerformanceService_UpdateThreshold_UpdatesExisting(t *testing.T) {
	perfRepo := newFakePerfRepo()
	existing := &domain.PerformanceThreshold{
		ID: "th1", ModelID: "m1", MetricName: "accuracy",
		WarningThreshold: 10.0, CriticalThreshold: 20.0,
		Direction: domain.ThresholdDirectionLower, Enabled: true,
	}
	perfRepo.thresholds = []*domain.PerformanceThreshold{existing}
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	newWarn := 7.0
	result, err := svc.UpdateThreshold("m1", &dto.UpdateThresholdRequest{
		MetricName:       "accuracy",
		WarningThreshold: &newWarn,
	})
	require.NoError(t, err)
	assert.Equal(t, newWarn, result.WarningThreshold)
	assert.Equal(t, 20.0, result.CriticalThreshold, "unchanged field should stay")
}

func TestPerformanceService_UpdateThreshold_InvalidValues_Rejected(t *testing.T) {
	svc := buildPerfSvc(&mockModelRepo{}, newFakePerfRepo())
	// warning > critical is invalid
	warn, crit := 30.0, 10.0
	_, err := svc.UpdateThreshold("m1", &dto.UpdateThresholdRequest{
		MetricName:        "accuracy",
		WarningThreshold:  &warn,
		CriticalThreshold: &crit,
	})
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetThresholdDefaults — merges DB overrides with hardcoded canonical set
// ---------------------------------------------------------------------------

func TestPerformanceService_GetThresholdDefaults_ReturnsHardcodedWhenNoDB(t *testing.T) {
	perfRepo := newFakePerfRepo() // no DB defaults
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	result, err := svc.GetThresholdDefaults("classification")
	require.NoError(t, err)
	assert.Equal(t, "classification", result.TaskType)
	// Should contain at least one metric (accuracy for classification)
	assert.NotEmpty(t, result.Defaults)
}

func TestPerformanceService_GetThresholdDefaults_DBOverridesHardcoded(t *testing.T) {
	perfRepo := newFakePerfRepo()
	// DB has a custom warning threshold for accuracy
	perfRepo.defaults = []*domain.PerformanceThresholdDefault{
		{
			TaskType: "classification", MetricName: "accuracy",
			WarningThreshold: 3.0, CriticalThreshold: 8.0,
			Direction: domain.ThresholdDirectionLower, Enabled: true,
		},
	}
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	result, err := svc.GetThresholdDefaults("classification")
	require.NoError(t, err)
	// Find accuracy in response
	var accuracyDefault *dto.PerformanceThresholdDefaultResponse
	for i, d := range result.Defaults {
		if d.MetricName == "accuracy" {
			accuracyDefault = &result.Defaults[i]
			break
		}
	}
	require.NotNil(t, accuracyDefault, "accuracy should appear in classification defaults")
	assert.Equal(t, 3.0, accuracyDefault.WarningThreshold, "DB value should override hardcoded")
}

// ---------------------------------------------------------------------------
// UpsertThresholdDefault
// ---------------------------------------------------------------------------

func TestPerformanceService_UpsertThresholdDefault_SeededFromHardcoded(t *testing.T) {
	perfRepo := newFakePerfRepo() // no existing DB defaults
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	warn := 5.0
	result, err := svc.UpsertThresholdDefault(&dto.UpdateThresholdDefaultRequest{
		TaskType:         "classification",
		MetricName:       "accuracy",
		WarningThreshold: &warn,
	}, "admin")
	require.NoError(t, err)
	assert.Equal(t, "accuracy", result.MetricName)
	assert.Equal(t, warn, result.WarningThreshold)
	assert.Len(t, perfRepo.defaults, 1, "should persist the upserted default")
}

func TestPerformanceService_UpsertThresholdDefault_InvalidValues_Rejected(t *testing.T) {
	svc := buildPerfSvc(&mockModelRepo{}, newFakePerfRepo())
	warn, crit := 50.0, 10.0 // warning > critical
	_, err := svc.UpsertThresholdDefault(&dto.UpdateThresholdDefaultRequest{
		TaskType:          "classification",
		MetricName:        "accuracy",
		WarningThreshold:  &warn,
		CriticalThreshold: &crit,
	}, "admin")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetPerformanceSummary
// ---------------------------------------------------------------------------

func TestPerformanceService_GetPerformanceSummary_HealthyWhenNoAlerts(t *testing.T) {
	perfRepo := newFakePerfRepo()
	model := &domain.Model{ID: "m1", ModelType: "classification"}
	mr := &mockModelRepo{
		getByID: func(id string) (*domain.Model, error) { return model, nil },
	}
	svc := buildPerfSvc(mr, perfRepo)

	result, err := svc.GetPerformanceSummary("m1")
	require.NoError(t, err)
	assert.Equal(t, "healthy", result.OverallHealthStatus)
	assert.False(t, result.HasBaseline)
}

func TestPerformanceService_GetPerformanceSummary_HasBaseline_WhenBaselinesPresent(t *testing.T) {
	perfRepo := newFakePerfRepo()
	perfRepo.baselines = []*domain.PerformanceBaseline{
		{ModelID: "m1", MetricName: "accuracy", MetricValue: 0.91},
	}
	model := &domain.Model{ID: "m1", ModelType: "classification"}
	mr := &mockModelRepo{
		getByID: func(id string) (*domain.Model, error) { return model, nil },
	}
	svc := buildPerfSvc(mr, perfRepo)

	result, err := svc.GetPerformanceSummary("m1")
	require.NoError(t, err)
	assert.True(t, result.HasBaseline)
	assert.Equal(t, 0.91, result.BaselineMetrics["accuracy"])
}

func TestPerformanceService_GetPerformanceSummary_ModelNotFound(t *testing.T) {
	svc := buildPerfSvc(&mockModelRepo{}, newFakePerfRepo())
	_, err := svc.GetPerformanceSummary("missing")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// DeleteByModelID — cascades all performance data
// ---------------------------------------------------------------------------

func TestPerformanceService_DeleteByModelID_NeverErrors(t *testing.T) {
	// DeleteByModelID deliberately swallows errors (logs warnings) and always returns nil.
	// Verify it completes without error even with data present.
	perfRepo := newFakePerfRepo()
	perfRepo.alerts = []*domain.PerformanceAlert{{ID: "a1", ModelID: "m1"}}
	perfRepo.records = []*domain.PerformanceRecord{{ModelID: "m1", MetricName: "accuracy"}}
	svc := buildPerfSvc(&mockModelRepo{}, perfRepo)

	err := svc.DeleteByModelID("m1")
	require.NoError(t, err, "DeleteByModelID should always return nil even when data exists")
}
