# ModelMatrix Server

A full-stack ML model management platform with a Go backend API and React web UI. Built with Gin framework, GORM ORM, PostgreSQL, React 19, and Tailwind CSS.

## Features

- **Datasource Management**: CRUD operations for datasources and collections, file uploads (CSV/Parquet), column role management
- **Model Building**: ML model training orchestration with parameter configuration
- **Model Management**: Model versioning, activation/deactivation, lifecycle management
- **Authentication**: LDAP-based authentication with JWT tokens
- **Authorization**: Role-based access control (RBAC) with admin/editor/viewer groups

## Architecture

The project follows a DDD-inspired layered architecture:

```
Controller/API Layer
    ↓ (accepts DTO, returns DTO)
Application Service Layer
    ↓ (converts DTO ↔ Domain Entity)
Domain Service Layer
    ↓ (core business logic on Domain Entities)
Repository Layer (Infrastructure)
    ↕ (converts Domain Entity ↔ GORM Model)
GORM Model Layer
    ↓
Database (PostgreSQL)
```

## Project Structure

```
modelmatrix-server/
├── cmd/
│   ├── api/
│   │   └── main.go              # Application entry point
│   └── migrate/
│       └── main.go              # Migration CLI
├── conf/
│   ├── dev.yaml                 # Development configuration
│   └── prod.yaml                # Production configuration
├── internal/
│   ├── infrastructure/
│   │   ├── auth/                # JWT & middleware
│   │   ├── db/                  # GORM initialization
│   │   ├── fileservice/         # File storage (MinIO/S3)
│   │   ├── folderservice/       # Folder & project hierarchy
│   │   └── ldap/                # LDAP client
│   └── module/
│       ├── datasource/          # Datasource management
│       │   ├── api/             # Controllers
│       │   ├── application/     # Application services
│       │   ├── domain/          # Domain entities & services
│       │   ├── dto/             # Data transfer objects
│       │   ├── model/           # GORM models
│       │   └── repository/      # Data access layer
│       ├── build/          # Model training
│       └── inventory/         # Model management
├── migrations/                  # Database migrations
├── pkg/
│   ├── config/                  # Configuration loader
│   ├── logger/                  # Structured logging
│   ├── response/                # Unified API responses
│   └── swagger/                 # Swagger setup
├── scripts/
│   └── init_db.sql             # Database initialization
├── web/                         # React web UI
│   ├── src/                     # React source code
│   ├── public/                  # Static assets
│   ├── package.json             # Node.js dependencies
│   ├── vite.config.ts           # Vite build configuration
│   └── tailwind.config.js       # Tailwind CSS configuration
├── dist/                        # Built UI assets (production)
├── go.mod
└── go.sum
```

## Prerequisites

- Go 1.23+
- Node.js 20+ (for web UI)
- PostgreSQL 14+
- MinIO (S3-compatible object storage)
- LDAP server (LLDAP recommended)
- Docker (for LDAP and MinIO)

## Web UI

The web UI is built with React 19, Vite, and Tailwind CSS. It's served by the Go server in production.

### Development Mode

Run the Vite dev server with hot reload (proxies API to Go backend):

```bash
# Terminal 1: Start Go backend
cd modelmatrix-server
go run cmd/api/main.go

# Terminal 2: Start Vite dev server
cd modelmatrix-server/web
npm install
npm run dev
```

- **UI**: http://localhost:3000 (Vite dev server)
- **API**: http://localhost:8080 (Go server)
- API calls from the UI are automatically proxied to the Go backend

### Production Mode

Build the UI and serve from the Go server:

```bash
# 1. Build the React UI
cd modelmatrix-server/web
npm install
npm run build    # Outputs to ../dist

# 2. Start the Go server (serves both API and UI)
cd modelmatrix-server
go build -o modelmatrix-server ./cmd/api
./modelmatrix-server
```

Access the application at http://localhost:8080 - the Go server serves:
- `/api/*` - REST API endpoints
- `/swagger/*` - Swagger documentation
- `/*` - React SPA (from `dist/` folder)

### Docker Production Build

```dockerfile
# Build stage for UI
FROM node:20-alpine AS ui-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Build stage for Go
FROM golang:1.23-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /app/dist ./dist
RUN go build -o modelmatrix-server ./cmd/api

# Final image
FROM alpine:latest
WORKDIR /app
COPY --from=go-builder /app/modelmatrix-server .
COPY --from=go-builder /app/dist ./dist
COPY --from=go-builder /app/conf ./conf
EXPOSE 8080
CMD ["./modelmatrix-server"]
```

## Quick Start

### 1. Start Infrastructure Services

