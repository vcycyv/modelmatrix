.PHONY: build run test clean migrate swagger docker-build docker-run

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

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

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

