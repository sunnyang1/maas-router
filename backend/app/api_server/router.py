"""
API Server router - OpenAI-compatible endpoints.

Thin route handlers that delegate business logic to services.
"""
import time
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, Header
from fastapi.responses import StreamingResponse
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.core.security import get_current_user
from app.models.user import User
from app.models.provider import Model, Provider
from app.schemas.chat import ChatCompletionRequest
from app.services.chat_service import ChatService
from app.services.billing_service import BillingService
from app.repositories.model_repo import ModelRepository
from app.repositories.api_key_repo import ApiKeyRepository
from app.repositories.request_log_repo import RequestLogRepository
from app.repositories.balance_repo import BalanceRepository

router = APIRouter(prefix="/v1", tags=["API"])

# ============================================
# GET /v1/models
# ============================================

@router.get("/models")
async def list_models(db: AsyncSession = Depends(get_db)):
    """List all available models (OpenAI-compatible)."""
    repo = ModelRepository(db)
    rows = await repo.list_active_with_provider()

    models_list = []
    for model, provider in rows:
        models_list.append({
            "id": model.id,
            "object": "model",
            "created": int(model.created_at.timestamp()) if model.created_at else 0,
            "owned_by": provider.name,
            "provider": {"id": provider.id, "name": provider.name},
            "context_window": model.context_window,
            "pricing": {"input": model.input_price, "output": model.output_price},
            "tags": model.tags or [],
            "features": model.features or [],
            "is_recommended": model.is_recommended,
        })

    return {"object": "list", "data": models_list}


@router.get("/models/{model_id}")
async def get_model(model_id: str, db: AsyncSession = Depends(get_db)):
    """Get a specific model by ID."""
    repo = ModelRepository(db)
    row = await repo.get_with_provider(model_id)
    if not row:
        raise HTTPException(status_code=404, detail=f"Model '{model_id}' not found")

    model, provider = row
    return {
        "id": model.id, "object": "model",
        "created": int(model.created_at.timestamp()) if model.created_at else 0,
        "owned_by": provider.name,
        "provider": {"id": provider.id, "name": provider.name},
        "context_window": model.context_window,
        "pricing": {"input": model.input_price, "output": model.output_price},
        "tags": model.tags or [],
        "features": model.features or [],
    }


# ============================================
# POST /v1/chat/completions
# ============================================

@router.post("/chat/completions")
async def chat_completions(
    body: ChatCompletionRequest,
    db: AsyncSession = Depends(get_db),
    user: User = Depends(get_current_user),
    x_idempotency_key: Optional[str] = Header(None, alias="X-Idempotency-Key"),
):
    """Chat completions endpoint (OpenAI-compatible).

    Delegates all business logic to ChatService.
    """
    service = ChatService(db)

    if body.stream:
        return await service.complete_stream(user, body, x_idempotency_key)
    else:
        return await service.complete(user, body, x_idempotency_key)


# ============================================
# API Key Management
# ============================================

@router.get("/keys")
async def list_api_keys(
    user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    """List user's API keys."""
    repo = ApiKeyRepository(db)
    keys = await repo.list_by_user(user.id)

    return {
        "object": "list",
        "data": [{
            "id": k.id, "name": k.name, "prefix": k.key_prefix,
            "status": k.status,
            "last_used_at": k.last_used_at.isoformat() if k.last_used_at else None,
            "created_at": k.created_at.isoformat() if k.created_at else None,
            "rate_limit_rpm": k.rate_limit_rpm,
            "rate_limit_tpm": k.rate_limit_tpm,
        } for k in keys]
    }


from pydantic import BaseModel

class CreateKeyRequest(BaseModel):
    name: str = "Default"

@router.post("/keys")
async def create_api_key(
    req: CreateKeyRequest,
    user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    """Create a new API key."""
    from app.core.security import generate_api_key
    from app.models.api_key import ApiKey

    full_key, prefix, key_hash = generate_api_key()

    api_key = ApiKey(
        user_id=user.id,
        name=req.name,
        key_hash=key_hash,
        key_prefix=prefix,
    )
    db.add(api_key)
    await db.flush()

    return {
        "id": api_key.id, "name": api_key.name,
        "key": full_key, "prefix": prefix,
        "status": api_key.status,
        "created_at": api_key.created_at.isoformat() if api_key.created_at else None,
    }


@router.delete("/keys/{key_id}")
async def revoke_api_key(
    key_id: str,
    user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    """Revoke an API key."""
    repo = ApiKeyRepository(db)
    from app.models.api_key import ApiKey

    # Find key belonging to user
    result = await db.execute(
        __import__('sqlalchemy').select(ApiKey).where(
            ApiKey.id == key_id, ApiKey.user_id == user.id
        )
    )
    api_key = result.scalar_one_or_none()
    if not api_key:
        raise HTTPException(status_code=404, detail="API key not found")

    api_key.status = "revoked"
    return {"id": key_id, "status": "revoked"}


# ============================================
# Balance
# ============================================

@router.get("/balance")
async def get_balance(
    user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    """Get user's $CRED balance."""
    repo = BalanceRepository(db)
    balance = await repo.get_by_user_id(user.id)

    if not balance:
        return {"cred_balance": 0.0, "usd_balance": 0.0}

    return {
        "cred_balance": balance.cred_balance,
        "usd_balance": balance.usd_balance,
        "frozen_cred": balance.frozen_cred,
    }


# ============================================
# Usage Summary
# ============================================

@router.get("/usage/summary")
async def usage_summary(
    user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    """Get user's usage summary."""
    service = BillingService(db)
    return await service.get_user_summary(user.id)


# ============================================
# Router Decision Log
# ============================================

@router.get("/router/decisions")
async def router_decisions(
    user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
    limit: int = 20,
):
    """Get recent routing decisions."""
    repo = RequestLogRepository(db)
    logs = await repo.list_by_user(user.id, limit)

    return {
        "object": "list",
        "data": [{
            "request_id": log.request_id, "model_id": log.model_id,
            "provider_id": log.provider_id,
            "complexity_score": log.complexity_score,
            "route_decision": log.route_decision,
            "prompt_tokens": log.prompt_tokens,
            "completion_tokens": log.completion_tokens,
            "latency_ms": log.latency_ms,
            "created_at": log.created_at.isoformat() if log.created_at else None,
        } for log in logs]
    }
