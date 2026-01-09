"""Service for training ML models."""
import uuid
from typing import Dict, Any, Optional
import pandas as pd
from sklearn.model_selection import train_test_split
import httpx

from src.services.data_loader import DataLoader
from src.services.model_saver import ModelSaver
from src.algorithms.base import BaseAlgorithm
from src.algorithms.decision_tree import DecisionTreeAlgorithm
from src.algorithms.random_forest import RandomForestAlgorithm
from src.algorithms.xgboost import XGBoostAlgorithm
from src.algorithms.linear_regression import LinearRegressionAlgorithm
from src.algorithms.polynomial_regression import PolynomialRegressionAlgorithm
from src.algorithms.kmeans import KMeansAlgorithm
from src.core.logger import logger


# Algorithm registry
ALGORITHMS: Dict[str, BaseAlgorithm] = {
    "decision_tree": DecisionTreeAlgorithm(),
    "random_forest": RandomForestAlgorithm(),
    "xgboost": XGBoostAlgorithm(),
    "linear_regression": LinearRegressionAlgorithm(),
    "polynomial_regression": PolynomialRegressionAlgorithm(),
    "kmeans": KMeansAlgorithm(),
}


class ModelTrainer:
    """Orchestrates model training workflow."""
    
    def __init__(self):
        """Initialize trainer with data loader and model saver."""
        self.data_loader = DataLoader()
        self.model_saver = ModelSaver()
        self._http_client = httpx.AsyncClient(timeout=30.0)
    
    def train(
        self,
        file_path: str,
        algorithm: str,
        hyperparameters: Dict[str, Any],
        target_column: str,
        input_columns: list,
        model_type: str = "classification",
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
            model_type: Model type (classification, regression, clustering)
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
            
            # Prepare features based on model type
            if model_type == "clustering":
                # Unsupervised learning - no target column needed
                X = self.data_loader.prepare_features_unsupervised(df, input_columns)
                y = None
                
                # Split data for evaluation (no stratification)
                X_train, X_test = train_test_split(
                    X, test_size=test_size, random_state=42
                )
                y_train, y_test = None, None
                
                logger.info(f"Training set: {len(X_train)} samples, Test set: {len(X_test)} samples")
            else:
                # Supervised learning - requires target column
                X, y = self.data_loader.prepare_features(df, target_column, input_columns)
                
                # Split data (stratify only for classification with categorical/discrete target)
                stratify = None
                if model_type == "classification" and (y.dtype == 'object' or y.nunique() < 10):
                    stratify = y
                
                X_train, X_test, y_train, y_test = train_test_split(
                    X, y, test_size=test_size, random_state=42, stratify=stratify
                )
                
                logger.info(f"Training set: {len(X_train)} samples, Test set: {len(X_test)} samples")
            
            # The model's input variables are the original input columns (before preprocessing)
            # One-hot encoding and other transformations are part of the scoring logic
            feature_names = input_columns
            logger.info(f"Model input variables: {len(feature_names)} columns: {feature_names[:10]}{'...' if len(feature_names) > 10 else ''}")
            
            # Train model
            logger.info(f"Training {algorithm} model for {model_type}...")
            model = algo.train(X_train, y_train, hyperparameters, model_type)
            
            # Evaluate model
            logger.info("Evaluating model...")
            metrics = algo.evaluate(model, X_test, y_test, model_type)
            
            # Extract feature importances
            logger.info("Extracting feature importances...")
            feature_importances = algo.get_feature_importances(
                model=model,
                feature_names=feature_names,
                X=X_test,
                y=y_test,
            )
            
            # Log which features are unused (importance = 0)
            unused_features = [name for name, imp in feature_importances.items() if imp == 0]
            if unused_features:
                logger.info(f"Unused features (importance=0): {unused_features}")
            
            # Save model with feature names
            logger.info("Saving model...")
            model_path = self.model_saver.save_model(
                model=model,
                job_id=job_id,
                algorithm=algorithm,
                feature_names=feature_names,
                target_column=target_column if model_type != "clustering" else None,
                model_type=model_type,
            )
            
            logger.info(f"Training completed successfully. Job ID: {job_id}")
            
            return {
                "job_id": job_id,
                "status": "completed",
                "model_path": model_path,
                "metrics": metrics,
                "feature_names": feature_names,
                "feature_count": len(feature_names),
                "feature_importances": feature_importances,
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

    async def train_and_notify(
        self,
        file_path: str,
        algorithm: str,
        hyperparameters: Dict[str, Any],
        target_column: str,
        input_columns: list,
        model_type: str = "classification",
        test_size: float = 0.2,
        callback_url: Optional[str] = None,
        build_id: Optional[str] = None,
    ) -> Dict[str, Any]:
        """
        Train a model and send callback notification when complete.
        
        This is the main entry point for training jobs that need to notify
        the backend of completion/failure.
        """
        # Run training (synchronous)
        result = self.train(
            file_path=file_path,
            algorithm=algorithm,
            hyperparameters=hyperparameters,
            target_column=target_column,
            input_columns=input_columns,
            model_type=model_type,
            test_size=test_size,
        )
        
        # Send callback if URL provided
        if callback_url:
            await self._send_callback(callback_url, build_id, result)
        
        return result
    
    async def _send_callback(
        self,
        callback_url: str,
        build_id: Optional[str],
        result: Dict[str, Any],
    ) -> None:
        """Send callback to backend with training results."""
        try:
            payload = {
                "build_id": build_id,
                "job_id": result.get("job_id"),
                "status": result.get("status", "failed"),
                "model_path": result.get("model_path"),
                "metrics": result.get("metrics"),
                "feature_names": result.get("feature_names"),
                "feature_count": result.get("feature_count"),
                "feature_importances": result.get("feature_importances"),
                "error": result.get("error"),
            }
            
            logger.info(f"Sending callback to {callback_url} for build {build_id}")
            response = await self._http_client.post(callback_url, json=payload)
            
            if response.status_code == 200:
                logger.info(f"Callback successful for build {build_id}")
            else:
                logger.warning(f"Callback returned status {response.status_code}: {response.text}")
                
        except Exception as e:
            logger.error(f"Failed to send callback for build {build_id}: {e}")


