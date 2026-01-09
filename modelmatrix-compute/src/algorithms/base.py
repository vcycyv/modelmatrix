"""Base interface for ML algorithms."""
from abc import ABC, abstractmethod
from typing import Dict, Any, List, Optional
import pandas as pd
from sklearn.metrics import (
    accuracy_score, precision_score, recall_score, f1_score, confusion_matrix,
    mean_squared_error, mean_absolute_error, r2_score
)
from sklearn.inspection import permutation_importance
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
    
    def get_feature_importances(
        self,
        model: Any,
        feature_names: List[str],
        X: Optional[pd.DataFrame] = None,
        y: Optional[pd.Series] = None,
    ) -> Dict[str, float]:
        """
        Extract feature importances from the trained model.
        
        For tree-based models: Uses feature_importances_ attribute
        For L1-regularized linear models: Uses absolute coef_ values
        For other models: Uses permutation importance (requires X, y)
        
        Handles one-hot encoded columns by aggregating back to original column names.
        
        Args:
            model: Trained model
            feature_names: List of feature names (original input columns)
            X: Test features (preprocessed, may have one-hot encoded columns)
            y: Test targets (required for permutation importance)
            
        Returns:
            Dictionary mapping feature names to importance scores (0.0 = unused)
        """
        # Get column names from X if available (these are the preprocessed column names)
        preprocessed_cols = list(X.columns) if X is not None else feature_names
        
        # Get raw importances from model
        raw_importances: Optional[np.ndarray] = None
        
        # Method 1: Tree-based models have feature_importances_
        if hasattr(model, 'feature_importances_'):
            raw_importances = model.feature_importances_
        
        # Method 2: Linear models with coef_
        elif hasattr(model, 'coef_'):
            coef = np.abs(model.coef_)
            if coef.ndim > 1:
                coef = np.mean(coef, axis=0)  # Multi-class: average across classes
            raw_importances = coef
        
        # Method 3: Permutation importance (model-agnostic)
        elif X is not None and y is not None:
            try:
                result = permutation_importance(model, X, y, n_repeats=10, random_state=42, n_jobs=-1)
                raw_importances = result.importances_mean
            except Exception:
                pass
        
        # If we got raw importances, aggregate to original column names
        if raw_importances is not None and len(raw_importances) == len(preprocessed_cols):
            return self._aggregate_importances(raw_importances, preprocessed_cols, feature_names)
        
        # Fallback: All features equally important (unknown)
        return {name: 1.0 / len(feature_names) for name in feature_names}
    
    def _aggregate_importances(
        self,
        raw_importances: np.ndarray,
        preprocessed_cols: List[str],
        original_cols: List[str],
    ) -> Dict[str, float]:
        """
        Aggregate importances from preprocessed columns back to original columns.
        
        One-hot encoded columns (e.g., gender_M, gender_F) are summed back to
        their original column (gender).
        
        Args:
            raw_importances: Array of importances for preprocessed columns
            preprocessed_cols: Column names after preprocessing (one-hot encoding)
            original_cols: Original column names before preprocessing
            
        Returns:
            Dictionary mapping original column names to aggregated importance
        """
        importances = {col: 0.0 for col in original_cols}
        
        for col_name, imp in zip(preprocessed_cols, raw_importances):
            # Find which original column this preprocessed column belongs to
            matched = False
            for orig_col in original_cols:
                # Check if preprocessed col is the original or a one-hot variant
                # One-hot columns are named like: original_value (e.g., gender_M)
                if col_name == orig_col or col_name.startswith(f"{orig_col}_"):
                    importances[orig_col] += float(imp)
                    matched = True
                    break
            
            if not matched:
                # Column not matched - shouldn't happen normally
                pass
        
        return importances
