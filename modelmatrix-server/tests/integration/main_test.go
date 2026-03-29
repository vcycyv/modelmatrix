package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"

	"modelmatrix-server/internal/infrastructure/auth"
	"modelmatrix-server/internal/infrastructure/compute"
	"modelmatrix-server/internal/infrastructure/db"
	"modelmatrix-server/internal/infrastructure/dbconnector"
	"modelmatrix-server/internal/infrastructure/fileservice"
	infraldap "modelmatrix-server/internal/infrastructure/ldap"
	"modelmatrix-server/migrations"

	dsApi "modelmatrix-server/internal/module/datasource/api"
	dsApp "modelmatrix-server/internal/module/datasource/application"
	dsDomain "modelmatrix-server/internal/module/datasource/domain"
	dsRepo "modelmatrix-server/internal/module/datasource/repository"

	buildApi "modelmatrix-server/internal/module/build/api"
	buildApp "modelmatrix-server/internal/module/build/application"
	buildDomain "modelmatrix-server/internal/module/build/domain"
	buildRepo "modelmatrix-server/internal/module/build/repository"

	invApi "modelmatrix-server/internal/module/inventory/api"
	invApp "modelmatrix-server/internal/module/inventory/application"
	invDomain "modelmatrix-server/internal/module/inventory/domain"
	invRepo "modelmatrix-server/internal/module/inventory/repository"

	folderApi "modelmatrix-server/internal/module/folder/api"
	folderApp "modelmatrix-server/internal/module/folder/application"
	folderRepo "modelmatrix-server/internal/module/folder/repository"

	searchApi "modelmatrix-server/internal/module/search/api"
	searchApp "modelmatrix-server/internal/module/search/application"
	searchRepoPkg "modelmatrix-server/internal/module/search/repository"

	"modelmatrix-server/pkg/config"
	"modelmatrix-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	testNetworkName = "modelmatrix-test-net"
	ldapBaseDN      = "dc=example,dc=org"
	ldapAdminDN     = "cn=admin,dc=example,dc=org"
	ldapAdminPass   = "admin"
)


// TestMain starts all infrastructure once and runs all suites.
// If TEST_DB_HOST is set, it uses existing services (dev mode).
// Otherwise it auto-starts containers (CI mode).
func TestMain(m *testing.M) {
	ctx := context.Background()

	var cleanups []func()

	// --- Auto-start containers when env vars are absent ---
	if os.Getenv("TEST_DB_HOST") == "" {
		cleanup := startContainers(ctx)
		cleanups = append(cleanups, cleanup)
	}

	// --- Build config from env vars (populated by containers or by dev .env.test) ---
	cfg := loadTestConfig()
	testConfig = cfg
	config.SetConfig(cfg)

	if err := logger.Init(cfg.Logging.Level, cfg.Logging.Format, "stdout", ""); err != nil {
		fmt.Fprintf(os.Stderr, "logger init: %v\n", err)
		os.Exit(1)
	}

	// --- Database ---
	database, err := db.Init(&cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db init: %v\n", err)
		os.Exit(1)
	}
	testDB = database
	if err := migrations.Migrate(database); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
	if err := migrations.CreateIndexes(database); err != nil {
		fmt.Fprintf(os.Stderr, "create indexes (warn): %v\n", err)
	}

	// --- Build the Gin router ---
	router, serverURL := buildTestRouter(ctx, cfg, database)
	testRouter = router
	testServerURL = serverURL

	// --- Run all suites ---
	code := m.Run()

	// --- Teardown ---
	if testServer != nil {
		testServer.Close()
	}
	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanups[i]()
	}

	os.Exit(code)
}

