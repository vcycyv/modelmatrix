"""Performance evaluation service for ML models."""
import httpx
import numpy as np
import pandas as pd
from typing import Dict, List, Optional, Any
from sklearn.metrics import (
    accuracy_score,
    precision_score,
    recall_score,
    f1_score,
    roc_auc_score,
    average_precision_score,
    confusion_matrix,
    mean_absolute_error,
    mean_squared_error,
    r2_score,
    mean_absolute_percentage_error,
)

from src.core.logger import logger
from src.services.data_loader import DataLoader


class PerformanceEvaluator:
    """Evaluates ML model performance by comparing predictions with actual values."""

    def __init__(self):
        """Initialize the performance evaluator."""
        self.data_loader = DataLoader()

    async def evaluate_and_notify(
        self,
        evaluation_id: str,
        model_id: str,
        model_file_path: str,
        datasource_file_path: str,
        input_columns: List[str],
        target_column: str,
        actual_column: str,
        prediction_column: Optional[str],
        model_type: str,
        callback_url: Optional[str] = None,
    ) -> Dict[str, Any]:
        """
        Evaluate model performance and send results to callback URL.

        Args:
            evaluation_id: Unique ID for this evaluation
            model_id: ID of the model being evaluated
            model_file_path: Path to the trained model file in MinIO
            datasource_file_path: Path to evaluation data with actuals
            input_columns: List of input feature column names
            target_column: Expected target column name in model
            actual_column: Column containing actual/true values in evaluation data
            prediction_column: Column containing predictions (if already scored)
            model_type: Type of model (classification, regression)
            callback_url: URL to call with results

        Returns:
            Dict containing evaluation metrics and status
        """
        result = {
            "evaluation_id": evaluation_id,
            "model_id": model_id,
            "status": "failed",
            "metrics": {},
            "sample_count": 0,
            "error": None,
        }

        try:
            logger.info(f"Starting performance evaluation {evaluation_id} for model {model_id}")
            
            # Load evaluation data
            logger.info(f"Loading evaluation data from {datasource_file_path}")
            eval_data = self.data_loader.load_data(datasource_file_path)
            
            if eval_data is None or len(eval_data) == 0:
                raise ValueError("Evaluation data is empty")

            result["sample_count"] = len(eval_data)
            logger.info(f"Loaded {len(eval_data)} samples for evaluation")

            # Check if actual column exists
            if actual_column not in eval_data.columns:
                raise ValueError(f"Actual column '{actual_column}' not found in evaluation data. "
                               f"Available columns: {list(eval_data.columns)}")

            y_actual = eval_data[actual_column]

            # Get predictions - either from existing column or by scoring
            if prediction_column and prediction_column in eval_data.columns:
                logger.info(f"Using existing predictions from column '{prediction_column}'")
                y_pred = eval_data[prediction_column]
            else:
                logger.info("Generating predictions using model")
                y_pred = await self._generate_predictions(
                    model_file_path,
                    eval_data,
                    input_columns,
                    model_type,
                )

            # Calculate metrics based on model type
            if model_type.lower() == "classification":
                metrics = self._calculate_classification_metrics(y_actual, y_pred, eval_data)
            elif model_type.lower() == "regression":
                metrics = self._calculate_regression_metrics(y_actual, y_pred)
            else:
                raise ValueError(f"Unsupported model type: {model_type}")

            result["metrics"] = metrics
            result["status"] = "completed"
            
            logger.info(f"Evaluation {evaluation_id} completed with metrics: {metrics}")

        except Exception as e:
            logger.error(f"Evaluation {evaluation_id} failed: {str(e)}", exc_info=True)
            result["status"] = "failed"
            result["error"] = str(e)

        # Send callback if URL provided
        if callback_url:
            await self._send_callback(callback_url, result)

        return result

    async def _generate_predictions(
        self,
        model_file_path: str,
        data: pd.DataFrame,
        input_columns: List[str],
        model_type: str,
    ) -> np.ndarray:
        """Generate predictions using the model."""
        model_data = self.data_loader.load_model(model_file_path)
        if model_data is None:
            raise ValueError(f"Failed to load model from {model_file_path}")

        # Support both dict format {"model": ..., "preprocessor": ...} and raw model
        if isinstance(model_data, dict) and "model" in model_data:
            model = model_data["model"]
            preprocessor = model_data.get("preprocessor")
        else:
            model = model_data
            preprocessor = None

        if model is None:
            raise ValueError("Model not found in model file")

        # Prepare input data with same encoding as training (one-hot for categoricals)
        if preprocessor:
            X = data[input_columns].copy()
            X = preprocessor.transform(X)
        else:
            # Match training pipeline: use DataLoader's prepare_features_unsupervised
            # so categoricals are one-hot encoded the same way as at fit time
            X = self.data_loader.prepare_features_unsupervised(data, input_columns)
            # Align columns to model's expected feature names (add missing as 0, reorder)
            expected = getattr(model, "feature_names_in_", None)
            if expected is not None:
                for col in expected:
                    if col not in X.columns:
                        X[col] = 0
                X = X[expected]

        predictions = model.predict(X)
        return predictions

    def _calculate_classification_metrics(
        self,
        y_actual: pd.Series,
        y_pred: np.ndarray,
        data: pd.DataFrame,
    ) -> Dict[str, float]:
        """Calculate classification metrics."""
        metrics = {}

        # Convert to numpy arrays for sklearn
        y_actual = np.array(y_actual)
        y_pred = np.array(y_pred)

        # Handle potential NaN values
        mask = ~(pd.isna(y_actual) | pd.isna(y_pred))
        y_actual = y_actual[mask]
        y_pred = y_pred[mask]

        if len(y_actual) == 0:
            raise ValueError("No valid samples after removing NaN values")

        # Determine if binary or multiclass
        unique_classes = np.unique(np.concatenate([y_actual, y_pred]))
        is_binary = len(unique_classes) <= 2

        # Basic metrics
        metrics["accuracy"] = float(accuracy_score(y_actual, y_pred))
        
        # Precision, Recall, F1
        average = "binary" if is_binary else "weighted"
        metrics["precision"] = float(precision_score(y_actual, y_pred, average=average, zero_division=0))
        metrics["recall"] = float(recall_score(y_actual, y_pred, average=average, zero_division=0))
        metrics["f1_score"] = float(f1_score(y_actual, y_pred, average=average, zero_division=0))

        # AUC-ROC (only for binary classification)
        if is_binary:
            try:
                metrics["auc_roc"] = float(roc_auc_score(y_actual, y_pred))
            except ValueError:
                logger.warning("Could not calculate AUC-ROC")

            try:
                metrics["auc_pr"] = float(average_precision_score(y_actual, y_pred))
            except ValueError:
                logger.warning("Could not calculate AUC-PR")

        # Confusion matrix (as flattened list for JSON)
        cm = confusion_matrix(y_actual, y_pred)
        metrics["confusion_matrix"] = cm.tolist()

        # Calculate PSI (Population Stability Index) if we have probability predictions
        # For now, skip PSI as it requires probability outputs

        return metrics

    def _calculate_regression_metrics(
        self,
        y_actual: pd.Series,
        y_pred: np.ndarray,
    ) -> Dict[str, float]:
        """Calculate regression metrics."""
        metrics = {}

        # Convert to numpy arrays
        y_actual = np.array(y_actual)
        y_pred = np.array(y_pred)

        # Handle potential NaN values
        mask = ~(np.isnan(y_actual) | np.isnan(y_pred))
        y_actual = y_actual[mask]
        y_pred = y_pred[mask]

        if len(y_actual) == 0:
            raise ValueError("No valid samples after removing NaN values")

        # Calculate metrics
        metrics["mae"] = float(mean_absolute_error(y_actual, y_pred))
        metrics["mse"] = float(mean_squared_error(y_actual, y_pred))
        metrics["rmse"] = float(np.sqrt(metrics["mse"]))
        metrics["r2"] = float(r2_score(y_actual, y_pred))

        # MAPE (avoiding division by zero)
        try:
            # Filter out zeros from actual values for MAPE calculation
            non_zero_mask = y_actual != 0
            if non_zero_mask.sum() > 0:
                metrics["mape"] = float(mean_absolute_percentage_error(
                    y_actual[non_zero_mask], y_pred[non_zero_mask]
                )) * 100  # Convert to percentage
        except Exception:
            logger.warning("Could not calculate MAPE")

        return metrics

    async def _send_callback(self, callback_url: str, result: Dict[str, Any]) -> None:
        """Send evaluation results to callback URL."""
        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                response = await client.post(callback_url, json=result)
                if response.status_code == 200:
                    logger.info(f"Callback sent successfully to {callback_url}")
                else:
                    logger.warning(f"Callback returned status {response.status_code}: {response.text}")
        except Exception as e:
            logger.error(f"Failed to send callback to {callback_url}: {str(e)}")


