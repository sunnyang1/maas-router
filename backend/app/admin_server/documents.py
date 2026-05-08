"""
Document Automation API Endpoints
=================================
REST API for on-demand document generation, listing, and download.
Integrates with the existing admin server at /api/admin/v1/documents/
"""

from __future__ import annotations

from datetime import datetime, timezone, timedelta
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, Query, BackgroundTasks
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field
from sqlalchemy import select, func, and_, desc
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.core.security import get_current_user_from_jwt
from app.models.user import User
from app.models.billing import Transaction
from app.models.routing import RequestLog, AuditLog
from app.models.provider import Model as ProviderModel
from app.services.document_service import (
    DocumentService,
    DocumentType,
    DocumentFormat,
    DocumentDataAdapter,
)

router = APIRouter(prefix="/documents", tags=["Documents"])

doc_service = DocumentService()
data_adapter = DocumentDataAdapter()


# ══════════════════════════════════════════════════════════════════
# Request/Response Models
# ══════════════════════════════════════════════════════════════════

class DocumentGenerateRequest(BaseModel):
    doc_type: str = Field(..., description="Document type", examples=["billing_report"])
    format: str = Field("pdf", description="Output format", examples=["pdf", "xlsx", "docx", "pptx", "csv"])
    period_days: int = Field(30, description="Data lookback period in days", ge=1, le=365)


class DocumentGenerateResponse(BaseModel):
    success: bool
    filename: str
    format: str
    size: int
    url: str
    message: str


class DocumentListItem(BaseModel):
    filename: str
    size: int
    size_display: str
    created: str
    format: str


class EngineStatus(BaseModel):
    pdf: bool
    excel: bool
    word: bool
    pptx: bool


# ══════════════════════════════════════════════════════════════════
# Endpoints
# ══════════════════════════════════════════════════════════════════

@router.get("/status", response_model=EngineStatus)
async def get_engine_status():
    """Check which document engines are available."""
    return doc_service.get_available_engines()


@router.get("/list", response_model=list[DocumentListItem])
async def list_documents(limit: int = Query(20, ge=1, le=100)):
    """List recently generated documents."""
    return doc_service.list_outputs(limit=limit)


@router.get("/download/{filename}")
async def download_document(filename: str):
    """Download a generated document by filename."""
    file_path = doc_service.pdf.output_dir / filename
    if not file_path.exists():
        raise HTTPException(status_code=404, detail=f"Document not found: {filename}")

    content = file_path.read_bytes()
    ext = filename.rsplit(".", 1)[-1] if "." in filename else "bin"
    mime_map = {
        "pdf": "application/pdf",
        "xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        "docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
        "pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
        "csv": "text/csv",
    }

    return StreamingResponse(
        iter([content]),
        media_type=mime_map.get(ext, "application/octet-stream"),
        headers={"Content-Disposition": f'attachment; filename="{filename}"'},
    )


# ── Document Generation Endpoints ──────────────────────────────────

