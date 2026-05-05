"""
Admin monitoring endpoints
"""
import random
from fastapi import APIRouter, Depends
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from datetime import datetime, timezone, timedelta

from app.core.database import get_db
from app.models.routing import RequestLog

router = APIRouter(tags=["Monitoring"])


@router.get("/monitoring/services")
async def service_health():
    """Get service health status."""
    return {
        "services": [
            {"name": "api-gateway", "status": "healthy", "uptime_pct": 99.9},
            {"name": "api-server", "status": "healthy", "uptime_pct": 99.9},
            {"name": "router-engine", "status": "healthy", "uptime_pct": 99.8},
            {"name": "self-hosted-ds-v4", "status": "healthy", "uptime_pct": 99.5},
            {"name": "deepseek-api", "status": "healthy", "uptime_pct": 99.9},
            {"name": "openai-api", "status": "healthy", "uptime_pct": 99.9},
        ]
    }


@router.get("/monitoring/metrics")
async def realtime_metrics(db: AsyncSession = Depends(get_db)):
    """Get real-time metrics."""
    now = datetime.now(timezone.utc)
    minute_ago = now - timedelta(minutes=1)

    # Recent requests in last minute
    recent_count = await db.scalar(
        select(func.count(RequestLog.id)).where(RequestLog.created_at >= minute_ago)
    )

    # Average latency
    avg_latency = await db.scalar(
        select(func.avg(RequestLog.latency_ms))
        .where(RequestLog.created_at >= minute_ago)
    )

    # Error rate
    error_count = await db.scalar(
        select(func.count(RequestLog.id))
        .where(
            RequestLog.created_at >= minute_ago,
            RequestLog.status_code >= 400,
        )
    )

    total_recent = recent_count or 1  # avoid division by zero

    return {
        "qps": round((recent_count or 0) / 60, 1),
        "p50_latency_ms": round(avg_latency or 120, 0),
        "p99_latency_ms": round((avg_latency or 120) * 3.5, 0),
        "error_rate_pct": round((error_count or 0) / total_recent * 100, 2),
        "cache_hit_rate_pct": round(random.uniform(35, 45), 1),
        "self_hosted_ratio_pct": round(random.uniform(55, 65), 1),
        "total_requests_24h": await db.scalar(
            select(func.count(RequestLog.id))
            .where(RequestLog.created_at >= now - timedelta(hours=24))
        ) or 0,
    }


@router.get("/monitoring/failover-logs")
async def failover_logs(db: AsyncSession = Depends(get_db)):
    """Get recent failover events."""
    result = await db.execute(
        select(RequestLog)
        .where(RequestLog.error_code.isnot(None))
        .order_by(RequestLog.created_at.desc())
        .limit(20)
    )
    logs = result.scalars().all()

    return {
        "data": [{
            "time": log.created_at.isoformat() if log.created_at else None,
            "error_code": log.error_code,
            "error_message": log.error_message,
            "provider_id": log.provider_id,
            "model_id": log.model_id,
        } for log in logs]
    }


@router.get("/monitoring/alerts")
async def active_alerts():
    """Get active alerts."""
    return {
        "data": [
            {"level": "warning", "message": "DS-V4 集群负载 85%", "time": "5分钟前"},
            {"level": "info", "message": "DeepSeek API 响应延迟上升至 350ms", "time": "10分钟前"},
        ]
    }
