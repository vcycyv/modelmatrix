"""Service for scoring data using trained models."""
import io
import pickle
import uuid
from typing import Dict, Any, Optional, List
import pandas as pd
from minio import Minio
from minio.error import S3Error
import httpx

from src.core.config import settings
from src.core.logger import logger


class ModelScorer:
    """Loads models and scores data."""

    def __init__(self):
        """Initialize MinIO client."""
        self.client = Minio(
            settings.minio_endpoint,
            access_key=settings.minio_access_key,
            secret_key=settings.minio_secret_key,
            secure=settings.minio_use_ssl
        )
        self.bucket = settings.minio_bucket
        self._http_client = httpx.AsyncClient(timeout=30.0)

    def load_model(self, model_file_path: str) -> Any:
        """
        Load a trained model from MinIO.

        Args:
            model_file_path: Path in format "minio://bucket/path" or just "path/to/file.pkl"

        Returns:
            Loaded model object
        """
        # Extract path from minio:// URL if present
        if model_file_path.startswith("minio://"):
            path = model_file_path.replace(f"minio://{self.bucket}/", "").lstrip("/")
        else:
            path = model_file_path.lstrip("/")

        try:
            logger.info(f"Loading model from MinIO: {path}")

            # Get object from MinIO
            response = self.client.get_object(self.bucket, path)
            model_bytes = response.read()
            response.close()
            response.release_conn()

            # Deserialize model
            model = pickle.loads(model_bytes)
            logger.info(f"Model loaded successfully")
            return model

        except S3Error as e:
            logger.error(f"Failed to load model from MinIO: {e}")
            raise ValueError(f"Failed to load model from MinIO: {e}")
        except Exception as e:
            logger.error(f"Error loading model: {e}")
            raise

    def load_data(self, file_path: str) -> pd.DataFrame:
        """Load data file from MinIO."""
        # Extract path from minio:// URL if present
        if file_path.startswith("minio://"):
            path = file_path.replace(f"minio://{self.bucket}/", "").lstrip("/")
        else:
            path = file_path.lstrip("/")

        try:
            logger.info(f"Loading data from MinIO: {path}")
            response = self.client.get_object(self.bucket, path)

            if path.endswith(".parquet"):
                df = pd.read_parquet(io.BytesIO(response.read()))
            elif path.endswith(".csv"):
                df = pd.read_csv(io.BytesIO(response.read()))
            else:
                raise ValueError(f"Unsupported file format: {path}")

            response.close()
            response.release_conn()

            logger.info(f"Loaded {len(df)} rows, {len(df.columns)} columns")
            return df

        except S3Error as e:
            logger.error(f"Failed to load data from MinIO: {e}")
            raise ValueError(f"Failed to load data from MinIO: {e}")

    def prepare_features(self, df: pd.DataFrame, input_columns: List[str]) -> pd.DataFrame:
        """
        Prepare features for scoring (same preprocessing as training).

        Args:
            df: Input DataFrame
            input_columns: List of input feature columns

        Returns:
            Preprocessed features DataFrame
        """
        # Validate columns exist
        missing_cols = [col for col in input_columns if col not in df.columns]
        if missing_cols:
            raise ValueError(f"Missing columns in input data: {missing_cols}")

        # Extract features
        X = df[input_columns].copy()

        # Handle missing values - fill numeric with median, categorical with mode
        for col in X.columns:
            if X[col].dtype == 'object' or X[col].dtype.name == 'category':
                mode_val = X[col].mode()
                if len(mode_val) > 0:
                    X[col] = X[col].fillna(mode_val[0])
                else:
                    X[col] = X[col].fillna('missing')
            else:
                X[col] = X[col].fillna(X[col].median())

        # Encode categorical features using one-hot encoding
        categorical_cols = X.select_dtypes(include=['object', 'category']).columns.tolist()
        if categorical_cols:
            logger.info(f"Encoding {len(categorical_cols)} categorical columns: {categorical_cols}")
            X = pd.get_dummies(X, columns=categorical_cols, drop_first=True, dtype=float)

        logger.info(f"Prepared features: {len(X)} samples, {len(X.columns)} features")
        return X

    def save_results(self, df: pd.DataFrame, output_path: str) -> str:
        """
        Save scored results to MinIO.

        Args:
            df: DataFrame with predictions
            output_path: Path in MinIO to save results

        Returns:
            Full path to saved file
        """
        try:
            logger.info(f"Saving scored results to MinIO: {output_path}")

            # Convert to parquet bytes
            buffer = io.BytesIO()
            df.to_parquet(buffer, index=False)
            buffer.seek(0)
            data_bytes = buffer.getvalue()

            # Upload to MinIO
            self.client.put_object(
                self.bucket,
                output_path,
                io.BytesIO(data_bytes),
                length=len(data_bytes),
                content_type="application/octet-stream"
            )

            full_path = f"minio://{self.bucket}/{output_path}"
            logger.info(f"Results saved: {full_path}")
            return full_path

        except S3Error as e:
            logger.error(f"Failed to save results to MinIO: {e}")
            raise ValueError(f"Failed to save results to MinIO: {e}")

    def score(
        self,
        model_file_path: str,
        input_file_path: str,
        output_path: str,
        input_columns: List[str],
        model_type: str,
    ) -> Dict[str, Any]:
        """
        Score data using a trained model.

        Args:
            model_file_path: Path to model file in MinIO
            input_file_path: Path to input data in MinIO
            output_path: Path to save results in MinIO
            input_columns: List of input feature columns
            model_type: Model type (classification, regression, clustering)

        Returns:
            Dictionary with scoring results
        """
        job_id = str(uuid.uuid4())
        logger.info(f"Starting scoring job {job_id}")

        try:
            # Load model
            model = self.load_model(model_file_path)

            # Load input data
            df = self.load_data(input_file_path)

            # Prepare features
            X = self.prepare_features(df, input_columns)

            # Make predictions
            logger.info(f"Scoring {len(X)} samples...")
            predictions = model.predict(X)

            # Build output DataFrame
            output_df = df.copy()

            if model_type == "classification":
                output_df["prediction"] = predictions
                # Add probability if available
                if hasattr(model, "predict_proba"):
                    try:
                        probas = model.predict_proba(X)
                        # For binary classification, use probability of positive class
                        if probas.shape[1] == 2:
                            output_df["probability"] = probas[:, 1]
                        else:
                            # For multi-class, add max probability
                            output_df["probability"] = probas.max(axis=1)
                    except Exception as e:
                        logger.warning(f"Could not get prediction probabilities: {e}")
            elif model_type == "regression":
                output_df["prediction"] = predictions
            elif model_type == "clustering":
                output_df["cluster"] = predictions

            # Save results
            output_file_path = self.save_results(output_df, output_path)

            logger.info(f"Scoring completed. Job ID: {job_id}")

            return {
                "job_id": job_id,
                "status": "completed",
                "output_file_path": output_file_path,
                "row_count": len(output_df),
                "error": None,
            }

        except Exception as e:
            logger.error(f"Scoring failed for job {job_id}: {e}", exc_info=True)
            return {
                "job_id": job_id,
                "status": "failed",
                "output_file_path": None,
                "row_count": 0,
                "error": str(e),
            }

    async def score_and_notify(
        self,
        model_file_path: str,
        input_file_path: str,
        output_path: str,
        input_columns: List[str],
        model_type: str,
        callback_url: Optional[str] = None,
        model_id: Optional[str] = None,
    ) -> Dict[str, Any]:
        """
        Score data and send callback notification when complete.
        """
        # Run scoring
        result = self.score(
            model_file_path=model_file_path,
            input_file_path=input_file_path,
            output_path=output_path,
            input_columns=input_columns,
            model_type=model_type,
        )

        # Send callback if URL provided
        if callback_url:
            await self._send_callback(callback_url, model_id, result)

        return result

    async def _send_callback(
        self,
        callback_url: str,
        model_id: Optional[str],
        result: Dict[str, Any],
    ) -> None:
        """Send callback to backend with scoring results."""
        try:
            payload = {
                "model_id": model_id,
                "job_id": result.get("job_id"),
                "status": result.get("status", "failed"),
                "output_file_path": result.get("output_file_path"),
                "row_count": result.get("row_count", 0),
                "error": result.get("error"),
            }

            logger.info(f"Sending scoring callback to {callback_url} for model {model_id}")
            response = await self._http_client.post(callback_url, json=payload)

            if response.status_code == 200:
                logger.info(f"Scoring callback successful for model {model_id}")
            else:
                logger.warning(f"Scoring callback returned status {response.status_code}: {response.text}")

        except Exception as e:
            logger.error(f"Failed to send scoring callback for model {model_id}: {e}")
