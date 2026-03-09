"""Service for loading data from MinIO."""
import io
import pickle
import pandas as pd
from minio import Minio
from minio.error import S3Error
from typing import Any, Tuple

from src.core.config import settings
from src.core.logger import logger


class DataLoader:
    """Loads data files from MinIO storage."""
    
    def __init__(self):
        """Initialize MinIO client."""
        self.client = Minio(
            settings.minio_endpoint,
            access_key=settings.minio_access_key,
            secret_key=settings.minio_secret_key,
            secure=settings.minio_use_ssl
        )
        self.bucket = settings.minio_bucket

    def load_data(self, file_path: str) -> pd.DataFrame:
        """
        Load a data file from MinIO by extension (.csv or .parquet).

        Args:
            file_path: Path in format "minio://bucket/path" or "path/to/file.csv" (or .parquet)

        Returns:
            pandas DataFrame
        """
        path = file_path.strip().lower()
        if path.endswith(".parquet"):
            return self.load_parquet(file_path)
        if path.endswith(".csv"):
            return self.load_csv(file_path)
        raise ValueError(f"Unsupported file format for {file_path}. Use .parquet or .csv")

    def load_model(self, model_file_path: str) -> Any:
        """
        Load a trained model (pickle) from MinIO.

        Args:
            model_file_path: Path in format "minio://bucket/path" or "path/to/file.pkl"

        Returns:
            Unpickled model object (may be raw model or dict with "model" / "preprocessor" keys)
        """
        if model_file_path.startswith("minio://"):
            path = model_file_path.replace(f"minio://{self.bucket}/", "").lstrip("/")
        else:
            path = model_file_path.lstrip("/")

        try:
            logger.info(f"Loading model from MinIO: {path}")
            response = self.client.get_object(self.bucket, path)
            model_bytes = response.read()
            response.close()
            response.release_conn()
            obj = pickle.loads(model_bytes)
            logger.info("Model loaded successfully")
            return obj
        except S3Error as e:
            logger.error(f"Failed to load model from MinIO: {e}")
            raise ValueError(f"Failed to load model from MinIO: {e}") from e
        except Exception as e:
            logger.error(f"Error loading model: {e}")
            raise

    def load_parquet(self, file_path: str) -> pd.DataFrame:
        """
        Load a Parquet file from MinIO.
        
        Args:
            file_path: Path in format "minio://bucket/path" or just "path/to/file.parquet"
            
        Returns:
            pandas DataFrame
        """
        # Extract path from minio:// URL if present
        if file_path.startswith("minio://"):
            path = file_path.replace(f"minio://{self.bucket}/", "").lstrip("/")
        else:
            path = file_path.lstrip("/")
        
        try:
            logger.info(f"Loading Parquet file from MinIO: {path}")
            
            # Get object from MinIO
            response = self.client.get_object(self.bucket, path)
            
            # Read into pandas
            df = pd.read_parquet(io.BytesIO(response.read()))
            
            response.close()
            response.release_conn()
            
            logger.info(f"Loaded {len(df)} rows, {len(df.columns)} columns")
            return df
            
        except S3Error as e:
            logger.error(f"Failed to load file from MinIO: {e}")
            raise ValueError(f"Failed to load file from MinIO: {e}")
        except Exception as e:
            logger.error(f"Error loading Parquet file: {e}")
            raise
    
    def load_csv(self, file_path: str) -> pd.DataFrame:
        """
        Load a CSV file from MinIO.
        
        Args:
            file_path: Path in format "minio://bucket/path" or just "path/to/file.csv"
            
        Returns:
            pandas DataFrame
        """
        # Extract path from minio:// URL if present
        if file_path.startswith("minio://"):
            path = file_path.replace(f"minio://{self.bucket}/", "").lstrip("/")
        else:
            path = file_path.lstrip("/")
        
        try:
            logger.info(f"Loading CSV file from MinIO: {path}")
            
            # Get object from MinIO
            response = self.client.get_object(self.bucket, path)
            
            # Read into pandas
            df = pd.read_csv(io.BytesIO(response.read()))
            
            response.close()
            response.release_conn()
            
            logger.info(f"Loaded {len(df)} rows, {len(df.columns)} columns")
            return df
            
        except S3Error as e:
            logger.error(f"Failed to load file from MinIO: {e}")
            raise ValueError(f"Failed to load file from MinIO: {e}")
        except Exception as e:
            logger.error(f"Error loading CSV file: {e}")
            raise
    
    def prepare_features(self, df: pd.DataFrame, target_column: str, input_columns: list) -> Tuple[pd.DataFrame, pd.Series]:
        """
        Prepare features and target from DataFrame.
        
        Args:
            df: Input DataFrame
            target_column: Name of target column
            input_columns: List of input feature columns
            
        Returns:
            Tuple of (X, y) where X is features DataFrame and y is target Series
        """
        # Validate columns exist
        missing_cols = [col for col in [target_column] + input_columns if col not in df.columns]
        if missing_cols:
            raise ValueError(f"Missing columns: {missing_cols}")
        
        # Extract features and target
        X = df[input_columns].copy()
        y = df[target_column].copy()
        
        # Handle missing values - fill numeric with median, categorical with mode
        for col in X.columns:
            if X[col].dtype == 'object' or X[col].dtype.name == 'category':
                # Fill categorical with mode (most frequent value)
                mode_val = X[col].mode()
                if len(mode_val) > 0:
                    X[col] = X[col].fillna(mode_val[0])
                else:
                    X[col] = X[col].fillna('missing')
            else:
                # Fill numeric with median
                X[col] = X[col].fillna(X[col].median())
        
        # Handle missing values in target
        if y.dtype == 'object' or y.dtype.name == 'category':
            mode_val = y.mode()
            if len(mode_val) > 0:
                y = y.fillna(mode_val[0])
        else:
            y = y.fillna(y.median())
        
        # Encode categorical features using one-hot encoding
        categorical_cols = X.select_dtypes(include=['object', 'category']).columns.tolist()
        if categorical_cols:
            logger.info(f"Encoding {len(categorical_cols)} categorical columns: {categorical_cols}")
            X = pd.get_dummies(X, columns=categorical_cols, drop_first=True, dtype=float)
        
        logger.info(f"Prepared features: {len(X)} samples, {len(X.columns)} features (after encoding)")
        
        return X, y

    def prepare_features_unsupervised(self, df: pd.DataFrame, input_columns: list) -> pd.DataFrame:
        """
        Prepare features for unsupervised learning (no target column).
        
        Args:
            df: Input DataFrame
            input_columns: List of input feature columns
            
        Returns:
            Features DataFrame
        """
        # Validate columns exist
        missing_cols = [col for col in input_columns if col not in df.columns]
        if missing_cols:
            raise ValueError(f"Missing columns: {missing_cols}")
        
        # Extract features
        X = df[input_columns].copy()
        
        # Handle missing values - fill numeric with median, categorical with mode
        for col in X.columns:
            if X[col].dtype == 'object' or X[col].dtype.name == 'category':
                # Fill categorical with mode (most frequent value)
                mode_val = X[col].mode()
                if len(mode_val) > 0:
                    X[col] = X[col].fillna(mode_val[0])
                else:
                    X[col] = X[col].fillna('missing')
            else:
                # Fill numeric with median
                X[col] = X[col].fillna(X[col].median())
        
        # Encode categorical features using one-hot encoding
        categorical_cols = X.select_dtypes(include=['object', 'category']).columns.tolist()
        if categorical_cols:
            logger.info(f"Encoding {len(categorical_cols)} categorical columns: {categorical_cols}")
            X = pd.get_dummies(X, columns=categorical_cols, drop_first=True, dtype=float)
        
        logger.info(f"Prepared features: {len(X)} samples, {len(X.columns)} features (after encoding)")
        
        return X

