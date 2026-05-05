"""
Background billing worker.

Handles async transaction recording without blocking the response.
"""
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import async_session_factory
from app.models.billing import Transaction
from app.repositories.balance_repo import BalanceRepository
from app.repositories.transaction_repo import TransactionRepository


async def record_transaction_async(
    user_id: str,
    tx_type: str,
    request_id: str | None = None,
    model_id: str | None = None,
    provider_id: str | None = None,
    prompt_tokens: int = 0,
    completion_tokens: int = 0,
    total_tokens: int = 0,
    amount: float = 0.0,
    currency: str = "CRED",
    route_reason: str | None = None,
    route_confidence: float | None = None,
) -> Transaction | None:
    """Record a transaction in a separate session (non-blocking)."""
    async with async_session_factory() as session:
        try:
            repo = TransactionRepository(session)
            txn = Transaction(
                user_id=user_id,
                type=tx_type,
                request_id=request_id,
                model_id=model_id,
                provider_id=provider_id,
                prompt_tokens=prompt_tokens,
                completion_tokens=completion_tokens,
                total_tokens=total_tokens,
                amount=amount,
                currency=currency,
                route_reason=route_reason,
                route_confidence=route_confidence,
            )
            await repo.create(txn)
            await session.commit()
            return txn
        except Exception:
            await session.rollback()
            return None


async def update_balance_async(user_id: str, delta: float) -> float | None:
    """Update user balance asynchronously."""
    async with async_session_factory() as session:
        try:
            repo = BalanceRepository(session)
            if delta > 0:
                new_bal = await repo.add(user_id, delta)
            else:
                bal = await repo.get_or_create(user_id)
                new_bal = bal.cred_balance + delta
            await session.commit()
            return new_bal
        except Exception:
            await session.rollback()
            return None
