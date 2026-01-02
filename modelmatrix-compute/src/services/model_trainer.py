"""Service for training ML models."""
import uuid
from typing import Dict, Any, Optional
import pandas as pd
from sklearn.model_selection import train_test_split

from src.services.data_loader import DataLoader
from src.services.model_saver import ModelSaver
from src.algorithms.base import BaseAlgorithm
from src.algorithms.decision_tree import DecisionTreeAlgorithm
from src.algorithms.random_forest import RandomForestAlgorithm
from src.algorithms.xgboost import XGBoostAlgorithm
from src.core.logger import logger


# Algorithm registry
ALGORITHMS: Dict[str, BaseAlgorithm] = {
    "decision_tree": DecisionTreeAlgorithm(),
    "random_forest": RandomForestAlgorithm(),
    "xgboost": XGBoostAlgorithm(),
}


class ModelTrainer:
    """Orchestrates model training workflow."""
    
    def __init__(self):
        """Initialize trainer with data loader and model saver."""
        self.data_loader = DataLoader()
        self.model_saver = ModelSaver()
    
    def train(
        self,
        file_path: str,
        algorithm: str,
        hyperparameters: Dict[str, Any],
        target_column: str,
        input_columns: list,
        test_size: float = 0.2,
    ) -> Dict[str, Any]:
        """
        Train a machine learning model.
        
        Args:
            file_path: Path to data file in MinIO
            algorithm: Algorithm name
            hyperparameters: Algorithm hyperparameters
            target_column: Target column name
            input_columns: Input feature columns
            test_size: Test set size ratio (default: 0.2)
            
        Returns:
            Dictionary with job_id, model_path, and metrics
        """
        job_id = str(uuid.uuid4())
        logger.info(f"Starting training job {job_id} with algorithm {algorithm}")
        
        try:
            # Get algorithm
            if algorithm not in ALGORITHMS:
                raise ValueError(f"Unknown algorithm: {algorithm}. Available: {list(ALGORITHMS.keys())}")
            
            algo = ALGORITHMS[algorithm]
            
            # Load data
            logger.info(f"Loading data from {file_path}")
            if file_path.endswith(".parquet"):
                df = self.data_loader.load_parquet(file_path)
            elif file_path.endswith(".csv"):
                df = self.data_loader.load_csv(file_path)
            else:
                raise ValueError(f"Unsupported file format. Use .parquet or .csv")
            
            # Prepare features
            X, y = self.data_loader.prepare_features(df, target_column, input_columns)
            
            # Split data
            X_train, X_test, y_train, y_test = train_test_split(
                X, y, test_size=test_size, random_state=42, stratify=y if y.dtype == 'object' or y.nunique() < 10 else None
            )
            
            logger.info(f"Training set: {len(X_train)} samples, Test set: {len(X_test)} samples")
            
            # Train model
            logger.info(f"Training {algorithm} model...")
            model = algo.train(X_train, y_train, hyperparameters)
            
            # Evaluate model
            logger.info("Evaluating model...")
            metrics = algo.evaluate(model, X_test, y_test)
            
            # Save model
            logger.info("Saving model...")
            model_path = self.model_saver.save_model(model, job_id, algorithm)
            
            logger.info(f"Training completed successfully. Job ID: {job_id}")
            
            return {
                "job_id": job_id,
                "status": "completed",
                "model_path": model_path,
                "metrics": metrics,
                "error": None,
            }
            
        except Exception as e:
            logger.error(f"Training failed for job {job_id}: {e}", exc_info=True)
            return {
                "job_id": job_id,
                "status": "failed",
                "model_path": None,
                "metrics": None,
                "error": str(e),
            }


