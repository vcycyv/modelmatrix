"""Logging configuration."""
import logging
import sys
from pythonjsonlogger import jsonlogger


def setup_logger(level: str = "info") -> logging.Logger:
    """Setup JSON logger for the application."""
    logger = logging.getLogger("modelmatrix-compute")
    logger.setLevel(getattr(logging, level.upper()))
    
    # Remove existing handlers
    logger.handlers = []
    
    # Create JSON formatter
    formatter = jsonlogger.JsonFormatter(
        "%(asctime)s %(name)s %(levelname)s %(message)s"
    )
    
    # Console handler
    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(formatter)
    logger.addHandler(handler)
    
    return logger


# Global logger instance
logger = setup_logger()


