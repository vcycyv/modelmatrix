package integration

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"modelmatrix-server/internal/infrastructure/auth"
	"modelmatrix-server/internal/infrastructure/db"
	"modelmatrix-server/internal/infrastructure/dbconnector"
	"modelmatrix-server/internal/infrastructure/fileservice"
	infraldap "modelmatrix-server/internal/infrastructure/ldap"
	"modelmatrix-server/migrations"

	"github.com/joho/godotenv"

	// Datasource module
	dsApi "modelmatrix-server/internal/module/datasource/api"
	dsApp "modelmatrix-server/internal/module/datasource/application"
	dsDomain "modelmatrix-server/internal/module/datasource/domain"
	dsRepo "modelmatrix-server/internal/module/datasource/repository"

	// Model Build module
	buildApi "modelmatrix-server/internal/module/build/api"
	buildApp "modelmatrix-server/internal/module/build/application"
	buildDomain "modelmatrix-server/internal/module/build/domain"
	buildRepo "modelmatrix-server/internal/module/build/repository"

	"modelmatrix-server/internal/infrastructure/compute"

	// Model Manage module
	invApi "modelmatrix-server/internal/module/inventory/api"
	invApp "modelmatrix-server/internal/module/inventory/application"
	invDomain "modelmatrix-server/internal/module/inventory/domain"
	invRepo "modelmatrix-server/internal/module/inventory/repository"

	"modelmatrix-server/pkg/config"
	"modelmatrix-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var (
	testDB     *gorm.DB
	testRouter *gin.Engine
	testConfig *config.Config
)

// setupTestServer creates a test HTTP server with all routes configured
func setupTestServer(t *testing.T) *httptest.Server {
	// Load test configuration
	cfg := loadTestConfig()

	// Set the global config so db.Init() can access it
	config.SetConfig(cfg)
	testConfig = cfg

	// Initialize logger
	err := logger.Init(cfg.Logging.Level, cfg.Logging.Format, "stdout", "")
	require.NoError(t, err, "Failed to initialize logger")

	// Initialize database
	database, err := db.Init(&cfg.Database)
	require.NoError(t, err, "Failed to connect to test database")
	testDB = database

	// Run migrations
	err = migrations.Migrate(database)
	require.NoError(t, err, "Failed to run migrations")

	// Create indexes
	err = migrations.CreateIndexes(database)
	if err != nil {
		t.Logf("Warning: Failed to create some indexes: %v", err)
	}

	// Initialize LDAP client
	ldapClient, err := infraldap.NewClient(&cfg.LDAP)
	require.NoError(t, err, "Failed to initialize LDAP client")

	// Initialize file service
	fileService, err := fileservice.NewFileService(&cfg.FileService)
	require.NoError(t, err, "Failed to initialize file service")

	// Initialize JWT token service
	tokenService := auth.NewTokenService(&cfg.JWT)

	// Initialize Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// API routes
	api := router.Group("/api")

	// Auth middleware
	authMiddleware := auth.Middleware(tokenService)

	// ===== Dependency Injection =====

	// --- Datasource Module ---
	dsDomainService := dsDomain.NewService()
	collectionRepo := dsRepo.NewCollectionRepository(database)
	datasourceRepo := dsRepo.NewDatasourceRepository(database)
	columnRepo := dsRepo.NewColumnRepository(database)
	externalDBConnector := dbconnector.NewExternalDBConnector()

	collectionService := dsApp.NewCollectionService(collectionRepo, dsDomainService)
	datasourceService := dsApp.NewDatasourceService(database, datasourceRepo, collectionRepo, columnRepo, dsDomainService, fileService, externalDBConnector)
	columnService := dsApp.NewColumnService(database, columnRepo, datasourceRepo, dsDomainService)

	authController := dsApi.NewAuthController(ldapClient, tokenService)
	collectionController := dsApi.NewCollectionController(collectionService)
	datasourceController := dsApi.NewDatasourceController(datasourceService, columnService)

	// Register datasource routes
	authController.RegisterRoutes(api)
	collectionController.RegisterRoutes(api, authMiddleware)
	datasourceController.RegisterRoutes(api, authMiddleware)

	// --- Model Manage Module (initialize first, needed by Build) ---
	invDomainService := invDomain.NewService()
	modelRepo := invRepo.NewModelRepository(database)
	modelService := invApp.NewModelService(modelRepo, invDomainService, fileService)
	modelController := invApi.NewModelController(modelService)

	// Register model manage routes
	modelController.RegisterRoutes(api, authMiddleware)

	// --- Model Build Module ---
	buildDomainService := buildDomain.NewService()
	buildRepo := buildRepo.NewBuildRepository(database)
	computeClient := &mockComputeClient{} // Mock client for integration tests
	buildService := buildApp.NewBuildService(buildRepo, buildDomainService, computeClient, datasourceService, modelService, nil, cfg)
	buildController := buildApi.NewBuildController(buildService)

	// Register model build routes
	buildController.RegisterRoutes(api, authMiddleware)

	testRouter = router
	testConfig = cfg

	return httptest.NewServer(router)
}

