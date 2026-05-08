"""
Alembic migration environment.

Supports both PostgreSQL (via psycopg2) and SQLite (via pysqlite).
Uses sync engine since Alembic doesn't support async drivers for DDL.

Usage:
    cd backend
    alembic revision --autogenerate -m "description"
    alembic upgrade head
    alembic downgrade -1
"""
from logging.config import fileConfig

from alembic import context
from sqlalchemy import engine_from_config, pool

from app.core.config import get_settings
from app.core.database import Base

# Import all models so Alembic can detect them for autogenerate
from app.models import (  # noqa: F401
    User,
    Team,
    TeamMember,
    ApiKey,
    Provider,
    Model,
    Balance,
    Transaction,
    RoutingRule,
    AuditLog,
    RequestLog,
)

# Alembic Config object
config = context.config

# Override sqlalchemy.url from settings
settings = get_settings()
db_url = settings.database_url

# Convert async drivers to sync drivers for Alembic
sync_url = (
    db_url
    .replace("+asyncpg", "+psycopg2")
    .replace("+aiosqlite", "+pysqlite")
)
config.set_main_option("sqlalchemy.url", sync_url)

# Set up logging
if config.config_file_name is not None:
    fileConfig(config.config_file_name)

# Target metadata for autogenerate
target_metadata = Base.metadata


def run_migrations_offline() -> None:
    """Run migrations in 'offline' mode (generate SQL scripts)."""
    url = config.get_main_option("sqlalchemy.url")
    context.configure(
        url=url,
        target_metadata=target_metadata,
        literal_binds=True,
        dialect_opts={"paramstyle": "named"},
    )
    with context.begin_transaction():
        context.run_migrations()


def run_migrations_online() -> None:
    """Run migrations in 'online' mode using a sync engine."""
    connectable = engine_from_config(
        config.get_section(config.config_ini_section, {}),
        prefix="sqlalchemy.",
        poolclass=pool.NullPool,
    )

    with connectable.connect() as connection:
        context.configure(
            connection=connection,
            target_metadata=target_metadata,
        )
        with context.begin_transaction():
            context.run_migrations()

    connectable.dispose()


if context.is_offline_mode():
    run_migrations_offline()
else:
    run_migrations_online()
