"""
User model
"""
import uuid
from datetime import datetime, timezone

from sqlalchemy import String, Boolean, DateTime, Text
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.core.database import Base


class User(Base):
    __tablename__ = "users"

    id: Mapped[str] = mapped_column(
        String(36), primary_key=True, default=lambda: str(uuid.uuid4())
    )
    email: Mapped[str] = mapped_column(String(255), unique=True, nullable=False, index=True)
    password_hash: Mapped[str] = mapped_column(String(255), nullable=False)
    display_name: Mapped[str | None] = mapped_column(String(100))
    avatar_url: Mapped[str | None] = mapped_column(Text)

    status: Mapped[str] = mapped_column(String(20), default="active")  # active, suspended, deleted
    email_verified: Mapped[bool] = mapped_column(Boolean, default=False)

    plan_id: Mapped[str] = mapped_column(String(20), default="free")  # free, pro, enterprise

    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), default=lambda: datetime.now(timezone.utc)
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), default=lambda: datetime.now(timezone.utc),
        onupdate=lambda: datetime.now(timezone.utc)
    )
    last_login_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))

    # Relationships
    api_keys: Mapped[list["ApiKey"]] = relationship(back_populates="user", lazy="selectin")
    balance: Mapped["Balance | None"] = relationship(back_populates="user", uselist=False, lazy="selectin")
