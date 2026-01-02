# ModelMatrix Compute Server

Python-based ML compute service for training machine learning models. This service receives training requests from the Go backend, loads data from MinIO, trains models using various algorithms, and returns model files and metrics.

## Features

- **REST API**: FastAPI-based HTTP service
- **Multiple Algorithms**: Decision Tree, Random Forest, XGBoost, etc.
- **MinIO Integration**: Loads training data and saves trained models
- **Async Training**: Supports long-running training jobs
- **Metrics**: Returns training metrics (accuracy, precision, recall, etc.)

## Architecture

```
Go Backend (Orchestrator)
    ↓ HTTP/REST
Python Compute Service (This)
    ↓
MinIO (Data Storage)
    ↓
Trained Models → MinIO
```

## Project Structure

```
modelmatrix-compute/
├── src/
│   ├── main.py              # FastAPI app entry point
│   ├── api/
│   │   ├── routes.py        # API endpoints
│   │   └── schemas.py       # Pydantic request/response models
│   ├── core/
│   │   ├── config.py         # Configuration management
│   │   └── logger.py         # Logging setup
│   ├── services/
│   │   ├── model_trainer.py  # Model training orchestration
│   │   ├── data_loader.py    # Load data from MinIO
│   │   └── model_saver.py    # Save models to MinIO
│   └── algorithms/
│       ├── base.py           # Base algorithm interface
│       ├── decision_tree.py
│       ├── random_forest.py
│       └── xgboost.py
├── tests/
│   ├── test_api.py
│   └── test_trainer.py
├── requirements.txt
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

## Quick Start

### 1. Install Dependencies

```bash
cd modelmatrix-compute
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r requirements.txt
```

### 2. Configure Environment

```bash
cp .env.example .env
# Edit .env with your settings
```

### 3. Run the Service

```bash
# Development
uvicorn src.main:app --reload --host 0.0.0.0 --port 8081

# Production
uvicorn src.main:app --host 0.0.0.0 --port 8081 --workers 4
```

### 4. Using Docker

```bash
docker-compose up -d
```

## API Endpoints

### POST /compute/train

Train a machine learning model.

**Request:**
```json
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
  "input_columns": ["feature1", "feature2", "feature3"]
}
```

**Response:**
```json
{
  "job_id": "uuid",
  "status": "training",
  "message": "Training started"
}
```

### GET /compute/status/{job_id}

Get training job status.

**Response:**
```json
{
  "job_id": "uuid",
  "status": "completed",
  "progress": 100,
  "model_path": "minio://bucket/models/model.pkl",
  "metrics": {
    "accuracy": 0.95,
    "precision": 0.93,
    "recall": 0.92,
    "f1_score": 0.925
  },
  "error": null
}
```

### GET /health

Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "service": "modelmatrix-compute",
  "version": "1.0.0"
}
```

## Configuration

Environment variables (see `.env.example`):

- `MINIO_ENDPOINT`: MinIO server endpoint
- `MINIO_ACCESS_KEY`: MinIO access key
- `MINIO_SECRET_KEY`: MinIO secret key
- `MINIO_BUCKET`: MinIO bucket name
- `MINIO_USE_SSL`: Use SSL for MinIO (true/false)
- `COMPUTE_HOST`: Service host (default: 0.0.0.0)
- `COMPUTE_PORT`: Service port (default: 8081)
- `LOG_LEVEL`: Logging level (default: info)

## Algorithms Supported

- **decision_tree**: Scikit-learn Decision Tree
- **random_forest**: Scikit-learn Random Forest
- **xgboost**: XGBoost Gradient Boosting
- More algorithms can be added by implementing the base algorithm interface

## Development

### Run Tests

```bash
pytest tests/
```

### Code Formatting

```bash
black src/ tests/
isort src/ tests/
```

### Type Checking

```bash
mypy src/
```

## Integration with Go Backend

The Go backend calls this service via HTTP. See `modelmatrix_backend/internal/infrastructure/compute/` for the Go client implementation.

## License

Same as ModelMatrix project.


