"""
Balance repository with locking for atomic deductions.
"""
from sqlalchemy import select, update
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.billing import Balance
from app.repositories.base import BaseRepository


class BalanceRepository(BaseRepository[Balance]):
    def __init__(self, session: AsyncSession):
        super().__init__(Balance, session)

    async def get_by_user_id(self, user_id: str) -> Balance | None:
        return await self.find_one(user_id=user_id)

    async def get_or_create(self, user_id: str) -> Balance:
        bal = await self.get_by_user_id(user_id)
        if not bal:
            bal = Balance(user_id=user_id)
            self.session.add(bal)
            await self.session.flush()
        return bal

    async def deduct(
        self, user_id: str, amount: float
    ) -> tuple[bool, float]:
        """
        Atomically deduct balance with validation.
        Returns (success, new_balance).
        Uses UPDATE ... WHERE for atomicity.
        """
        result = await self.session.execute(
            update(Balance)
            .where(
                Balance.user_id == user_id,
                Balance.cred_balance >= amount,
            )
            .values(cred_balance=Balance.cred_balance - amount)
            .returning(Balance.cred_balance)
        )
        new_balance = result.scalar_one_or_none()
        if new_balance is not None:
            return True, round(float(new_balance), 6)
        return False, 0.0

    async def add(self, user_id: str, amount: float) -> float:
        """Add credits to a user's balance."""
        result = await self.session.execute(
            update(Balance)
            .where(Balance.user_id == user_id)
            .values(cred_balance=Balance.cred_balance + amount)
            .returning(Balance.cred_balance)
        )
        new_balance = result.scalar_one_or_none()
        return round(float(new_balance or 0), 6)

    async def total_cred(self) -> float:
        from sqlalchemy import func
        result = await self.session.execute(
            select(func.coalesce(func.sum(Balance.cred_balance), 0))
        )
        return round(float(result.scalar() or 0), 4)

    async def user_count_with_balance(self) -> int:
        from sqlalchemy import func
        result = await self.session.execute(
            select(func.count(Balance.id)).where(Balance.cred_balance > 0)
        )
        return result.scalar() or 0
