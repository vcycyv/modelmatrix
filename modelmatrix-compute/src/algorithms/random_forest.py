"""Random Forest algorithm implementation."""
from typing import Dict, Any
import pandas as pd
from sklearn.ensemble import RandomForestClassifier, RandomForestRegressor

from src.algorithms.base import BaseAlgorithm


class RandomForestAlgorithm(BaseAlgorithm):
    """Random Forest classifier/regressor."""
    
    def train(self, X: pd.DataFrame, y: pd.Series, hyperparameters: Dict[str, Any], model_type: str = "classification") -> Any:
        """Train a Random Forest model."""
        # Default hyperparameters
        params = {
            "n_estimators": 100,
            "max_depth": 10,
            "min_samples_split": 2,
            "min_samples_leaf": 1,
            "random_state": 42,
        }
        params.update(hyperparameters)
        
        # Create and train model based on model type
        if model_type == "regression":
            model = RandomForestRegressor(**params)
        else:
            model = RandomForestClassifier(**params)
        
        model.fit(X, y)
        return model
    
    def get_name(self) -> str:
        return "random_forest"
