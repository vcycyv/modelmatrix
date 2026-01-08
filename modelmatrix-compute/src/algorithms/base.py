"""Base interface for ML algorithms."""
from abc import ABC, abstractmethod
from typing import Dict, Any
import pandas as pd
from sklearn.metrics import (
    accuracy_score, precision_score, recall_score, f1_score, confusion_matrix,
    mean_squared_error, mean_absolute_error, r2_score
)
import numpy as np


class BaseAlgorithm(ABC):
    """Base class for all ML algorithms."""
    
    @abstractmethod
    def train(self, X: pd.DataFrame, y: pd.Series, hyperparameters: Dict[str, Any], model_type: str = "classification") -> Any:
        """
        Train the model.
        
        Args:
            X: Feature DataFrame
            y: Target Series
            hyperparameters: Algorithm-specific hyperparameters
            model_type: "classification" or "regression"
            
        Returns:
            Trained model object
        """
        pass
    
    @abstractmethod
    def get_name(self) -> str:
        """Get algorithm name."""
        pass
    
    def evaluate(self, model: Any, X_test: pd.DataFrame, y_test: pd.Series, model_type: str = "classification") -> Dict[str, Any]:
        """
        Evaluate model and return metrics.
        
        Args:
            model: Trained model
            X_test: Test features
            y_test: Test targets
            model_type: "classification" or "regression"
            
        Returns:
            Dictionary of metrics
        """
        # Predict
        y_pred = model.predict(X_test)
        
        if model_type == "regression":
            # Regression metrics
            metrics = {
                "mse": float(mean_squared_error(y_test, y_pred)),
                "rmse": float(np.sqrt(mean_squared_error(y_test, y_pred))),
                "mae": float(mean_absolute_error(y_test, y_pred)),
                "r2": float(r2_score(y_test, y_pred)),
            }
        else:
            # Classification metrics
            metrics = {
                "accuracy": float(accuracy_score(y_test, y_pred)),
                "precision": float(precision_score(y_test, y_pred, average="weighted", zero_division=0)),
                "recall": float(recall_score(y_test, y_pred, average="weighted", zero_division=0)),
                "f1_score": float(f1_score(y_test, y_pred, average="weighted", zero_division=0)),
            }
            
            # Confusion matrix
            cm = confusion_matrix(y_test, y_pred)
            metrics["confusion_matrix"] = cm.tolist()
        
        return metrics
