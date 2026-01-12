"""Random Forest algorithm implementation."""
from typing import Dict, Any
import pandas as pd
from sklearn.ensemble import RandomForestClassifier, RandomForestRegressor

from src.algorithms.base import BaseAlgorithm


class RandomForestAlgorithm(BaseAlgorithm):
    """Random Forest classifier/regressor."""
    
    # Valid hyperparameters for RandomForest
    VALID_PARAMS = {
        "n_estimators", "max_depth", "min_samples_split", "min_samples_leaf",
        "max_features", "random_state", "criterion", "min_weight_fraction_leaf",
        "max_leaf_nodes", "min_impurity_decrease", "bootstrap", "oob_score",
        "n_jobs", "warm_start", "class_weight", "ccp_alpha", "max_samples"
    }
    
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
        # Only update with valid hyperparameters for Random Forest
        valid_hyperparams = {k: v for k, v in hyperparameters.items() if k in self.VALID_PARAMS}
        params.update(valid_hyperparams)
        
        # Create and train model based on model type
        if model_type == "regression":
            model = RandomForestRegressor(**params)
        else:
            model = RandomForestClassifier(**params)
        
        model.fit(X, y)
        return model
    
    def get_name(self) -> str:
        return "random_forest"
