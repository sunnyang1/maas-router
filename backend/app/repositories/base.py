"""
Base repository with common CRUD operations.
"""
from typing import Generic, TypeVar, Optional
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import DeclarativeBase

ModelType = TypeVar("ModelType", bound=DeclarativeBase)


class BaseRepository(Generic[ModelType]):
    """Generic base repository providing common data access patterns."""

    def __init__(self, model: type[ModelType], session: AsyncSession):
        self.model = model
        self.session = session

    async def get_by_id(self, id_val: str) -> Optional[ModelType]:
        """Get a single record by primary key."""
        result = await self.session.execute(
            select(self.model).where(self.model.id == id_val)
        )
        return result.scalar_one_or_none()

    async def get_all(
        self,
        offset: int = 0,
        limit: int = 100,
        order_by=None,
    ) -> list[ModelType]:
        """Get all records with pagination."""
        query = select(self.model)
        if order_by is not None:
            query = query.order_by(order_by)
        query = query.offset(offset).limit(limit)
        result = await self.session.execute(query)
        return list(result.scalars().all())

    async def count(self, *filters) -> int:
        """Count records matching optional filters."""
        from sqlalchemy import func
        query = select(func.count()).select_from(self.model)
        for f in filters:
            query = query.where(f)
        result = await self.session.execute(query)
        return result.scalar() or 0

    async def create(self, instance: ModelType) -> ModelType:
        """Add a new record and flush to get generated values."""
        self.session.add(instance)
        await self.session.flush()
        return instance

    async def update(self, instance: ModelType) -> ModelType:
        """Mark an instance as modified (caller must have set fields)."""
        await self.session.flush()
        return instance

    async def delete(self, instance: ModelType) -> None:
        """Delete a record."""
        await self.session.delete(instance)
        await self.session.flush()

    async def exists(self, **kwargs) -> bool:
        """Check if any record matches the given column filters."""
        from sqlalchemy import func
        query = select(func.count()).select_from(self.model)
        for col_name, value in kwargs.items():
            query = query.where(getattr(self.model, col_name) == value)
        result = await self.session.execute(query)
        return (result.scalar() or 0) > 0

    async def find_one(self, **kwargs) -> Optional[ModelType]:
        """Find one record matching column filters."""
        query = select(self.model)
        for col_name, value in kwargs.items():
            query = query.where(getattr(self.model, col_name) == value)
        result = await self.session.execute(query)
        return result.scalar_one_or_none()
