"""XGBoost algorithm implementation."""
from typing import Dict, Any
import pandas as pd
from xgboost import XGBClassifier, XGBRegressor

from src.algorithms.base import BaseAlgorithm


class XGBoostAlgorithm(BaseAlgorithm):
    """XGBoost classifier/regressor."""
    
    def train(self, X: pd.DataFrame, y: pd.Series, hyperparameters: Dict[str, Any], model_type: str = "classification") -> Any:
        """Train an XGBoost model."""
        # Default hyperparameters
        params = {
            "n_estimators": 100,
            "max_depth": 6,
            "learning_rate": 0.1,
            "random_state": 42,
        }
        params.update(hyperparameters)
        
        # Create and train model based on model type
        if model_type == "regression":
            model = XGBRegressor(**params)
        else:
            model = XGBClassifier(**params)
        
        model.fit(X, y)
        return model
    
    def get_name(self) -> str:
        return "xgboost"
