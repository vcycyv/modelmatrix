"""K-Means clustering algorithm implementation."""
from typing import Dict, Any
import pandas as pd
import numpy as np
from sklearn.cluster import KMeans
from sklearn.metrics import silhouette_score, calinski_harabasz_score, davies_bouldin_score

from src.algorithms.base import BaseAlgorithm


class KMeansAlgorithm(BaseAlgorithm):
    """K-Means clustering algorithm."""
    
    # Valid hyperparameters for KMeans
    VALID_PARAMS = {
        "n_clusters", "init", "n_init", "max_iter", "random_state",
        "tol", "algorithm", "copy_x"
    }
    
    def train(self, X: pd.DataFrame, y: pd.Series, hyperparameters: Dict[str, Any], model_type: str = "clustering") -> Any:
        """
        Train a K-Means model.
        
        Note: For clustering, y is ignored (unsupervised learning).
        """
        # Filter to valid hyperparameters only
        valid_hyperparams = {k: v for k, v in hyperparameters.items() if k in self.VALID_PARAMS}
        
        # Default hyperparameters
        params = {
            "n_clusters": valid_hyperparams.get("n_clusters", 3),
            "init": valid_hyperparams.get("init", "k-means++"),
            "n_init": valid_hyperparams.get("n_init", 10),
            "max_iter": valid_hyperparams.get("max_iter", 300),
            "random_state": valid_hyperparams.get("random_state", 42),
        }
        
        # Create and train model
        model = KMeans(**params)
        model.fit(X)
        
        return model
    
    def evaluate(self, model: Any, X_test: pd.DataFrame, y_test: pd.Series, model_type: str = "clustering") -> Dict[str, Any]:
        """
        Evaluate clustering model using internal validation metrics.
        
        Note: For clustering, y_test is ignored.
        """
        # Get cluster labels for test data
        labels = model.predict(X_test)
        
        metrics = {}
        
        # Only calculate metrics if we have more than 1 cluster and more samples than clusters
        n_clusters = len(set(labels))
        if n_clusters > 1 and len(X_test) > n_clusters:
            try:
                # Silhouette Score: -1 to 1, higher is better
                metrics["silhouette_score"] = float(silhouette_score(X_test, labels))
            except Exception:
                metrics["silhouette_score"] = None
            
            try:
                # Calinski-Harabasz Index: higher is better
                metrics["calinski_harabasz_score"] = float(calinski_harabasz_score(X_test, labels))
            except Exception:
                metrics["calinski_harabasz_score"] = None
            
            try:
                # Davies-Bouldin Index: lower is better
                metrics["davies_bouldin_score"] = float(davies_bouldin_score(X_test, labels))
            except Exception:
                metrics["davies_bouldin_score"] = None
        
        # Add cluster distribution
        unique, counts = np.unique(labels, return_counts=True)
        metrics["cluster_distribution"] = {int(k): int(v) for k, v in zip(unique, counts)}
        metrics["n_clusters"] = n_clusters
        metrics["inertia"] = float(model.inertia_)
        
        return metrics
    
    def get_name(self) -> str:
        return "kmeans"

