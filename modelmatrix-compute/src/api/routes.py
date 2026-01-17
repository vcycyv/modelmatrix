"""API routes for the compute service."""
from fastapi import APIRouter, HTTPException, BackgroundTasks
from typing import Dict
import uuid

from src.api.schemas import (
    TrainRequest,
    TrainResponse,
    ScoreRequest,
    ScoreResponse,
    EvaluateRequest,
    EvaluateResponse,
    JobStatusResponse,
    HealthResponse,
    JobStatus,
)
from src.services.model_trainer import ModelTrainer
from src.services.model_scorer import ModelScorer
from src.services.performance_evaluator import PerformanceEvaluator
from src.core.logger import logger

# In-memory job storage (in production, use Redis or database)
_jobs: Dict[str, Dict] = {}

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
        
        # Train model and notify backend
        trainer = ModelTrainer()
        result = await trainer.train_and_notify(
            file_path=request.file_path,
            algorithm=request.algorithm,
            hyperparameters=request.hyperparameters,
            target_column=request.target_column,
            input_columns=request.input_columns,
            model_type=request.model_type.value if hasattr(request.model_type, 'value') else str(request.model_type),
            callback_url=request.callback_url,
            build_id=request.build_id,
        )
        
        # Update job status
        _jobs[job_id].update(result)
        _jobs[job_id]["status"] = JobStatus.COMPLETED if result["status"] == "completed" else JobStatus.FAILED
        _jobs[job_id]["progress"] = 100
        
        logger.info(f"Training job {job_id} completed with status {result['status']}")
        
    except Exception as e:
        logger.error(f"Training job {job_id} failed: {e}", exc_info=True)
        _jobs[job_id]["status"] = JobStatus.FAILED
        _jobs[job_id]["error"] = str(e)
        _jobs[job_id]["progress"] = 0


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


@router.post("/score", response_model=ScoreResponse, status_code=202)
async def score_model(request: ScoreRequest, background_tasks: BackgroundTasks):
    """
    Start a model scoring job.
    
    The scoring runs in the background. Use GET /compute/status/{job_id} to check progress.
    """
    job_id = str(uuid.uuid4())
    
    # Initialize job status
    _jobs[job_id] = {
        "job_id": job_id,
        "status": JobStatus.PENDING,
        "progress": 0,
        "output_file_path": None,
        "error": None,
    }
    
    # Start scoring in background
    background_tasks.add_task(run_scoring, job_id, request)
    
    logger.info(f"Scoring job {job_id} started for model {request.model_id}")
    
    return ScoreResponse(
        job_id=job_id,
        status="scoring",
        message="Scoring job started"
    )


async def run_scoring(job_id: str, request: ScoreRequest):
    """Background task to run model scoring."""
    try:
        # Update status
        _jobs[job_id]["status"] = JobStatus.TRAINING  # Reuse TRAINING status for scoring
        _jobs[job_id]["progress"] = 10
        
        # Score data and notify backend
        scorer = ModelScorer()
        result = await scorer.score_and_notify(
            model_file_path=request.model_file_path,
            input_file_path=request.input_file_path,
            output_path=request.output_path,
            input_columns=request.input_columns,
            model_type=request.model_type,
            callback_url=request.callback_url,
            model_id=request.model_id,
        )
        
        # Update job status
        _jobs[job_id].update(result)
        _jobs[job_id]["status"] = JobStatus.COMPLETED if result["status"] == "completed" else JobStatus.FAILED
        _jobs[job_id]["progress"] = 100
        
        logger.info(f"Scoring job {job_id} completed with status {result['status']}")
        
    except Exception as e:
        logger.error(f"Scoring job {job_id} failed: {e}", exc_info=True)
        _jobs[job_id]["status"] = JobStatus.FAILED
        _jobs[job_id]["error"] = str(e)
        _jobs[job_id]["progress"] = 0


@router.post("/evaluate", response_model=EvaluateResponse, status_code=202)
async def evaluate_performance(request: EvaluateRequest, background_tasks: BackgroundTasks):
    """
    Start a performance evaluation job.
    
    Evaluates model performance by comparing predictions with actual values.
    Use GET /compute/status/{job_id} to check progress.
    """
    job_id = str(uuid.uuid4())
    
    # Initialize job status
    _jobs[job_id] = {
        "job_id": job_id,
        "status": JobStatus.PENDING,
        "progress": 0,
        "metrics": None,
        "error": None,
    }
    
    # Start evaluation in background
    background_tasks.add_task(run_evaluation, job_id, request)
    
    logger.info(f"Evaluation job {job_id} started for model {request.model_id}")
    
    return EvaluateResponse(
        job_id=job_id,
        status="evaluating",
        message="Evaluation job started"
    )


async def run_evaluation(job_id: str, request: EvaluateRequest):
    """Background task to run performance evaluation."""
    try:
        # Update status
        _jobs[job_id]["status"] = JobStatus.TRAINING  # Reuse TRAINING status
        _jobs[job_id]["progress"] = 10
        
        # Run evaluation
        evaluator = PerformanceEvaluator()
        result = await evaluator.evaluate_and_notify(
            evaluation_id=request.evaluation_id,
            model_id=request.model_id,
            model_file_path=request.model_file_path,
            datasource_file_path=request.datasource_file_path,
            input_columns=request.input_columns,
            target_column=request.target_column,
            actual_column=request.actual_column,
            prediction_column=request.prediction_column,
            model_type=request.model_type,
            callback_url=request.callback_url,
        )
        
        # Update job status
        _jobs[job_id]["metrics"] = result.get("metrics")
        _jobs[job_id]["status"] = JobStatus.COMPLETED if result["status"] == "completed" else JobStatus.FAILED
        _jobs[job_id]["progress"] = 100
        if result.get("error"):
            _jobs[job_id]["error"] = result["error"]
        
        logger.info(f"Evaluation job {job_id} completed with status {result['status']}")
        
    except Exception as e:
        logger.error(f"Evaluation job {job_id} failed: {e}", exc_info=True)
        _jobs[job_id]["status"] = JobStatus.FAILED
        _jobs[job_id]["error"] = str(e)
        _jobs[job_id]["progress"] = 0


@router.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint."""
    return HealthResponse()


