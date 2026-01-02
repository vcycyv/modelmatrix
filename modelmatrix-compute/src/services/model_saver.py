"""Service for saving trained models to MinIO."""
import io
import pickle
from datetime import datetime
from minio import Minio
from minio.error import S3Error
from typing import Any

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
    
    def save_model(self, model: Any, job_id: str, algorithm: str) -> str:
        """
        Save a trained model to MinIO.
        
        Args:
            model: Trained model object (scikit-learn, xgboost, etc.)
            job_id: Training job ID
            algorithm: Algorithm name
            
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
            
            # Upload to MinIO
            self.client.put_object(
                self.bucket,
                model_path,
                model_stream,
                length=len(model_bytes),
                content_type="application/octet-stream"
            )
            
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


