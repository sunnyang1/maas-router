"""
Billing service - balance management and transaction recording.
"""
from datetime import datetime, timezone, timedelta
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.billing import Transaction
from app.repositories.balance_repo import BalanceRepository
from app.repositories.transaction_repo import TransactionRepository


class BillingService:
    """Handles balance checks, deductions, and transaction recording."""

    def __init__(self, session: AsyncSession):
        self.balance_repo = BalanceRepository(session)
        self.transaction_repo = TransactionRepository(session)

    async def check_balance(self, user_id: str) -> float:
        """Get user's CRED balance. Returns 0 if no balance record."""
        bal = await self.balance_repo.get_by_user_id(user_id)
        return round(float(bal.cred_balance if bal else 0), 6)

    async def has_sufficient_balance(
        self, user_id: str, required: float
    ) -> bool:
        """Check if user has at least `required` CRED."""
        current = await self.check_balance(user_id)
        return current >= required

    async def deduct(
        self, user_id: str, amount: float
    ) -> tuple[bool, float]:
        """
        Atomically deduct balance.
        Returns (success, new_balance).
        """
        await self.balance_repo.get_or_create(user_id)
        return await self.balance_repo.deduct(user_id, amount)

    async def add_credits(
        self, user_id: str, amount: float
    ) -> float:
        """Add credits to a user's balance."""
        await self.balance_repo.get_or_create(user_id)
        return await self.balance_repo.add(user_id, amount)

    async def record_usage(
        self,
        user_id: str,
        request_id: str,
        model_id: str,
        provider_id: str,
        prompt_tokens: int,
        completion_tokens: int,
        cost_cred: float,
        route_reason: str = "",
        route_confidence: float = 1.0,
    ) -> Transaction:
        """Record a usage transaction."""
        total = prompt_tokens + completion_tokens
        txn = Transaction(
            user_id=user_id,
            type="usage",
            request_id=request_id,
            model_id=model_id,
            provider_id=provider_id,
            prompt_tokens=prompt_tokens,
            completion_tokens=completion_tokens,
            total_tokens=total,
            amount=-cost_cred,
            currency="CRED",
            route_reason=route_reason,
            route_confidence=route_confidence,
        )
        self.transaction_repo.session.add(txn)
        await self.transaction_repo.session.flush()
        return txn

    async def record_topup(
        self, user_id: str, amount: float, reason: str = ""
    ) -> Transaction:
        """Record a topup/refund transaction."""
        txn = Transaction(
            user_id=user_id,
            type="topup" if amount > 0 else "refund",
            amount=amount,
            currency="CRED",
        )
        self.transaction_repo.session.add(txn)
        await self.transaction_repo.session.flush()
        return txn

    async def get_user_summary(self, user_id: str) -> dict:
        """Get usage summary for a user."""
        return await self.transaction_repo.get_user_summary(user_id)

    async def get_daily_cost_trends(self, days: int = 7) -> dict:
        """Get daily cost trends from transactions."""
        now = datetime.now(timezone.utc)
        start = now - timedelta(days=days)

        from sqlalchemy import select, func, text
        result = await self.transaction_repo.session.execute(
            select(
                func.date_trunc("day", Transaction.created_at).label("day"),
                func.coalesce(func.sum(Transaction.amount), 0).label("cost"),
            )
            .where(
                Transaction.type == "usage",
                Transaction.created_at >= start,
            )
            .group_by(text("day"))
            .order_by(text("day"))
        )

        cost_map = {}
        for day, cost in result.all():
            key = day.strftime("%m-%d") if day else "unknown"
            cost_map[key] = round(abs(float(cost or 0)), 4)

        # Build full date range
        trends = []
        for i in range(days):
            day = start + timedelta(days=i)
            key = day.strftime("%m-%d")
            trends.append({
                "date": key,
                "cost": cost_map.get(key, 0),
            })

        return {"data": trends, "days": days}
