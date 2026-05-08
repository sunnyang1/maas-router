"""
Model repository.
"""
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.provider import Model, Provider
from app.repositories.base import BaseRepository


class ModelRepository(BaseRepository[Model]):
    def __init__(self, session: AsyncSession):
        super().__init__(Model, session)

    async def list_active_with_provider(self) -> list[tuple[Model, Provider]]:
        result = await self.session.execute(
            select(Model, Provider)
            .join(Provider, Model.provider_id == Provider.id)
            .where(Model.status == "active")
            .order_by(Model.popularity.desc())
        )
        return result.all()

    async def get_with_provider(self, model_id: str) -> tuple[Model, Provider] | None:
        result = await self.session.execute(
            select(Model, Provider)
            .join(Provider, Model.provider_id == Provider.id)
            .where(Model.id == model_id)
        )
        return result.one_or_none()
