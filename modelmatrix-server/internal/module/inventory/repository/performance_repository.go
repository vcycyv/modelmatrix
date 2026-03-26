package repository

import (
	"time"

	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/model"

	"gorm.io/gorm"
)

// PerformanceRepository defines the interface for performance data access
type PerformanceRepository interface {
	// Baseline operations
	CreateBaseline(baseline *domain.PerformanceBaseline) error
	UpdateBaseline(baseline *domain.PerformanceBaseline) error
	GetBaselineByModelAndMetric(modelID, metricName string) (*domain.PerformanceBaseline, error)
	GetBaselinesByModelID(modelID string) ([]domain.PerformanceBaseline, error)
	DeleteBaselinesByModelID(modelID string) error

	// Record operations
	CreateRecord(record *domain.PerformanceRecord) error
	CreateRecords(records []domain.PerformanceRecord) error
	GetRecordsByModelID(modelID string, limit int, startTime, endTime *time.Time) ([]domain.PerformanceRecord, error)
	GetRecordsByMetric(modelID, metricName string, limit int) ([]domain.PerformanceRecord, error)
	GetLatestRecordsByModelID(modelID string) ([]domain.PerformanceRecord, error)
	DeleteRecordsByModelID(modelID string) error

	// Alert operations
	CreateAlert(alert *domain.PerformanceAlert) error
	UpdateAlert(alert *domain.PerformanceAlert) error
	GetAlertByID(id string) (*domain.PerformanceAlert, error)
	GetActiveAlertsByModelID(modelID string) ([]domain.PerformanceAlert, error)
	GetAlertsByModelID(modelID string, status string, limit int) ([]domain.PerformanceAlert, error)
	CountActiveAlertsByModelID(modelID string) (int, int, int, error) // Returns total, warning, critical counts
	DeleteAlertsByModelID(modelID string) error

	// Threshold operations
	CreateThreshold(threshold *domain.PerformanceThreshold) error
	UpdateThreshold(threshold *domain.PerformanceThreshold) error
	GetThresholdByModelAndMetric(modelID, metricName string) (*domain.PerformanceThreshold, error)
	GetThresholdsByModelID(modelID string) ([]domain.PerformanceThreshold, error)
	DeleteThresholdsByModelID(modelID string) error

	// Global threshold default operations
	GetThresholdDefaultsByTaskType(taskType string) ([]domain.PerformanceThresholdDefault, error)
	UpsertThresholdDefault(d *domain.PerformanceThresholdDefault) error

	// Evaluation operations
	CreateEvaluation(evaluation *domain.PerformanceEvaluation) error
	UpdateEvaluation(evaluation *domain.PerformanceEvaluation) error
	GetEvaluationByID(id string) (*domain.PerformanceEvaluation, error)
	GetEvaluationsByModelID(modelID string, limit int) ([]domain.PerformanceEvaluation, error)
	GetPendingEvaluations() ([]domain.PerformanceEvaluation, error)
	DeleteEvaluationsByModelID(modelID string) error
}

// PerformanceRepositoryImpl implements PerformanceRepository
type PerformanceRepositoryImpl struct {
	db *gorm.DB
}

// NewPerformanceRepository creates a new performance repository
func NewPerformanceRepository(db *gorm.DB) PerformanceRepository {
	return &PerformanceRepositoryImpl{db: db}
}

// === Baseline Operations ===

func (r *PerformanceRepositoryImpl) CreateBaseline(baseline *domain.PerformanceBaseline) error {
	dbModel := r.baselineToModel(baseline)
	if err := r.db.Create(dbModel).Error; err != nil {
		return err
	}
	baseline.ID = dbModel.ID
	baseline.CreatedAt = dbModel.CreatedAt
	baseline.UpdatedAt = dbModel.UpdatedAt
	return nil
}

func (r *PerformanceRepositoryImpl) UpdateBaseline(baseline *domain.PerformanceBaseline) error {
	dbModel := r.baselineToModel(baseline)
	return r.db.Save(dbModel).Error
}

