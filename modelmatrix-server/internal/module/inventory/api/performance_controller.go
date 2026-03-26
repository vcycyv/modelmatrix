package api

import (
	"modelmatrix-server/internal/infrastructure/auth"
	"modelmatrix-server/internal/module/inventory/application"
	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"
	"modelmatrix-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// PerformanceController handles model performance monitoring HTTP requests
type PerformanceController struct {
	performanceService application.PerformanceService
}

// NewPerformanceController creates a new performance controller
func NewPerformanceController(performanceService application.PerformanceService) *PerformanceController {
	return &PerformanceController{
		performanceService: performanceService,
	}
}

// RegisterRoutes registers performance monitoring routes
func (c *PerformanceController) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// All performance routes are under /models/:id/performance
	perf := router.Group("/models/:id/performance")
	perf.Use(authMiddleware)
	{
		// Summary
		perf.GET("", auth.RequireViewer(), c.GetSummary)

		// Baseline operations
		perf.GET("/baselines", auth.RequireViewer(), c.GetBaselines)
		perf.POST("/baselines", auth.RequireEditor(), c.CreateBaseline)

		// Performance records
		perf.GET("/history", auth.RequireViewer(), c.GetHistory)
		perf.POST("/record", auth.RequireEditor(), c.RecordPerformance)
		perf.GET("/metrics/:metricName/series", auth.RequireViewer(), c.GetMetricTimeSeries)

		// Evaluation
		perf.POST("/evaluate", auth.RequireEditor(), c.StartEvaluation)
		perf.GET("/evaluations", auth.RequireViewer(), c.GetEvaluations)
		perf.GET("/evaluations/:evaluationId", auth.RequireViewer(), c.GetEvaluation)

		// Alerts
		perf.GET("/alerts", auth.RequireViewer(), c.GetAlerts)
		perf.PUT("/alerts/:alertId", auth.RequireEditor(), c.UpdateAlert)

		// Thresholds
		perf.GET("/thresholds", auth.RequireViewer(), c.GetThresholds)
		perf.PUT("/thresholds", auth.RequireEditor(), c.UpdateThreshold)
	}

	// Callback endpoint (no auth, called by compute service)
	router.POST("/models/:id/performance/evaluations/:evaluationId/callback", c.EvaluationCallback)

	// Global threshold default settings (admin only)
	defaults := router.Group("/performance/threshold-defaults")
	defaults.Use(authMiddleware)
	{
		defaults.GET("", auth.RequireViewer(), c.GetThresholdDefaults)
		defaults.PUT("", auth.RequireAdmin(), c.UpsertThresholdDefault)
	}
}