// startContainers auto-starts PostgreSQL, MinIO, LDAP and Compute via testcontainers.
// It sets the corresponding TEST_* environment variables so loadTestConfig picks them up.
func startContainers(ctx context.Context) func() {
	net, err := network.New(ctx,
		network.WithDriver("bridge"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create docker network: %v\n", err)
		os.Exit(1)
	}

	var containers []tc.Container

	// --- PostgreSQL ---
	pgC := mustStartContainer(ctx, tc.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "dayang",
			"POSTGRES_DB":       "modelmatrixtest",
		},
		Networks:       []string{net.Name},
		NetworkAliases: map[string][]string{net.Name: {"postgres"}},
		WaitingFor:     wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60 * time.Second),
	})
	containers = append(containers, pgC)
	pgHost := mustHost(ctx, pgC)
	pgPort := mustMappedPort(ctx, pgC, "5432/tcp")
	os.Setenv("TEST_DB_HOST", pgHost)
	os.Setenv("TEST_DB_PORT", pgPort)
	os.Setenv("TEST_DB_USER", "postgres")
	os.Setenv("TEST_DB_PASSWORD", "dayang")
	os.Setenv("TEST_DB_NAME", "modelmatrixtest")

	// --- MinIO ---
	minioC := mustStartContainer(ctx, tc.ContainerRequest{
		Image:        "minio/minio:latest",
		Cmd:          []string{"server", "/data", "--console-address", ":9001"},
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadmin",
			"MINIO_ROOT_PASSWORD": "minioadmin123",
		},
		Networks:       []string{net.Name},
		NetworkAliases: map[string][]string{net.Name: {"minio"}},
		WaitingFor:     wait.ForHTTP("/minio/health/live").WithPort("9000/tcp").WithStartupTimeout(60 * time.Second),
	})
	containers = append(containers, minioC)
	minioHost := mustHost(ctx, minioC)
	minioPort := mustMappedPort(ctx, minioC, "9000/tcp")
	os.Setenv("TEST_MINIO_ENDPOINT", minioHost+":"+minioPort)
	os.Setenv("TEST_MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("TEST_MINIO_SECRET_KEY", "minioadmin123")
	os.Setenv("TEST_MINIO_BUCKET", "modelmatrixtest")

	// --- LDAP ---
	ldapC := mustStartContainer(ctx, tc.ContainerRequest{
		Image:        "osixia/openldap:latest",
		ExposedPorts: []string{"389/tcp"},
		Env: map[string]string{
			"LDAP_ORGANISATION":      "Example Org",
			"LDAP_DOMAIN":            "example.org",
			"LDAP_ADMIN_PASSWORD":    ldapAdminPass,
			"LDAP_CONFIG_PASSWORD":   "configpassword",
			"LDAP_RFC2307BIS_SCHEMA": "true",
			"LDAP_REMOVE_CONFIG_AFTER_SETUP": "false",
		},
		Networks:       []string{net.Name},
		NetworkAliases: map[string][]string{net.Name: {"ldap"}},
		WaitingFor:     wait.ForLog("slapd starting").WithStartupTimeout(60 * time.Second),
	})
	containers = append(containers, ldapC)
	ldapHost := mustHost(ctx, ldapC)
	ldapPort := mustMappedPort(ctx, ldapC, "389/tcp")
	os.Setenv("TEST_LDAP_HOST", ldapHost)
	os.Setenv("TEST_LDAP_PORT", ldapPort)
	os.Setenv("TEST_LDAP_BASE_DN", ldapBaseDN)
	os.Setenv("TEST_LDAP_BIND_DN", ldapAdminDN)
	os.Setenv("TEST_LDAP_BIND_PASSWORD", ldapAdminPass)

	// Bootstrap LDAP test users
	bootstrapLDAP(ctx, ldapC)

	// --- Compute service ---
	computeURL := startComputeContainer(ctx, net.Name, containers)
	if computeURL != "" {
		os.Setenv("TEST_COMPUTE_URL", computeURL)
	}

	return func() {
		for i := len(containers) - 1; i >= 0; i-- {
			_ = containers[i].Terminate(ctx)
		}
		_ = net.Remove(ctx)
	}
}

// startComputeContainer builds and starts the compute service container.
// Returns the compute service URL or empty string if unavailable.
func startComputeContainer(ctx context.Context, netName string, containers []tc.Container) string {
	// Determine the compute service Dockerfile context path
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	computeCtx := filepath.Join(repoRoot, "modelmatrix-compute")

	if _, err := os.Stat(filepath.Join(computeCtx, "Dockerfile")); err != nil {
		fmt.Fprintf(os.Stderr, "compute Dockerfile not found at %s, skipping compute container\n", computeCtx)
		return ""
	}

	// Start a temporary listener to get an available port for the test server callback
	// The actual test server is started later; we use host.docker.internal for callbacks
	computeC, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			FromDockerfile: tc.FromDockerfile{
				Context:    computeCtx,
				Dockerfile: "Dockerfile",
				KeepImage:  true,
			},
			ExposedPorts: []string{"8081/tcp"},
			Env: map[string]string{
				"MINIO_ENDPOINT":  "minio:9000",
				"MINIO_ACCESS_KEY": "minioadmin",
				"MINIO_SECRET_KEY": "minioadmin123",
				"MINIO_BUCKET":    "modelmatrixtest",
				"MINIO_USE_SSL":   "false",
				"COMPUTE_HOST":    "0.0.0.0",
				"COMPUTE_PORT":    "8081",
			},
			Networks:       []string{netName},
			NetworkAliases: map[string][]string{netName: {"compute"}},
			WaitingFor:     wait.ForHTTP("/compute/health").WithPort("8081/tcp").WithStartupTimeout(180 * time.Second),
			HostConfigModifier: func(hc *container.HostConfig) {
				// Allow container to reach host via host.docker.internal
				hc.ExtraHosts = []string{"host.docker.internal:host-gateway"}
			},
		},
		Started: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "compute container start (non-fatal): %v\n", err)
		return ""
	}
	containers = append(containers, computeC)

	computeHost := mustHost(ctx, computeC)
	computePort := mustMappedPort(ctx, computeC, "8081/tcp")
	return "http://" + computeHost + ":" + computePort
}

