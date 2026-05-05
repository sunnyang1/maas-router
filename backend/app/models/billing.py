"""
Balance and Transaction models
"""
import uuid
from datetime import datetime, timezone

from sqlalchemy import JSON, String, Integer, Float, Text, ForeignKey, DateTime
from sqlalchemy.orm import Mapped, mapped_column, relationship


from app.core.database import Base


class Balance(Base):
    __tablename__ = "balances"

    id: Mapped[str] = mapped_column(
        String(36), primary_key=True, default=lambda: str(uuid.uuid4())
    )
    user_id: Mapped[str] = mapped_column(
        String(36), ForeignKey("users.id"), unique=True, nullable=False
    )
    team_id: Mapped[str | None] = mapped_column(
        String(36), ForeignKey("teams.id"), nullable=True
    )
    cred_balance: Mapped[float] = mapped_column(Float, default=0.0)
    usd_balance: Mapped[float] = mapped_column(Float, default=0.0)
    frozen_cred: Mapped[float] = mapped_column(Float, default=0.0)
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), default=lambda: datetime.now(timezone.utc),
        onupdate=lambda: datetime.now(timezone.utc)
    )

    user: Mapped["User"] = relationship(back_populates="balance")


class Transaction(Base):
    __tablename__ = "transactions"

    id: Mapped[str] = mapped_column(
        String(36), primary_key=True, default=lambda: str(uuid.uuid4())
    )
    user_id: Mapped[str] = mapped_column(
        String(36), ForeignKey("users.id"), nullable=False
    )
    team_id: Mapped[str | None] = mapped_column(
        String(36), ForeignKey("teams.id"), nullable=True
    )
    api_key_id: Mapped[str | None] = mapped_column(
        String(36), ForeignKey("api_keys.id"), nullable=True
    )
    type: Mapped[str] = mapped_column(String(20), nullable=False)  # usage, topup, refund, bonus
    request_id: Mapped[str | None] = mapped_column(String(100))
    model_id: Mapped[str | None] = mapped_column(String(100))
    provider_id: Mapped[str | None] = mapped_column(String(50))
    prompt_tokens: Mapped[int] = mapped_column(Integer, default=0)
    completion_tokens: Mapped[int] = mapped_column(Integer, default=0)
    total_tokens: Mapped[int] = mapped_column(Integer, default=0)
    amount: Mapped[float] = mapped_column(Float, nullable=False)
    currency: Mapped[str] = mapped_column(String(10), default="CRED")
    unit_price: Mapped[float | None] = mapped_column(Float)
    route_reason: Mapped[str | None] = mapped_column(Text)
    route_confidence: Mapped[float | None] = mapped_column(Float)
    status: Mapped[str] = mapped_column(String(20), default="completed")
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), default=lambda: datetime.now(timezone.utc)
    )
