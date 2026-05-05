"""
Admin model & provider management endpoints
"""
from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from pydantic import BaseModel

from app.core.database import get_db
from app.models.provider import Provider, Model
from app.models.routing import RoutingRule

router = APIRouter(tags=["Models & Providers"])


# ============================================
# Providers
# ============================================

@router.get("/providers")
async def list_providers(db: AsyncSession = Depends(get_db)):
    """List all providers."""
    result = await db.execute(select(Provider).order_by(Provider.name))
    providers = result.scalars().all()

    # Get model counts per provider
    count_result = await db.execute(
        select(Model.provider_id, func.count(Model.id))
        .group_by(Model.provider_id)
    )
    counts = {row[0]: row[1] for row in count_result.all()}

    return {
        "data": [{
            "id": p.id,
            "name": p.name,
            "logo_url": p.logo_url,
            "description": p.description,
            "api_base_url": p.api_base_url,
            "status": p.status,
            "model_count": counts.get(p.id, 0),
            "created_at": p.created_at.isoformat() if p.created_at else None,
        } for p in providers]
    }


class ProviderUpdate(BaseModel):
    name: str | None = None
    description: str | None = None
    api_base_url: str | None = None
    status: str | None = None
    config: dict | None = None


@router.put("/providers/{provider_id}")
async def update_provider(provider_id: str, req: ProviderUpdate, db: AsyncSession = Depends(get_db)):
    """Update provider."""
    result = await db.execute(select(Provider).where(Provider.id == provider_id))
    p = result.scalar_one_or_none()
    if not p:
        raise HTTPException(status_code=404, detail="Provider not found")

    for key, value in req.model_dump(exclude_unset=True).items():
        setattr(p, key, value)

    return {"id": p.id, "status": "updated"}


@router.put("/providers/{provider_id}/status")
async def toggle_provider_status(provider_id: str, status: str = "active", db: AsyncSession = Depends(get_db)):
    """Enable or disable a provider."""
    result = await db.execute(select(Provider).where(Provider.id == provider_id))
    p = result.scalar_one_or_none()
    if not p:
        raise HTTPException(status_code=404, detail="Provider not found")

    p.status = status
    return {"id": p.id, "status": p.status}


# ============================================
# Models
# ============================================

@router.get("/models")
async def list_models_admin(
    provider_id: str | None = None,
    search: str | None = None,
    status: str | None = None,
    db: AsyncSession = Depends(get_db),
):
    """List all models for admin."""
    query = select(Model, Provider.name).join(Provider, Model.provider_id == Provider.id)

    if provider_id:
        query = query.where(Model.provider_id == provider_id)
    if status:
        query = query.where(Model.status == status)
    if search:
        query = query.where(Model.name.ilike(f"%{search}%"))

    result = await db.execute(query.order_by(Model.popularity.desc()))
    rows = result.all()

    return {
        "data": [{
            "id": m.id,
            "provider_id": m.provider_id,
            "provider_name": p_name,
            "name": m.name,
            "description": m.description,
            "tags": m.tags,
            "context_window": m.context_window,
            "input_price": m.input_price,
            "output_price": m.output_price,
            "features": m.features,
            "status": m.status,
            "popularity": m.popularity,
            "is_recommended": m.is_recommended,
        } for m, p_name in rows]
    }


class ModelUpdate(BaseModel):
    name: str | None = None
    description: str | None = None
    input_price: float | None = None
    output_price: float | None = None
    status: str | None = None
    is_recommended: bool | None = None
    tags: list | None = None
    context_window: int | None = None


@router.put("/models/{model_id}")
async def update_model(model_id: str, req: ModelUpdate, db: AsyncSession = Depends(get_db)):
    """Update model."""
    result = await db.execute(select(Model).where(Model.id == model_id))
    m = result.scalar_one_or_none()
    if not m:
        raise HTTPException(status_code=404, detail="Model not found")

    for key, value in req.model_dump(exclude_unset=True).items():
        setattr(m, key, value)

    return {"id": m.id, "status": "updated"}


@router.put("/models/{model_id}/status")
async def toggle_model_status(model_id: str, status: str = "active", db: AsyncSession = Depends(get_db)):
    """Enable or disable a model."""
    result = await db.execute(select(Model).where(Model.id == model_id))
    m = result.scalar_one_or_none()
    if not m:
        raise HTTPException(status_code=404, detail="Model not found")

    m.status = status
    return {"id": m.id, "status": m.status}


# ============================================
# Routing Rules
# ============================================

@router.get("/routing-rules")
async def list_routing_rules(db: AsyncSession = Depends(get_db)):
    """List routing rules."""
    result = await db.execute(
        select(RoutingRule).order_by(RoutingRule.priority.desc())
    )
    rules = result.scalars().all()

    return {
        "data": [{
            "id": r.id,
            "name": r.name,
            "description": r.description,
            "priority": r.priority,
            "condition": r.condition,
            "action": r.action,
            "status": r.status,
            "created_at": r.created_at.isoformat() if r.created_at else None,
        } for r in rules]
    }


class RoutingRuleCreate(BaseModel):
    name: str
    description: str | None = None
    priority: int = 0
    condition: dict
    action: dict


@router.post("/routing-rules")
async def create_routing_rule(req: RoutingRuleCreate, db: AsyncSession = Depends(get_db)):
    """Create a routing rule."""
    rule = RoutingRule(**req.model_dump())
    db.add(rule)
    await db.flush()
    return {"id": rule.id, "name": rule.name}


@router.delete("/routing-rules/{rule_id}")
async def delete_routing_rule(rule_id: str, db: AsyncSession = Depends(get_db)):
    """Delete a routing rule."""
    result = await db.execute(select(RoutingRule).where(RoutingRule.id == rule_id))
    rule = result.scalar_one_or_none()
    if not rule:
        raise HTTPException(status_code=404, detail="Rule not found")

    await db.delete(rule)
    return {"id": rule_id, "status": "deleted"}
