package application

import (
	"fmt"
	"time"

	"modelmatrix-server/internal/infrastructure/compute"
	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"
	"modelmatrix-server/internal/module/inventory/repository"
	"modelmatrix-server/pkg/config"
	"modelmatrix-server/pkg/logger"
)

// PerformanceService defines the interface for performance monitoring
type PerformanceService interface {
	// Baseline operations
	CreateBaseline(modelID string, req *dto.CreateBaselineRequest, createdBy string) (*dto.BaselinesListResponse, error)
	GetBaselines(modelID string) (*dto.BaselinesListResponse, error)

	// Performance recording
	RecordPerformance(modelID string, req *dto.RecordPerformanceRequest, createdBy string) (*dto.PerformanceHistoryResponse, error)
	GetPerformanceHistory(modelID string, params *dto.GetPerformanceHistoryParams) (*dto.PerformanceHistoryResponse, error)
	GetMetricTimeSeries(modelID, metricName string, limit int) (*dto.MetricTimeSeriesResponse, error)

	// Evaluation
	StartEvaluation(modelID string, req *dto.EvaluatePerformanceRequest, createdBy string) (*dto.PerformanceEvaluationResponse, error)
	HandleEvaluationCallback(req *dto.EvaluationCallbackRequest) error
	GetEvaluations(modelID string, limit int) (*dto.EvaluationsListResponse, error)
	GetEvaluation(evaluationID string) (*dto.PerformanceEvaluationResponse, error)

	// Alert operations
	GetAlerts(modelID string, params *dto.GetAlertsParams) (*dto.AlertsListResponse, error)
	UpdateAlert(alertID string, req *dto.UpdateAlertRequest, username string) (*dto.PerformanceAlertResponse, error)

	// Threshold operations
	GetThresholds(modelID string) (*dto.ThresholdsListResponse, error)
	UpdateThreshold(modelID string, req *dto.UpdateThresholdRequest) (*dto.PerformanceThresholdResponse, error)
	InitializeDefaultThresholds(modelID string, taskType domain.TaskType) error

	// Summary
	GetPerformanceSummary(modelID string) (*dto.PerformanceSummaryResponse, error)

	// Configuration
	ConfigureCompute(computeClient compute.Client, datasourceGetter DatasourceGetter, cfg *config.Config)

	// Cleanup
	DeleteByModelID(modelID string) error
}

// PerformanceServiceImpl implements PerformanceService
type PerformanceServiceImpl struct {
	performanceRepo  repository.PerformanceRepository
	modelRepo        repository.ModelRepository
	computeClient    compute.Client
	datasourceGetter DatasourceGetter
	config           *config.Config
}

// NewPerformanceService creates a new performance service
func NewPerformanceService(
	performanceRepo repository.PerformanceRepository,
	modelRepo repository.ModelRepository,
) PerformanceService {
	return &PerformanceServiceImpl{
		performanceRepo: performanceRepo,
		modelRepo:       modelRepo,
	}
}

// ConfigureCompute adds compute capabilities to the service
func (s *PerformanceServiceImpl) ConfigureCompute(
	computeClient compute.Client,
	datasourceGetter DatasourceGetter,
	cfg *config.Config,
) {
	s.computeClient = computeClient
	s.datasourceGetter = datasourceGetter
	s.config = cfg
}

