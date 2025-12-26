# Prompt: Generate Complete `modelmatrix_backend` Project (Golang + Gin + GORM v2 + PostgreSQL)
## Project Overview
I need to build the backend for a project named "ModelMatrix" (separated into `modelmatrix_backend` and `modelmatrix_ui`; this prompt focuses **exclusively on `modelmatrix_backend`**). The backend must implement 3 core business modules with a strict, layered DDD-inspired architecture, and integrate with LDAP (running in WSL2 via `ldap/docker-compose.yml`) for authentication/authorization.

### Core Business Modules
1. **Datasource Management**  
   - Full CRUD for datasources (support PostgreSQL, MySQL, CSV, and Apache Parquet formats).  
   - Full CRUD for **Datasource Collections** (logical groups for datasources).  
   - Data column role management (assign `target`/`input`/`output`/`ignore` roles to datasource columns).  
   - Persist datasource files via an `infrastructure/fileservice` layer (infrastructure-level file handling).
2. **Build Machine Learning Models**  
   - Orchestrate ML model training workflows.  
   - Manage training parameter configurations (skeleton code with consistent layered structure).
3. **Manage Models**  
   - Model versioning, activation, deactivation.  
   - Enforce business rules (e.g., "active models cannot be deleted"; skeleton code with consistent layered structure).

### Tech Stack Requirements
- **Core Language**: Golang (latest stable version, e.g., 1.22+)  
- **Web Framework**: Gin (HTTP web framework, with `gin-swagger` for OpenAPI docs)  
- **Persistence**: GORM v2 (ORM tool for PostgreSQL interactions)  
- **Database**: PostgreSQL (connection details: host=localhost, port=5432, username=postgres, password=dayang, dbname=modelmatrix)  
- **Infrastructure**: 
  - File storage: `fileservice` (infrastructure layer for datasource file persistence).  
  - Authentication/Authorization: LDAP (config sourced from `ldap/docker-compose.yml` in the project root).  
- **Mandatory Deliverables (Non-Negotiable)**:  
  1. PostgreSQL database initialization SQL script.  
  2. GORM migration scripts for all 3 modules (with tables, indexes, and constraints).  
  3. Complete folder structure with strict layer boundaries.  
  4. Dependency injection (DI) for service/repository instantiation (no hardcoded dependencies).  
  5. Unified API response format (consistent across all endpoints).  
  6. Structured logging (audit + system events) and basic error handling.  
  7. Swagger/OpenAPI documentation for all API endpoints.  
  8. Health check endpoint for PostgreSQL + LDAP connectivity.

### Mandatory Layered Architecture (Per Module)
Follow this strict layer flow (with a clear boundary between business logic and infrastructure):
Controller / API Layer↓ (accepts DTO, returns DTO)
Application Service Layer↓ (converts DTO ↔ Domain Entity, calls Domain Service)
Domain Service Layer↓ (only operates on Domain Entity, core business logic)
---------------------------------- ← Boundary (Business Logic ↔ Infrastructure)Repository Layer (Infrastructure)↕ (converts Domain Entity ↔ GORM Model)
GORM Model Layer (Infrastructure)↓ (interacts with Database, has GORM tags)Database (PostgreSQL)

Additional Infrastructure Layers (Parallel to Repository)
Fileservice (Infrastructure) ← For datasource file persistence
LDAP Client (Infrastructure) ← For auth/authorization


### Key Layer Responsibilities & Rules
#### 1. Layer Boundaries & Dependencies
- **Controller/API Layer**  
  - Only handles HTTP requests/responses, binds DTOs to request bodies/params, and calls Application Services (**never call Repositories/Fileservice directly**).  
  - Enforces JWT authentication (validates Bearer tokens in `Authorization` header for protected endpoints).  
  - Returns responses via the unified API format (no business logic here).

- **Application Service Layer**  
  - Knows DTOs, Repositories, and Fileservice (orchestrates workflow: load Domain Entity → call Domain Service → save via Repository/Fileservice).  
  - Manages database transactions (for atomic operations, e.g., create collection + associate datasource).  
  - Converts DTOs ↔ Domain Entities (mapping logic for presentation ↔ business layers).  
  - Minimal business logic (only orchestration; no core rules).  
  - Implements RBAC checks (uses JWT claims to verify LDAP group permissions).

- **Domain Service Layer**  
  - **Does NOT know DTOs, Repositories, Fileservice, LDAP, or Database** (pure business logic).  
  - Contains **core business rules** (e.g., "datasource name must be unique per collection", "only 1 target column per datasource", "active models cannot be deleted").  
  - Only operates on Domain Entities (no infrastructure code/GORM tags).  
  - High reusability (works with HTTP/gRPC/CLI entry points).

