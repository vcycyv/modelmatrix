package main

import (
	"fmt"
	"os"

	"modelmatrix_backend/internal/infrastructure/auth"
	"modelmatrix_backend/internal/infrastructure/db"
	"modelmatrix_backend/internal/infrastructure/fileservice"
	infraldap "modelmatrix_backend/internal/infrastructure/ldap"
	"modelmatrix_backend/migrations"

	// Datasource module
	dsApi "modelmatrix_backend/internal/module/datasource/api"
	dsApp "modelmatrix_backend/internal/module/datasource/application"
	dsDomain "modelmatrix_backend/internal/module/datasource/domain"
	dsRepo "modelmatrix_backend/internal/module/datasource/repository"

	// Model Build module
	mbApi "modelmatrix_backend/internal/module/modelbuild/api"
	mbApp "modelmatrix_backend/internal/module/modelbuild/application"
	mbDomain "modelmatrix_backend/internal/module/modelbuild/domain"
	mbRepo "modelmatrix_backend/internal/module/modelbuild/repository"

	// Model Manage module
	mmApi "modelmatrix_backend/internal/module/modelmanage/api"
	mmApp "modelmatrix_backend/internal/module/modelmanage/application"
	mmDomain "modelmatrix_backend/internal/module/modelmanage/domain"
	mmRepo "modelmatrix_backend/internal/module/modelmanage/repository"

	"modelmatrix_backend/pkg/config"
	"modelmatrix_backend/pkg/logger"
	"modelmatrix_backend/pkg/response"
	"modelmatrix_backend/pkg/swagger"

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

	collectionService := dsApp.NewCollectionService(collectionRepo, dsDomainService)
	datasourceService := dsApp.NewDatasourceService(database, datasourceRepo, collectionRepo, columnRepo, dsDomainService, fileService)
	columnService := dsApp.NewColumnService(database, columnRepo, datasourceRepo, dsDomainService)

	authController := dsApi.NewAuthController(ldapClient, tokenService)
	collectionController := dsApi.NewCollectionController(collectionService)
	datasourceController := dsApi.NewDatasourceController(datasourceService, columnService)

	// Register datasource routes
	authController.RegisterRoutes(api)
	collectionController.RegisterRoutes(api, authMiddleware)
	datasourceController.RegisterRoutes(api, authMiddleware)

	// --- Model Build Module ---
	mbDomainService := mbDomain.NewService()
	buildRepo := mbRepo.NewBuildRepository(database)
	buildService := mbApp.NewBuildService(buildRepo, mbDomainService)
	buildController := mbApi.NewBuildController(buildService)

	// Register model build routes
	buildController.RegisterRoutes(api, authMiddleware)

	// --- Model Manage Module ---
	mmDomainService := mmDomain.NewService()
	modelRepo := mmRepo.NewModelRepository(database)
	versionRepo := mmRepo.NewVersionRepository(database)
	modelService := mmApp.NewModelService(modelRepo, versionRepo, mmDomainService)
	modelController := mmApi.NewModelController(modelService)

	// Register model manage routes
	modelController.RegisterRoutes(api, authMiddleware)

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info("Server listening on %s", addr)
	logger.Info("Swagger UI available at http://%s/swagger/index.html", addr)

	if err := router.Run(addr); err != nil {
		logger.Fatal("Failed to start server: %v", err)
	}
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

