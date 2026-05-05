"""
Authentication & Security utilities
"""
import hashlib
import secrets
from datetime import datetime, timedelta, timezone

from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from jose import JWTError, jwt
from passlib.context import CryptContext
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.config import get_settings
from app.core.database import get_db

settings = get_settings()
pwd_context = CryptContext(schemes=["bcrypt"], deprecated="auto")
bearer_scheme = HTTPBearer(auto_error=False)


# ============================================
# Password Hashing
# ============================================

def hash_password(password: str) -> str:
    return pwd_context.hash(password)


def verify_password(plain: str, hashed: str) -> bool:
    return pwd_context.verify(plain, hashed)


# ============================================
# JWT Token
# ============================================

def create_jwt_token(data: dict, expires_minutes: int = None) -> str:
    if expires_minutes is None:
        expires_minutes = settings.jwt_expire_minutes
    to_encode = data.copy()
    expire = datetime.now(timezone.utc) + timedelta(minutes=expires_minutes)
    to_encode.update({"exp": expire})
    return jwt.encode(to_encode, settings.jwt_secret_key, algorithm=settings.jwt_algorithm)


def decode_jwt_token(token: str) -> dict:
    try:
        return jwt.decode(token, settings.jwt_secret_key, algorithms=[settings.jwt_algorithm])
    except JWTError:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid token")


# ============================================
# API Key
# ============================================

def generate_api_key() -> tuple[str, str, str]:
    """Generate API key. Returns (full_key, prefix, key_hash)."""
    random_part = secrets.token_hex(24)  # 48 chars
    prefix = f"{settings.api_key_prefix}{secrets.token_hex(4)}"
    full_key = f"{prefix}-{random_part}"
    key_hash = hashlib.sha256(full_key.encode()).hexdigest()
    return full_key, prefix, key_hash


def hash_api_key(key: str) -> str:
    return hashlib.sha256(key.encode()).hexdigest()


# ============================================
# Auth Dependencies
# ============================================

async def get_current_user_from_jwt(
    credentials: HTTPAuthorizationCredentials = Depends(bearer_scheme),
    db: AsyncSession = Depends(get_db),
):
    """Validate JWT token and return current user."""
    if not credentials:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Missing token")

    payload = decode_jwt_token(credentials.credentials)
    user_id = payload.get("sub")
    if not user_id:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid token payload")

    from app.models.user import User
    result = await db.execute(select(User).where(User.id == user_id))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="User not found")
    if user.status != "active":
        raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail="Account suspended")

    return user


async def get_current_user_from_api_key(
    credentials: HTTPAuthorizationCredentials = Depends(bearer_scheme),
    db: AsyncSession = Depends(get_db),
):
    """Validate API Key and return current user."""
    if not credentials:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Missing API key")

    from app.models.api_key import ApiKey
    key_hash = hash_api_key(credentials.credentials)
    result = await db.execute(select(ApiKey).where(ApiKey.key_hash == key_hash))
    api_key = result.scalar_one_or_none()

    if not api_key:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid API key")
    if api_key.status != "active":
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="API key revoked")

    # Update last used
    api_key.last_used_at = datetime.now(timezone.utc)
    await db.flush()

    return api_key


async def get_current_user(
    credentials: HTTPAuthorizationCredentials = Depends(bearer_scheme),
    db: AsyncSession = Depends(get_db),
):
    """Try API key first, fallback to JWT."""
    if not credentials:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Missing credentials")

    token = credentials.credentials

    # Try API Key (has prefix "sk-mr-")
    if token.startswith(settings.api_key_prefix):
        from app.models.api_key import ApiKey
        from app.models.user import User
        key_hash = hash_api_key(token)
        result = await db.execute(select(ApiKey).where(ApiKey.key_hash == key_hash))
        api_key = result.scalar_one_or_none()
        if api_key and api_key.status == "active":
            api_key.last_used_at = datetime.now(timezone.utc)
            result_user = await db.execute(select(User).where(User.id == api_key.user_id))
            user = result_user.scalar_one_or_none()
            return user
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid API key")

    # Try JWT
    try:
        payload = decode_jwt_token(token)
        user_id = payload.get("sub")
        from app.models.user import User
        result = await db.execute(select(User).where(User.id == user_id))
        user = result.scalar_one_or_none()
        return user
    except Exception:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid credentials")
