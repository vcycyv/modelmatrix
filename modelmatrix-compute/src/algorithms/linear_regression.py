"""Linear Regression algorithm implementation."""
from typing import Dict, Any
import pandas as pd
from sklearn.linear_model import LinearRegression, Ridge, Lasso

from src.algorithms.base import BaseAlgorithm


class LinearRegressionAlgorithm(BaseAlgorithm):
    """Linear Regression for regression tasks."""
    
    def train(self, X: pd.DataFrame, y: pd.Series, hyperparameters: Dict[str, Any], model_type: str = "regression") -> Any:
        """Train a Linear Regression model."""
        # Get regularization type (none, ridge, lasso)
        regularization = hyperparameters.pop("regularization", "none")
        
        # Default hyperparameters for regularized versions
        params = {
            "fit_intercept": True,
        }
        
        if regularization in ["ridge", "lasso"]:
            params["alpha"] = hyperparameters.get("alpha", 1.0)
        
        params.update(hyperparameters)
        
        # Create model based on regularization
        if regularization == "ridge":
            model = Ridge(**params)
        elif regularization == "lasso":
            model = Lasso(**params)
        else:
            # Remove alpha if present for basic LinearRegression
            params.pop("alpha", None)
            model = LinearRegression(**params)
        
        model.fit(X, y)
        return model
    
    def get_name(self) -> str:
        return "linear_regression"