- **Domain Entity**  
  - Pure business object (no GORM tags, no infrastructure-related code).  
  - May contain simple business behavior (e.g., `CalculateModelAccuracy()`, `ValidateCurrencyMismatch()`).  
  - Represents business meaning (not storage structure, e.g., `Datasource`, `Collection`, `Model`).

- **Repository Layer (Infrastructure)**  
  - **Only knows Domain Entities and GORM Models** (no DTOs/business logic).  
  - Handles data access (CRUD operations for PostgreSQL) and converts Domain Entities ↔ GORM Models (mapping logic lives here).  
  - Implements repository interfaces defined in the Application Service layer (for DI compatibility).  


- **GORM Model Layer (Infrastructure)**  
  - Infrastructure object with GORM tags (for PostgreSQL mapping: column names, types, constraints, indexes).  
  - Optimized for storage (e.g., `collection_id` foreign key for datasources).  
  - No business logic (pure data structure with `CreatedAt`/`UpdatedAt`/`DeletedAt` for audit).

- **Fileservice (Infrastructure)**  
  - Dedicated to datasource file persistence (local/S3 paths).  
  - Exposes interfaces for saving/loading/deleting datasource files (called by Application Services).  
  - Implements file validation (e.g., check Parquet format validity) and path management.

- **LDAP Client (Infrastructure)**  
  - Dedicated to LDAP interactions (bind, user/group search).  
  - Uses config from `ldap/docker-compose.yml` (host, port, base DN, bind credentials).  
  - Exposes interfaces for user authentication and group retrieval (called by Auth Controller/Application Service).

#### 2. Critical Distinctions
- **Domain Entity ≠ GORM Model**: Domain Entities have no GORM tags; GORM Models have only storage-related GORM tags. All conversion happens in the Repository layer.  
- **Application Service calls Repositories/Fileservice**; Domain Service does NOT call any infrastructure layers (Repositories/Fileservice/LDAP).  


#### 3. Aspect Comparison (Must Adhere To)
| Aspect           | Application Service | Domain Service |
| ---------------- | ------------------- | -------------- |
| Knows DTO        | ✅ Yes               | ❌ No           |
| Knows DB/LDAP/Fileservice | ❌ No                | ❌ No           |
| Knows Repository/Fileservice Interfaces | ✅ Yes               | ❌ No           |
| Business Rules   | ❌ Minimal (orchestration only) | ✅ Core (all business constraints) |
| Transactions     | ✅ Yes (database transactions) | ❌ No           |
| Reusable         | Medium (HTTP-bound)  | High (agnostic to entry point) |

### Deliverables Required
#### 1. Complete `modelmatrix_backend` Folder Structure (Tree View)
Organize by 3 business modules + shared infrastructure/utilities. Example structure (AI must expand with all layers):
modelmatrix_backend/
├── cmd/                          # Application entry point
│   └── api/
│       └── main.go               # Initialize config, logger, GORM, LDAP, DI, Gin router
├── conf/                         # Environment-specific configs
│   ├── dev.yaml                  # Development config (loads via ENV=dev)
│   ├── prod.yaml                 # Production config (loads via ENV=prod)
│   ├── database.yaml             # PostgreSQL config (imported by dev/prod.yaml)
│   └── ldap.yaml                 # LDAP config (references env vars for bind password)
├── internal/                     # Private application code
│   ├── module/
│   │   ├── datasource/           # Full working code (reference for other modules)
│   │   │   ├── api/              # Controller layer (Gin handlers)
│   │   │   ├── dto/              # Request/Response DTOs (moved from model for clarity)
│   │   │   ├── application/      # Application Service layer (interface + impl)
│   │   │   ├── domain/           # Domain Service + Domain Entities
│   │   │   ├── repository/       # Repository layer (interface + GORM impl)
│   │   │   └── model/            # GORM Models (infrastructure)
│   │   ├── modelbuild/           # Skeleton code (consistent structure)
│   │   └── modelmanage/          # Skeleton code (consistent structure)
│   └── infrastructure/           # Infrastructure layers
│       ├── fileservice/          # Datasource file persistence (interface + impl)
│       ├── ldap/                 # LDAP client (interface + impl, uses ldap/docker-compose.yml config)
│       └── db/                   # GORM initialization (shared)
├── pkg/                          # Shared public utilities
│   ├── config/                   # Env config loader (supports dev/prod)
│   ├── logger/                   # Structured logging (audit + system events)
│   ├── response/                 # Unified API response format
│   └── swagger/                  # Swagger/OpenAPI docs setup
├── migrations/                   # GORM migration scripts (all 3 modules)
├── scripts/                      # Database initialization
│   └── init_db.sql               # Create PostgreSQL database if not exists
├── ldap/                         # LDAP config (docker-compose.yml, project root)
│   └── docker-compose.yml        # LLDAP config (referenced for LDAP client setup)
├── go.mod                        # Golang module dependencies
└── go.sum                        # Dependency locks

