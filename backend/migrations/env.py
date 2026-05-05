"""
Alembic configuration
"""
from app.core.config import get_settings

settings = get_settings()

# Alembic Config object
config = type("Config", (), {
    "sqlalchemy": type("obj", (), {"url": settings.database_url}),
})()

# Metadata for autogenerate
from app.models import Base
target_metadata = Base.metadata
