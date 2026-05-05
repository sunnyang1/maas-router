"""
Admin dashboard endpoints
"""
from fastapi import APIRouter, Depends
from sqlalchemy import select, func, and_
from sqlalchemy.ext.asyncio import AsyncSession
from datetime import datetime, timezone, timedelta

from app.core.database import get_db
from app.core.security import get_current_user_from_jwt
from app.models.user import User
from app.models.api_key import ApiKey
from app.models.billing import Transaction
from app.models.routing import RequestLog

router = APIRouter(tags=["Dashboard"])


@router.get("/dashboard/overview")
async def dashboard_overview(db: AsyncSession = Depends(get_db)):
    """Get dashboard overview statistics."""
    now = datetime.now(timezone.utc)
    today_start = now.replace(hour=0, minute=0, second=0, microsecond=0)
    month_start = now.replace(day=1, hour=0, minute=0, second=0, microsecond=0)

    # Total users
    total_users = await db.scalar(select(func.count(User.id)))

    # Active today
    active_today = await db.scalar(
        select(func.count(func.distinct(RequestLog.user_id)))
        .where(RequestLog.created_at >= today_start)
    )

    # Today revenue (positive sum of transactions = topups for today)
    today_revenue_result = await db.execute(
        select(func.coalesce(func.sum(Transaction.amount), 0))
        .where(
            Transaction.type == "topup",
            Transaction.created_at >= today_start,
            Transaction.status == "completed",
        )
    )
    today_revenue = today_revenue_result.scalar() or 0

    # Active API keys
    active_keys = await db.scalar(
        select(func.count(ApiKey.id)).where(ApiKey.status == "active")
    )

    # Today requests
    today_requests = await db.scalar(
        select(func.count(RequestLog.id)).where(RequestLog.created_at >= today_start)
    )

    # Monthly revenue
    monthly_revenue_result = await db.execute(
        select(func.coalesce(func.sum(Transaction.amount), 0))
        .where(
            Transaction.type == "topup",
            Transaction.created_at >= month_start,
            Transaction.status == "completed",
        )
    )
    monthly_revenue = monthly_revenue_result.scalar() or 0

    return {
        "total_users": total_users or 0,
        "active_today": active_today or 0,
        "today_revenue": round(float(today_revenue), 2),
        "monthly_revenue": round(float(monthly_revenue), 2),
        "active_api_keys": active_keys or 0,
        "today_requests": today_requests or 0,
    }


@router.get("/dashboard/trends")
async def dashboard_trends(days: int = 7, db: AsyncSession = Depends(get_db)):
    """Get trend data for charts."""
    now = datetime.now(timezone.utc)
    trend_data = []

    for i in range(days):
        day = now - timedelta(days=days - 1 - i)
        day_start = day.replace(hour=0, minute=0, second=0, microsecond=0)
        day_end = day_start + timedelta(days=1)

        requests = await db.scalar(
            select(func.count(RequestLog.id))
            .where(RequestLog.created_at >= day_start, RequestLog.created_at < day_end)
        )

        tokens = await db.scalar(
            select(func.coalesce(func.sum(RequestLog.prompt_tokens + RequestLog.completion_tokens), 0))
            .where(RequestLog.created_at >= day_start, RequestLog.created_at < day_end)
        )

        cost = await db.scalar(
            select(func.coalesce(func.sum(Transaction.amount), 0))
            .where(
                Transaction.type == "usage",
                Transaction.created_at >= day_start,
                Transaction.created_at < day_end,
            )
        )

        trend_data.append({
            "date": day.strftime("%m-%d"),
            "requests": requests or 0,
            "tokens": int(tokens or 0),
            "cost": round(abs(float(cost or 0)), 4),
        })

    return {"data": trend_data, "days": days}


@router.get("/dashboard/model-distribution")
async def model_distribution(db: AsyncSession = Depends(get_db)):
    """Get model usage distribution (pie chart data)."""
    result = await db.execute(
        select(
            RequestLog.model_id,
            func.count(RequestLog.id).label("count"),
        )
        .group_by(RequestLog.model_id)
        .order_by(func.count(RequestLog.id).desc())
        .limit(10)
    )
    rows = result.all()

    return {
        "data": [
            {"model": row.model_id or "unknown", "count": row.count}
            for row in rows
        ]
    }


@router.get("/dashboard/recent-requests")
async def recent_requests(limit: int = 10, db: AsyncSession = Depends(get_db)):
    """Get recent requests with router decisions."""
    result = await db.execute(
        select(RequestLog)
        .order_by(RequestLog.created_at.desc())
        .limit(limit)
    )
    logs = result.scalars().all()

    return {
        "data": [{
            "request_id": log.request_id,
            "model_id": log.model_id,
            "provider_id": log.provider_id,
            "status_code": log.status_code,
            "latency_ms": log.latency_ms,
            "complexity_score": log.complexity_score,
            "route_decision": log.route_decision,
            "prompt_tokens": log.prompt_tokens,
            "completion_tokens": log.completion_tokens,
            "error_code": log.error_code,
            "created_at": log.created_at.isoformat() if log.created_at else None,
        } for log in logs]
    }