func (r *PerformanceRepositoryImpl) GetBaselineByModelAndMetric(modelID, metricName string) (*domain.PerformanceBaseline, error) {
	var dbModel model.PerformanceBaseline
	if err := r.db.Where("model_id = ? AND metric_name = ?", modelID, metricName).First(&dbModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrBaselineNotFound
		}
		return nil, err
	}
	return r.baselineToDomain(&dbModel), nil
}

func (r *PerformanceRepositoryImpl) GetBaselinesByModelID(modelID string) ([]domain.PerformanceBaseline, error) {
	var dbModels []model.PerformanceBaseline
	if err := r.db.Where("model_id = ?", modelID).Find(&dbModels).Error; err != nil {
		return nil, err
	}
	result := make([]domain.PerformanceBaseline, len(dbModels))
	for i, m := range dbModels {
		result[i] = *r.baselineToDomain(&m)
	}
	return result, nil
}

func (r *PerformanceRepositoryImpl) DeleteBaselinesByModelID(modelID string) error {
	return r.db.Delete(&model.PerformanceBaseline{}, "model_id = ?", modelID).Error
}

// === Record Operations ===

func (r *PerformanceRepositoryImpl) CreateRecord(record *domain.PerformanceRecord) error {
	dbModel := r.recordToModel(record)
	if err := r.db.Create(dbModel).Error; err != nil {
		return err
	}
	record.ID = dbModel.ID
	record.CreatedAt = dbModel.CreatedAt
	return nil
}

func (r *PerformanceRepositoryImpl) CreateRecords(records []domain.PerformanceRecord) error {
	if len(records) == 0 {
		return nil
	}
	dbModels := make([]model.PerformanceRecord, len(records))
	for i, rec := range records {
		dbModels[i] = *r.recordToModel(&rec)
	}
	if err := r.db.Create(&dbModels).Error; err != nil {
		return err
	}
	for i := range dbModels {
		records[i].ID = dbModels[i].ID
	}
	return nil
}

func (r *PerformanceRepositoryImpl) GetRecordsByModelID(modelID string, limit int, startTime, endTime *time.Time) ([]domain.PerformanceRecord, error) {
	var dbModels []model.PerformanceRecord
	query := r.db.Where("model_id = ?", modelID)

	if startTime != nil {
		query = query.Where("window_start >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("window_end <= ?", *endTime)
	}

	query = query.Order("window_start DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&dbModels).Error; err != nil {
		return nil, err
	}

	result := make([]domain.PerformanceRecord, len(dbModels))
	for i, m := range dbModels {
		result[i] = *r.recordToDomain(&m)
	}
	return result, nil
}

func (r *PerformanceRepositoryImpl) GetRecordsByMetric(modelID, metricName string, limit int) ([]domain.PerformanceRecord, error) {
	var dbModels []model.PerformanceRecord
	query := r.db.Where("model_id = ? AND metric_name = ?", modelID, metricName).Order("window_start DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&dbModels).Error; err != nil {
		return nil, err
	}
	result := make([]domain.PerformanceRecord, len(dbModels))
	for i, m := range dbModels {
		result[i] = *r.recordToDomain(&m)
	}
	return result, nil
}

func (r *PerformanceRepositoryImpl) GetLatestRecordsByModelID(modelID string) ([]domain.PerformanceRecord, error) {
	// Get the latest record for each metric
	var dbModels []model.PerformanceRecord
	subQuery := r.db.Model(&model.PerformanceRecord{}).
		Select("MAX(created_at)").
		Where("model_id = ?", modelID).
		Group("metric_name")

	if err := r.db.Where("model_id = ? AND created_at IN (?)", modelID, subQuery).Find(&dbModels).Error; err != nil {
		return nil, err
	}

	result := make([]domain.PerformanceRecord, len(dbModels))
	for i, m := range dbModels {
		result[i] = *r.recordToDomain(&m)
	}
	return result, nil
}

func (r *PerformanceRepositoryImpl) DeleteRecordsByModelID(modelID string) error {
	return r.db.Delete(&model.PerformanceRecord{}, "model_id = ?", modelID).Error
}

// === Alert Operations ===

func (r *PerformanceRepositoryImpl) CreateAlert(alert *domain.PerformanceAlert) error {
	dbModel := r.alertToModel(alert)
	if err := r.db.Create(dbModel).Error; err != nil {
		return err
	}
	alert.ID = dbModel.ID
	alert.CreatedAt = dbModel.CreatedAt
	alert.UpdatedAt = dbModel.UpdatedAt
	return nil
}

func (r *PerformanceRepositoryImpl) UpdateAlert(alert *domain.PerformanceAlert) error {
	dbModel := r.alertToModel(alert)
	return r.db.Save(dbModel).Error
}

func (r *PerformanceRepositoryImpl) GetAlertByID(id string) (*domain.PerformanceAlert, error) {
	var dbModel model.PerformanceAlert
	if err := r.db.Where("id = ?", id).First(&dbModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrAlertNotFound
		}
		return nil, err
	}
	return r.alertToDomain(&dbModel), nil
}

