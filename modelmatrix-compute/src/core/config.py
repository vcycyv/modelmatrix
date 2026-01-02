"""Configuration management for the compute service."""
from pydantic_settings import BaseSettings
from typing import Optional


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""
    
    # MinIO Configuration
    minio_endpoint: str = "localhost:9000"
    minio_access_key: str = "minioadmin"
    minio_secret_key: str = "minioadmin123"
    minio_bucket: str = "modelmatrix"
    minio_use_ssl: bool = False
    
    # Service Configuration
    compute_host: str = "0.0.0.0"
    compute_port: int = 8081
    log_level: str = "info"
    
    # Backend API (optional, for callbacks)
    backend_api_url: Optional[str] = None
    backend_api_key: Optional[str] = None
    
    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"
        case_sensitive = False


# Global settings instance
settings = Settings()