// bootstrapLDAP adds test users to the running LDAP container.
func bootstrapLDAP(ctx context.Context, ldapC tc.Container) {
	// Give LDAP a moment to fully initialize
	time.Sleep(3 * time.Second)

	_, thisFile, _, _ := runtime.Caller(0)
	ldifPath := filepath.Join(filepath.Dir(thisFile), "..", "testdata", "test-users.ldif")

	if err := ldapC.CopyFileToContainer(ctx, ldifPath, "/tmp/test-users.ldif", 0644); err != nil {
		fmt.Fprintf(os.Stderr, "LDAP copy ldif (warn): %v\n", err)
		return
	}

	// Retry ldapadd a few times since slapd may still be starting
	for i := 0; i < 5; i++ {
		_, _, err := ldapC.Exec(ctx, []string{
			"ldapadd", "-x",
			"-D", ldapAdminDN,
			"-w", ldapAdminPass,
			"-f", "/tmp/test-users.ldif",
		})
		if err == nil {
			return
		}
		time.Sleep(2 * time.Second)
	}
	fmt.Fprintf(os.Stderr, "LDAP bootstrap failed after retries (warn)\n")
}

// buildTestRouter constructs the Gin router and starts the test HTTP server.
// When containers are active, the server binds to 0.0.0.0 so compute can reach it.
func buildTestRouter(ctx context.Context, cfg *config.Config, database *gorm.DB) (*gin.Engine, string) {
	ldapClient, err := infraldap.NewClient(&cfg.LDAP)
	if err != nil {
		fmt.Fprintf(os.Stderr, "LDAP client init: %v\n", err)
		os.Exit(1)
	}

	fileSvc, err := fileservice.NewFileService(&cfg.FileService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "file service init: %v\n", err)
		os.Exit(1)
	}

	tokenSvc := auth.NewTokenService(&cfg.JWT)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())
	api := router.Group("/api")
	authMiddleware := auth.Middleware(tokenSvc)

	// --- Datasource module ---
	dsDomainSvc := dsDomain.NewService()
	collectionRepo := dsRepo.NewCollectionRepository(database)
	datasourceRepo := dsRepo.NewDatasourceRepository(database)
	columnRepo := dsRepo.NewColumnRepository(database)
	externalDBConn := dbconnector.NewExternalDBConnector()
	collectionSvc := dsApp.NewCollectionService(collectionRepo, datasourceRepo, dsDomainSvc, fileSvc)
	datasourceSvc := dsApp.NewDatasourceService(database, datasourceRepo, collectionRepo, columnRepo, dsDomainSvc, fileSvc, externalDBConn)
	columnSvc := dsApp.NewColumnService(database, columnRepo, datasourceRepo, dsDomainSvc)

	dsApi.NewAuthController(ldapClient, tokenSvc).RegisterRoutes(api)
	dsApi.NewCollectionController(collectionSvc).RegisterRoutes(api, authMiddleware)
	dsApi.NewDatasourceController(datasourceSvc, columnSvc).RegisterRoutes(api, authMiddleware)

	// --- Inventory module ---
	invDomainSvc := invDomain.NewService()
	modelRepo := invRepo.NewModelRepository(database)
	modelSvc := invApp.NewModelService(modelRepo, invDomainSvc, fileSvc)

	// --- Version module ---
	versionRepo := invRepo.NewModelVersionRepository(database)
	versionSvc := invApp.NewModelVersionService(modelRepo, versionRepo, fileSvc.(fileservice.VersionStore))
	invApi.NewVersionController(versionSvc).RegisterRoutes(api, authMiddleware)

	// --- Folder module (service only — controller registered after build service) ---
	folderRepoImpl := folderRepo.NewFolderRepository(database)
	projectRepoImpl := folderRepo.NewProjectRepository(database)
	folderSvc := folderApp.NewFolderService(database, folderRepoImpl, projectRepoImpl)

	// --- Performance ---
	perfRepo := invRepo.NewPerformanceRepository(database)
	perfSvc := invApp.NewPerformanceService(perfRepo, modelRepo)
	invApi.NewPerformanceController(perfSvc).RegisterRoutes(api, authMiddleware)

	// --- Compute client ---
	var computeClient compute.Client
	computeURL := os.Getenv("TEST_COMPUTE_URL")
	if computeURL == "" {
		computeURL = cfg.Compute.ServiceURL
	}
	if computeURL != "" {
		cfg.Compute.ServiceURL = computeURL
		computeClient = compute.NewClient(&cfg.Compute)
	}

	// --- Build module ---
	buildDomainSvc := buildDomain.NewService()
	buildRepoImpl := buildRepo.NewBuildRepository(database)
	buildSvc := buildApp.NewBuildService(buildRepoImpl, buildDomainSvc, computeClient, datasourceSvc, modelSvc, versionSvc, folderSvc, perfSvc, cfg)
	buildApi.NewBuildController(buildSvc).RegisterRoutes(api, authMiddleware)

	// --- Folder controller (needs buildSvc + modelSvc for cascade deletes) ---
	folderApi.NewFolderController(folderSvc, buildSvc, modelSvc).RegisterRoutes(api, authMiddleware)

	// --- Model controller (with retrain) ---
	modelCtrl := invApi.NewModelControllerWithRetrain(modelSvc, buildSvc)
	modelCtrl.RegisterRoutes(api, authMiddleware)

	// --- Search ---
	searchRepository := searchRepoPkg.NewGormSearchRepository(database)
	searchService := searchApp.NewSearchService(searchRepository)
	searchApi.NewSearchController(searchService).RegisterRoutes(api, authMiddleware)

	// --- Start HTTP server ---
	// Bind to 0.0.0.0 so Docker containers can reach the callback endpoint.
	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}
	srv := httptest.NewUnstartedServer(router)
	srv.Listener = ln
	srv.Start()
	testServer = srv

	port := ln.Addr().(*net.TCPAddr).Port
	// Callback URL must be reachable from inside Docker containers
	cfg.Server.BaseURL = fmt.Sprintf("http://host.docker.internal:%d", port)
	// Server URL for test clients (always localhost)
	serverURL := fmt.Sprintf("http://localhost:%d", port)

	return router, serverURL
}

