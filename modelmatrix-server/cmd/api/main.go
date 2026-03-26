package main

import (
	"fmt"
	"os"
	"strings"

	"modelmatrix-server/internal/infrastructure/auth"
	"modelmatrix-server/internal/infrastructure/compute"
	"modelmatrix-server/internal/infrastructure/db"
	"modelmatrix-server/internal/infrastructure/dbconnector"
	"modelmatrix-server/internal/infrastructure/fileservice"
	infraldap "modelmatrix-server/internal/infrastructure/ldap"
	"modelmatrix-server/migrations"

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

	// Model Manage module
	invApi "modelmatrix-server/internal/module/inventory/api"
	invApp "modelmatrix-server/internal/module/inventory/application"
	invDomain "modelmatrix-server/internal/module/inventory/domain"
	invRepo "modelmatrix-server/internal/module/inventory/repository"

	// Folder module
	folderApi "modelmatrix-server/internal/module/folder/api"
	folderApp "modelmatrix-server/internal/module/folder/application"
	folderRepo "modelmatrix-server/internal/module/folder/repository"

	// Search module
	searchApi "modelmatrix-server/internal/module/search/api"

	"modelmatrix-server/pkg/config"
	"modelmatrix-server/pkg/logger"
	"modelmatrix-server/pkg/response"
	"modelmatrix-server/pkg/swagger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// @title ModelMatrix API
// @version 1.0
// @description ModelMatrix Backend API for ML Model Management
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@modelmatrix.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format: Bearer {token}

func main() {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output, cfg.Logging.FilePath); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting ModelMatrix Backend API")
	logger.Info("Environment: %s", cfg.Env)

	// Initialize database
	database, err := db.Init(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to initialize database: %v", err)
	}
	defer db.Close()

	logger.Info("Database connection established")

	// Run migrations
	if err := runMigrations(database); err != nil {
		logger.Fatal("Failed to run migrations: %v", err)
	}

	// Initialize LDAP client
	ldapClient, err := infraldap.NewClient(&cfg.LDAP)
	if err != nil {
		logger.Fatal("Failed to initialize LDAP client: %v", err)
	}
	defer ldapClient.Close()

	logger.Info("LDAP client initialized")

	// Initialize file service
	fileService, err := fileservice.NewFileService(&cfg.FileService)
	if err != nil {
		logger.Fatal("Failed to initialize file service: %v", err)
	}

	logger.Info("File service initialized")

	// Initialize JWT token service
	tokenService := auth.NewTokenService(&cfg.JWT)

	// Initialize Gin router
	if cfg.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(requestLogger())

	// Setup Swagger
	swagger.Setup(router)

	// API routes
	api := router.Group("/api")

	// Health check endpoint
	api.GET("/health", healthCheckHandler(ldapClient, fileService))

	// Auth middleware
	authMiddleware := auth.Middleware(tokenService)

	// ===== Dependency Injection =====

	// --- Datasource Module ---
	dsDomainService := dsDomain.NewService()
	collectionRepo := dsRepo.NewCollectionRepository(database)
	datasourceRepo := dsRepo.NewDatasourceRepository(database)
	columnRepo := dsRepo.NewColumnRepository(database)
	externalDBConnector := dbconnector.NewExternalDBConnector()

	collectionService := dsApp.NewCollectionService(collectionRepo, datasourceRepo, dsDomainService, fileService)
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

	// --- Model Version Module ---
	versionRepo := invRepo.NewModelVersionRepository(database)
	versionService := invApp.NewModelVersionService(modelRepo, versionRepo, fileService.(fileservice.VersionStore))
	versionController := invApi.NewVersionController(versionService)

	// --- Performance Monitoring (part of inventory module) ---
	performanceRepo := invRepo.NewPerformanceRepository(database)
	performanceService := invApp.NewPerformanceService(performanceRepo, modelRepo)
	performanceController := invApi.NewPerformanceController(performanceService)

	// Register version routes (before model so /models/:id/versions are available)
	versionController.RegisterRoutes(api, authMiddleware)

	// Register performance monitoring routes
	performanceController.RegisterRoutes(api, authMiddleware)

	// --- Folder Module (initialized early as it's needed by build service) ---
	folderRepoImpl := folderRepo.NewFolderRepository(database)
	projectRepoImpl := folderRepo.NewProjectRepository(database)
	folderSvc := folderApp.NewFolderService(database, folderRepoImpl, projectRepoImpl)

	// --- Model Build Module ---
	buildDomainService := buildDomain.NewService()
	buildRepo := buildRepo.NewBuildRepository(database)

	// Initialize compute service client
	computeClient := compute.NewClient(&cfg.Compute)

	// Configure scoring for model service
	dsGetter := &datasourceGetterAdapter{svc: datasourceService}
	dsCreator := &datasourceCreatorAdapter{svc: datasourceService}
	modelService.ConfigureScoring(computeClient, dsGetter, dsCreator, cfg)

	// Configure compute for performance service
	performanceService.ConfigureCompute(computeClient, dsGetter, cfg)
	buildService := buildApp.NewBuildService(buildRepo, buildDomainService, computeClient, datasourceService, modelService, versionService, folderSvc, performanceService, cfg)
	buildController := buildApi.NewBuildController(buildService)

	// Wire up cascade delete dependencies (after services are created)
	folderSvc.SetModelDeleter(modelService)
	folderSvc.SetBuildDeleter(buildService)

	// Re-register model controller with retrain (registers all model routes including POST /:id/retrain)
	modelControllerWithRetrain := invApi.NewModelControllerWithRetrain(modelService, buildService)
	modelControllerWithRetrain.RegisterRoutes(api, authMiddleware)

	// Register model build routes
	buildController.RegisterRoutes(api, authMiddleware)

	// --- Folder Controller ---
	folderController := folderApi.NewFolderController(folderSvc, buildService, modelService)

	// Register folder routes
	folderController.RegisterRoutes(api, authMiddleware)

	// --- Search ---
	searchController := searchApi.NewSearchController(database)
	searchController.RegisterRoutes(api, authMiddleware)

	// Serve static files for the web UI
	setupStaticFileServing(router)

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info("Server listening on %s", addr)
	logger.Info("Swagger UI available at http://%s/swagger/index.html", addr)

	if err := router.Run(addr); err != nil {
		logger.Fatal("Failed to start server: %v", err)
	}
}

