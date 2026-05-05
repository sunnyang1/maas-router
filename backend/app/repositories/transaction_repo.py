"""
Transaction repository.
"""
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.billing import Transaction
from app.models.user import User
from app.repositories.base import BaseRepository


class TransactionRepository(BaseRepository[Transaction]):
    def __init__(self, session: AsyncSession):
        super().__init__(Transaction, session)

    async def list_with_user(
        self,
        tx_type: str | None = None,
        user_id: str | None = None,
        offset: int = 0,
        limit: int = 20,
    ) -> tuple[list[tuple[Transaction, str, str]], int]:
        """List transactions with user info."""
        base = select(
            Transaction, User.email, User.display_name
        ).join(User, Transaction.user_id == User.id)
        count_q = select(func.count(Transaction.id))

        if tx_type:
            base = base.where(Transaction.type == tx_type)
            count_q = count_q.where(Transaction.type == tx_type)
        if user_id:
            base = base.where(Transaction.user_id == user_id)
            count_q = count_q.where(Transaction.user_id == user_id)

        total = await self.session.execute(count_q)
        total_count = total.scalar() or 0

        result = await self.session.execute(
            base.order_by(Transaction.created_at.desc()).offset(offset).limit(limit)
        )
        return result.all(), total_count

    async def get_user_summary(self, user_id: str) -> dict:
        result = await self.session.execute(
            select(
                func.count(Transaction.id).label("total_requests"),
                func.sum(Transaction.total_tokens).label("total_tokens"),
                func.sum(Transaction.amount).label("total_cost"),
            ).where(
                Transaction.user_id == user_id,
                Transaction.type == "usage",
            )
        )
        row = result.one()
        return {
            "total_requests": row.total_requests or 0,
            "total_tokens": int(row.total_tokens or 0),
            "total_cost_cred": round(abs(float(row.total_cost or 0)), 6),
        }
