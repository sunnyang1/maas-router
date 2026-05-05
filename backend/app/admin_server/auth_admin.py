"""
Admin authentication endpoints
"""
from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from pydantic import BaseModel

from app.core.database import get_db
from app.core.security import verify_password, create_jwt_token, get_current_user_from_jwt
from app.models.user import User

router = APIRouter(tags=["Admin Auth"])


class LoginRequest(BaseModel):
    email: str
    password: str


class LoginResponse(BaseModel):
    access_token: str
    token_type: str = "bearer"
    user: dict


@router.post("/auth/login")
async def login(req: LoginRequest, db: AsyncSession = Depends(get_db)):
    """Admin login endpoint."""
    result = await db.execute(select(User).where(User.email == req.email))
    user = result.scalar_one_or_none()

    if not user or not verify_password(req.password, user.password_hash):
        raise HTTPException(status_code=401, detail="邮箱或密码错误")

    if user.status != "active":
        raise HTTPException(status_code=403, detail="账户已被禁用")

    token = create_jwt_token({"sub": user.id, "email": user.email, "role": "admin"})

    return {
        "access_token": token,
        "token_type": "bearer",
        "user": {
            "id": user.id,
            "email": user.email,
            "display_name": user.display_name,
            "plan_id": user.plan_id,
        }
    }


@router.get("/auth/me")
async def me(user: User = Depends(get_current_user_from_jwt)):
    """Get current user info."""
    return {
        "id": user.id,
        "email": user.email,
        "display_name": user.display_name,
        "plan_id": user.plan_id,
        "status": user.status,
        "created_at": user.created_at.isoformat() if user.created_at else None,
    }