// CreateBaseline creates or updates baseline metrics for a model
func (s *PerformanceServiceImpl) CreateBaseline(modelID string, req *dto.CreateBaselineRequest, createdBy string) (*dto.BaselinesListResponse, error) {
	// Get model to verify it exists and get task type
	model, err := s.modelRepo.GetByID(modelID)
	if err != nil {
		return nil, err
	}

	taskType := domain.TaskType(model.ModelType)
	if !taskType.IsValid() {
		return nil, domain.ErrInvalidTaskType
	}

	// Create or update baselines for each metric
	for metricName, metricValue := range req.Metrics {
		existing, err := s.performanceRepo.GetBaselineByModelAndMetric(modelID, metricName)
		if err == domain.ErrBaselineNotFound {
			// Create new baseline
			baseline := &domain.PerformanceBaseline{
				ModelID:     modelID,
				TaskType:    taskType,
				MetricName:  metricName,
				MetricValue: metricValue,
				SampleCount: req.SampleCount,
				Description: req.Description,
				CreatedBy:   createdBy,
			}
			if err := s.performanceRepo.CreateBaseline(baseline); err != nil {
				return nil, fmt.Errorf("failed to create baseline for %s: %w", metricName, err)
			}
		} else if err != nil {
			return nil, err
		} else {
			// Update existing baseline
			existing.MetricValue = metricValue
			existing.SampleCount = req.SampleCount
			if req.Description != "" {
				existing.Description = req.Description
			}
			if err := s.performanceRepo.UpdateBaseline(existing); err != nil {
				return nil, fmt.Errorf("failed to update baseline for %s: %w", metricName, err)
			}
		}
	}

	// Initialize default thresholds if not already set
	if err := s.InitializeDefaultThresholds(modelID, taskType); err != nil {
		logger.Warn("Failed to initialize default thresholds: %v", err)
	}

	logger.Audit(createdBy, "create_baseline", "model_performance", modelID, "success", nil)

	return s.GetBaselines(modelID)
}

// GetBaselines retrieves all baselines for a model
func (s *PerformanceServiceImpl) GetBaselines(modelID string) (*dto.BaselinesListResponse, error) {
	baselines, err := s.performanceRepo.GetBaselinesByModelID(modelID)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.PerformanceBaselineResponse, len(baselines))
	for i, b := range baselines {
		responses[i] = toBaselineResponse(&b)
	}

	return &dto.BaselinesListResponse{Baselines: responses}, nil
}

// RecordPerformance manually records performance metrics
func (s *PerformanceServiceImpl) RecordPerformance(modelID string, req *dto.RecordPerformanceRequest, createdBy string) (*dto.PerformanceHistoryResponse, error) {
	// Get baselines for drift calculation
	baselines, err := s.performanceRepo.GetBaselinesByModelID(modelID)
	if err != nil {
		return nil, err
	}

	baselineMap := make(map[string]*domain.PerformanceBaseline)
	for i := range baselines {
		baselineMap[baselines[i].MetricName] = &baselines[i]
	}

	// Get thresholds (ensure defaults exist if model has baselines but no thresholds)
	thresholds, err := s.performanceRepo.GetThresholdsByModelID(modelID)
	if err != nil {
		logger.Warn("Failed to get thresholds: %v", err)
	}
	if len(thresholds) == 0 && len(baselines) > 0 {
		model, modelErr := s.modelRepo.GetByID(modelID)
		if modelErr == nil {
			taskType := domain.TaskType(model.ModelType)
			if taskType.IsValid() {
				if initErr := s.InitializeDefaultThresholds(modelID, taskType); initErr != nil {
					logger.Warn("Failed to initialize default thresholds when recording: %v", initErr)
				} else {
					thresholds, err = s.performanceRepo.GetThresholdsByModelID(modelID)
					if err != nil {
						logger.Warn("Failed to get thresholds after init: %v", err)
					}
				}
			}
		}
	}
	thresholdMap := make(map[string]*domain.PerformanceThreshold)
	for i := range thresholds {
		thresholdMap[thresholds[i].MetricName] = &thresholds[i]
	}

	now := time.Now()
	windowStart := now.Add(-24 * time.Hour)
	windowEnd := now
	if req.WindowStart != nil {
		windowStart = *req.WindowStart
	}
	if req.WindowEnd != nil {
		windowEnd = *req.WindowEnd
	}

	// Create records for each metric
	records := make([]domain.PerformanceRecord, 0, len(req.Metrics))
	for metricName, metricValue := range req.Metrics {
		record := domain.PerformanceRecord{
			ModelID:      modelID,
			DatasourceID: req.DatasourceID,
			MetricName:   metricName,
			MetricValue:  metricValue,
			SampleCount:  req.SampleCount,
			WindowStart:  windowStart,
			WindowEnd:    windowEnd,
			CreatedBy:    createdBy,
		}

		// Calculate drift if baseline exists
		if baseline, ok := baselineMap[metricName]; ok {
			record.BaselineValue = &baseline.MetricValue
			direction := domain.ThresholdDirectionLower
			if threshold, ok := thresholdMap[metricName]; ok {
				direction = threshold.Direction
			}
			drift := record.CalculateDrift(baseline.MetricValue, direction)
			record.DriftPercentage = &drift
		}

		records = append(records, record)
	}

	if err := s.performanceRepo.CreateRecords(records); err != nil {
		return nil, fmt.Errorf("failed to create performance records: %w", err)
	}

	// Check thresholds and create alerts
	logger.Info("RecordPerformance: model=%s baselines=%d thresholds=%d records=%d (with drift: %d)",
		modelID, len(baselineMap), len(thresholdMap), len(records), countRecordsWithDrift(records))
	for i := range records {
		if threshold, ok := thresholdMap[records[i].MetricName]; ok {
			s.checkAndCreateAlert(modelID, &records[i], threshold)
		} else {
			logger.Debug("RecordPerformance: no threshold for metric %s, skipping alert check", records[i].MetricName)
		}
	}

	logger.Audit(createdBy, "record_performance", "model_performance", modelID, "success", nil)

	return s.GetPerformanceHistory(modelID, &dto.GetPerformanceHistoryParams{Limit: 50})
}

