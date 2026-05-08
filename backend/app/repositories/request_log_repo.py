"""
Request log repository.
"""
from datetime import datetime, timedelta, timezone
from sqlalchemy import select, func, text
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.routing import RequestLog
from app.repositories.base import BaseRepository


class RequestLogRepository(BaseRepository[RequestLog]):
    def __init__(self, session: AsyncSession):
        super().__init__(RequestLog, session)

    async def count_recent(self, minutes: int = 1) -> int:
        since = datetime.now(timezone.utc) - timedelta(minutes=minutes)
        result = await self.session.execute(
            select(func.count(RequestLog.id)).where(RequestLog.created_at >= since)
        )
        return result.scalar() or 0

    async def count_today(self) -> int:
        today = datetime.now(timezone.utc).replace(hour=0, minute=0, second=0, microsecond=0)
        result = await self.session.execute(
            select(func.count(RequestLog.id)).where(RequestLog.created_at >= today)
        )
        return result.scalar() or 0

    async def active_users_today(self) -> int:
        today = datetime.now(timezone.utc).replace(hour=0, minute=0, second=0, microsecond=0)
        result = await self.session.execute(
            select(func.count(func.distinct(RequestLog.user_id)))
            .where(RequestLog.created_at >= today)
        )
        return result.scalar() or 0

    async def get_trends(
        self, days: int = 7
    ) -> list[tuple[str, int, int]]:
        """Daily aggregated trends (date, requests, tokens)."""
        since = datetime.now(timezone.utc) - timedelta(days=days)
        result = await self.session.execute(
            select(
                func.date_trunc("day", RequestLog.created_at).label("day"),
                func.count(RequestLog.id).label("requests"),
                func.coalesce(
                    func.sum(RequestLog.prompt_tokens + RequestLog.completion_tokens), 0
                ).label("tokens"),
            )
            .where(RequestLog.created_at >= since)
            .group_by(text("day"))
            .order_by(text("day"))
        )
        rows = result.all()
        return [
            (day.strftime("%m-%d") if day else "unknown", reqs or 0, int(tok or 0))
            for day, reqs, tok in rows
        ]

    async def get_model_distribution(self, limit: int = 10) -> list[tuple[str, int]]:
        result = await self.session.execute(
            select(
                RequestLog.model_id,
                func.count(RequestLog.id).label("count"),
            )
            .group_by(RequestLog.model_id)
            .order_by(func.count(RequestLog.id).desc())
            .limit(limit)
        )
        return [(row.model_id or "unknown", row.count) for row in result.all()]

    async def get_recent(
        self, limit: int = 10
    ) -> list[RequestLog]:
        result = await self.session.execute(
            select(RequestLog)
            .order_by(RequestLog.created_at.desc())
            .limit(limit)
        )
        return list(result.scalars().all())

    async def get_failover_events(self, limit: int = 20) -> list[RequestLog]:
        result = await self.session.execute(
            select(RequestLog)
            .where(RequestLog.error_code.isnot(None))
            .order_by(RequestLog.created_at.desc())
            .limit(limit)
        )
        return list(result.scalars().all())

    async def avg_latency_recent(self, minutes: int = 1) -> float:
        since = datetime.now(timezone.utc) - timedelta(minutes=minutes)
        result = await self.session.execute(
            select(func.avg(RequestLog.latency_ms))
            .where(RequestLog.created_at >= since)
        )
        return float(result.scalar() or 0)

    async def error_rate_recent(self, minutes: int = 1) -> float:
        since = datetime.now(timezone.utc) - timedelta(minutes=minutes)
        total = await self.session.execute(
            select(func.count(RequestLog.id))
            .where(RequestLog.created_at >= since)
        )
        errors = await self.session.execute(
            select(func.count(RequestLog.id))
            .where(
                RequestLog.created_at >= since,
                RequestLog.status_code >= 400,
            )
        )
        total_n = total.scalar() or 1
        error_n = errors.scalar() or 0
        return round(error_n / total_n * 100, 2)

    async def count_24h(self) -> int:
        since = datetime.now(timezone.utc) - timedelta(hours=24)
        result = await self.session.execute(
            select(func.count(RequestLog.id)).where(RequestLog.created_at >= since)
        )
        return result.scalar() or 0

    async def list_by_user(
        self, user_id: str, limit: int = 20
    ) -> list[RequestLog]:
        result = await self.session.execute(
            select(RequestLog)
            .where(RequestLog.user_id == user_id)
            .order_by(RequestLog.created_at.desc())
            .limit(limit)
        )
        return list(result.scalars().all())

    async def count_by_user(self, user_id: str) -> int:
        from sqlalchemy import func
        result = await self.session.execute(
            select(func.count(RequestLog.id)).where(RequestLog.user_id == user_id)
        )
        return result.scalar() or 0
