"""
Admin user management endpoints.
"""
from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from pydantic import BaseModel

from app.core.database import get_db
from app.core.security import get_current_user_from_jwt, hash_password
from app.models.user import User
from app.models.billing import Balance
from app.repositories.user_repo import UserRepository
from app.repositories.balance_repo import BalanceRepository
from app.repositories.api_key_repo import ApiKeyRepository
from app.repositories.request_log_repo import RequestLogRepository

router = APIRouter(tags=["Users"])


class UserCreate(BaseModel):
    email: str
    password: str
    display_name: str | None = None
    plan_id: str = "free"


class UserUpdate(BaseModel):
    display_name: str | None = None
    plan_id: str | None = None
    status: str | None = None


@router.get("/users")
async def list_users(
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    search: str | None = None,
    db: AsyncSession = Depends(get_db),
):
    """List users with pagination and search."""
    repo = UserRepository(db)
    offset = (page - 1) * page_size
    rows, total = await repo.list_with_balance(search, offset, page_size)

    users_data = []
    for u, cred in rows:
        users_data.append({
            "id": u.id, "email": u.email,
            "display_name": u.display_name, "plan_id": u.plan_id,
            "status": u.status, "email_verified": u.email_verified,
            "cred_balance": round(float(cred), 4),
            "created_at": u.created_at.isoformat() if u.created_at else None,
            "last_login_at": u.last_login_at.isoformat() if u.last_login_at else None,
        })

    return {"total": total, "page": page, "page_size": page_size, "data": users_data}


@router.get("/users/{user_id}")
async def get_user(user_id: str, db: AsyncSession = Depends(get_db)):
    """Get user details."""
    user_repo = UserRepository(db)
    user = await user_repo.get_by_id(user_id)
    if not user:
        raise HTTPException(status_code=404, detail="User not found")

    bal_repo = BalanceRepository(db)
    bal = await bal_repo.get_by_user_id(user.id)

    key_repo = ApiKeyRepository(db)
    key_count = await key_repo.count_by_user(user.id)

    log_repo = RequestLogRepository(db)
    total_requests = await log_repo.count_by_user(user.id)

    return {
        "id": user.id, "email": user.email,
        "display_name": user.display_name, "plan_id": user.plan_id,
        "status": user.status, "email_verified": user.email_verified,
        "cred_balance": round(float(bal.cred_balance if bal else 0), 4),
        "api_key_count": key_count or 0,
        "total_requests": total_requests or 0,
        "created_at": user.created_at.isoformat() if user.created_at else None,
        "last_login_at": user.last_login_at.isoformat() if user.last_login_at else None,
    }


@router.post("/users")
async def create_user(req: UserCreate, db: AsyncSession = Depends(get_db)):
    """Create a new user."""
    repo = UserRepository(db)
    existing = await repo.get_by_email(req.email)
    if existing:
        raise HTTPException(status_code=400, detail="Email already exists")

    user = User(
        email=req.email,
        password_hash=hash_password(req.password),
        display_name=req.display_name,
        plan_id=req.plan_id,
    )
    await repo.create(user)

    bal_repo = BalanceRepository(db)
    await bal_repo.create(Balance(user_id=user.id))

    return {
        "id": user.id, "email": user.email,
        "display_name": user.display_name, "plan_id": user.plan_id,
    }


@router.put("/users/{user_id}")
async def update_user(user_id: str, req: UserUpdate, db: AsyncSession = Depends(get_db)):
    """Update user."""
    repo = UserRepository(db)
    user = await repo.get_by_id(user_id)
    if not user:
        raise HTTPException(status_code=404, detail="User not found")

    if req.display_name is not None:
        user.display_name = req.display_name
    if req.plan_id is not None:
        user.plan_id = req.plan_id
    if req.status is not None:
        user.status = req.status

    return {"id": user.id, "status": "updated"}


@router.get("/users/{user_id}/api-keys")
async def get_user_api_keys(user_id: str, db: AsyncSession = Depends(get_db)):
    """Get user's API keys."""
    repo = ApiKeyRepository(db)
    keys = await repo.list_by_user(user_id)

    return {
        "data": [{
            "id": k.id, "name": k.name, "prefix": k.key_prefix,
            "status": k.status,
            "rate_limit_rpm": k.rate_limit_rpm,
            "rate_limit_tpm": k.rate_limit_tpm,
            "last_used_at": k.last_used_at.isoformat() if k.last_used_at else None,
            "created_at": k.created_at.isoformat() if k.created_at else None,
        } for k in keys]
    }


@router.get("/users/{user_id}/transactions")
async def get_user_transactions(
    user_id: str,
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
):
    """Get user's transaction history."""
    from app.repositories.transaction_repo import TransactionRepository
    repo = TransactionRepository(db)
    offset = (page - 1) * page_size
    rows, total = await repo.list_with_user(
        user_id=user_id, offset=offset, limit=page_size
    )

    return {
        "total": total, "page": page, "page_size": page_size,
        "data": [{
            "id": txn.id, "type": txn.type, "amount": txn.amount,
            "currency": txn.currency, "model_id": txn.model_id,
            "total_tokens": txn.total_tokens,
            "route_reason": txn.route_reason, "status": txn.status,
            "created_at": txn.created_at.isoformat() if txn.created_at else None,
        } for txn, email, name in rows]
    }