func countRecordsWithDrift(records []domain.PerformanceRecord) int {
	n := 0
	for i := range records {
		if records[i].DriftPercentage != nil {
			n++
		}
	}
	return n
}

// checkAndCreateAlert checks threshold and creates alert if breached
func (s *PerformanceServiceImpl) checkAndCreateAlert(modelID string, record *domain.PerformanceRecord, threshold *domain.PerformanceThreshold) {
	if record.DriftPercentage == nil || record.BaselineValue == nil {
		logger.Debug("checkAndCreateAlert: skip metric %s (no drift or baseline)", record.MetricName)
		return
	}

	breached, severity := threshold.CheckBreach(*record.DriftPercentage)
	if !breached {
		logger.Info("No alert: metric %s drift %.2f%% below threshold (warning=%.1f%%, critical=%.1f%%)",
			record.MetricName, *record.DriftPercentage, threshold.WarningThreshold, threshold.CriticalThreshold)
		return
	}

	// Create alert (RecordID links to the performance record; use nil if ID not set to avoid invalid UUID)
	var recordID *string
	if record.ID != "" {
		recordID = &record.ID
	}
	alert := &domain.PerformanceAlert{
		ModelID:             modelID,
		RecordID:            recordID,
		AlertType:           domain.AlertTypePerformanceDrift,
		Severity:            severity,
		MetricName:          record.MetricName,
		BaselineValue:       *record.BaselineValue,
		CurrentValue:        record.MetricValue,
		ThresholdPercentage: threshold.WarningThreshold,
		DriftPercentage:     *record.DriftPercentage,
		Message: fmt.Sprintf("%s drifted %.2f%% from baseline (threshold: %.1f%%)",
			record.MetricName, *record.DriftPercentage, threshold.WarningThreshold),
		Status: domain.AlertStatusActive,
	}

	if severity == domain.AlertSeverityCritical {
		alert.ThresholdPercentage = threshold.CriticalThreshold
	}

	if err := s.performanceRepo.CreateAlert(alert); err != nil {
		logger.Error("Failed to create performance alert: %v", err)
	} else {
		logger.Info("Created %s alert for model %s: %s", severity, modelID, alert.Message)
	}
}

