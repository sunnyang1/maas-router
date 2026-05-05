"""
Routing rule repository.
"""
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.routing import RoutingRule
from app.repositories.base import BaseRepository


class RoutingRuleRepository(BaseRepository[RoutingRule]):
    def __init__(self, session: AsyncSession):
        super().__init__(RoutingRule, session)

    async def get_active_rules(
        self, user_id: str | None = None
    ) -> list[RoutingRule]:
        """Get active routing rules ordered by priority."""
        query = (
            select(RoutingRule)
            .where(RoutingRule.status == "active")
            .order_by(RoutingRule.priority.desc())
        )
        result = await self.session.execute(query)
        return list(result.scalars().all())
