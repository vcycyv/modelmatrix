# Postman Testing Guide: Building a Model

This guide walks you through testing the model building workflow using Postman.

## Prerequisites

- Backend API running on `http://localhost:8080`
- Compute service running on `http://localhost:8081`
- MinIO running (for file storage)
- PostgreSQL database running
- LDAP server running

## Step-by-Step Testing

### Step 1: Authenticate and Get JWT Token

**Request:**
```
POST http://localhost:8080/api/auth/login
Content-Type: application/json
```

**Body:**
```json
{
  "username": "michael.jordan",
  "password": "111222333"
}
```

**Response:**
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

**Save the token** - You'll need it for all subsequent requests in the `Authorization` header:
```
Authorization: Bearer <your-token>
```

---

### Step 2: Create a Collection (if you don't have one)

**Request:**
```
POST http://localhost:8080/api/collections
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Body:**
```json
{
  "name": "Test Collection",
  "description": "Collection for testing model builds"
}
```

**Response:**
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Test Collection",
    ...
  }
}
```

**Save the collection_id** for the next step.

---

### Step 3: Create a Datasource (or use existing one)

#### Option A: Create Datasource from CSV File

**Request:**
```
POST http://localhost:8080/api/datasources
Authorization: Bearer <your-token>
Content-Type: multipart/form-data
```

**Form Data:**
- `collection_id`: `<collection-id-from-step-2>`
- `name`: `HMEQ Dataset`
- `description`: `Home Equity dataset for testing`
- `type`: `csv`
- `file`: (Select file) - Upload your CSV file (e.g., `hmeq.csv`)

**Response:**
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "name": "HMEQ Dataset",
    "file_path": "datasources/660e8400-e29b-41d4-a716-446655440001/hmeq.csv",
    ...
  }
}
```

**Save the datasource_id** for later steps.

#### Option B: Create Datasource from PostgreSQL Database

**Request:**
```
POST http://localhost:8080/api/datasources
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Body:**
```json
{
  "collection_id": "<collection-id>",
  "name": "iris",
  "description": "iris description",
  "type": "postgresql",
  "connection_config": {
    "host": "localhost",
    "port": 5432,
    "database": "datasets",
    "username": "postgres",
    "password": "dayang",
    "schema": "public",
    "table": "iris",
    "sslmode": "disable"
  }
}
```

---

### Step 4: Set Column Roles (Target and Input)

The datasource needs at least:
- **1 target column** (the variable you want to predict)
- **1+ input columns** (features used for prediction)

**Request:**
```
PUT http://localhost:8080/api/datasources/<datasource-id>/columns/roles
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Body:**
```json
{
  "columns": [
    {
      "column_id": "<column-id-1>",
      "role": "target"
    },
    {
      "column_id": "<column-id-2>",
      "role": "input"
    },
    {
      "column_id": "<column-id-3>",
      "role": "input"
    }
  ]
}
```

**To get column IDs, first get datasource details:**
```
GET http://localhost:8080/api/datasources/<datasource-id>
Authorization: Bearer <your-token>
```

This returns all columns with their IDs.

---

### Step 5: Create a Model Build

**Request:**
```
POST http://localhost:8080/api/builds
Authorization: Bearer <your-token>
Content-Type: application/json
```

**Body:**
```json
{
  "name": "Sales Predictor v1",
  "description": "Random forest model for sales prediction",
  "datasource_id": "<datasource-id-from-step-3>",
  "model_type": "classification",
  "parameters": {
    "algorithm": "random_forest",
    "hyperparameters": {
      "n_estimators": 100,
      "max_depth": 10,
      "min_samples_split": 2
    },
    "train_test_split": 0.8,
    "random_seed": 42
  }
}
```

**Available model_type values:**
- `classification`
- `regression`
- `clustering`

**Available algorithm values:**
- `decision_tree`
- `random_forest`
- `xgboost`

**Response:**
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "name": "Sales Predictor v1",
    "status": "pending",
    "datasource_id": "...",
    ...
  }
}
```

**Save the build_id** for the next step.

---

### Step 6: Start the Model Build

**Request:**
```
POST http://localhost:8080/api/builds/<build-id>/start
Authorization: Bearer <your-token>
```

**Response:**
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "name": "Sales Predictor v1",
    "status": "running",
    "started_at": "2024-01-15T10:30:00Z",
    ...
  }
}
```

This will:
1. Call the compute service to start training
2. Update build status to "running"
3. Store the job ID for tracking

---

### Step 7: Check Build Status

**Request:**
```
GET http://localhost:8080/api/builds/<build-id>
Authorization: Bearer <your-token>
```

**Response (while training):**
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": "...",
    "status": "running",
    "started_at": "2024-01-15T10:30:00Z",
    ...
  }
}
```