// GetPerformanceHistory retrieves historical performance records
func (s *PerformanceServiceImpl) GetPerformanceHistory(modelID string, params *dto.GetPerformanceHistoryParams) (*dto.PerformanceHistoryResponse, error) {
	var records []domain.PerformanceRecord
	var err error

	limit := 100
	if params.Limit > 0 {
		limit = params.Limit
	}

	if params.MetricName != "" {
		records, err = s.performanceRepo.GetRecordsByMetric(modelID, params.MetricName, limit)
	} else {
		records, err = s.performanceRepo.GetRecordsByModelID(modelID, limit, params.StartTime, params.EndTime)
	}

	if err != nil {
		return nil, err
	}

	responses := make([]dto.PerformanceRecordResponse, len(records))
	for i, r := range records {
		responses[i] = toRecordResponse(&r)
	}

	return &dto.PerformanceHistoryResponse{
		ModelID:    modelID,
		Records:    responses,
		TotalCount: len(responses),
	}, nil
}

// GetMetricTimeSeries retrieves time-series data for a specific metric
func (s *PerformanceServiceImpl) GetMetricTimeSeries(modelID, metricName string, limit int) (*dto.MetricTimeSeriesResponse, error) {
	if limit <= 0 {
		limit = 100
	}

	records, err := s.performanceRepo.GetRecordsByMetric(modelID, metricName, limit)
	if err != nil {
		return nil, err
	}

	// Get baseline
	var baseline *float64
	baselineRecord, err := s.performanceRepo.GetBaselineByModelAndMetric(modelID, metricName)
	if err == nil {
		baseline = &baselineRecord.MetricValue
	}

	dataPoints := make([]dto.MetricDataPointResponse, len(records))
	for i, r := range records {
		dataPoints[i] = dto.MetricDataPointResponse{
			Timestamp:       r.WindowEnd,
			Value:           r.MetricValue,
			DriftPercentage: r.DriftPercentage,
			SampleCount:     r.SampleCount,
		}
	}

	return &dto.MetricTimeSeriesResponse{
		MetricName: metricName,
		Baseline:   baseline,
		DataPoints: dataPoints,
	}, nil
}

// StartEvaluation starts a performance evaluation job
func (s *PerformanceServiceImpl) StartEvaluation(modelID string, req *dto.EvaluatePerformanceRequest, createdBy string) (*dto.PerformanceEvaluationResponse, error) {
	// Get model details
	model, err := s.modelRepo.GetByIDWithRelations(modelID)
	if err != nil {
		return nil, err
	}

	taskType := domain.TaskType(model.ModelType)
	if !taskType.IsValid() {
		return nil, domain.ErrInvalidTaskType
	}

	// Create evaluation record
	evaluation := &domain.PerformanceEvaluation{
		ModelID:      modelID,
		DatasourceID: req.DatasourceID,
		Status:       domain.EvaluationStatusPending,
		TaskType:     taskType,
		CreatedBy:    createdBy,
	}

	if err := s.performanceRepo.CreateEvaluation(evaluation); err != nil {
		return nil, fmt.Errorf("failed to create evaluation: %w", err)
	}

	// If compute client is configured, trigger async evaluation
	if s.computeClient != nil && s.datasourceGetter != nil {
		go s.runEvaluation(evaluation, model, req)
	} else {
		logger.Warn("Compute client not configured, evaluation will remain pending")
	}

	return toEvaluationResponse(evaluation), nil
}