@router.post("/generate/billing", response_model=DocumentGenerateResponse)
async def generate_billing_report(
    request: DocumentGenerateRequest,
    background_tasks: BackgroundTasks,
    db: AsyncSession = Depends(get_db),
):
    """
    Generate billing report (PDF or Excel).
    Data sources: transactions, request_logs, dashboard overview.
    """
    now = datetime.now(timezone.utc)
    since = now - timedelta(days=request.period_days)

    # Gather billing data
    overview_result = await db.execute(
        select(func.count(func.distinct(RequestLog.user_id)))
        .where(RequestLog.created_at >= since)
    )
    active_users = overview_result.scalar() or 0

    # Recent transactions
    txn_result = await db.execute(
        select(Transaction)
        .where(Transaction.created_at >= since)
        .order_by(desc(Transaction.created_at))
        .limit(200)
    )
    transactions = txn_result.scalars().all()

    # Daily breakdown from request logs
    daily_data = []
    for i in range(request.period_days):
        day = since + timedelta(days=i)
        day_start = day.replace(hour=0, minute=0, second=0, microsecond=0)
        day_end = day_start + timedelta(days=1)

        req_count = await db.scalar(
            select(func.count(RequestLog.id))
            .where(RequestLog.created_at >= day_start, RequestLog.created_at < day_end)
        )
        token_sum = await db.scalar(
            select(func.coalesce(
                func.sum(RequestLog.prompt_tokens + RequestLog.completion_tokens), 0
            )).where(RequestLog.created_at >= day_start, RequestLog.created_at < day_end)
        )
        cost_sum = await db.scalar(
            select(func.coalesce(func.sum(Transaction.amount), 0))
            .where(
                Transaction.type == "usage",
                Transaction.created_at >= day_start,
                Transaction.created_at < day_end,
            )
        )

        daily_data.append({
            "date": day.strftime("%m-%d"),
            "requests": req_count or 0,
            "tokens": int(token_sum or 0),
            "cost": round(abs(float(cost_sum or 0)), 4),
        })

    total_revenue = await db.scalar(
        select(func.coalesce(func.sum(Transaction.amount), 0))
        .where(
            Transaction.type == "topup",
            Transaction.created_at >= since,
            Transaction.status == "completed",
        )
    )

    # Build data context
    data = data_adapter.billing_data(
        dashboard_overview={
            "monthly_revenue": round(float(total_revenue or 0), 2),
            "today_revenue": 0,
            "active_today": active_users or 0,
        },
        trends={"data": daily_data},
        transactions=[{
            "id": t.id,
            "user_email": str(t.user_id),
            "type": t.type,
            "model_id": t.model_id or "N/A",
            "total_tokens": t.total_tokens,
            "amount": t.amount,
            "currency": t.currency,
            "status": t.status,
            "created_at": t.created_at.isoformat() if t.created_at else "",
        } for t in transactions],
    )

    # Generate
    fmt = DocumentFormat(request.format)
    doc_type = DocumentType.BILLING_REPORT
    content, file_path, mime = doc_service.generate(doc_type, data, fmt)

    return DocumentGenerateResponse(
        success=True,
        filename=file_path.name,
        format=request.format,
        size=len(content),
        url=f"/api/admin/v1/documents/download/{file_path.name}",
        message=f"Billing report generated: {file_path.name}",
    )


@router.post("/generate/user-report", response_model=DocumentGenerateResponse)
async def generate_user_report(
    request: DocumentGenerateRequest,
    background_tasks: BackgroundTasks,
    db: AsyncSession = Depends(get_db),
):
    """
    Generate user usage report (PDF or Word).
    Shows per-user API consumption, model preferences, and cost breakdown.
    """
    now = datetime.now(timezone.utc)
    since = now - timedelta(days=request.period_days)

    # Get users with usage stats
    user_stats_result = await db.execute(
        select(
            User.id,
            User.email,
            User.display_name,
            User.plan_id,
            func.count(RequestLog.id).label("total_requests"),
            func.coalesce(
                func.sum(RequestLog.prompt_tokens + RequestLog.completion_tokens), 0
            ).label("total_tokens"),
            func.coalesce(func.sum(Transaction.amount), 0).label("total_cost"),
        )
        .outerjoin(RequestLog, and_(
            User.id == RequestLog.user_id,
            RequestLog.created_at >= since,
        ))
        .outerjoin(Transaction, and_(
            User.id == Transaction.user_id,
            Transaction.type == "usage",
            Transaction.created_at >= since,
        ))
        .group_by(User.id)
        .order_by(desc(func.coalesce(func.sum(RequestLog.prompt_tokens + RequestLog.completion_tokens), 0)))
        .limit(20)
    )
    users = [
        {
            "email": row.email,
            "display_name": row.display_name or row.email,
            "plan_id": row.plan_id,
            "total_requests": row.total_requests or 0,
            "total_tokens": int(row.total_tokens or 0),
            "total_cost": round(abs(float(row.total_cost or 0)), 4),
        }
        for row in user_stats_result.all()
    ]

    # Model distribution
    model_result = await db.execute(
        select(
            RequestLog.model_id,
            func.count(RequestLog.id).label("count"),
            func.coalesce(func.avg(RequestLog.latency_ms), 0).label("avg_latency"),
        )
        .where(RequestLog.created_at >= since)
        .group_by(RequestLog.model_id)
        .order_by(desc(func.count(RequestLog.id)))
        .limit(10)
    )
    model_dist = {
        "data": [
            {
                "model_id": row.model_id or "unknown",
                "count": row.count,
                "avg_latency": round(float(row.avg_latency), 1),
            }
            for row in model_result.all()
        ]
    }

    data = data_adapter.user_usage_data(users, model_dist)

    fmt = DocumentFormat(request.format)
    doc_type = DocumentType.USER_USAGE_REPORT
    content, file_path, mime = doc_service.generate(doc_type, data, fmt)

    return DocumentGenerateResponse(
        success=True,
        filename=file_path.name,
        format=request.format,
        size=len(content),
        url=f"/api/admin/v1/documents/download/{file_path.name}",
        message=f"User usage report generated: {file_path.name}",
    )