func (r *PerformanceRepositoryImpl) GetActiveAlertsByModelID(modelID string) ([]domain.PerformanceAlert, error) {
	var dbModels []model.PerformanceAlert
	if err := r.db.Where("model_id = ? AND status = ?", modelID, domain.AlertStatusActive).
		Order("created_at DESC").Find(&dbModels).Error; err != nil {
		return nil, err
	}
	result := make([]domain.PerformanceAlert, len(dbModels))
	for i, m := range dbModels {
		result[i] = *r.alertToDomain(&m)
	}
	return result, nil
}

func (r *PerformanceRepositoryImpl) GetAlertsByModelID(modelID string, status string, limit int) ([]domain.PerformanceAlert, error) {
	var dbModels []model.PerformanceAlert
	query := r.db.Where("model_id = ?", modelID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query = query.Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&dbModels).Error; err != nil {
		return nil, err
	}
	result := make([]domain.PerformanceAlert, len(dbModels))
	for i, m := range dbModels {
		result[i] = *r.alertToDomain(&m)
	}
	return result, nil
}

func (r *PerformanceRepositoryImpl) CountActiveAlertsByModelID(modelID string) (int, int, int, error) {
	type result struct {
		Severity string
		Count    int
	}
	var results []result

	if err := r.db.Model(&model.PerformanceAlert{}).
		Select("severity, COUNT(*) as count").
		Where("model_id = ? AND status = ?", modelID, domain.AlertStatusActive).
		Group("severity").
		Scan(&results).Error; err != nil {
		return 0, 0, 0, err
	}

	var total, warning, critical int
	for _, res := range results {
		total += res.Count
		switch domain.AlertSeverity(res.Severity) {
		case domain.AlertSeverityWarning:
			warning = res.Count
		case domain.AlertSeverityCritical:
			critical = res.Count
		}
	}
	return total, warning, critical, nil
}

func (r *PerformanceRepositoryImpl) DeleteAlertsByModelID(modelID string) error {
	return r.db.Delete(&model.PerformanceAlert{}, "model_id = ?", modelID).Error
}

// === Threshold Operations ===

func (r *PerformanceRepositoryImpl) CreateThreshold(threshold *domain.PerformanceThreshold) error {
	dbModel := r.thresholdToModel(threshold)
	if err := r.db.Create(dbModel).Error; err != nil {
		return err
	}
	threshold.ID = dbModel.ID
	threshold.CreatedAt = dbModel.CreatedAt
	threshold.UpdatedAt = dbModel.UpdatedAt
	return nil
}

func (r *PerformanceRepositoryImpl) UpdateThreshold(threshold *domain.PerformanceThreshold) error {
	dbModel := r.thresholdToModel(threshold)
	return r.db.Save(dbModel).Error
}

func (r *PerformanceRepositoryImpl) GetThresholdByModelAndMetric(modelID, metricName string) (*domain.PerformanceThreshold, error) {
	var dbModel model.PerformanceThreshold
	if err := r.db.Where("model_id = ? AND metric_name = ?", modelID, metricName).First(&dbModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrThresholdNotFound
		}
		return nil, err
	}
	return r.thresholdToDomain(&dbModel), nil
}

