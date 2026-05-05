"""
API Key repository.
"""
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.api_key import ApiKey
from app.repositories.base import BaseRepository


class ApiKeyRepository(BaseRepository[ApiKey]):
    def __init__(self, session: AsyncSession):
        super().__init__(ApiKey, session)

    async def get_by_hash(self, key_hash: str) -> ApiKey | None:
        return await self.find_one(key_hash=key_hash)

    async def list_by_user(self, user_id: str) -> list[ApiKey]:
        result = await self.session.execute(
            select(ApiKey)
            .where(ApiKey.user_id == user_id)
            .order_by(ApiKey.created_at.desc())
        )
        return list(result.scalars().all())

    async def count_by_user(self, user_id: str) -> int:
        from sqlalchemy import func
        result = await self.session.execute(
            select(func.count(ApiKey.id)).where(ApiKey.user_id == user_id)
        )
        return result.scalar() or 0

    async def count_active(self) -> int:
        from sqlalchemy import func
        result = await self.session.execute(
            select(func.count(ApiKey.id)).where(ApiKey.status == "active")
        )
        return result.scalar() or 0