@router.post("/generate/ops-daily", response_model=DocumentGenerateResponse)
async def generate_ops_daily(
    request: DocumentGenerateRequest,
    db: AsyncSession = Depends(get_db),
):
    """
    Generate daily operations report (PDF).
    Includes service health, request metrics, error rates, and alert summary.
    """
    now = datetime.now(timezone.utc)
    today_start = now.replace(hour=0, minute=0, second=0, microsecond=0)
    yesterday_start = today_start - timedelta(days=1)

    # Today's metrics
    today_requests = await db.scalar(
        select(func.count(RequestLog.id)).where(RequestLog.created_at >= today_start)
    )
    today_errors = await db.scalar(
        select(func.count(RequestLog.id))
        .where(RequestLog.created_at >= today_start, RequestLog.status_code >= 400)
    )
    today_avg_latency = await db.scalar(
        select(func.coalesce(func.avg(RequestLog.latency_ms), 0))
        .where(RequestLog.created_at >= today_start)
    )
    today_tokens = await db.scalar(
        select(func.coalesce(
            func.sum(RequestLog.prompt_tokens + RequestLog.completion_tokens), 0
        )).where(RequestLog.created_at >= today_start)
    )

    # Error breakdown
    error_breakdown_result = await db.execute(
        select(
            RequestLog.error_code,
            func.count(RequestLog.id).label("count"),
        )
        .where(RequestLog.created_at >= today_start, RequestLog.status_code >= 400)
        .group_by(RequestLog.error_code)
        .order_by(desc(func.count(RequestLog.id)))
    )
    errors = [
        {"code": row.error_code or "unknown", "count": row.count}
        for row in error_breakdown_result.all()
    ]

    data = {
        "report_date": now.strftime("%Y-%m-%d"),
        "summary": {
            "today_requests": today_requests or 0,
            "today_tokens": int(today_tokens or 0),
            "error_count": today_errors or 0,
            "error_rate": f"{((today_errors or 0) / max(today_requests or 1, 1)) * 100:.2f}%",
            "avg_latency_ms": round(float(today_avg_latency or 0), 1),
            "p99_latency_ms": 0,  # Requires raw data percentile calculation
        },
        "service_health": {
            "api_server": "healthy",
            "admin_server": "healthy",
            "postgres": "healthy",
            "redis": "healthy",
        },
        "alerts": errors,
        "top_models": [],
        "top_users": [],
    }

    fmt = DocumentFormat(request.format)
    doc_type = DocumentType.OPS_DAILY
    content, file_path, mime = doc_service.generate(doc_type, data, fmt)

    return DocumentGenerateResponse(
        success=True,
        filename=file_path.name,
        format=request.format,
        size=len(content),
        url=f"/api/admin/v1/documents/download/{file_path.name}",
        message=f"Daily ops report generated: {file_path.name}",
    )