**Response (after completion):**
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": "...",
    "status": "completed",
    "started_at": "2024-01-15T10:30:00Z",
    "completed_at": "2024-01-15T10:35:00Z",
    "metrics": {
      "accuracy": 0.95,
      "precision": 0.94,
      "recall": 0.93,
      "f1_score": 0.935
    },
    ...
  }
}
```

---

### Step 8: (Optional) Check Compute Service Job Status Directly

**Request:**
```
GET http://localhost:8081/compute/status/<job-id>
```

**Note:** The job_id is stored in the build's hyperparameters as `_job_id`. You can extract it from the build response.

**Response:**
```json
{
  "job_id": "...",
  "status": "completed",
  "progress": 100,
  "model_path": "minio://modelmatrix/models/model.pkl",
  "metrics": {
    "accuracy": 0.95,
    ...
  }
}
```

---

## Complete Example Workflow

Here's a complete example using the HMEQ dataset:

### 1. Login
```bash
POST http://localhost:8080/api/auth/login
{
  "username": "michael.jordan",
  "password": "111222333"
}
# Save token: <TOKEN>
```

### 2. Create Collection
```bash
POST http://localhost:8080/api/collections
Authorization: Bearer <TOKEN>
{
  "name": "ML Training Data",
  "description": "Collection for ML models"
}
# Save collection_id: <COLLECTION_ID>
```

### 3. Create Datasource (CSV Upload)
```bash
POST http://localhost:8080/api/datasources
Authorization: Bearer <TOKEN>
Content-Type: multipart/form-data

collection_id: <COLLECTION_ID>
name: HMEQ Dataset
description: Home Equity dataset
type: csv
file: [Upload hmeq.csv]
# Save datasource_id: <DATASOURCE_ID>
```

### 4. Get Datasource Columns
```bash
GET http://localhost:8080/api/datasources/<DATASOURCE_ID>
Authorization: Bearer <TOKEN>
# Note column IDs, e.g., BAD column (target), LOAN, MORTDUE, etc. (inputs)
```

### 5. Set Column Roles
```bash
PUT http://localhost:8080/api/datasources/<DATASOURCE_ID>/columns/roles
Authorization: Bearer <TOKEN>
{
  "columns": [
    {"column_id": "<BAD_COLUMN_ID>", "role": "target"},
    {"column_id": "<LOAN_COLUMN_ID>", "role": "input"},
    {"column_id": "<MORTDUE_COLUMN_ID>", "role": "input"},
    {"column_id": "<VALUE_COLUMN_ID>", "role": "input"}
  ]
}
```

### 6. Create Build
```bash
POST http://localhost:8080/api/builds
Authorization: Bearer <TOKEN>
{
  "name": "HMEQ Classifier",
  "description": "Classification model for home equity",
  "datasource_id": "<DATASOURCE_ID>",
  "model_type": "classification",
  "parameters": {
    "algorithm": "random_forest",
    "hyperparameters": {
      "n_estimators": 100,
      "max_depth": 10
    },
    "train_test_split": 0.8,
    "random_seed": 42
  }
}
# Save build_id: <BUILD_ID>
```

### 7. Start Build
```bash
POST http://localhost:8080/api/builds/<BUILD_ID>/start
Authorization: Bearer <TOKEN>
```

### 8. Monitor Build Status
```bash
GET http://localhost:8080/api/builds/<BUILD_ID>
Authorization: Bearer <TOKEN>
# Poll this endpoint until status is "completed" or "failed"
```

---

## Troubleshooting

### Error: "no target column found"
- Make sure you've set at least one column role to "target" in Step 4

### Error: "no input columns found"
- Make sure you've set at least one column role to "input" in Step 4

### Error: "datasource does not have a file path"
- The datasource must have a file uploaded (for CSV/Parquet) or data fetched from database

### Error: "Failed to start training job"
- Check that the compute service is running on port 8081
- Check compute service logs for errors
- Verify MinIO is accessible and configured correctly

### Build Status Stuck on "running"
- Check compute service logs: `http://localhost:8081/compute/status/<job-id>`
- Verify the compute service can access MinIO to read the data file

---

## Postman Collection Setup Tips

1. **Create Environment Variables:**
   - `base_url`: `http://localhost:8080`
   - `compute_url`: `http://localhost:8081`
   - `token`: (set after login)
   - `collection_id`: (set after creating collection)
   - `datasource_id`: (set after creating datasource)
   - `build_id`: (set after creating build)

2. **Use Pre-request Scripts:**
   - Automatically add `Authorization: Bearer {{token}}` header

3. **Use Tests Scripts:**
   - Extract IDs from responses and save to environment variables
   - Example: `pm.environment.set("build_id", pm.response.json().data.id);`

