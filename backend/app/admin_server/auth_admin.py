"""
Admin authentication endpoints.
"""
from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.core.security import get_current_user_from_jwt
from app.models.user import User
from app.services.auth_service import AuthService

router = APIRouter(tags=["Admin Auth"])


from pydantic import BaseModel

class LoginRequest(BaseModel):
    email: str
    password: str


@router.post("/auth/login")
async def login(req: LoginRequest, db: AsyncSession = Depends(get_db)):
    """Admin login endpoint."""
    result = await AuthService(db).authenticate(req.email, req.password)

    if result is None:
        raise HTTPException(status_code=401, detail="邮箱或密码错误")

    token, user_info = result
    return {
        "access_token": token,
        "token_type": "bearer",
        "user": user_info,
    }


@router.get("/auth/me")
async def me(user: User = Depends(get_current_user_from_jwt)):
    """Get current user info."""
    return {
        "id": user.id, "email": user.email,
        "display_name": user.display_name, "plan_id": user.plan_id,
        "status": user.status,
        "created_at": user.created_at.isoformat() if user.created_at else None,
    }