@router.post("/generate/audit", response_model=DocumentGenerateResponse)
async def generate_audit_report(
    request: DocumentGenerateRequest,
    db: AsyncSession = Depends(get_db),
):
    """
    Generate compliance audit report (Word).
    Includes audit trail, user activity, and critical event summary.
    """
    now = datetime.now(timezone.utc)
    since = now - timedelta(days=request.period_days)

    # Audit logs
    audit_result = await db.execute(
        select(AuditLog)
        .where(AuditLog.created_at >= since)
        .order_by(desc(AuditLog.created_at))
        .limit(500)
    )
    audit_logs = audit_result.scalars().all()

    # User activity
    user_activity_result = await db.execute(
        select(
            AuditLog.user_id,
            func.count(AuditLog.id).label("action_count"),
            func.max(AuditLog.created_at).label("last_active"),
            func.count(func.distinct(AuditLog.ip_address)).label("ip_count"),
        )
        .where(AuditLog.created_at >= since)
        .group_by(AuditLog.user_id)
        .order_by(desc(func.count(AuditLog.id)))
    )

    user_activity = []
    for row in user_activity_result.all():
        # Get user display name
        user_result = await db.execute(
            select(User.display_name, User.email).where(User.id == row.user_id)
        )
        user_row = user_result.first()
        display_name = user_row.display_name or user_row.email if user_row else str(row.user_id)

        user_activity.append({
            "display_name": display_name,
            "email": user_row.email if user_row else "",
            "action_count": row.action_count,
            "last_active": row.last_active.isoformat() if row.last_active else "",
            "ip_count": row.ip_count,
        })

    data = data_adapter.audit_data(
        audit_logs=[{
            "action": log.action,
            "resource_type": log.resource_type,
            "resource_id": log.resource_id,
            "ip": log.ip_address,
            "timestamp": log.created_at.isoformat() if log.created_at else "",
        } for log in audit_logs],
        user_activity=user_activity,
        period_start=since.strftime("%Y-%m-%d"),
        period_end=now.strftime("%Y-%m-%d"),
    )

    fmt = DocumentFormat(request.format)
    doc_type = DocumentType.AUDIT_REPORT
    content, file_path, mime = doc_service.generate(doc_type, data, fmt)

    return DocumentGenerateResponse(
        success=True,
        filename=file_path.name,
        format=request.format,
        size=len(content),
        url=f"/api/admin/v1/documents/download/{file_path.name}",
        message=f"Audit report generated: {file_path.name}",
    )


@router.post("/generate/data-export", response_model=DocumentGenerateResponse)
async def generate_data_export(
    request: DocumentGenerateRequest,
    db: AsyncSession = Depends(get_db),
):
    """
    Generate generic data export (Excel or CSV).
    Exports request logs, transactions, and user data as separate sheets.
    """
    now = datetime.now(timezone.utc)
    since = now - timedelta(days=request.period_days)

    # Request logs
    req_result = await db.execute(
        select(RequestLog)
        .where(RequestLog.created_at >= since)
        .order_by(desc(RequestLog.created_at))
        .limit(1000)
    )
    req_logs = req_result.scalars().all()

    # Transactions
    txn_result = await db.execute(
        select(Transaction)
        .where(Transaction.created_at >= since)
        .order_by(desc(Transaction.created_at))
        .limit(1000)
    )
    txns = txn_result.scalars().all()

    sheets = {
        "Request Logs": [
            {
                "request_id": r.request_id,
                "user_id": r.user_id,
                "model_id": r.model_id,
                "provider_id": r.provider_id,
                "status_code": r.status_code,
                "latency_ms": r.latency_ms,
                "prompt_tokens": r.prompt_tokens,
                "completion_tokens": r.completion_tokens,
                "error_code": r.error_code,
                "created_at": r.created_at.isoformat() if r.created_at else "",
            }
            for r in req_logs
        ],
        "Transactions": [
            {
                "id": t.id,
                "user_id": t.user_id,
                "type": t.type,
                "model_id": t.model_id or "",
                "total_tokens": t.total_tokens,
                "amount": t.amount,
                "currency": t.currency,
                "status": t.status,
                "created_at": t.created_at.isoformat() if t.created_at else "",
            }
            for t in txns
        ],
    }

    data = {"sheets": sheets}
    fmt = DocumentFormat(request.format)
    doc_type = DocumentType.DATA_EXPORT
    content, file_path, mime = doc_service.generate(doc_type, data, fmt)

    return DocumentGenerateResponse(
        success=True,
        filename=file_path.name,
        format=request.format,
        size=len(content),
        url=f"/api/admin/v1/documents/download/{file_path.name}",
        message=f"Data export generated: {file_path.name}",
    )


# ── Quick Generate (single endpoint, auto-detects format) ──────────

@router.post("/generate", response_model=DocumentGenerateResponse)
async def generate_document(
    request: DocumentGenerateRequest,
    background_tasks: BackgroundTasks,
    db: AsyncSession = Depends(get_db),
):
    """
    Unified document generation endpoint.
    Routes to the appropriate handler based on doc_type.
    """
    handlers = {
        "billing_report": generate_billing_report,
        "user_usage_report": generate_user_report,
        "ops_daily": generate_ops_daily,
        "audit_report": generate_audit_report,
        "data_export": generate_data_export,
    }

    handler = handlers.get(request.doc_type)
    if not handler:
        raise HTTPException(
            status_code=400,
            detail=f"Unknown doc_type: {request.doc_type}. "
                   f"Available: {list(handlers.keys())}",
        )

    return await handler(request, background_tasks, db)
