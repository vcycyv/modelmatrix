"""Decision Tree algorithm implementation."""
from typing import Dict, Any
import pandas as pd
from sklearn.tree import DecisionTreeClassifier
from sklearn.model_selection import train_test_split

from src.algorithms.base import BaseAlgorithm


class DecisionTreeAlgorithm(BaseAlgorithm):
    """Decision Tree classifier."""
    
    def train(self, X: pd.DataFrame, y: pd.Series, hyperparameters: Dict[str, Any]) -> DecisionTreeClassifier:
        """Train a Decision Tree model."""
        # Default hyperparameters
        params = {
            "max_depth": 10,
            "min_samples_split": 2,
            "min_samples_leaf": 1,
            "criterion": "gini",
            "random_state": 42,
        }
        params.update(hyperparameters)
        
        # Create and train model
        model = DecisionTreeClassifier(**params)
        model.fit(X, y)
        
        return model
    
    def get_name(self) -> str:
        return "decision_tree"