def calculate_psi(expected: np.ndarray, actual: np.ndarray, bins: int = 10) -> float:
    """
    Calculate Population Stability Index (PSI).
    
    PSI measures the shift in distribution between two samples.
    PSI < 0.1: No significant change
    0.1 <= PSI < 0.25: Moderate change
    PSI >= 0.25: Significant change
    
    Args:
        expected: Expected/baseline distribution (e.g., training data)
        actual: Actual distribution (e.g., production data)
        bins: Number of bins for discretization
        
    Returns:
        PSI value
    """
    # Calculate percentile boundaries based on expected distribution
    breakpoints = np.percentile(expected, np.linspace(0, 100, bins + 1))
    breakpoints[0] = -np.inf
    breakpoints[-1] = np.inf

    # Calculate proportions in each bin
    expected_counts = np.histogram(expected, bins=breakpoints)[0]
    actual_counts = np.histogram(actual, bins=breakpoints)[0]

    # Convert to proportions
    expected_prop = expected_counts / len(expected)
    actual_prop = actual_counts / len(actual)

    # Avoid division by zero and log(0)
    expected_prop = np.clip(expected_prop, 1e-10, 1)
    actual_prop = np.clip(actual_prop, 1e-10, 1)

    # Calculate PSI
    psi = np.sum((actual_prop - expected_prop) * np.log(actual_prop / expected_prop))

    return float(psi)


