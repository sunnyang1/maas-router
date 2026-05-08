"""
Dashboard service — aggregated statistics for the admin dashboard.
"""
from datetime import datetime, timezone, timedelta
from sqlalchemy import select, func, text
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.billing import Transaction
from app.repositories.user_repo import UserRepository
from app.repositories.api_key_repo import ApiKeyRepository
from app.repositories.balance_repo import BalanceRepository
from app.repositories.transaction_repo import TransactionRepository
from app.repositories.request_log_repo import RequestLogRepository


class DashboardService:
    """Provides aggregated dashboard data."""

    def __init__(self, session: AsyncSession):
        self.session = session
        self.user_repo = UserRepository(session)
        self.api_key_repo = ApiKeyRepository(session)
        self.balance_repo = BalanceRepository(session)
        self.transaction_repo = TransactionRepository(session)
        self.log_repo = RequestLogRepository(session)

    async def get_overview(self) -> dict:
        """Get dashboard overview statistics."""
        today_start = datetime.now(timezone.utc).replace(
            hour=0, minute=0, second=0, microsecond=0
        )
        month_start = today_start.replace(day=1)

        total_users = await self.user_repo.count()
        active_today = await self.log_repo.active_users_today()
        active_keys = await self.api_key_repo.count_active()
        today_requests = await self.log_repo.count_today()
        total_cred = await self.balance_repo.total_cred()
        balance_users = await self.balance_repo.user_count_with_balance()

        today_revenue = await self.session.scalar(
            select(func.coalesce(func.sum(Transaction.amount), 0))
            .where(
                Transaction.type == "topup",
                Transaction.created_at >= today_start,
                Transaction.status == "completed",
            )
        )
        monthly_revenue = await self.session.scalar(
            select(func.coalesce(func.sum(Transaction.amount), 0))
            .where(
                Transaction.type == "topup",
                Transaction.created_at >= month_start,
                Transaction.status == "completed",
            )
        )

        return {
            "total_users": total_users,
            "active_today": active_today,
            "today_revenue": round(float(today_revenue or 0), 2),
            "monthly_revenue": round(float(monthly_revenue or 0), 2),
            "active_api_keys": active_keys,
            "today_requests": today_requests,
            "total_cred_circulation": total_cred,
            "balance_users": balance_users,
        }

    async def get_trends(self, days: int = 7) -> dict:
        """Get daily trend data with requests, tokens, and cost."""
        trends_raw = await self.log_repo.get_trends(days)
        req_map = {
            date: {"requests": reqs, "tokens": toks}
            for date, reqs, toks in trends_raw
        }

        # Cost data from transactions
        cost_result = await self.session.execute(
            select(
                func.date_trunc("day", Transaction.created_at).label("day"),
                func.coalesce(func.sum(Transaction.amount), 0).label("cost"),
            )
            .where(
                Transaction.type == "usage",
                Transaction.created_at >= datetime.now(timezone.utc) - timedelta(days=days),
            )
            .group_by(text("day"))
            .order_by(text("day"))
        )
        cost_map = {}
        for day, cost in cost_result.all():
            key = day.strftime("%m-%d") if day else "unknown"
            cost_map[key] = round(abs(float(cost or 0)), 4)

        start = datetime.now(timezone.utc) - timedelta(days=days)
        trend_data = []
        for i in range(days):
            day = start + timedelta(days=i)
            key = day.strftime("%m-%d")
            entry = req_map.get(key, {"requests": 0, "tokens": 0})
            trend_data.append({
                "date": key,
                "requests": entry["requests"],
                "tokens": entry["tokens"],
                "cost": cost_map.get(key, 0),
            })
        return {"data": trend_data, "days": days}

    async def get_model_distribution(self) -> dict:
        """Get model usage distribution for pie chart."""
        dist = await self.log_repo.get_model_distribution(limit=10)
        return {
            "data": [
                {"model": model or "unknown", "count": count}
                for model, count in dist
            ]
        }

    async def get_recent_requests(self, limit: int = 10) -> dict:
        """Get recent requests with router decisions."""
        logs = await self.log_repo.get_recent(limit)
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
