.PHONY: build run test test-unit test-integration test-integration-coverage test-frontend test-e2e clean migrate swagger docker-build docker-run

# Build the application
build:
	go build -o bin/modelmatrix-api cmd/api/main.go
	go build -o bin/modelmatrix-migrate cmd/migrate/main.go

# Run the application
run:
	go run cmd/api/main.go

# Run migrations
migrate:
	go run cmd/migrate/main.go

# Drop and recreate database
migrate-fresh:
	go run cmd/migrate/main.go -drop

# Run all Go tests (unit)
test:
	go test -v ./internal/...

# Run Go unit tests only (fast, no Docker) — domain, application, and controller layers
test-unit:
	go test -v \
	  ./internal/module/inventory/domain/... \
	  ./internal/module/inventory/application/... \
	  ./internal/module/inventory/api/... \
	  ./internal/module/build/domain/... \
	  ./internal/module/build/application/... \
	  ./internal/module/build/api/... \
	  ./internal/module/datasource/domain/... \
	  ./internal/module/datasource/application/... \
	  ./internal/module/datasource/api/...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html

# Run integration tests (starts containers automatically via testcontainers)
test-integration:
	go test -v -timeout 300s ./tests/integration/...

# Run frontend unit tests
test-frontend:
	cd web && npm run test

# Run E2E tests (requires running app + E2E_BASE_URL)
test-e2e:
	cd web && npm run test:e2e

# Run integration tests with coverage
test-integration-coverage:
	go test -v -coverprofile=coverage-integration.out ./tests/integration/...
	go tool cover -html=coverage-integration.out -o coverage-integration.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html coverage-integration.out coverage-integration.html

# Generate Swagger documentation
swagger:
	swag init -g cmd/api/main.go -o docs

# Install dependencies
deps:
	go mod download
	go mod tidy

# Verify dependencies
verify:
	go mod verify

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Start LDAP server
ldap-up:
	cd ../ldap && docker-compose up -d

# Stop LDAP server
ldap-down:
	cd ../ldap && docker-compose down

# Check LDAP status
ldap-status:
	cd ../ldap && docker-compose ps

# Start MinIO server
minio-up:
	docker run -d --name minio \
		-p 9000:9000 -p 9001:9001 \
		-e MINIO_ROOT_USER=minioadmin \
		-e MINIO_ROOT_PASSWORD=minioadmin123 \
		minio/minio server /data --console-address ":9001"

# Stop MinIO server
minio-down:
	docker stop minio && docker rm minio

# Check MinIO status
minio-status:
	docker ps -f name=minio

# Development mode (with hot reload using air)
dev:
	air

# Install development tools
tools:
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  run           - Run the application"
	@echo "  migrate       - Run database migrations"
	@echo "  migrate-fresh - Drop and recreate database"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  test-integration - Run integration tests"
	@echo "  test-integration-coverage - Run integration tests with coverage"
	@echo "  clean         - Clean build artifacts"
	@echo "  swagger       - Generate Swagger documentation"
	@echo "  deps          - Download dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  ldap-up       - Start LDAP server"
	@echo "  ldap-down     - Stop LDAP server"
	@echo "  minio-up      - Start MinIO server"
	@echo "  minio-down    - Stop MinIO server"
	@echo "  dev           - Run with hot reload"
	@echo "  tools         - Install development tools"