**LDAP Server:**
```bash
cd ldap
docker-compose up -d
```

When encountering the error:
```bash
ERROR: for lldap  Cannot create container for service lldap: Conflict.
```
Simply start the existing container:
```bash
docker stop lldap
docker rm lldap
```

Access LLDAP admin UI at http://localhost:17170 (admin/dayangdayang)

**MinIO Server:**
```bash
docker run -d --name minio \
  -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin123 \
  minio/minio server /data --console-address ":9001"
```

When encountering the error:
``` bash
docker: Error response from daemon: Conflict. The container name "/minio" is already in use by container..
```
Simply start the existing container:
``` bash
docker start minio
```

Access MinIO Console at http://localhost:9001 (minioadmin/minioadmin123)

### 2. Create PostgreSQL Database

```bash
psql -U postgres -c "CREATE DATABASE modelmatrix;"
```

Or run the init script:

```bash
psql -U postgres -f scripts/init_db.sql
```

### 3. Configure Environment

For development, the default `conf/dev.yaml` should work out of the box.

For production, set environment variables:

```bash
export ENV=prod
export DB_HOST=your-db-host
export DB_PORT=5432
export DB_USERNAME=postgres
export DB_PASSWORD=your-password
export DB_NAME=modelmatrix
export LDAP_HOST=your-ldap-host
export LDAP_PORT=3890
export LDAP_BIND_PASSWORD=your-ldap-password
export JWT_SECRET=your-jwt-secret
export MINIO_ENDPOINT=your-minio-host:9000
export MINIO_ACCESS_KEY=your-access-key
export MINIO_SECRET_KEY=your-secret-key
export MINIO_BUCKET=modelmatrix
export MINIO_USE_SSL=true
```

### 4. Run Migrations

```bash
cd modelmatrix-server
go run cmd/migrate/main.go
```

### 5. Start the Server

```bash
go run cmd/api/main.go
```

The API will be available at http://localhost:8080

## API Documentation

Swagger UI is available at http://localhost:8080/swagger/index.html

### Key Endpoints

#### Authentication

- `POST /api/auth/login` - Login with LDAP credentials
- `POST /api/auth/refresh` - Refresh JWT token

#### Collections

- `GET /api/collections` - List collections
- `POST /api/collections` - Create collection
- `GET /api/collections/:id` - Get collection
- `PUT /api/collections/:id` - Update collection
- `DELETE /api/collections/:id` - Delete collection (admin only)

#### Datasources

- `GET /api/datasources` - List datasources
- `POST /api/datasources` - Create datasource
- `GET /api/datasources/:id` - Get datasource with columns
- `PUT /api/datasources/:id` - Update datasource
- `DELETE /api/datasources/:id` - Delete datasource (admin only)
- `POST /api/datasources/:id/upload` - Upload file
- `GET /api/datasources/:id/columns` - Get columns
- `PUT /api/datasources/:id/columns/:column_id/role` - Update column role

#### Model Builds

- `GET /api/builds` - List model builds
- `POST /api/builds` - Create model build
- `GET /api/builds/:id` - Get model build
- `POST /api/builds/:id/start` - Start training
- `POST /api/builds/:id/cancel` - Cancel training

#### Models

- `GET /api/models` - List models
- `POST /api/models` - Create model
- `GET /api/models/:id` - Get model with versions
- `POST /api/models/:id/activate` - Activate model
- `POST /api/models/:id/deactivate` - Deactivate model
- `GET /api/models/:id/versions` - Get model versions

#### Health Check

- `GET /api/health` - Check PostgreSQL and LDAP connectivity

## RBAC Groups

Create these groups in LDAP for access control:

- `modelmatrix_admin` - Full access (create, read, update, delete)
- `modelmatrix_editor` - Create, read, update (no delete)
- `modelmatrix_viewer` - Read only

## Development

### Generate Swagger Docs

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/api/main.go
```

### Run Tests

```bash
go test ./...
```

### Build

```bash
go build -o modelmatrix-api cmd/api/main.go
```

## Configuration

### Database (`conf/dev.yaml`)

```yaml
database:
  host: localhost
  port: 5432
  username: postgres
  password: dayang
  dbname: modelmatrix
  sslmode: disable
```

### LDAP (`conf/dev.yaml`)

```yaml
ldap:
  host: localhost
  port: 3890
  base_dn: "dc=modelmatrix,dc=local"
  bind_dn: "uid=admin,ou=people,dc=modelmatrix,dc=local"
  bind_password: "dayangdayang"
```

### JWT (`conf/dev.yaml`)

```yaml
jwt:
  secret: "your-secret-key"
  expiration_hours: 24
```

## License

Apache 2.0