// runEvaluation runs the evaluation asynchronously
func (s *PerformanceServiceImpl) runEvaluation(evaluation *domain.PerformanceEvaluation, model *domain.Model, req *dto.EvaluatePerformanceRequest) {
	// Mark as running
	evaluation.Start()
	if err := s.performanceRepo.UpdateEvaluation(evaluation); err != nil {
		logger.Error("Failed to update evaluation status: %v", err)
		return
	}

	// Get model file path
	var modelFilePath string
	for _, f := range model.Files {
		if f.FileType == domain.FileTypeModel {
			modelFilePath = f.FilePath
			break
		}
	}

	if modelFilePath == "" {
		evaluation.Fail("Model file not found")
		s.performanceRepo.UpdateEvaluation(evaluation)
		return
	}

	// Get input columns
	var inputColumns []string
	for _, v := range model.Variables {
		if v.Role == domain.VariableRoleInput {
			inputColumns = append(inputColumns, v.Name)
		}
	}

	// Get datasource file path
	datasourcePath, err := s.datasourceGetter.GetFilePath(req.DatasourceID)
	if err != nil {
		evaluation.Fail(fmt.Sprintf("Failed to get datasource: %v", err))
		s.performanceRepo.UpdateEvaluation(evaluation)
		return
	}

	// Build callback URL
	callbackURL := ""
	if s.config != nil {
		callbackURL = fmt.Sprintf("%s/api/models/%s/performance/evaluations/%s/callback",
			s.config.Server.BaseURL, model.ID, evaluation.ID)
	}

	// Call compute service
	evalReq := &compute.EvaluateRequest{
		EvaluationID:     evaluation.ID,
		ModelID:          model.ID,
		ModelFilePath:    modelFilePath,
		DatasourceFilePath: datasourcePath,
		InputColumns:     inputColumns,
		TargetColumn:     model.TargetColumn,
		ActualColumn:     req.ActualColumn,
		PredictionColumn: req.PredictionColumn,
		ModelType:        model.ModelType,
		CallbackURL:      callbackURL,
	}

	_, err = s.computeClient.EvaluatePerformance(evalReq)
	if err != nil {
		evaluation.Fail(fmt.Sprintf("Failed to start evaluation: %v", err))
		s.performanceRepo.UpdateEvaluation(evaluation)
		logger.Error("Failed to start performance evaluation: %v", err)
	}
}

// HandleEvaluationCallback processes callback from compute service
func (s *PerformanceServiceImpl) HandleEvaluationCallback(req *dto.EvaluationCallbackRequest) error {
	evaluation, err := s.performanceRepo.GetEvaluationByID(req.EvaluationID)
	if err != nil {
		return err
	}

	if req.Status == "failed" {
		evaluation.Fail(req.Error)
		return s.performanceRepo.UpdateEvaluation(evaluation)
	}

	if req.Status == "completed" {
		evaluation.Complete(req.Metrics, req.SampleCount)
		if err := s.performanceRepo.UpdateEvaluation(evaluation); err != nil {
			return err
		}

		// Convert metrics and record performance (JSON numbers can be float64 or int)
		metrics := make(map[string]float64)
		for k, v := range req.Metrics {
			switch val := v.(type) {
			case float64:
				metrics[k] = val
			case int:
				metrics[k] = float64(val)
			case int64:
				metrics[k] = float64(val)
			}
		}

		if len(metrics) == 0 {
			logger.Warn("Evaluation callback: no numeric metrics (raw keys=%d), cannot record performance for model %s", len(req.Metrics), evaluation.ModelID)
		} else {
			recordReq := &dto.RecordPerformanceRequest{
				DatasourceID: evaluation.DatasourceID,
				Metrics:      metrics,
				SampleCount:  req.SampleCount,
			}
			_, err = s.RecordPerformance(evaluation.ModelID, recordReq, evaluation.CreatedBy)
			if err != nil {
				logger.Error("Failed to record performance from evaluation: %v", err)
			}
		}

		logger.Info("Performance evaluation completed for model %s: %d metrics recorded", evaluation.ModelID, len(metrics))
	}

	return nil
}