// loadTestConfig loads test configuration from environment variables
// It first tries to load .env.test file, then falls back to environment variables
func loadTestConfig() *config.Config {
	// Try to load .env.test file (ignore error if file doesn't exist)
	// Try multiple possible paths relative to common working directories
	possiblePaths := []string{
		filepath.Join("tests", "integration", ".env.test"), // From project root
		filepath.Join("integration", ".env.test"),          // From tests/ directory
		".env.test", // From tests/integration/ directory
		filepath.Join("..", "..", "tests", "integration", ".env.test"), // From deeper directories
	}

	for _, envPath := range possiblePaths {
		if _, err := os.Stat(envPath); err == nil {
			_ = godotenv.Load(envPath)
			break
		}
	}

	dbPort, _ := strconv.Atoi(getEnv("TEST_DB_PORT", "5432"))
	ldapPort, _ := strconv.Atoi(getEnv("TEST_LDAP_PORT", "3890"))

	cfg := &config.Config{
		Env: "test",
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: config.DatabaseConfig{
			Host:         getEnv("TEST_DB_HOST", "localhost"),
			Port:         dbPort,
			Username:     getEnv("TEST_DB_USER", "postgres"),
			Password:     getEnv("TEST_DB_PASSWORD", "dayang"),
			DBName:       getEnv("TEST_DB_NAME", "modelmatrixtest"),
			SSLMode:      "disable",
			MaxIdleConns: 10,
			MaxOpenConns: 100,
		},
		LDAP: config.LDAPConfig{
			Host:         getEnv("TEST_LDAP_HOST", "localhost"),
			Port:         ldapPort,
			BaseDN:       getEnv("TEST_LDAP_BASE_DN", "dc=example,dc=org"),
			BindDN:       getEnv("TEST_LDAP_BIND_DN", "cn=admin,dc=example,dc=org"),
			BindPassword: getEnv("TEST_LDAP_BIND_PASSWORD", "admin"),
			UserFilter:   "(uid=%s)",
			GroupFilter:  "(|(member=%s)(uniqueMember=%s))",
			UseTLS:       false,
		},
		FileService: config.FileServiceConfig{
			MinioEndpoint:  getEnv("TEST_MINIO_ENDPOINT", "localhost:9000"),
			MinioAccessKey: getEnv("TEST_MINIO_ACCESS_KEY", "minioadmin"),
			MinioSecretKey: getEnv("TEST_MINIO_SECRET_KEY", "minioadmin123"),
			MinioUseSSL:    false,
			MinioBucket:    getEnv("TEST_MINIO_BUCKET", "modelmatrixtest"),
		},
		JWT: config.JWTConfig{
			Secret:          getEnv("TEST_JWT_SECRET", "test-secret-key-change-in-production"),
			ExpirationHours: 24,
		},
		Logging: config.LoggingConfig{
			Level:    getEnv("TEST_LOG_LEVEL", "info"),
			Format:   "json",
			Output:   "stdout",
			FilePath: "",
		},
	}
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// cleanupTestDB cleans up test data (drops all tables or truncates)
func cleanupTestDB(t *testing.T) {
	if testDB == nil {
		return
	}

	// Option 1: Drop all tables (fresh start)
	// This is slower but ensures complete isolation
	err := testDB.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;").Error
	if err != nil {
		t.Logf("Warning: Failed to drop schema: %v", err)
	}

	// Re-run migrations
	err = migrations.Migrate(testDB)
	require.NoError(t, err, "Failed to re-run migrations after cleanup")

	// Re-create indexes
	err = migrations.CreateIndexes(testDB)
	if err != nil {
		t.Logf("Warning: Failed to re-create indexes: %v", err)
	}
}

// authenticate performs login and returns JWT token
func authenticate(t *testing.T, client *http.Client, baseURL, username, password string) string {
	loginReq := map[string]string{
		"username": username,
		"password": password,
	}
	body, _ := json.Marshal(loginReq)

	resp, err := client.Post(baseURL+"/api/auth/login", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.Equal(t, 200, result.Code)
	require.NotEmpty(t, result.Data.Token)

	return result.Data.Token
}

// makeRequest is a helper to make authenticated HTTP requests
func makeRequest(t *testing.T, client *http.Client, method, url, token string, body interface{}) *http.Response {
	var req *http.Request
	var err error

	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req, err = http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		require.NoError(t, err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

// parseResponse parses JSON response into a struct
func parseResponse(t *testing.T, resp *http.Response, result interface{}) {
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(result)
	require.NoError(t, err, "Failed to parse response")
}

// requireSuccess asserts that response has code 200
func requireSuccess(t *testing.T, resp *http.Response) {
	require.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")
}

// requireNoContent asserts that response has code 204
func requireNoContent(t *testing.T, resp *http.Response) {
	require.Equal(t, http.StatusNoContent, resp.StatusCode, "Expected 204 No Content")
}

// requireCreated asserts that response has code 201
func requireCreated(t *testing.T, resp *http.Response) {
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Expected 201 Created")
}

// requireBadRequest asserts that response has code 400
func requireBadRequest(t *testing.T, resp *http.Response) {
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Expected 400 Bad Request")
}

// requireUnauthorized asserts that response has code 401
func requireUnauthorized(t *testing.T, resp *http.Response) {
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Expected 401 Unauthorized")
}

// requireForbidden asserts that response has code 403
func requireForbidden(t *testing.T, resp *http.Response) {
	require.Equal(t, http.StatusForbidden, resp.StatusCode, "Expected 403 Forbidden")
}

// requireNotFound asserts that response has code 404
func requireNotFound(t *testing.T, resp *http.Response) {
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Expected 404 Not Found")
}

// makeMultipartRequest is a helper to make authenticated multipart/form-data requests
func makeMultipartRequest(t *testing.T, client *http.Client, method, url, token string, formData map[string]string, fileField, filePath string) *http.Response {
	// Read the file
	fileData, err := os.ReadFile(filePath)
	require.NoError(t, err, "Failed to read file: %s", filePath)

	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add form fields
	for key, value := range formData {
		err := writer.WriteField(key, value)
		require.NoError(t, err, "Failed to write form field: %s", key)
	}

	// Add file
	part, err := writer.CreateFormFile(fileField, filepath.Base(filePath))
	require.NoError(t, err, "Failed to create form file")
	_, err = part.Write(fileData)
	require.NoError(t, err, "Failed to write file data")

	err = writer.Close()
	require.NoError(t, err, "Failed to close multipart writer")

	// Create request
	req, err := http.NewRequest(method, url, &body)
	require.NoError(t, err, "Failed to create request")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	require.NoError(t, err, "Failed to execute request")
	return resp
}

// mockComputeClient is a mock implementation of compute.Client for integration tests
type mockComputeClient struct{}

func (m *mockComputeClient) TrainModel(req *compute.TrainRequest) (*compute.TrainResponse, error) {
	return &compute.TrainResponse{
		JobID:   "mock-job-id",
		Status:  "accepted",
		Message: "Training job accepted (mock)",
	}, nil
}

func (m *mockComputeClient) GetStatus(jobID string) (*compute.JobStatusResponse, error) {
	return &compute.JobStatusResponse{
		JobID:    jobID,
		Status:   "completed",
		Progress: 100,
	}, nil
}

func (m *mockComputeClient) HealthCheck() error {
	return nil
}
