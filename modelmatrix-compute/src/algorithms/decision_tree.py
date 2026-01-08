"""Decision Tree algorithm implementation."""
from typing import Dict, Any
import pandas as pd
from sklearn.tree import DecisionTreeClassifier, DecisionTreeRegressor

from src.algorithms.base import BaseAlgorithm


class DecisionTreeAlgorithm(BaseAlgorithm):
    """Decision Tree classifier/regressor."""
    
    def train(self, X: pd.DataFrame, y: pd.Series, hyperparameters: Dict[str, Any], model_type: str = "classification") -> Any:
        """Train a Decision Tree model."""
        # Default hyperparameters
        params = {
            "max_depth": 10,
            "min_samples_split": 2,
            "min_samples_leaf": 1,
            "random_state": 42,
        }
        
        # Add criterion based on model type
        if model_type == "regression":
            params["criterion"] = "squared_error"
        else:
            params["criterion"] = "gini"
        
        params.update(hyperparameters)
        
        # Create and train model based on model type
        if model_type == "regression":
            model = DecisionTreeRegressor(**params)
        else:
            model = DecisionTreeClassifier(**params)
        
        model.fit(X, y)
        return model
    
    def get_name(self) -> str:
        return "decision_tree"
