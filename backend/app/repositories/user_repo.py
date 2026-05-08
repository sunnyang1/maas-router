"""
User repository.
"""
from sqlalchemy import select, func, or_
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.user import User
from app.models.billing import Balance
from app.repositories.base import BaseRepository


class UserRepository(BaseRepository[User]):
    def __init__(self, session: AsyncSession):
        super().__init__(User, session)

    async def get_by_email(self, email: str) -> User | None:
        return await self.find_one(email=email)

    async def list_with_balance(
        self,
        search: str | None = None,
        offset: int = 0,
        limit: int = 20,
    ) -> tuple[list[tuple[User, float]], int]:
        """List users with their balance in a single query."""
        base = (
            select(User, func.coalesce(Balance.cred_balance, 0).label("cred"))
            .outerjoin(Balance, Balance.user_id == User.id)
        )
        count_q = select(func.count(User.id))

        if search:
            cond = or_(
                User.email.ilike(f"%{search}%"),
                User.display_name.ilike(f"%{search}%"),
            )
            base = base.where(cond)
            count_q = count_q.where(cond)

        total = await self.session.execute(count_q)
        total_count = total.scalar() or 0

        result = await self.session.execute(
            base.order_by(User.created_at.desc()).offset(offset).limit(limit)
        )
        return result.all(), total_count

    async def is_email_taken(self, email: str) -> bool:
        return await self.exists(email=email)
