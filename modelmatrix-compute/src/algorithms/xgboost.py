"""XGBoost algorithm implementation."""
from typing import Dict, Any
import pandas as pd
from xgboost import XGBClassifier, XGBRegressor

from src.algorithms.base import BaseAlgorithm


class XGBoostAlgorithm(BaseAlgorithm):
    """XGBoost classifier/regressor."""
    
    # Valid hyperparameters for XGBoost
    VALID_PARAMS = {
        "n_estimators", "max_depth", "learning_rate", "random_state",
        "min_child_weight", "gamma", "subsample", "colsample_bytree",
        "colsample_bylevel", "colsample_bynode", "reg_alpha", "reg_lambda",
        "scale_pos_weight", "base_score", "n_jobs", "tree_method",
        "grow_policy", "max_leaves", "max_bin", "objective", "eval_metric",
        "early_stopping_rounds", "verbosity"
    }
    
    def train(self, X: pd.DataFrame, y: pd.Series, hyperparameters: Dict[str, Any], model_type: str = "classification") -> Any:
        """Train an XGBoost model."""
        # Default hyperparameters
        params = {
            "n_estimators": 100,
            "max_depth": 6,
            "learning_rate": 0.1,
            "random_state": 42,
        }
        # Only update with valid hyperparameters for XGBoost
        valid_hyperparams = {k: v for k, v in hyperparameters.items() if k in self.VALID_PARAMS}
        params.update(valid_hyperparams)
        
        # Create and train model based on model type
        if model_type == "regression":
            model = XGBRegressor(**params)
        else:
            model = XGBClassifier(**params)
        
        model.fit(X, y)
        return model
    
    def get_name(self) -> str:
        return "xgboost"
