"""Polynomial Regression algorithm implementation."""
from typing import Dict, Any
import pandas as pd
import numpy as np
from sklearn.linear_model import LinearRegression, Ridge
from sklearn.preprocessing import PolynomialFeatures
from sklearn.pipeline import Pipeline

from src.algorithms.base import BaseAlgorithm


class PolynomialRegressionAlgorithm(BaseAlgorithm):
    """Polynomial Regression for regression tasks."""
    
    def train(self, X: pd.DataFrame, y: pd.Series, hyperparameters: Dict[str, Any], model_type: str = "regression") -> Any:
        """Train a Polynomial Regression model."""
        # Default hyperparameters
        degree = hyperparameters.get("degree", 2)
        include_bias = hyperparameters.get("include_bias", False)
        regularization = hyperparameters.get("regularization", "ridge")  # Default to ridge to prevent overfitting
        alpha = hyperparameters.get("alpha", 1.0)
        
        # Create polynomial features transformer
        poly_features = PolynomialFeatures(
            degree=degree,
            include_bias=include_bias,
            interaction_only=hyperparameters.get("interaction_only", False)
        )
        
        # Create regressor (use Ridge by default to prevent overfitting with polynomial features)
        if regularization == "ridge":
            regressor = Ridge(alpha=alpha, fit_intercept=True)
        else:
            regressor = LinearRegression(fit_intercept=True)
        
        # Create pipeline
        model = Pipeline([
            ("poly_features", poly_features),
            ("regressor", regressor)
        ])
        
        model.fit(X, y)
        return model
    
    def get_name(self) -> str:
        return "polynomial_regression"

