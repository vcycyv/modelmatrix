"""API routes for the compute service."""
from fastapi import APIRouter, HTTPException, BackgroundTasks
from typing import Dict, Optional
import uuid
import httpx

from src.api.schemas import (
    TrainRequest,
    TrainResponse,
    JobStatusResponse,
    HealthResponse,
    JobStatus,
)
from src.services.model_trainer import ModelTrainer
from src.core.logger import logger

# In-memory job storage (in production, use Redis or database)
_jobs: Dict[str, Dict] = {}

# HTTP client for callbacks
_http_client = httpx.AsyncClient(timeout=30.0)

router = APIRouter(prefix="/compute", tags=["compute"])


@router.post("/train", response_model=TrainResponse, status_code=202)
async def train_model(request: TrainRequest, background_tasks: BackgroundTasks):
    """
    Start a model training job.
    
    The training runs in the background. Use GET /compute/status/{job_id} to check progress.
    """
    job_id = str(uuid.uuid4())
    
    # Initialize job status
    _jobs[job_id] = {
        "job_id": job_id,
        "status": JobStatus.PENDING,
        "progress": 0,
        "model_path": None,
        "metrics": None,
        "error": None,
    }
    
    # Start training in background
    background_tasks.add_task(run_training, job_id, request)
    
    logger.info(f"Training job {job_id} started for datasource {request.datasource_id}")
    
    return TrainResponse(
        job_id=job_id,
        status="training",
        message="Training job started"
    )


async def run_training(job_id: str, request: TrainRequest):
    """Background task to run model training."""
    try:
        # Update status to training
        _jobs[job_id]["status"] = JobStatus.TRAINING
        _jobs[job_id]["progress"] = 10
        
        # Train model
        trainer = ModelTrainer()
        result = trainer.train(
            file_path=request.file_path,
            algorithm=request.algorithm,
            hyperparameters=request.hyperparameters,
            target_column=request.target_column,
            input_columns=request.input_columns,
        )
        
        # Update job status
        _jobs[job_id].update(result)
        _jobs[job_id]["status"] = JobStatus.COMPLETED if result["status"] == "completed" else JobStatus.FAILED
        _jobs[job_id]["progress"] = 100
        
        logger.info(f"Training job {job_id} completed with status {result['status']}")
        
        # Call webhook to notify backend
        if request.callback_url:
            await send_callback(request.callback_url, request.build_id, job_id, result)
        
    except Exception as e:
        logger.error(f"Training job {job_id} failed: {e}", exc_info=True)
        _jobs[job_id]["status"] = JobStatus.FAILED
        _jobs[job_id]["error"] = str(e)
        _jobs[job_id]["progress"] = 0
        
        # Notify failure via callback
        if request.callback_url:
            await send_callback(
                request.callback_url,
                request.build_id,
                job_id,
                {"status": "failed", "error": str(e), "model_path": None, "metrics": None}
            )


async def send_callback(callback_url: str, build_id: str, job_id: str, result: Dict):
    """Send callback to backend with training results."""
    try:
        payload = {
            "build_id": build_id,
            "job_id": job_id,
            "status": result.get("status", "failed"),
            "model_path": result.get("model_path"),
            "metrics": result.get("metrics"),
            "error": result.get("error"),
        }
        
        logger.info(f"Sending callback to {callback_url} for build {build_id}")
        response = await _http_client.post(callback_url, json=payload)
        
        if response.status_code == 200:
            logger.info(f"Callback successful for build {build_id}")
        else:
            logger.warning(f"Callback returned status {response.status_code}: {response.text}")
            
    except Exception as e:
        logger.error(f"Failed to send callback for build {build_id}: {e}")


@router.get("/status/{job_id}", response_model=JobStatusResponse)
async def get_job_status(job_id: str):
    """Get the status of a training job."""
    if job_id not in _jobs:
        raise HTTPException(status_code=404, detail=f"Job {job_id} not found")
    
    job = _jobs[job_id]
    
    return JobStatusResponse(
        job_id=job["job_id"],
        status=job["status"],
        progress=job["progress"],
        model_path=job.get("model_path"),
        metrics=job.get("metrics"),
        error=job.get("error"),
    )


@router.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint."""
    return HealthResponse()


