package compute

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"modelmatrix-server/pkg/config"
	"modelmatrix-server/pkg/logger"
)

// Client is the interface for communicating with the compute service
type Client interface {
	TrainModel(req *TrainRequest) (*TrainResponse, error)
	ScoreModel(req *ScoreRequest) (*ScoreResponse, error)
	EvaluatePerformance(req *EvaluateRequest) (*EvaluateResponse, error)
	GetStatus(jobID string) (*JobStatusResponse, error)
	HealthCheck() error
}

// TrainRequest represents a model training request
type TrainRequest struct {
	DatasourceID    string                 `json:"datasource_id"`
	BuildID         string                 `json:"build_id"`
	FilePath        string                 `json:"file_path"`
	Algorithm       string                 `json:"algorithm"`
	ModelType       string                 `json:"model_type"` // classification, regression, clustering
	Hyperparameters map[string]interface{} `json:"hyperparameters"`
	TargetColumn    string                 `json:"target_column"`
	InputColumns    []string               `json:"input_columns"`
	CallbackURL     string                 `json:"callback_url,omitempty"`
}

// CallbackPayload represents the payload sent by compute service when job completes
type CallbackPayload struct {
	BuildID   string                 `json:"build_id"`
	JobID     string                 `json:"job_id"`
	Status    string                 `json:"status"` // "completed" or "failed"
	ModelPath *string                `json:"model_path,omitempty"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
	Error     *string                `json:"error,omitempty"`
}

// TrainResponse represents the response from a training request
type TrainResponse struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// JobStatusResponse represents the status of a training job
type JobStatusResponse struct {
	JobID     string                 `json:"job_id"`
	Status    string                 `json:"status"`
	Progress  int                    `json:"progress"`
	ModelPath *string                `json:"model_path,omitempty"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
	Error     *string                `json:"error,omitempty"`
}

// ScoreRequest represents a model scoring request
type ScoreRequest struct {
	ModelID       string   `json:"model_id"`
	ModelFilePath string   `json:"model_file_path"`
	InputFilePath string   `json:"input_file_path"`
	OutputPath    string   `json:"output_path"`
	InputColumns  []string `json:"input_columns"`
	ModelType     string   `json:"model_type"` // classification, regression, clustering
	Algorithm     string   `json:"algorithm"`
	CallbackURL   string   `json:"callback_url,omitempty"`
}

// ScoreResponse represents the response from a scoring request
type ScoreResponse struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// EvaluateRequest represents a performance evaluation request
type EvaluateRequest struct {
	EvaluationID       string   `json:"evaluation_id"`
	ModelID            string   `json:"model_id"`
	ModelFilePath      string   `json:"model_file_path"`
	DatasourceFilePath string   `json:"datasource_file_path"`
	InputColumns       []string `json:"input_columns"`
	TargetColumn       string   `json:"target_column"`      // Expected target column name
	ActualColumn       string   `json:"actual_column"`      // Actual values column in evaluation data
	PredictionColumn   string   `json:"prediction_column"`  // Optional: if predictions already exist
	ModelType          string   `json:"model_type"`         // classification, regression
	CallbackURL        string   `json:"callback_url,omitempty"`
}

// EvaluateResponse represents the response from an evaluation request
type EvaluateResponse struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// EvaluateCallbackPayload represents the callback payload for evaluation results
type EvaluateCallbackPayload struct {
	EvaluationID string                 `json:"evaluation_id"`
	ModelID      string                 `json:"model_id"`
	Status       string                 `json:"status"` // "completed" or "failed"
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	SampleCount  int                    `json:"sample_count,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// HTTPClient implements the Client interface using HTTP
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewClient creates a new compute service client
func NewClient(cfg *config.ComputeConfig) Client {
	return &HTTPClient{
		baseURL: cfg.ServiceURL,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		apiKey: cfg.APIKey,
	}
}

// TrainModel sends a training request to the compute service
func (c *HTTPClient) TrainModel(req *TrainRequest) (*TrainResponse, error) {
	url := fmt.Sprintf("%s/compute/train", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("compute service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var trainResp TrainResponse
	if err := json.NewDecoder(resp.Body).Decode(&trainResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.Info("Training job started: %s", trainResp.JobID)
	return &trainResp, nil
}

// ScoreModel sends a scoring request to the compute service
func (c *HTTPClient) ScoreModel(req *ScoreRequest) (*ScoreResponse, error) {
	url := fmt.Sprintf("%s/compute/score", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("compute service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var scoreResp ScoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&scoreResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.Info("Scoring job started: %s", scoreResp.JobID)
	return &scoreResp, nil
}

// EvaluatePerformance sends a performance evaluation request to the compute service
func (c *HTTPClient) EvaluatePerformance(req *EvaluateRequest) (*EvaluateResponse, error) {
	url := fmt.Sprintf("%s/compute/evaluate", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("compute service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var evalResp EvaluateResponse
	if err := json.NewDecoder(resp.Body).Decode(&evalResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.Info("Evaluation job started: %s", evalResp.JobID)
	return &evalResp, nil
}

// GetStatus retrieves the status of a training job
func (c *HTTPClient) GetStatus(jobID string) (*JobStatusResponse, error) {
	url := fmt.Sprintf("%s/compute/status/%s", c.baseURL, jobID)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		httpReq.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("job %s not found", jobID)
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("compute service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var statusResp JobStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &statusResp, nil
}

// HealthCheck checks if the compute service is healthy
func (c *HTTPClient) HealthCheck() error {
	url := fmt.Sprintf("%s/compute/health", c.baseURL)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("compute service health check failed with status %d", resp.StatusCode)
	}

	return nil
}


