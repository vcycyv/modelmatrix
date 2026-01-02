# ModelMatrix Compute Server Architecture

## Recommendation: Separate Project

Create the Python compute server as a **separate project** at the same level as `modelmatrix_backend`:

```
/home/chuyang/ws/modelmatrix/
├── modelmatrix_backend/          # Go API service
├── modelmatrix-compute/           # Python ML compute service (NEW)
└── requirement.md
```

## Why Separate Project?

### 1. **Different Technology Stacks**
- **Backend**: Go (Gin, GORM, PostgreSQL)
- **Compute**: Python (FastAPI/Flask, scikit-learn, pandas, etc.)

### 2. **Independent Deployment & Scaling**
- Backend: Stateless API server (horizontal scaling)
- Compute: Resource-intensive ML workloads (may need GPUs, more memory)
- Can deploy compute on different infrastructure (GPU nodes, cloud ML services)

### 3. **Different Dependencies**
- Backend: Go modules
- Compute: Python packages (numpy, pandas, scikit-learn, xgboost, etc.)
- Different virtual environments, Docker images

### 4. **Clear Service Boundaries**
- **Backend**: Orchestrates workflows, manages state, handles HTTP requests
- **Compute**: Pure computation, stateless model training

### 5. **Independent Versioning**
- Backend and compute can have different release cycles
- API versioning between services

## Integration Pattern

### Communication: REST API

The Go backend calls the Python compute service via HTTP REST API:

```
┌─────────────────┐         HTTP/REST         ┌──────────────────┐
│  Go Backend     │ ────────────────────────> │  Python Compute  │
│  (Orchestrator) │ <──────────────────────── │  (ML Engine)     │
└─────────────────┘      JSON Request/Response └──────────────────┘
```

### Example Flow:

1. **User Request**: `POST /api/builds` with datasource ID and parameters
2. **Go Backend**:
   - Validates request
   - Fetches datasource metadata from database
   - Gets file path from MinIO
   - Calls Python compute service: `POST /compute/train`
3. **Python Compute**:
   - Receives: datasource file path, algorithm, hyperparameters
   - Downloads data from MinIO (or receives data directly)
   - Trains model
   - Returns: model file path, metrics, training status
4. **Go Backend**:
   - Saves model metadata to database
   - Stores model file in MinIO
   - Returns build result to user

## Project Structure

### `modelmatrix-compute/` (Python Project)

```
modelmatrix-compute/
├── README.md
├── requirements.txt
├── pyproject.toml
├── Dockerfile
├── docker-compose.yml          # For local development
├── .env.example
├── src/
│   ├── main.py                # FastAPI/Flask app entry point
│   ├── api/
│   │   ├── routes.py          # API endpoints
│   │   └── schemas.py         # Pydantic models
│   ├── core/
│   │   ├── config.py           # Configuration
│   │   └── logger.py           # Logging
│   ├── services/
│   │   ├── model_trainer.py    # Model training logic
│   │   ├── data_loader.py      # Data loading from MinIO
│   │   └── model_saver.py      # Save trained models
│   └── algorithms/
│       ├── decision_tree.py
│       ├── random_forest.py
│       ├── xgboost.py
│       └── base.py             # Base algorithm interface
├── tests/
│   ├── test_api.py
│   └── test_trainer.py
└── scripts/
    └── setup.sh
```

## API Contract

### Python Compute Service Endpoints

```python
# POST /compute/train
{
  "datasource_id": "uuid",
  "file_path": "minio://bucket/path/to/data.parquet",
  "algorithm": "decision_tree",
  "hyperparameters": {
    "max_depth": 10,
    "min_samples_split": 2,
    "criterion": "gini"
  },
  "target_column": "target",
  "input_columns": ["feature1", "feature2"]
}

# Response
{
  "job_id": "uuid",
  "status": "training",
  "model_path": "minio://bucket/models/model.pkl",
  "metrics": {
    "accuracy": 0.95,
    "precision": 0.93,
    "recall": 0.92
  }
}

# GET /compute/status/{job_id}
{
  "job_id": "uuid",
  "status": "completed",
  "progress": 100,
  "model_path": "minio://bucket/models/model.pkl"
}
```

## Configuration

### Environment Variables

**Python Compute Service**:
```bash
# MinIO connection (to read data, save models)
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin123
MINIO_BUCKET=modelmatrix

# Backend API (for callbacks/status updates)
BACKEND_API_URL=http://localhost:8080
BACKEND_API_KEY=secret-key

# Compute service
COMPUTE_HOST=0.0.0.0
COMPUTE_PORT=8081
LOG_LEVEL=info
```

**Go Backend** (add to config):
```yaml
compute:
  service_url: "http://localhost:8081"
  timeout: 300  # 5 minutes
  api_key: "secret-key"
```

## Go Backend Integration

### Add to `modelmatrix_backend/internal/infrastructure/compute/`

```go
// compute/client.go
type Client interface {
    TrainModel(req *TrainRequest) (*TrainResponse, error)
    GetStatus(jobID string) (*JobStatus, error)
}

type TrainRequest struct {
    DatasourceID   string
    FilePath       string
    Algorithm      string
    Hyperparameters map[string]interface{}
    TargetColumn   string
    InputColumns   []string
}
```

### Update `modelbuild` module

The `ModelBuildService` will:
1. Validate build request
2. Get datasource info from repository
3. Call compute service to train model
4. Save model metadata and file path
5. Update build status

## Deployment Options

### Option 1: Docker Compose (Development)
```yaml
# docker-compose.yml
services:
  backend:
    build: ./modelmatrix_backend
    ports: ["8080:8080"]
  
  compute:
    build: ./modelmatrix-compute
    ports: ["8081:8081"]
    # GPU support if needed
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
```

### Option 2: Kubernetes (Production)
- Backend: Stateless deployment, auto-scaling
- Compute: Job-based pods (for training), can use GPU nodes

## Alternative: Monorepo Approach

If you prefer everything in one repo:

```
modelmatrix/
├── backend/              # Go service
├── compute/              # Python service
├── shared/               # Shared configs, scripts
│   ├── docker-compose.yml
│   └── scripts/
└── README.md
```

**Pros**: Easier local development, shared configs  
**Cons**: Mixed languages, harder to scale independently

## Recommendation Summary

✅ **Separate Project** (`modelmatrix-compute/`)
- Better separation of concerns
- Independent scaling and deployment
- Clearer service boundaries
- More maintainable long-term

The services communicate via REST API, which is simple, language-agnostic, and allows independent evolution.


