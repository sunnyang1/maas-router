"""
Provider and Model models
"""
from datetime import datetime, timezone

from sqlalchemy import JSON, String, Text, Integer, Float, DateTime, ForeignKey

from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.core.database import Base


class Provider(Base):
    __tablename__ = "providers"

    id: Mapped[str] = mapped_column(String(50), primary_key=True)
    name: Mapped[str] = mapped_column(String(100), nullable=False)
    logo_url: Mapped[str | None] = mapped_column(Text)
    description: Mapped[str | None] = mapped_column(Text)
    api_base_url: Mapped[str | None] = mapped_column(Text)
    status: Mapped[str] = mapped_column(String(20), default="active")  # active, degraded, offline
    config: Mapped[dict | None] = mapped_column(JSON, default=dict)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), default=lambda: datetime.now(timezone.utc)
    )

    models: Mapped[list["Model"]] = relationship(back_populates="provider", lazy="selectin")


class Model(Base):
    __tablename__ = "models"

    id: Mapped[str] = mapped_column(String(100), primary_key=True)
    provider_id: Mapped[str] = mapped_column(String(50), ForeignKey("providers.id"), nullable=False)
    name: Mapped[str] = mapped_column(String(100), nullable=False)
    description: Mapped[str | None] = mapped_column(Text)
    tags: Mapped[list | None] = mapped_column(JSON)
    context_window: Mapped[int | None] = mapped_column(Integer)
    input_price: Mapped[float | None] = mapped_column(Float)  # $/1M tokens
    output_price: Mapped[float | None] = mapped_column(Float)
    features: Mapped[list | None] = mapped_column(JSON)
    status: Mapped[str] = mapped_column(String(20), default="active")
    popularity: Mapped[int] = mapped_column(Integer, default=0)
    is_recommended: Mapped[bool] = mapped_column(default=False)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), default=lambda: datetime.now(timezone.utc)
    )

    provider: Mapped["Provider"] = relationship(back_populates="models", lazy="selectin")