#### 2. Code for Each Layer (Per Module)
- **Datasource Management Module (Full Working Code)**:  
  Implement all features (Collection CRUD, Datasource CRUD, Parquet validation, column role management) with:  
  - Controller: Gin handlers for `/api/collections` and `/api/datasources` (JWT protected).  
  - DTO: `CreateCollectionRequest`, `DatasourceResponse`, `ColumnRoleRequest` (with validation tags).  
  - Application Service: Orchestrate workflow (e.g., create collection → validate via Domain Service → save via Repository).  
  - Domain Service: Enforce rules (e.g., "collection name unique", "1 target column max").  
  - Domain Entity: `Collection`, `Datasource`, `Column` (pure business structs, no GORM tags).  
  - Repository: Convert Domain Entity ↔ GORM Model, implement CRUD with indexes.  
  - GORM Model: `CollectionModel`, `DatasourceModel`, `ColumnModel` (with GORM tags/constraints).  
  - Fileservice: Save Parquet/CSV files to local/S3 paths (called by Application Service).  
- **Model Build & Model Manage Modules (Skeleton Code)**:  
  Mirror Datasource module structure with placeholder comments for core logic (e.g., `// Implement model training orchestration here`).

#### 3. Support Files (Full Working Code)
- **`conf/dev.yaml`/`conf/prod.yaml`**: Environment configs (load via `ENV` environment variable; prod uses env vars for secrets).  
- **`conf/database.yaml`**: PostgreSQL connection config (host=localhost, port=5432, username=postgres, password=dayang, dbname=modelmatrix).  
- **`conf/ldap.yaml`**: LDAP config (host=localhost, port=3890, base_dn=dc=modelmatrix,dc=local, bind_dn=uid=admin,ou=people,dc=modelmatrix,dc=local, bind_password=${LDAP_BIND_PASSWORD} (env var)).  
- **`scripts/init_db.sql`**: PostgreSQL initialization (create database if not exists).  
- **`migrations/`**: GORM scripts for `collections`, `datasources`, `datasource_columns`, `model_builds`, `model_versions` (with indexes/unique constraints).  
- **`cmd/api/main.go`**: Initialize all dependencies (config, logger, GORM, LDAP, Fileservice, DI), register Gin routes, and start server.  
- **`pkg/`**: Shared utilities (config loader, structured logger, unified response, swagger setup).  
- **`internal/infrastructure/fileservice/`**: File persistence (local/S3) with interface `FileService` and implementation `LocalFileService`.  
- **`internal/infrastructure/ldap/`**: LDAP client (uses `ldap/docker-compose.yml` config) with interface `LDAPClient` (authenticate user, get groups).  
- **`go.mod`/`go.sum`**: Dependencies (gin, gorm, postgres driver, gin-swagger, jwt-go, parquet-go, ldap).

#### 4. Additional Requirements (Non-Negotiable)
- **Unified API Response Format**:  
  Success: `{"code": 200, "msg": "success", "data": {...}}`  
  Error: `{"code": >200, "msg": "error detail", "data": null}`  
- **Structured Logging**: Log user actions (audit) and system events with fields: `user`, `action`, `resource_type`, `resource_id`, `status`, `error`.  
- **Health Check**: `/api/health` endpoint (returns 200 OK if PostgreSQL/LDAP are healthy; 503 otherwise).  
- **Secret Management**: Production LDAP bind password is referenced via environment variable (no plaintext in YAML).  
- **Swagger/OpenAPI**: All endpoints are documented with `gin-swagger` (access via `/swagger/index.html`).  
- **Unique Constraints & Indexes**:  
  - Unique: `collections.name`, `datasources.name` (per collection), `datasource_columns.datasource_id + datasource_columns.column_name`.  
  - Indexes: `datasources.collection_id`, `datasource_columns.datasource_id`, `models.status`.  
- **RBAC Enforcement**: Use LDAP groups (`modelmatrix_admin`/`modelmatrix_editor`/`modelmatrix_viewer`) to restrict endpoint access (e.g., viewers can’t delete collections).

### Output Format
1. First, list the **complete `modelmatrix_backend` folder structure (tree view)** (expanded to all files).  
2. Then, provide **code for each file** with full file paths as headings (e.g., `### modelmatrix_backend/cmd/api/main.go`).  
3. Add **detailed explanations** for key files (layer interactions, DTO ↔ Domain Entity ↔ GORM Model conversion, LDAP/JWT flow).  
4. Ensure all code is **runnable with minor local adjustments** (e.g., set `LDAP_BIND_PASSWORD` env var, create `lldap_data` folder).  
5. Highlight **critical implementation details** (e.g., Fileservice usage, LDAP config sourcing from `ldap/docker-compose.yml`).