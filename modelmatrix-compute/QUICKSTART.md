# Quick Start Guide

## Prerequisites

- Python 3.11+
- MinIO running (or use docker-compose)
- Go backend running (for integration)

## Setup

### 1. Create Virtual Environment

```bash
cd modelmatrix-compute
python -m venv venv
source venv/bin/activate
```

### 2. Install Dependencies

```bash
pip install -r requirements.txt
```

### 3. Configure Environment

```bash
cp .env.example .env
# Edit .env with your MinIO settings
```

### 4. Run the Service

```bash
# Development mode (with auto-reload)
uvicorn src.main:app --reload --host 0.0.0.0 --port 8081

# Or use Python directly
python -m src.main
```

### 5. Test the Service

```bash
# Health check
curl http://localhost:8081/compute/health

# Should return:
# {"status":"healthy","service":"modelmatrix-compute","version":"1.0.0"}
```

## Using Docker

```bash
# Build and run with docker-compose (includes MinIO)
docker-compose up -d

# View logs
docker-compose logs -f compute
```

## Example API Usage

### Start Training Job

```bash
curl -X POST http://localhost:8081/compute/train \
  -H "Content-Type: application/json" \
  -d '{
    "datasource_id": "550e8400-e29b-41d4-a716-446655440000",
    "file_path": "minio://modelmatrix/datasources/data.parquet",
    "algorithm": "decision_tree",
    "hyperparameters": {
      "max_depth": 10,
      "min_samples_split": 2,
      "criterion": "gini"
    },
    "target_column": "target",
    "input_columns": ["feature1", "feature2", "feature3"]
  }'
```

### Check Job Status

```bash
curl http://localhost:8081/compute/status/{job_id}
```

## Integration with Go Backend

The Go backend is already configured to call this service. Make sure:

1. Compute service is running on port 8081
2. Go backend config has `compute.service_url: "http://localhost:8081"`
3. Both services can access the same MinIO instance

## Next Steps

- See `README.md` for full documentation
- Check `src/algorithms/` to add more ML algorithms
- Review `src/api/routes.py` for API endpoints