// GetEvaluations retrieves evaluations for a model
func (s *PerformanceServiceImpl) GetEvaluations(modelID string, limit int) (*dto.EvaluationsListResponse, error) {
	if limit <= 0 {
		limit = 20
	}

	evaluations, err := s.performanceRepo.GetEvaluationsByModelID(modelID, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.PerformanceEvaluationResponse, len(evaluations))
	for i, e := range evaluations {
		responses[i] = *toEvaluationResponse(&e)
	}

	return &dto.EvaluationsListResponse{
		Evaluations: responses,
		TotalCount:  len(responses),
	}, nil
}

// GetEvaluation retrieves a single evaluation
func (s *PerformanceServiceImpl) GetEvaluation(evaluationID string) (*dto.PerformanceEvaluationResponse, error) {
	evaluation, err := s.performanceRepo.GetEvaluationByID(evaluationID)
	if err != nil {
		return nil, err
	}
	return toEvaluationResponse(evaluation), nil
}

// GetAlerts retrieves alerts for a model
func (s *PerformanceServiceImpl) GetAlerts(modelID string, params *dto.GetAlertsParams) (*dto.AlertsListResponse, error) {
	limit := 50
	if params.Limit > 0 {
		limit = params.Limit
	}

	alerts, err := s.performanceRepo.GetAlertsByModelID(modelID, params.Status, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.PerformanceAlertResponse, len(alerts))
	for i, a := range alerts {
		responses[i] = toAlertResponse(&a)
	}

	return &dto.AlertsListResponse{
		Alerts:     responses,
		TotalCount: len(responses),
	}, nil
}

// UpdateAlert updates an alert status
func (s *PerformanceServiceImpl) UpdateAlert(alertID string, req *dto.UpdateAlertRequest, username string) (*dto.PerformanceAlertResponse, error) {
	alert, err := s.performanceRepo.GetAlertByID(alertID)
	if err != nil {
		return nil, err
	}

	switch req.Status {
	case "acknowledged":
		alert.Acknowledge(username)
	case "resolved":
		alert.Resolve()
	default:
		return nil, domain.ErrInvalidAlertStatus
	}

	if err := s.performanceRepo.UpdateAlert(alert); err != nil {
		return nil, err
	}

	return &dto.PerformanceAlertResponse{
		ID:                  alert.ID,
		ModelID:             alert.ModelID,
		RecordID:            alert.RecordID,
		AlertType:           string(alert.AlertType),
		Severity:            string(alert.Severity),
		MetricName:          alert.MetricName,
		BaselineValue:       alert.BaselineValue,
		CurrentValue:        alert.CurrentValue,
		ThresholdPercentage: alert.ThresholdPercentage,
		DriftPercentage:     alert.DriftPercentage,
		Message:             alert.Message,
		Status:              string(alert.Status),
		AcknowledgedBy:      alert.AcknowledgedBy,
		AcknowledgedAt:      alert.AcknowledgedAt,
		ResolvedAt:          alert.ResolvedAt,
		CreatedAt:           alert.CreatedAt,
		UpdatedAt:           alert.UpdatedAt,
	}, nil
}

// GetThresholds retrieves thresholds for a model
func (s *PerformanceServiceImpl) GetThresholds(modelID string) (*dto.ThresholdsListResponse, error) {
	thresholds, err := s.performanceRepo.GetThresholdsByModelID(modelID)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.PerformanceThresholdResponse, len(thresholds))
	for i, t := range thresholds {
		responses[i] = toThresholdResponse(&t)
	}

	return &dto.ThresholdsListResponse{Thresholds: responses}, nil
}

