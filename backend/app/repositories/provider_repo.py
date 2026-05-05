"""
Provider repository.
"""
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.provider import Provider
from app.repositories.base import BaseRepository


class ProviderRepository(BaseRepository[Provider]):
    def __init__(self, session: AsyncSession):
        super().__init__(Provider, session)

    async def get_active_providers(self) -> list[Provider]:
        result = await self.session.execute(
            select(Provider).where(Provider.status == "active")
        )
        return list(result.scalars().all())

    async def get_with_models(self, provider_id: str) -> Provider | None:
        from sqlalchemy.orm import selectinload
        from app.models.provider import Model
        result = await self.session.execute(
            select(Provider)
            .options(selectinload(Provider.models))
            .where(Provider.id == provider_id)
        )
        return result.scalar_one_or_none()
