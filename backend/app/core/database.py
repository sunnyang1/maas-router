"""
Database setup with async SQLAlchemy
"""
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker
from sqlalchemy.orm import DeclarativeBase

from app.core.config import get_settings

settings = get_settings()


def _get_engine(database_url: str | None = None):
    """Create an async engine with proper pool configuration."""
    url = database_url or settings.database_url

    if "sqlite" in url:
        return create_async_engine(
            url,
            echo=settings.debug,
            connect_args={"check_same_thread": False},
        )
    else:
        return create_async_engine(
            url,
            echo=False,  # Disable echo even in debug; use logging instead
            pool_size=settings.db_pool_size,
            max_overflow=settings.db_max_overflow,
            pool_pre_ping=True,
            pool_recycle=settings.db_pool_recycle,
            pool_timeout=settings.db_pool_timeout,
        )


# Primary (write) engine
engine = _get_engine()

# Read replica engine (falls back to primary if not configured)
read_engine = _get_engine(settings.database_url_read) if settings.database_url_read else engine

async_session_factory = async_sessionmaker(
    engine,
    class_=AsyncSession,
    expire_on_commit=False,
)

read_session_factory = async_sessionmaker(
    read_engine,
    class_=AsyncSession,
    expire_on_commit=False,
)


class Base(DeclarativeBase):
    pass


async def get_db() -> AsyncSession:
    """Dependency to get async DB session (primary/write)."""
    async with async_session_factory() as session:
        try:
            yield session
            await session.commit()
        except Exception:
            await session.rollback()
            raise


async def get_db_read() -> AsyncSession:
    """Dependency to get async DB session (read replica, or primary if no replica)."""
    async with read_session_factory() as session:
        try:
            yield session
            await session.commit()
        except Exception:
            await session.rollback()
            raise


async def init_db():
    """
    Create all tables. Safe for development only.
    In production, use Alembic migrations instead.
    """
    if settings.environment in ("staging", "production"):
        import sys
        print(
            "WARNING: Using Base.metadata.create_all in non-development environment. "
            "Use Alembic migrations for production deployments.",
            file=sys.stderr,
        )
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
