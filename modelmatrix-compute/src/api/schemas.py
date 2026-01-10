"""Pydantic schemas for API requests and responses."""
from pydantic import BaseModel, Field, ConfigDict
from typing import Dict, List, Optional, Any
from enum import Enum


class JobStatus(str, Enum):
    """Training job status."""
    PENDING = "pending"
    TRAINING = "training"
    COMPLETED = "completed"
    FAILED = "failed"


class ModelType(str, Enum):
    """Model type for training."""
    CLASSIFICATION = "classification"
    REGRESSION = "regression"
    CLUSTERING = "clustering"


class TrainRequest(BaseModel):
    """Request schema for model training."""
    datasource_id: str = Field(..., description="Datasource UUID")
    build_id: str = Field(..., description="Model build UUID for callback")
    file_path: str = Field(..., description="Path to data file in MinIO")
    algorithm: str = Field(..., description="Algorithm name (decision_tree, random_forest, xgboost)")
    model_type: ModelType = Field(ModelType.CLASSIFICATION, description="Model type (classification, regression, clustering)")
    hyperparameters: Dict[str, Any] = Field(default_factory=dict, description="Algorithm hyperparameters")
    target_column: str = Field(..., description="Name of the target column")
    input_columns: List[str] = Field(..., description="List of input feature columns")
    callback_url: Optional[str] = Field(None, description="URL to call when training completes")
    
    class Config:
        json_schema_extra = {
            "example": {
                "datasource_id": "550e8400-e29b-41d4-a716-446655440000",
                "build_id": "660e8400-e29b-41d4-a716-446655440001",
                "file_path": "minio://modelmatrix/datasources/data.parquet",
                "algorithm": "decision_tree",
                "hyperparameters": {
                    "max_depth": 10,
                    "min_samples_split": 2,
                    "criterion": "gini"
                },
                "target_column": "target",
                "input_columns": ["feature1", "feature2", "feature3"],
                "callback_url": "http://localhost:8080/api/builds/callback"
            }
        }


class TrainResponse(BaseModel):
    """Response schema for training request."""
    job_id: str = Field(..., description="Training job ID")
    status: str = Field(..., description="Job status")
    message: str = Field(..., description="Status message")


class MetricsResponse(BaseModel):
    """Model training metrics."""
    accuracy: Optional[float] = None
    precision: Optional[float] = None
    recall: Optional[float] = None
    f1_score: Optional[float] = None
    confusion_matrix: Optional[List[List[int]]] = None


class JobStatusResponse(BaseModel):
    """Response schema for job status."""
    model_config = ConfigDict(protected_namespaces=())
    
    job_id: str = Field(..., description="Job ID")
    status: JobStatus = Field(..., description="Current status")
    progress: int = Field(0, ge=0, le=100, description="Progress percentage")
    model_path: Optional[str] = Field(None, description="Path to trained model in MinIO")
    metrics: Optional[MetricsResponse] = Field(None, description="Training metrics")
    error: Optional[str] = Field(None, description="Error message if failed")


class HealthResponse(BaseModel):
    """Health check response."""
    status: str = "healthy"
    service: str = "modelmatrix-compute"
    version: str = "1.0.0"


class ScoreRequest(BaseModel):
    """Request schema for model scoring."""
    model_id: str = Field(..., description="Model UUID")
    model_file_path: str = Field(..., description="Path to model file in MinIO")
    input_file_path: str = Field(..., description="Path to input data file in MinIO")
    output_path: str = Field(..., description="Output path in MinIO for scored data")
    input_columns: List[str] = Field(..., description="List of input feature columns")
    model_type: str = Field(..., description="Model type (classification, regression, clustering)")
    algorithm: str = Field(..., description="Algorithm name")
    callback_url: Optional[str] = Field(None, description="URL to call when scoring completes")

    class Config:
        json_schema_extra = {
            "example": {
                "model_id": "550e8400-e29b-41d4-a716-446655440000",
                "model_file_path": "minio://modelmatrix/models/random_forest/abc123.pkl",
                "input_file_path": "minio://modelmatrix/datasources/data.parquet",
                "output_path": "scored/550e8400/scored_data.parquet",
                "input_columns": ["feature1", "feature2", "feature3"],
                "model_type": "classification",
                "algorithm": "random_forest",
                "callback_url": "http://localhost:8080/api/models/550e8400/score/callback"
            }
        }


class ScoreResponse(BaseModel):
    """Response schema for scoring request."""
    job_id: str = Field(..., description="Scoring job ID")
    status: str = Field(..., description="Job status")
    message: str = Field(..., description="Status message")