// UpdateThreshold updates or creates a threshold
func (s *PerformanceServiceImpl) UpdateThreshold(modelID string, req *dto.UpdateThresholdRequest) (*dto.PerformanceThresholdResponse, error) {
	threshold, err := s.performanceRepo.GetThresholdByModelAndMetric(modelID, req.MetricName)
	if err == domain.ErrThresholdNotFound {
		// Create new threshold
		threshold = &domain.PerformanceThreshold{
			ModelID:             modelID,
			MetricName:          req.MetricName,
			WarningThreshold:    10.0,
			CriticalThreshold:   20.0,
			Direction:           domain.ThresholdDirectionLower,
			Enabled:             true,
			ConsecutiveBreaches: 1,
		}
	} else if err != nil {
		return nil, err
	}

	// Apply updates
	if req.WarningThreshold != nil {
		threshold.WarningThreshold = *req.WarningThreshold
	}
	if req.CriticalThreshold != nil {
		threshold.CriticalThreshold = *req.CriticalThreshold
	}
	if req.Direction != nil {
		threshold.Direction = domain.ThresholdDirection(*req.Direction)
	}
	if req.Enabled != nil {
		threshold.Enabled = *req.Enabled
	}
	if req.ConsecutiveBreaches != nil {
		threshold.ConsecutiveBreaches = *req.ConsecutiveBreaches
	}

	if threshold.ID == "" {
		if err := s.performanceRepo.CreateThreshold(threshold); err != nil {
			return nil, err
		}
	} else {
		if err := s.performanceRepo.UpdateThreshold(threshold); err != nil {
			return nil, err
		}
	}

	return &dto.PerformanceThresholdResponse{
		ID:                  threshold.ID,
		ModelID:             threshold.ModelID,
		MetricName:          threshold.MetricName,
		WarningThreshold:    threshold.WarningThreshold,
		CriticalThreshold:   threshold.CriticalThreshold,
		Direction:           string(threshold.Direction),
		Enabled:             threshold.Enabled,
		ConsecutiveBreaches: threshold.ConsecutiveBreaches,
		CreatedAt:           threshold.CreatedAt,
		UpdatedAt:           threshold.UpdatedAt,
	}, nil
}

// InitializeDefaultThresholds creates default thresholds for a model
func (s *PerformanceServiceImpl) InitializeDefaultThresholds(modelID string, taskType domain.TaskType) error {
	defaults := domain.GetDefaultThresholds(taskType)

	for _, def := range defaults {
		// Check if already exists
		_, err := s.performanceRepo.GetThresholdByModelAndMetric(modelID, def.MetricName)
		if err == nil {
			continue // Already exists
		}
		if err != domain.ErrThresholdNotFound {
			return err
		}

		// Create threshold
		threshold := &domain.PerformanceThreshold{
			ModelID:             modelID,
			MetricName:          def.MetricName,
			WarningThreshold:    def.WarningThreshold,
			CriticalThreshold:   def.CriticalThreshold,
			Direction:           def.Direction,
			Enabled:             def.Enabled,
			ConsecutiveBreaches: def.ConsecutiveBreaches,
		}
		if err := s.performanceRepo.CreateThreshold(threshold); err != nil {
			return err
		}
	}

	return nil
}

// GetPerformanceSummary returns a summary of model performance status
func (s *PerformanceServiceImpl) GetPerformanceSummary(modelID string) (*dto.PerformanceSummaryResponse, error) {
	// Get model to get task type
	model, err := s.modelRepo.GetByID(modelID)
	if err != nil {
		return nil, err
	}

	summary := &dto.PerformanceSummaryResponse{
		ModelID:             modelID,
		TaskType:            model.ModelType,
		HasBaseline:         false,
		LatestMetrics:       make(map[string]float64),
		BaselineMetrics:     make(map[string]float64),
		DriftPercentages:    make(map[string]float64),
		OverallHealthStatus: "healthy",
	}

	// Get baselines
	baselines, err := s.performanceRepo.GetBaselinesByModelID(modelID)
	if err == nil && len(baselines) > 0 {
		summary.HasBaseline = true
		for _, b := range baselines {
			summary.BaselineMetrics[b.MetricName] = b.MetricValue
		}
	}

	// Get latest records
	latestRecords, err := s.performanceRepo.GetLatestRecordsByModelID(modelID)
	if err == nil {
		for _, r := range latestRecords {
			summary.LatestMetrics[r.MetricName] = r.MetricValue
			if r.DriftPercentage != nil {
				summary.DriftPercentages[r.MetricName] = *r.DriftPercentage
			}
		}
		summary.RecordCount = len(latestRecords)
	}

	// Get alert counts
	total, warning, critical, err := s.performanceRepo.CountActiveAlertsByModelID(modelID)
	if err == nil {
		summary.ActiveAlerts = total
		summary.WarningAlerts = warning
		summary.CriticalAlerts = critical

		// Determine health status
		if critical > 0 {
			summary.OverallHealthStatus = "critical"
		} else if warning > 0 {
			summary.OverallHealthStatus = "warning"
		}
	}

	// Get last evaluation time
	evaluations, err := s.performanceRepo.GetEvaluationsByModelID(modelID, 1)
	if err == nil && len(evaluations) > 0 {
		summary.LastEvaluationAt = &evaluations[0].CreatedAt
	}

	return summary, nil
}

