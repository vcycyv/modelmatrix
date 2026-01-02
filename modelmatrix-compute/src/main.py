"""FastAPI application entry point."""
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from src.core.config import settings
from src.core.logger import logger, setup_logger
from src.api.routes import router

# Setup logger with configured level
setup_logger(settings.log_level)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Lifespan event handler for startup and shutdown."""
    # Startup
    logger.info("ModelMatrix Compute Service starting up")
    logger.info(f"MinIO endpoint: {settings.minio_endpoint}")
    logger.info(f"Service listening on {settings.compute_host}:{settings.compute_port}")
    yield
    # Shutdown
    logger.info("ModelMatrix Compute Service shutting down")


# Create FastAPI app with lifespan
app = FastAPI(
    title="ModelMatrix Compute Service",
    description="ML model training service for ModelMatrix",
    version="1.0.0",
    lifespan=lifespan,
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # In production, specify allowed origins
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include routers
app.include_router(router)


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "src.main:app",
        host=settings.compute_host,
        port=settings.compute_port,
        reload=True,
        log_level=settings.log_level,
    )


