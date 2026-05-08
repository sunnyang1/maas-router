"""
Authentication service — JWT token creation and password management.
"""
from app.core.security import (
    hash_password,
    verify_password,
    create_jwt_token,
)


class AuthService:
    """Authentication logic extracted from route handlers."""

    @staticmethod
    def hash_password(password: str) -> str:
        return hash_password(password)

    @staticmethod
    def verify_password(plain: str, hashed: str) -> bool:
        return verify_password(plain, hashed)

    @staticmethod
    def create_token(user_id: str, email: str, role: str = "user") -> str:
        return create_jwt_token({
            "sub": user_id,
            "email": email,
            "role": role,
        })
