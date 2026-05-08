"""
Admin system settings endpoints
"""
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from pydantic import BaseModel

from app.core.database import get_db
from app.models.routing import AuditLog
from app.models.user import User
from app.core.security import hash_password

router = APIRouter(tags=["Settings"])


# ============================================
# System Config
# ============================================

@router.get("/system/config")
async def get_system_config():
    """Get system configuration."""
    return {
        "rate_limit": {
            "free_rpm": 100,
            "free_tpm": 10000,
            "pro_rpm": 1000,
            "pro_tpm": 100000,
        },
        "router": {
            "complexity_threshold": 5,
            "strategy": "cost_optimized",
            "cache_ttl_seconds": 3600,
        },
        "settlement": {
            "time_utc": "00:00",
            "l2_network": "Polygon",
        },
        "pricing": {
            "self_hosted_input": 0.15,
            "self_hosted_output": 0.50,
            "take_rate_pct": 3.0,
            "cred_discount_pct": 30,
        },
    }


@router.put("/system/config/{key}")
async def update_system_config(key: str):
    """Update system config (placeholder)."""
    return {"key": key, "status": "updated"}


# ============================================
# Admin Users
# ============================================

@router.get("/system/admins")
async def list_admins(db: AsyncSession = Depends(get_db)):
    """List admin users."""
    result = await db.execute(select(User).order_by(User.created_at.desc()).limit(50))
    users = result.scalars().all()

    return {
        "data": [{
            "id": u.id,
            "email": u.email,
            "display_name": u.display_name,
            "role": "admin" if u.plan_id == "enterprise" else "user",
            "status": u.status,
            "last_login_at": u.last_login_at.isoformat() if u.last_login_at else None,
        } for u in users]
    }


class CreateAdminRequest(BaseModel):
    email: str
    password: str
    display_name: str


@router.post("/system/admins")
async def create_admin(req: CreateAdminRequest, db: AsyncSession = Depends(get_db)):
    """Create a new admin user."""
    existing = await db.scalar(select(User.id).where(User.email == req.email))
    if existing:
        raise HTTPException(status_code=400, detail="Email already exists")

    user = User(
        email=req.email,
        password_hash=hash_password(req.password),
        display_name=req.display_name,
        plan_id="enterprise",
        email_verified=True,
    )
    db.add(user)
    await db.flush()

    return {"id": user.id, "email": user.email}


# ============================================
# Audit Logs
# ============================================

@router.get("/audit-logs")
async def list_audit_logs(
    page: int = 1,
    page_size: int = 20,
    user_id: str | None = None,
    action: str | None = None,
    db: AsyncSession = Depends(get_db),
):
    """List audit logs."""
    query = select(AuditLog)
    count_query = select(func.count(AuditLog.id))

    if user_id:
        query = query.where(AuditLog.user_id == user_id)
        count_query = count_query.where(AuditLog.user_id == user_id)
    if action:
        query = query.where(AuditLog.action == action)
        count_query = count_query.where(AuditLog.action == action)

    total = await db.scalar(count_query)
    result = await db.execute(
        query.order_by(AuditLog.created_at.desc())
        .offset((page - 1) * page_size)
        .limit(page_size)
    )
    logs = result.scalars().all()

    return {
        "total": total,
        "page": page,
        "page_size": page_size,
        "data": [{
            "id": log.id,
            "user_id": log.user_id,
            "action": log.action,
            "resource_type": log.resource_type,
            "resource_id": log.resource_id,
            "ip_address": log.ip_address,
            "created_at": log.created_at.isoformat() if log.created_at else None,
        } for log in logs]
    }