// setupStaticFileServing configures serving of the React SPA from the dist folder
func setupStaticFileServing(router *gin.Engine) {
	// Path to the dist folder containing built React app
	distPath := "./dist"

	// Check if dist folder exists
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		logger.Warn("Static files directory 'dist' not found. Web UI will not be available.")
		logger.Warn("Run 'cd web && npm install && npm run build' to build the UI")
		return
	}

	// Serve static assets (js, css, images, etc.)
	router.Static("/assets", distPath+"/assets")

	// Serve other static files from the root
	router.StaticFile("/vite.svg", distPath+"/vite.svg")

	// For SPA routing: serve index.html for all non-API, non-asset routes
	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Don't serve index.html for API routes or swagger
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/swagger") {
			c.JSON(404, gin.H{"error": "Not found"})
			return
		}

		// Check if it's a request for a static file that exists
		filePath := distPath + path
		if _, err := os.Stat(filePath); err == nil {
			c.File(filePath)
			return
		}

		// For all other routes, serve index.html (SPA client-side routing)
		c.File(distPath + "/index.html")
	})

	logger.Info("Static file serving configured from '%s'", distPath)
}

// runMigrations runs database migrations
func runMigrations(database interface{}) error {
	gormDB := database.(*gorm.DB)
	
	// Import and run migrations
	if err := migrations.Migrate(gormDB); err != nil {
		return err
	}
	
	// Create additional indexes
	if err := migrations.CreateIndexes(gormDB); err != nil {
		logger.Warn("Some indexes may already exist: %v", err)
	}

	logger.Info("Database migrations completed")
	return nil
}

// healthCheckHandler returns a health check handler
func healthCheckHandler(ldapClient infraldap.Client, fileService fileservice.FileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		health := struct {
			Status   string `json:"status"`
			Database string `json:"database"`
			LDAP     string `json:"ldap"`
			MinIO    string `json:"minio"`
		}{
			Status:   "healthy",
			Database: "healthy",
			LDAP:     "healthy",
			MinIO:    "healthy",
		}

		// Check database
		if err := db.HealthCheck(); err != nil {
			health.Status = "unhealthy"
			health.Database = "unhealthy: " + err.Error()
		}

		// Check LDAP
		if err := ldapClient.HealthCheck(); err != nil {
			health.Status = "unhealthy"
			health.LDAP = "unhealthy: " + err.Error()
		}

		// Check MinIO
		if err := fileService.HealthCheck(); err != nil {
			health.Status = "unhealthy"
			health.MinIO = "unhealthy: " + err.Error()
		}

		if health.Status == "healthy" {
			response.Success(c, health)
		} else {
			response.ServiceUnavailable(c, health.Status)
		}
	}
}

// corsMiddleware adds CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// requestLogger logs incoming requests
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Debug("Request: %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
	}
}

// datasourceGetterAdapter adapts DatasourceService to DatasourceGetter interface
type datasourceGetterAdapter struct {
	svc dsApp.DatasourceService
}

func (a *datasourceGetterAdapter) GetFilePath(datasourceID string) (string, error) {
	ds, err := a.svc.GetByID(datasourceID)
	if err != nil {
		return "", err
	}
	if ds.FilePath == "" {
		return "", fmt.Errorf("datasource %s has no file path", datasourceID)
	}
	return ds.FilePath, nil
}

// datasourceCreatorAdapter adapts DatasourceService to DatasourceCreator interface
type datasourceCreatorAdapter struct {
	svc dsApp.DatasourceService
}

func (a *datasourceCreatorAdapter) CreateScoredOutput(collectionID, name, filePath string, rowCount int, createdBy string) (string, error) {
	resp, err := a.svc.CreateFromExistingFile(collectionID, name, filePath, rowCount, createdBy)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}