func (r *PerformanceRepositoryImpl) GetThresholdsByModelID(modelID string) ([]domain.PerformanceThreshold, error) {
	var dbModels []model.PerformanceThreshold
	if err := r.db.Where("model_id = ?", modelID).Find(&dbModels).Error; err != nil {
		return nil, err
	}
	result := make([]domain.PerformanceThreshold, len(dbModels))
	for i, m := range dbModels {
		result[i] = *r.thresholdToDomain(&m)
	}
	return result, nil
}

func (r *PerformanceRepositoryImpl) DeleteThresholdsByModelID(modelID string) error {
	return r.db.Delete(&model.PerformanceThreshold{}, "model_id = ?", modelID).Error
}

// === Evaluation Operations ===

func (r *PerformanceRepositoryImpl) CreateEvaluation(evaluation *domain.PerformanceEvaluation) error {
	dbModel := r.evaluationToModel(evaluation)
	if err := r.db.Create(dbModel).Error; err != nil {
		return err
	}
	evaluation.ID = dbModel.ID
	evaluation.CreatedAt = dbModel.CreatedAt
	evaluation.UpdatedAt = dbModel.UpdatedAt
	return nil
}

func (r *PerformanceRepositoryImpl) UpdateEvaluation(evaluation *domain.PerformanceEvaluation) error {
	dbModel := r.evaluationToModel(evaluation)
	return r.db.Save(dbModel).Error
}

func (r *PerformanceRepositoryImpl) GetEvaluationByID(id string) (*domain.PerformanceEvaluation, error) {
	var dbModel model.PerformanceEvaluation
	if err := r.db.Where("id = ?", id).First(&dbModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrEvaluationNotFound
		}
		return nil, err
	}
	return r.evaluationToDomain(&dbModel), nil
}

func (r *PerformanceRepositoryImpl) GetEvaluationsByModelID(modelID string, limit int) ([]domain.PerformanceEvaluation, error) {
	var dbModels []model.PerformanceEvaluation
	query := r.db.Where("model_id = ?", modelID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&dbModels).Error; err != nil {
		return nil, err
	}
	result := make([]domain.PerformanceEvaluation, len(dbModels))
	for i, m := range dbModels {
		result[i] = *r.evaluationToDomain(&m)
	}
	return result, nil
}

func (r *PerformanceRepositoryImpl) GetPendingEvaluations() ([]domain.PerformanceEvaluation, error) {
	var dbModels []model.PerformanceEvaluation
	if err := r.db.Where("status IN ?", []string{string(domain.EvaluationStatusPending), string(domain.EvaluationStatusRunning)}).
		Find(&dbModels).Error; err != nil {
		return nil, err
	}
	result := make([]domain.PerformanceEvaluation, len(dbModels))
	for i, m := range dbModels {
		result[i] = *r.evaluationToDomain(&m)
	}
	return result, nil
}

func (r *PerformanceRepositoryImpl) DeleteEvaluationsByModelID(modelID string) error {
	return r.db.Delete(&model.PerformanceEvaluation{}, "model_id = ?", modelID).Error
}

// === Conversion Functions ===

func (r *PerformanceRepositoryImpl) baselineToModel(b *domain.PerformanceBaseline) *model.PerformanceBaseline {
	return &model.PerformanceBaseline{
		ID:          b.ID,
		ModelID:     b.ModelID,
		TaskType:    string(b.TaskType),
		MetricName:  b.MetricName,
		MetricValue: b.MetricValue,
		SampleCount: b.SampleCount,
		Description: b.Description,
		CreatedBy:   b.CreatedBy,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
	}
}

