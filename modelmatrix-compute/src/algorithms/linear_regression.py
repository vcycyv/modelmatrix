"""Linear Regression algorithm implementation."""
from typing import Dict, Any
import pandas as pd
from sklearn.linear_model import LinearRegression, Ridge, Lasso

from src.algorithms.base import BaseAlgorithm


class LinearRegressionAlgorithm(BaseAlgorithm):
    """Linear Regression for regression tasks."""
    
    # Valid hyperparameters for Linear Regression variants
    VALID_PARAMS = {
        "fit_intercept", "copy_X", "n_jobs", "positive",  # LinearRegression
        "alpha", "max_iter", "tol", "solver", "random_state",  # Ridge/Lasso
        "selection", "warm_start"  # Lasso
    }
    
    def train(self, X: pd.DataFrame, y: pd.Series, hyperparameters: Dict[str, Any], model_type: str = "regression") -> Any:
        """Train a Linear Regression model."""
        # Make a copy to avoid modifying the original
        hyperparameters = hyperparameters.copy()
        
        # Get regularization type (none, ridge, lasso)
        regularization = hyperparameters.pop("regularization", "none")
        
        # Default hyperparameters for regularized versions
        params = {
            "fit_intercept": True,
        }
        
        if regularization in ["ridge", "lasso"]:
            params["alpha"] = hyperparameters.get("alpha", 1.0)
        
        # Only update with valid hyperparameters
        valid_hyperparams = {k: v for k, v in hyperparameters.items() if k in self.VALID_PARAMS}
        params.update(valid_hyperparams)
        
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