// waitForComputeHealth polls the compute service health endpoint until ready or timeout.
func waitForComputeHealth(computeURL string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(computeURL + "/compute/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return true
		}
		time.Sleep(2 * time.Second)
	}
	return false
}

// mustStartContainer starts a container or exits.
func mustStartContainer(ctx context.Context, req tc.ContainerRequest) tc.Container {
	c, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "start container %s: %v\n", req.Image, err)
		os.Exit(1)
	}
	return c
}

func mustHost(ctx context.Context, c tc.Container) string {
	h, err := c.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "container host: %v\n", err)
		os.Exit(1)
	}
	return h
}

func mustMappedPort(ctx context.Context, c tc.Container, port string) string {
	p, err := c.MappedPort(ctx, nat.Port(port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "mapped port %s: %v\n", port, err)
		os.Exit(1)
	}
	return p.Port()
}

// loadTestConfig builds a *config.Config from TEST_* environment variables.
func loadTestConfig() *config.Config {
	dbPort, _ := strconv.Atoi(getEnv("TEST_DB_PORT", "5432"))
	ldapPort, _ := strconv.Atoi(getEnv("TEST_LDAP_PORT", "3890"))

	return &config.Config{
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
			BaseDN:       getEnv("TEST_LDAP_BASE_DN", ldapBaseDN),
			BindDN:       getEnv("TEST_LDAP_BIND_DN", ldapAdminDN),
			BindPassword: getEnv("TEST_LDAP_BIND_PASSWORD", ldapAdminPass),
			UserFilter:   "(uid=%s)",
			GroupFilter:  "(|(member=%s)(uniqueMember=%s))",
			UseTLS:       false,
		},
		FileService: config.FileServiceConfig{
			MinioEndpoint:  getEnv("TEST_MINIO_ENDPOINT", "localhost:9000"),
			MinioAccessKey: getEnv("TEST_MINIO_ACCESS_KEY", "minioadmin"),
			MinioSecretKey: getEnv("TEST_MINIO_SECRET_KEY", "minioadmin123"),
			MinioBucket:    getEnv("TEST_MINIO_BUCKET", "modelmatrixtest"),
			MinioUseSSL:    false,
		},
		JWT: config.JWTConfig{
			Secret:          getEnv("TEST_JWT_SECRET", "test-secret-key-change-in-production"),
			ExpirationHours: 24,
		},
		Logging: config.LoggingConfig{
			Level:  getEnv("TEST_LOG_LEVEL", "warn"),
			Format: "json",
			Output: "stdout",
		},
		Compute: config.ComputeConfig{
			ServiceURL: getEnv("TEST_COMPUTE_URL", ""),
			Timeout:    120,
		},
	}
}