// GetSummary godoc
// @Summary Get performance summary
// @Description Retrieves a summary of model performance status including alerts, baselines, and latest metrics
// @Tags Model Performance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Success 200 {object} response.Response{data=dto.PerformanceSummaryResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/performance [get]
func (c *PerformanceController) GetSummary(ctx *gin.Context) {
	modelID := ctx.Param("id")

	result, err := c.performanceService.GetPerformanceSummary(modelID)
	if err != nil {
		handlePerformanceError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// GetBaselines godoc
// @Summary Get performance baselines
// @Description Retrieves all baseline metrics for a model
// @Tags Model Performance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Success 200 {object} response.Response{data=dto.BaselinesListResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/performance/baselines [get]
func (c *PerformanceController) GetBaselines(ctx *gin.Context) {
	modelID := ctx.Param("id")

	result, err := c.performanceService.GetBaselines(modelID)
	if err != nil {
		handlePerformanceError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// CreateBaseline godoc
// @Summary Create or update performance baselines
// @Description Creates or updates baseline metrics for a model
// @Tags Model Performance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param baseline body dto.CreateBaselineRequest true "Baseline metrics"
// @Success 200 {object} response.Response{data=dto.BaselinesListResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/performance/baselines [post]
func (c *PerformanceController) CreateBaseline(ctx *gin.Context) {
	modelID := ctx.Param("id")

	var req dto.CreateBaselineRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	username := auth.GetUsername(ctx)
	result, err := c.performanceService.CreateBaseline(modelID, &req, username)
	if err != nil {
		handlePerformanceError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// GetHistory godoc
// @Summary Get performance history
// @Description Retrieves historical performance records for a model
// @Tags Model Performance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param metric_name query string false "Filter by metric name"
// @Param limit query int false "Maximum number of records" default(100)
// @Param start_time query string false "Start time filter (RFC3339)"
// @Param end_time query string false "End time filter (RFC3339)"
// @Success 200 {object} response.Response{data=dto.PerformanceHistoryResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/performance/history [get]
func (c *PerformanceController) GetHistory(ctx *gin.Context) {
	modelID := ctx.Param("id")

	var params dto.GetPerformanceHistoryParams
	if err := ctx.ShouldBindQuery(&params); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.performanceService.GetPerformanceHistory(modelID, &params)
	if err != nil {
		handlePerformanceError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// RecordPerformance godoc
// @Summary Record performance metrics
// @Description Manually records performance metrics for a model
// @Tags Model Performance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param record body dto.RecordPerformanceRequest true "Performance metrics"
// @Success 200 {object} response.Response{data=dto.PerformanceHistoryResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/performance/record [post]
func (c *PerformanceController) RecordPerformance(ctx *gin.Context) {
	modelID := ctx.Param("id")

	var req dto.RecordPerformanceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	username := auth.GetUsername(ctx)
	result, err := c.performanceService.RecordPerformance(modelID, &req, username)
	if err != nil {
		handlePerformanceError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// GetMetricTimeSeries godoc
// @Summary Get metric time series
// @Description Retrieves time-series data for a specific metric
// @Tags Model Performance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param metricName path string true "Metric name"
// @Param limit query int false "Maximum number of data points" default(100)
// @Success 200 {object} response.Response{data=dto.MetricTimeSeriesResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/performance/metrics/{metricName}/series [get]
func (c *PerformanceController) GetMetricTimeSeries(ctx *gin.Context) {
	modelID := ctx.Param("id")
	metricName := ctx.Param("metricName")

	limit := 100
	if l := ctx.Query("limit"); l != "" {
		// Parse limit if provided
		if _, err := ctx.GetQuery("limit"); err {
			limit = 100 // Default
		}
	}

	result, err := c.performanceService.GetMetricTimeSeries(modelID, metricName, limit)
	if err != nil {
		handlePerformanceError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// StartEvaluation godoc
// @Summary Start performance evaluation
// @Description Starts an asynchronous performance evaluation job
// @Tags Model Performance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param evaluation body dto.EvaluatePerformanceRequest true "Evaluation request"
// @Success 202 {object} response.Response{data=dto.PerformanceEvaluationResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/performance/evaluate [post]
func (c *PerformanceController) StartEvaluation(ctx *gin.Context) {
	modelID := ctx.Param("id")

	var req dto.EvaluatePerformanceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	username := auth.GetUsername(ctx)
	result, err := c.performanceService.StartEvaluation(modelID, &req, username)
	if err != nil {
		handlePerformanceError(ctx, err)
		return
	}

	response.Accepted(ctx, result)
}

// GetEvaluations godoc
// @Summary Get evaluations list
// @Description Retrieves evaluation jobs for a model
// @Tags Model Performance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param limit query int false "Maximum number of evaluations" default(20)
// @Success 200 {object} response.Response{data=dto.EvaluationsListResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/performance/evaluations [get]
func (c *PerformanceController) GetEvaluations(ctx *gin.Context) {
	modelID := ctx.Param("id")

	limit := 20
	if l := ctx.Query("limit"); l != "" {
		limit = 20 // Use default for now
	}

	result, err := c.performanceService.GetEvaluations(modelID, limit)
	if err != nil {
		handlePerformanceError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// GetEvaluation godoc
// @Summary Get evaluation details
// @Description Retrieves a specific evaluation job
// @Tags Model Performance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param evaluationId path string true "Evaluation ID (UUID)"
// @Success 200 {object} response.Response{data=dto.PerformanceEvaluationResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/performance/evaluations/{evaluationId} [get]
func (c *PerformanceController) GetEvaluation(ctx *gin.Context) {
	evaluationID := ctx.Param("evaluationId")

	result, err := c.performanceService.GetEvaluation(evaluationID)
	if err != nil {
		handlePerformanceError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// EvaluationCallback godoc
// @Summary Handle evaluation callback
// @Description Receives callback from compute service when evaluation completes
// @Tags Model Performance
// @Accept json
// @Produce json
// @Param id path string true "Model ID (UUID)"
// @Param evaluationId path string true "Evaluation ID (UUID)"
// @Param callback body dto.EvaluationCallbackRequest true "Callback data"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/models/{id}/performance/evaluations/{evaluationId}/callback [post]
func (c *PerformanceController) EvaluationCallback(ctx *gin.Context) {
	var req dto.EvaluationCallbackRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	// Use IDs from path if not in body
	if req.EvaluationID == "" {
		req.EvaluationID = ctx.Param("evaluationId")
	}
	if req.ModelID == "" {
		req.ModelID = ctx.Param("id")
	}

	if err := c.performanceService.HandleEvaluationCallback(&req); err != nil {
		response.InternalError(ctx, err.Error())
		return
	}

	response.Success(ctx, map[string]string{"status": "acknowledged"})
}

// GetAlerts godoc
// @Summary Get performance alerts
// @Description Retrieves performance alerts for a model
// @Tags Model Performance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param status query string false "Filter by status (active, acknowledged, resolved)"
// @Param limit query int false "Maximum number of alerts" default(50)
// @Success 200 {object} response.Response{data=dto.AlertsListResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/performance/alerts [get]
func (c *PerformanceController) GetAlerts(ctx *gin.Context) {
	modelID := ctx.Param("id")

	var params dto.GetAlertsParams
	if err := ctx.ShouldBindQuery(&params); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.performanceService.GetAlerts(modelID, &params)
	if err != nil {
		handlePerformanceError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// UpdateAlert godoc
// @Summary Update alert status
// @Description Updates the status of a performance alert
// @Tags Model Performance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param alertId path string true "Alert ID (UUID)"
// @Param update body dto.UpdateAlertRequest true "Alert update"
// @Success 200 {object} response.Response{data=dto.PerformanceAlertResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/performance/alerts/{alertId} [put]
func (c *PerformanceController) UpdateAlert(ctx *gin.Context) {
	alertID := ctx.Param("alertId")

	var req dto.UpdateAlertRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	username := auth.GetUsername(ctx)
	result, err := c.performanceService.UpdateAlert(alertID, &req, username)
	if err != nil {
		handlePerformanceError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// GetThresholds godoc
// @Summary Get performance thresholds
// @Description Retrieves threshold configurations for a model
// @Tags Model Performance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Success 200 {object} response.Response{data=dto.ThresholdsListResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/performance/thresholds [get]
func (c *PerformanceController) GetThresholds(ctx *gin.Context) {
	modelID := ctx.Param("id")

	result, err := c.performanceService.GetThresholds(modelID)
	if err != nil {
		handlePerformanceError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// UpdateThreshold godoc
// @Summary Update performance threshold
// @Description Updates or creates a performance threshold configuration
// @Tags Model Performance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param threshold body dto.UpdateThresholdRequest true "Threshold configuration"
// @Success 200 {object} response.Response{data=dto.PerformanceThresholdResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/performance/thresholds [put]
func (c *PerformanceController) UpdateThreshold(ctx *gin.Context) {
	modelID := ctx.Param("id")

	var req dto.UpdateThresholdRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.performanceService.UpdateThreshold(modelID, &req)
	if err != nil {
		handlePerformanceError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// GetThresholdDefaults godoc
// @Summary Get global threshold defaults
// @Description Retrieves org-wide default threshold settings for a task type
// @Tags Performance Settings
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param task_type query string true "Task type" Enums(classification,regression)
// @Success 200 {object} response.Response{data=dto.ThresholdDefaultsListResponse}
// @Failure 401 {object} response.Response
// @Router /api/performance/threshold-defaults [get]
func (c *PerformanceController) GetThresholdDefaults(ctx *gin.Context) {
	taskType := ctx.Query("task_type")
	if taskType == "" {
		response.BadRequest(ctx, "task_type query parameter is required")
		return
	}
	result, err := c.performanceService.GetThresholdDefaults(taskType)
	if err != nil {
		handlePerformanceError(ctx, err)
		return
	}
	response.Success(ctx, result)
}

// UpsertThresholdDefault godoc
// @Summary Update global threshold default
// @Description Creates or updates an org-wide threshold default (admin only)
// @Tags Performance Settings
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param body body dto.UpdateThresholdDefaultRequest true "Default threshold settings"
// @Success 200 {object} response.Response{data=dto.PerformanceThresholdDefaultResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /api/performance/threshold-defaults [put]
func (c *PerformanceController) UpsertThresholdDefault(ctx *gin.Context) {
	var req dto.UpdateThresholdDefaultRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}
	username := auth.GetUsername(ctx)
	result, err := c.performanceService.UpsertThresholdDefault(&req, username)
	if err != nil {
		handlePerformanceError(ctx, err)
		return
	}
	response.Success(ctx, result)
}

// handlePerformanceError maps domain errors to HTTP responses
func handlePerformanceError(ctx *gin.Context, err error) {
	switch err {
	case domain.ErrModelNotFound:
		response.NotFound(ctx, "model not found")
	case domain.ErrBaselineNotFound:
		response.NotFound(ctx, "baseline not found")
	case domain.ErrRecordNotFound:
		response.NotFound(ctx, "performance record not found")
	case domain.ErrAlertNotFound:
		response.NotFound(ctx, "alert not found")
	case domain.ErrEvaluationNotFound:
		response.NotFound(ctx, "evaluation not found")
	case domain.ErrThresholdNotFound:
		response.NotFound(ctx, "threshold not found")
	case domain.ErrInvalidTaskType:
		response.BadRequest(ctx, "invalid task type")
	case domain.ErrInvalidAlertSeverity:
		response.BadRequest(ctx, "invalid alert severity")
	case domain.ErrInvalidAlertStatus:
		response.BadRequest(ctx, "invalid alert status")
	case domain.ErrEvaluationRunning:
		response.Conflict(ctx, "evaluation is already running")
	case domain.ErrNoActualTargetColumn:
		response.BadRequest(ctx, "no actual target column found in evaluation data")
	case domain.ErrInvalidThresholdValues:
		response.BadRequest(ctx, "thresholds must be positive and warning must not exceed critical")
	default:
		response.InternalError(ctx, err.Error())
	}
}
