"""
Admin dashboard endpoints — thin handlers delegating to DashboardService.
"""
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.services.dashboard_service import DashboardService

router = APIRouter(tags=["Dashboard"])


def _get_dashboard_service(db: AsyncSession = Depends(get_db)) -> DashboardService:
    return DashboardService(db)


@router.get("/dashboard/overview")
async def dashboard_overview(svc: DashboardService = Depends(_get_dashboard_service)):
    """Get dashboard overview statistics."""
    return await svc.get_overview()


@router.get("/dashboard/trends")
async def dashboard_trends(
    days: int = 7,
    svc: DashboardService = Depends(_get_dashboard_service),
):
    """Get trend data for charts."""
    return await svc.get_trends(days)


@router.get("/dashboard/model-distribution")
async def model_distribution(svc: DashboardService = Depends(_get_dashboard_service)):
    """Get model usage distribution (pie chart data)."""
    return await svc.get_model_distribution()


@router.get("/dashboard/recent-requests")
async def recent_requests(
    limit: int = 10,
    svc: DashboardService = Depends(_get_dashboard_service),
):
    """Get recent requests with router decisions."""
    return await svc.get_recent_requests(limit)
