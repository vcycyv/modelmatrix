"""Service for saving trained models to MinIO."""
import io
import pickle
from datetime import datetime
from minio import Minio
from minio.error import S3Error
from typing import Any, List, Optional

from src.core.config import settings
from src.core.logger import logger


class ModelSaver:
    """Saves trained models to MinIO storage."""
    
    def __init__(self):
        """Initialize MinIO client."""
        self.client = Minio(
            settings.minio_endpoint,
            access_key=settings.minio_access_key,
            secret_key=settings.minio_secret_key,
            secure=settings.minio_use_ssl
        )
        self.bucket = settings.minio_bucket
    
    def save_model(
        self,
        model: Any,
        job_id: str,
        algorithm: str,
        feature_names: Optional[List[str]] = None,
        target_column: Optional[str] = None,
        model_type: Optional[str] = None,
    ) -> str:
        """
        Save a trained model to MinIO along with metadata.
        
        Args:
            model: Trained model object (scikit-learn, xgboost, etc.)
            job_id: Training job ID
            algorithm: Algorithm name
            feature_names: List of input feature names the model expects
            target_column: Name of target column (None for clustering)
            model_type: Type of model (classification, regression, clustering)
            
        Returns:
            Path to saved model in MinIO (minio://bucket/path format)
        """
        # Generate model path
        timestamp = datetime.utcnow().strftime("%Y%m%d_%H%M%S")
        model_path = f"models/{algorithm}/{job_id}_{timestamp}.pkl"
        
        try:
            logger.info(f"Saving model to MinIO: {model_path}")
            
            # Serialize model
            model_bytes = pickle.dumps(model)
            model_stream = io.BytesIO(model_bytes)
            
            # Upload model to MinIO
            self.client.put_object(
                self.bucket,
                model_path,
                model_stream,
                length=len(model_bytes),
                content_type="application/octet-stream"
            )
            
            logger.info(f"Model uses {len(feature_names) if feature_names else 0} features")
            
            # Return full path
            full_path = f"minio://{self.bucket}/{model_path}"
            logger.info(f"Model saved successfully: {full_path}")
            
            return full_path
            
        except S3Error as e:
            logger.error(f"Failed to save model to MinIO: {e}")
            raise ValueError(f"Failed to save model to MinIO: {e}")
        except Exception as e:
            logger.error(f"Error saving model: {e}")
            raise

    def save_training_code(
        self,
        job_id: str,
        algorithm: str,
        hyperparameters: dict,
        target_column: Optional[str],
        input_columns: List[str],
        model_type: str,
        file_path: str,
    ) -> str:
        """
        Generate and save Python training code to MinIO.
        
        Args:
            job_id: Training job ID
            algorithm: Algorithm name
            hyperparameters: Hyperparameters used
            target_column: Target column name
            input_columns: Input feature columns
            model_type: Model type (classification, regression, clustering)
            file_path: Path to the training data file
            
        Returns:
            Path to saved code in MinIO
        """
        timestamp = datetime.utcnow().strftime("%Y%m%d_%H%M%S")
        code_path = f"models/{algorithm}/{job_id}_{timestamp}_train.py"
        
        try:
            # Generate Python code
            code = self._generate_training_code(
                algorithm=algorithm,
                hyperparameters=hyperparameters,
                target_column=target_column,
                input_columns=input_columns,
                model_type=model_type,
                file_path=file_path,
            )
            
            logger.info(f"Saving training code to MinIO: {code_path}")
            
            # Upload to MinIO
            code_bytes = code.encode('utf-8')
            code_stream = io.BytesIO(code_bytes)
            
            self.client.put_object(
                self.bucket,
                code_path,
                code_stream,
                length=len(code_bytes),
                content_type="text/x-python"
            )
            
            full_path = f"minio://{self.bucket}/{code_path}"
            logger.info(f"Training code saved successfully: {full_path}")
            
            return full_path
            
        except Exception as e:
            logger.error(f"Error saving training code: {e}")
            raise

    def _generate_training_code(
        self,
        algorithm: str,
        hyperparameters: dict,
        target_column: Optional[str],
        input_columns: List[str],
        model_type: str,
        file_path: str,
    ) -> str:
        """Generate Python code that replicates the training process."""
        
        # Filter out internal hyperparameters (starting with _)
        clean_hyperparams = {k: v for k, v in hyperparameters.items() if not k.startswith('_')}
        
        # Algorithm-specific imports and model creation
        algo_code = self._get_algorithm_code(algorithm, model_type, clean_hyperparams)
        
        code = f'''"""
Auto-generated training code for {algorithm} model.
Model Type: {model_type}
Generated by ModelMatrix
"""
import pandas as pd
import pickle
from sklearn.model_selection import train_test_split
{algo_code["imports"]}

# Configuration
FILE_PATH = "{file_path}"
TARGET_COLUMN = {repr(target_column)}
INPUT_COLUMNS = {repr(input_columns)}
MODEL_TYPE = "{model_type}"
TEST_SIZE = 0.2
RANDOM_STATE = 42

# Hyperparameters
HYPERPARAMETERS = {repr(clean_hyperparams)}

def load_data(file_path: str) -> pd.DataFrame:
    """Load data from file."""
    if file_path.endswith('.parquet'):
        return pd.read_parquet(file_path)
    elif file_path.endswith('.csv'):
        return pd.read_csv(file_path)
    else:
        raise ValueError(f"Unsupported file format: {{file_path}}")

def prepare_features(df: pd.DataFrame):
    """Prepare features for training."""
    X = df[INPUT_COLUMNS].copy()
    
    # Handle categorical columns with one-hot encoding
    categorical_cols = X.select_dtypes(include=['object', 'category']).columns.tolist()
    if categorical_cols:
        X = pd.get_dummies(X, columns=categorical_cols, drop_first=False)
    
    # Fill missing values
    for col in X.columns:
        if X[col].dtype in ['float64', 'int64']:
            X[col] = X[col].fillna(X[col].median())
        else:
            X[col] = X[col].fillna(X[col].mode().iloc[0] if not X[col].mode().empty else 0)
    
    {"y = None" if model_type == "clustering" else "y = df[TARGET_COLUMN]"}
    return X, y

def train():
    """Train the model."""
    # Load data
    print(f"Loading data from {{FILE_PATH}}...")
    df = load_data(FILE_PATH)
    print(f"Loaded {{len(df)}} rows")
    
    # Prepare features
    X, y = prepare_features(df)
    print(f"Prepared {{X.shape[1]}} features")
    
{self._get_split_code(model_type)}
    
    # Create and train model
    print("Training model...")
{algo_code["model_creation"]}
{algo_code["fit_code"]}
    
    print("Training complete!")
    return model

def save_model(model, output_path: str = "model.pkl"):
    """Save the trained model."""
    with open(output_path, 'wb') as f:
        pickle.dump(model, f)
    print(f"Model saved to {{output_path}}")

if __name__ == "__main__":
    model = train()
    save_model(model)
'''
        return code

    def _get_algorithm_code(self, algorithm: str, model_type: str, hyperparams: dict) -> dict:
        """Get algorithm-specific code snippets."""
        
        if algorithm == "random_forest":
            if model_type == "regression":
                imports = "from sklearn.ensemble import RandomForestRegressor"
                model_class = "RandomForestRegressor"
            else:
                imports = "from sklearn.ensemble import RandomForestClassifier"
                model_class = "RandomForestClassifier"
            
            params = {"n_estimators": 100, "max_depth": 10, "random_state": 42}
            params.update(hyperparams)
            
            return {
                "imports": imports,
                "model_creation": f"    model = {model_class}(**{repr(params)})",
                "fit_code": "    model.fit(X_train, y_train)",
            }
        
        elif algorithm == "decision_tree":
            if model_type == "regression":
                imports = "from sklearn.tree import DecisionTreeRegressor"
                model_class = "DecisionTreeRegressor"
            else:
                imports = "from sklearn.tree import DecisionTreeClassifier"
                model_class = "DecisionTreeClassifier"
            
            params = {"max_depth": 10, "random_state": 42}
            params.update(hyperparams)
            
            return {
                "imports": imports,
                "model_creation": f"    model = {model_class}(**{repr(params)})",
                "fit_code": "    model.fit(X_train, y_train)",
            }
        
        elif algorithm == "xgboost":
            if model_type == "regression":
                imports = "from xgboost import XGBRegressor"
                model_class = "XGBRegressor"
            else:
                imports = "from xgboost import XGBClassifier"
                model_class = "XGBClassifier"
            
            params = {"n_estimators": 100, "max_depth": 6, "learning_rate": 0.1, "random_state": 42}
            params.update(hyperparams)
            
            return {
                "imports": imports,
                "model_creation": f"    model = {model_class}(**{repr(params)})",
                "fit_code": "    model.fit(X_train, y_train)",
            }
        
        elif algorithm == "linear_regression":
            regularization = hyperparams.pop("regularization", "none")
            if regularization == "ridge":
                imports = "from sklearn.linear_model import Ridge"
                model_class = "Ridge"
            elif regularization == "lasso":
                imports = "from sklearn.linear_model import Lasso"
                model_class = "Lasso"
            else:
                imports = "from sklearn.linear_model import LinearRegression"
                model_class = "LinearRegression"
            
            params = {}
            if regularization in ["ridge", "lasso"]:
                params["alpha"] = hyperparams.get("alpha", 1.0)
            
            return {
                "imports": imports,
                "model_creation": f"    model = {model_class}(**{repr(params)})" if params else f"    model = {model_class}()",
                "fit_code": "    model.fit(X_train, y_train)",
            }
        
        elif algorithm == "polynomial_regression":
            imports = """from sklearn.preprocessing import PolynomialFeatures
from sklearn.pipeline import Pipeline
from sklearn.linear_model import Ridge, LinearRegression"""
            
            degree = hyperparams.get("degree", 2)
            regularization = hyperparams.get("regularization", "ridge")
            alpha = hyperparams.get("alpha", 1.0)
            
            if regularization == "ridge":
                regressor = f"Ridge(alpha={alpha})"
            else:
                regressor = "LinearRegression()"
            
            return {
                "imports": imports,
                "model_creation": f"""    model = Pipeline([
        ('poly_features', PolynomialFeatures(degree={degree}, include_bias=False)),
        ('regressor', {regressor})
    ])""",
                "fit_code": "    model.fit(X_train, y_train)",
            }
        
        elif algorithm == "kmeans":
            imports = "from sklearn.cluster import KMeans"
            
            params = {"n_clusters": 3, "random_state": 42}
            params.update(hyperparams)
            
            return {
                "imports": imports,
                "model_creation": f"    model = KMeans(**{repr(params)})",
                "fit_code": "    model.fit(X_train)",
            }
        
        else:
            # Generic fallback
            return {
                "imports": "# Unknown algorithm - customize as needed",
                "model_creation": f"    # model = create_{algorithm}_model(HYPERPARAMETERS)",
                "fit_code": "    # model.fit(X_train, y_train)",
            }

    def _get_split_code(self, model_type: str) -> str:
        """Get train/test split code based on model type."""
        if model_type == "clustering":
            return """    # Split data (no target for clustering)
    X_train, X_test = train_test_split(X, test_size=TEST_SIZE, random_state=RANDOM_STATE)
    print(f"Train: {len(X_train)}, Test: {len(X_test)}")"""
        else:
            return """    # Split data
    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=TEST_SIZE, random_state=RANDOM_STATE
    )
    print(f"Train: {len(X_train)}, Test: {len(X_test)}")"""
