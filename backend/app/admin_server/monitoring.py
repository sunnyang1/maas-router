"""
Admin monitoring endpoints - delegates to MonitoringService.
Uses real database metrics instead of mock/hardcoded data.
"""
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.services.monitoring_service import MonitoringService

router = APIRouter(tags=["Monitoring"])


@router.get("/monitoring/services")
async def service_health(db: AsyncSession = Depends(get_db)):
    """Get real service health status with circuit breaker states."""
    service = MonitoringService(db)
    return await service.get_service_health()


@router.get("/monitoring/metrics")
async def realtime_metrics(db: AsyncSession = Depends(get_db)):
    """Get real-time metrics from the database."""
    service = MonitoringService(db)
    return await service.get_realtime_metrics()


@router.get("/monitoring/failover-logs")
async def failover_logs(db: AsyncSession = Depends(get_db)):
    """Get recent failover events from the database."""
    service = MonitoringService(db)
    return await service.get_failover_logs()


@router.get("/monitoring/alerts")
async def active_alerts():
    """Get active alerts from circuit breaker states."""
    from app.core.database import async_session_factory

    async with async_session_factory() as session:
        service = MonitoringService(session)
        return await service.get_active_alerts()