// DeleteByModelID removes all performance data for a model
func (s *PerformanceServiceImpl) DeleteByModelID(modelID string) error {
	// Delete in order: alerts, records, evaluations, thresholds, baselines
	if err := s.performanceRepo.DeleteAlertsByModelID(modelID); err != nil {
		logger.Warn("Failed to delete alerts for model %s: %v", modelID, err)
	}
	if err := s.performanceRepo.DeleteRecordsByModelID(modelID); err != nil {
		logger.Warn("Failed to delete records for model %s: %v", modelID, err)
	}
	if err := s.performanceRepo.DeleteEvaluationsByModelID(modelID); err != nil {
		logger.Warn("Failed to delete evaluations for model %s: %v", modelID, err)
	}
	if err := s.performanceRepo.DeleteThresholdsByModelID(modelID); err != nil {
		logger.Warn("Failed to delete thresholds for model %s: %v", modelID, err)
	}
	if err := s.performanceRepo.DeleteBaselinesByModelID(modelID); err != nil {
		logger.Warn("Failed to delete baselines for model %s: %v", modelID, err)
	}

	return nil
}

// === Helper Functions ===

func toBaselineResponse(b *domain.PerformanceBaseline) dto.PerformanceBaselineResponse {
	return dto.PerformanceBaselineResponse{
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

func toRecordResponse(r *domain.PerformanceRecord) dto.PerformanceRecordResponse {
	return dto.PerformanceRecordResponse{
		ID:              r.ID,
		ModelID:         r.ModelID,
		DatasourceID:    r.DatasourceID,
		MetricName:      r.MetricName,
		MetricValue:     r.MetricValue,
		BaselineValue:   r.BaselineValue,
		DriftPercentage: r.DriftPercentage,
		SampleCount:     r.SampleCount,
		WindowStart:     r.WindowStart,
		WindowEnd:       r.WindowEnd,
		CreatedBy:       r.CreatedBy,
		CreatedAt:       r.CreatedAt,
	}
}

func toAlertResponse(a *domain.PerformanceAlert) dto.PerformanceAlertResponse {
	return dto.PerformanceAlertResponse{
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

func toThresholdResponse(t *domain.PerformanceThreshold) dto.PerformanceThresholdResponse {
	return dto.PerformanceThresholdResponse{
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

func toEvaluationResponse(e *domain.PerformanceEvaluation) *dto.PerformanceEvaluationResponse {
	return &dto.PerformanceEvaluationResponse{
		ID:           e.ID,
		ModelID:      e.ModelID,
		DatasourceID: e.DatasourceID,
		Status:       string(e.Status),
		TaskType:     string(e.TaskType),
		Metrics:      e.Metrics,
		SampleCount:  e.SampleCount,
		ErrorMessage: e.ErrorMessage,
		StartedAt:    e.StartedAt,
		CompletedAt:  e.CompletedAt,
		CreatedBy:    e.CreatedBy,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}