def calculate_feature_drift(
    baseline_data: pd.DataFrame,
    current_data: pd.DataFrame,
    feature_columns: List[str],
) -> Dict[str, Dict[str, Any]]:
    """
    Calculate drift metrics for each feature.
    
    Args:
        baseline_data: Baseline/training data
        current_data: Current/production data
        feature_columns: List of feature column names to analyze
        
    Returns:
        Dict mapping feature names to drift metrics
    """
    drift_metrics = {}

    for col in feature_columns:
        if col not in baseline_data.columns or col not in current_data.columns:
            continue

        baseline = baseline_data[col].dropna()
        current = current_data[col].dropna()

        if len(baseline) == 0 or len(current) == 0:
            continue

        feature_drift = {
            "baseline_mean": float(baseline.mean()) if pd.api.types.is_numeric_dtype(baseline) else None,
            "current_mean": float(current.mean()) if pd.api.types.is_numeric_dtype(current) else None,
            "baseline_std": float(baseline.std()) if pd.api.types.is_numeric_dtype(baseline) else None,
            "current_std": float(current.std()) if pd.api.types.is_numeric_dtype(current) else None,
        }

        # Calculate PSI for numeric features
        if pd.api.types.is_numeric_dtype(baseline):
            try:
                feature_drift["psi"] = calculate_psi(baseline.values, current.values)
            except Exception:
                pass

        drift_metrics[col] = feature_drift

    return drift_metrics
