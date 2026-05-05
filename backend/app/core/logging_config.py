"""
Structured logging configuration using structlog.

Provides JSON-formatted logs with context for production observability.
Falls back gracefully to standard logging if structlog is not installed.
"""
import logging
import sys

try:
    import structlog
    STRUCTLOG_AVAILABLE = True
except ImportError:
    STRUCTLOG_AVAILABLE = False


def setup_logging(environment: str = "development"):
    """Configure structured logging for the application."""
    if not STRUCTLOG_AVAILABLE:
        # Basic logging fallback
        logging.basicConfig(
            level=logging.INFO,
            format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
            stream=sys.stderr,
        )
        return

    if environment == "development":
        # Human-readable console output
        structlog.configure(
            processors=[
                structlog.stdlib.filter_by_level,
                structlog.stdlib.add_logger_name,
                structlog.stdlib.add_log_level,
                structlog.stdlib.PositionalArgumentsFormatter(),
                structlog.processors.TimeStamper(fmt="%H:%M:%S"),
                structlog.dev.ConsoleRenderer(colors=True),
            ],
            context_class=dict,
            logger_factory=structlog.stdlib.LoggerFactory(),
            cache_logger_on_first_use=True,
        )
    else:
        # JSON output for production (ELK/Splunk/etc.)
        structlog.configure(
            processors=[
                structlog.stdlib.filter_by_level,
                structlog.stdlib.add_logger_name,
                structlog.stdlib.add_log_level,
                structlog.stdlib.PositionalArgumentsFormatter(),
                structlog.processors.TimeStamper(fmt="iso"),
                structlog.processors.StackInfoRenderer(),
                structlog.processors.format_exc_info,
                structlog.processors.UnicodeDecoder(),
                structlog.processors.JSONRenderer(),
            ],
            context_class=dict,
            logger_factory=structlog.stdlib.LoggerFactory(),
            cache_logger_on_first_use=True,
        )


def get_logger(name: str = __name__):
    """Get a structured logger instance."""
    if STRUCTLOG_AVAILABLE:
        return structlog.get_logger(name)
    else:
        return logging.getLogger(name)
