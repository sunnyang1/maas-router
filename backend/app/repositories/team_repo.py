"""
Team repository.
"""
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.team import Team, TeamMember
from app.repositories.base import BaseRepository


class TeamRepository(BaseRepository[Team]):
    def __init__(self, session: AsyncSession):
        super().__init__(Team, session)

    async def list_by_user(self, user_id: str) -> list[Team]:
        result = await self.session.execute(
            select(Team)
            .join(TeamMember, Team.id == TeamMember.team_id)
            .where(TeamMember.user_id == user_id)
        )
        return list(result.scalars().all())

    async def list_members(self, team_id: str) -> list[TeamMember]:
        result = await self.session.execute(
            select(TeamMember).where(TeamMember.team_id == team_id)
        )
        return list(result.scalars().all())
