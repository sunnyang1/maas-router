"""
Admin billing & finance endpoints
"""
from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from pydantic import BaseModel
from datetime import datetime, timezone

from app.core.database import get_db
from app.models.billing import Balance, Transaction
from app.models.user import User

router = APIRouter(tags=["Billing"])


@router.get("/billing/overview")
async def billing_overview(db: AsyncSession = Depends(get_db)):
    """Get billing overview."""
    now = datetime.now(timezone.utc)
    today_start = now.replace(hour=0, minute=0, second=0, microsecond=0)
    month_start = now.replace(day=1, hour=0, minute=0, second=0, microsecond=0)

    # Today usage revenue
    today_usage = await db.scalar(
        select(func.coalesce(func.sum(func.abs(Transaction.amount)), 0))
        .where(
            Transaction.type == "usage",
            Transaction.created_at >= today_start,
        )
    )

    # Today topup
    today_topup = await db.scalar(
        select(func.coalesce(func.sum(Transaction.amount), 0))
        .where(
            Transaction.type == "topup",
            Transaction.created_at >= today_start,
        )
    )

    # Month usage
    month_usage = await db.scalar(
        select(func.coalesce(func.sum(func.abs(Transaction.amount)), 0))
        .where(
            Transaction.type == "usage",
            Transaction.created_at >= month_start,
        )
    )

    # Total CRED in circulation
    total_cred = await db.scalar(select(func.sum(Balance.cred_balance)))

    # Total users with balance
    total_users_with_balance = await db.scalar(
        select(func.count(Balance.id))
        .where(Balance.cred_balance > 0)
    )

    return {
        "today_usage_revenue": round(float(today_usage or 0), 4),
        "today_topup": round(float(today_topup or 0), 2),
        "monthly_usage_revenue": round(float(month_usage or 0), 4),
        "total_cred_circulation": round(float(total_cred or 0), 4),
        "active_balance_users": total_users_with_balance or 0,
    }


@router.get("/billing/transactions")
async def list_transactions(
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    type: str | None = None,
    user_id: str | None = None,
    db: AsyncSession = Depends(get_db),
):
    """List transactions."""
    query = select(Transaction, User.email, User.display_name).join(
        User, Transaction.user_id == User.id
    )
    count_query = select(func.count(Transaction.id))

    if type:
        query = query.where(Transaction.type == type)
        count_query = count_query.where(Transaction.type == type)
    if user_id:
        query = query.where(Transaction.user_id == user_id)
        count_query = count_query.where(Transaction.user_id == user_id)

    total = await db.scalar(count_query)
    result = await db.execute(
        query.order_by(Transaction.created_at.desc())
        .offset((page - 1) * page_size)
        .limit(page_size)
    )
    rows = result.all()

    return {
        "total": total,
        "page": page,
        "page_size": page_size,
        "data": [{
            "id": txn.id,
            "user_email": email,
            "user_name": name,
            "type": txn.type,
            "amount": txn.amount,
            "currency": txn.currency,
            "model_id": txn.model_id,
            "total_tokens": txn.total_tokens,
            "route_reason": txn.route_reason,
            "status": txn.status,
            "created_at": txn.created_at.isoformat() if txn.created_at else None,
        } for txn, email, name in rows]
    }


class BalanceAdjust(BaseModel):
    user_id: str
    amount: float
    reason: str = "管理员调整"


@router.post("/billing/adjust")
async def adjust_balance(req: BalanceAdjust, db: AsyncSession = Depends(get_db)):
    """Adjust user balance (admin action)."""
    result = await db.execute(select(Balance).where(Balance.user_id == req.user_id))
    balance = result.scalar_one_or_none()

    if not balance:
        balance = Balance(user_id=req.user_id)
        db.add(balance)

    balance.cred_balance = round(balance.cred_balance + req.amount, 6)

    # Record transaction
    txn = Transaction(
        user_id=req.user_id,
        type="topup" if req.amount > 0 else "refund",
        amount=req.amount,
        currency="CRED",
    )
    db.add(txn)

    return {
        "user_id": req.user_id,
        "new_balance": balance.cred_balance,
        "adjustment": req.amount,
    }


@router.get("/cred/supply")
async def cred_supply(db: AsyncSession = Depends(get_db)):
    """Get $CRED supply info."""
    total_supply = await db.scalar(select(func.sum(Balance.cred_balance)))
    user_count = await db.scalar(select(func.count(Balance.id)))

    return {
        "total_supply": round(float(total_supply or 0), 4),
        "holders": user_count or 0,
        "reserve_ratio": "100% (demo)",
    }