func (r *PerformanceRepositoryImpl) baselineToDomain(m *model.PerformanceBaseline) *domain.PerformanceBaseline {
	return &domain.PerformanceBaseline{
		ID:          m.ID,
		ModelID:     m.ModelID,
		TaskType:    domain.TaskType(m.TaskType),
		MetricName:  m.MetricName,
		MetricValue: m.MetricValue,
		SampleCount: m.SampleCount,
		Description: m.Description,
		CreatedBy:   m.CreatedBy,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func (r *PerformanceRepositoryImpl) recordToModel(rec *domain.PerformanceRecord) *model.PerformanceRecord {
	return &model.PerformanceRecord{
		ID:              rec.ID,
		ModelID:         rec.ModelID,
		DatasourceID:    rec.DatasourceID,
		MetricName:      rec.MetricName,
		MetricValue:     rec.MetricValue,
		BaselineValue:   rec.BaselineValue,
		DriftPercentage: rec.DriftPercentage,
		SampleCount:     rec.SampleCount,
		WindowStart:     rec.WindowStart,
		WindowEnd:       rec.WindowEnd,
		CreatedBy:       rec.CreatedBy,
		CreatedAt:       rec.CreatedAt,
	}
}

func (r *PerformanceRepositoryImpl) recordToDomain(m *model.PerformanceRecord) *domain.PerformanceRecord {
	return &domain.PerformanceRecord{
		ID:              m.ID,
		ModelID:         m.ModelID,
		DatasourceID:    m.DatasourceID,
		MetricName:      m.MetricName,
		MetricValue:     m.MetricValue,
		BaselineValue:   m.BaselineValue,
		DriftPercentage: m.DriftPercentage,
		SampleCount:     m.SampleCount,
		WindowStart:     m.WindowStart,
		WindowEnd:       m.WindowEnd,
		CreatedBy:       m.CreatedBy,
		CreatedAt:       m.CreatedAt,
	}
}

func (r *PerformanceRepositoryImpl) alertToModel(a *domain.PerformanceAlert) *model.PerformanceAlert {
	return &model.PerformanceAlert{
		ID:                  a.ID,
		ModelID:             a.ModelID,
		RecordID:            a.RecordID,
		AlertType:           string(a.AlertType),
		Severity:            string(a.Severity),
		MetricName:          a.MetricName,
		BaselineValue:       a.BaselineValue,
		CurrentValue:        a.CurrentValue,
		ThresholdPercentage: a.ThresholdPercentage,
		DriftPercentage:     a.DriftPercentage,
		Message:             a.Message,
		Status:              string(a.Status),
		AcknowledgedBy:      a.AcknowledgedBy,
		AcknowledgedAt:      a.AcknowledgedAt,
		ResolvedAt:          a.ResolvedAt,
		CreatedAt:           a.CreatedAt,
		UpdatedAt:           a.UpdatedAt,
	}
}

func (r *PerformanceRepositoryImpl) alertToDomain(m *model.PerformanceAlert) *domain.PerformanceAlert {
	return &domain.PerformanceAlert{
		ID:                  m.ID,
		ModelID:             m.ModelID,
		RecordID:            m.RecordID,
		AlertType:           domain.AlertType(m.AlertType),
		Severity:            domain.AlertSeverity(m.Severity),
		MetricName:          m.MetricName,
		BaselineValue:       m.BaselineValue,
		CurrentValue:        m.CurrentValue,
		ThresholdPercentage: m.ThresholdPercentage,
		DriftPercentage:     m.DriftPercentage,
		Message:             m.Message,
		Status:              domain.AlertStatus(m.Status),
		AcknowledgedBy:      m.AcknowledgedBy,
		AcknowledgedAt:      m.AcknowledgedAt,
		ResolvedAt:          m.ResolvedAt,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
}

func (r *PerformanceRepositoryImpl) thresholdToModel(t *domain.PerformanceThreshold) *model.PerformanceThreshold {
	return &model.PerformanceThreshold{
		ID:                  t.ID,
		ModelID:             t.ModelID,
		MetricName:          t.MetricName,
		WarningThreshold:    t.WarningThreshold,
		CriticalThreshold:   t.CriticalThreshold,
		Direction:           string(t.Direction),
		Enabled:             t.Enabled,
		ConsecutiveBreaches: t.ConsecutiveBreaches,
		CreatedAt:           t.CreatedAt,
		UpdatedAt:           t.UpdatedAt,
	}
}

func (r *PerformanceRepositoryImpl) thresholdToDomain(m *model.PerformanceThreshold) *domain.PerformanceThreshold {
	return &domain.PerformanceThreshold{
		ID:                  m.ID,
		ModelID:             m.ModelID,
		MetricName:          m.MetricName,
		WarningThreshold:    m.WarningThreshold,
		CriticalThreshold:   m.CriticalThreshold,
		Direction:           domain.ThresholdDirection(m.Direction),
		Enabled:             m.Enabled,
		ConsecutiveBreaches: m.ConsecutiveBreaches,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
}

func (r *PerformanceRepositoryImpl) evaluationToModel(e *domain.PerformanceEvaluation) *model.PerformanceEvaluation {
	dbModel := &model.PerformanceEvaluation{
		ID:           e.ID,
		ModelID:      e.ModelID,
		DatasourceID: e.DatasourceID,
		Status:       string(e.Status),
		TaskType:     string(e.TaskType),
		SampleCount:  e.SampleCount,
		ErrorMessage: e.ErrorMessage,
		StartedAt:    e.StartedAt,
		CompletedAt:  e.CompletedAt,
		CreatedBy:    e.CreatedBy,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
	if e.Metrics != nil {
		dbModel.Metrics = model.JSONMap(e.Metrics)
	}
	return dbModel
}

func (r *PerformanceRepositoryImpl) evaluationToDomain(m *model.PerformanceEvaluation) *domain.PerformanceEvaluation {
	eval := &domain.PerformanceEvaluation{
		ID:           m.ID,
		ModelID:      m.ModelID,
		DatasourceID: m.DatasourceID,
		Status:       domain.EvaluationStatus(m.Status),
		TaskType:     domain.TaskType(m.TaskType),
		SampleCount:  m.SampleCount,
		ErrorMessage: m.ErrorMessage,
		StartedAt:    m.StartedAt,
		CompletedAt:  m.CompletedAt,
		CreatedBy:    m.CreatedBy,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
	if m.Metrics != nil {
		eval.Metrics = map[string]interface{}(m.Metrics)
	}
	return eval
}

// === Global threshold defaults ===

func (r *PerformanceRepositoryImpl) GetThresholdDefaultsByTaskType(taskType string) ([]domain.PerformanceThresholdDefault, error) {
	var rows []model.PerformanceThresholdDefault
	if err := r.db.Where("task_type = ?", taskType).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.PerformanceThresholdDefault, len(rows))
	for i, m := range rows {
		result[i] = r.thresholdDefaultToDomain(&m)
	}
	return result, nil
}

func (r *PerformanceRepositoryImpl) UpsertThresholdDefault(d *domain.PerformanceThresholdDefault) error {
	existing := &model.PerformanceThresholdDefault{}
	err := r.db.Where("task_type = ? AND metric_name = ?", string(d.TaskType), d.MetricName).First(existing).Error
	if err == gorm.ErrRecordNotFound {
		dbModel := r.thresholdDefaultToModel(d)
		return r.db.Create(dbModel).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(existing).Updates(map[string]interface{}{
		"warning_threshold":    d.WarningThreshold,
		"critical_threshold":   d.CriticalThreshold,
		"direction":            string(d.Direction),
		"enabled":              d.Enabled,
		"consecutive_breaches": d.ConsecutiveBreaches,
		"updated_by":           d.UpdatedBy,
	}).Error
}

func (r *PerformanceRepositoryImpl) thresholdDefaultToModel(d *domain.PerformanceThresholdDefault) *model.PerformanceThresholdDefault {
	return &model.PerformanceThresholdDefault{
		ID:                  d.ID,
		TaskType:            string(d.TaskType),
		MetricName:          d.MetricName,
		WarningThreshold:    d.WarningThreshold,
		CriticalThreshold:   d.CriticalThreshold,
		Direction:           string(d.Direction),
		Enabled:             d.Enabled,
		ConsecutiveBreaches: d.ConsecutiveBreaches,
		UpdatedBy:           d.UpdatedBy,
	}
}

func (r *PerformanceRepositoryImpl) thresholdDefaultToDomain(m *model.PerformanceThresholdDefault) domain.PerformanceThresholdDefault {
	return domain.PerformanceThresholdDefault{
		ID:                  m.ID,
		TaskType:            domain.TaskType(m.TaskType),
		MetricName:          m.MetricName,
		WarningThreshold:    m.WarningThreshold,
		CriticalThreshold:   m.CriticalThreshold,
		Direction:           domain.ThresholdDirection(m.Direction),
		Enabled:             m.Enabled,
		ConsecutiveBreaches: m.ConsecutiveBreaches,
		UpdatedBy:           m.UpdatedBy,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
}
