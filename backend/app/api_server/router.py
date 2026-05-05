"""
API Server router - OpenAI-compatible endpoints
"""
import time
import uuid
import json
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, Request, status
from fastapi.responses import StreamingResponse
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.core.security import get_current_user
from app.models.user import User
from app.models.provider import Model, Provider
from app.models.api_key import ApiKey
from app.models.billing import Balance, Transaction
from app.models.routing import RequestLog

router = APIRouter(prefix="/v1", tags=["API"])

# ============================================
# GET /v1/models
# ============================================

@router.get("/models")
async def list_models(db: AsyncSession = Depends(get_db)):
    """List all available models (OpenAI-compatible)."""
    result = await db.execute(
        select(Model, Provider)
        .join(Provider, Model.provider_id == Provider.id)
        .where(Model.status == "active")
        .order_by(Model.popularity.desc())
    )
    rows = result.all()

    models_list = []
    for model, provider in rows:
        models_list.append({
            "id": model.id,
            "object": "model",
            "created": int(model.created_at.timestamp()) if model.created_at else 0,
            "owned_by": provider.name,
            "provider": {
                "id": provider.id,
                "name": provider.name,
            },
            "context_window": model.context_window,
            "pricing": {
                "input": model.input_price,
                "output": model.output_price,
            },
            "tags": model.tags or [],
            "features": model.features or [],
            "is_recommended": model.is_recommended,
        })

    return {"object": "list", "data": models_list}


@router.get("/models/{model_id}")
async def get_model(model_id: str, db: AsyncSession = Depends(get_db)):
    """Get a specific model by ID."""
    result = await db.execute(
        select(Model, Provider)
        .join(Provider, Model.provider_id == Provider.id)
        .where(Model.id == model_id)
    )
    row = result.one_or_none()
    if not row:
        raise HTTPException(status_code=404, detail=f"Model '{model_id}' not found")

    model, provider = row
    return {
        "id": model.id,
        "object": "model",
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
    request: Request,
    db: AsyncSession = Depends(get_db),
    user: User = Depends(get_current_user),
):
    """Chat completions endpoint (OpenAI-compatible)."""
    body = await request.json()

    model_id = body.get("model", "gpt-4o-mini")
    messages = body.get("messages", [])
    stream = body.get("stream", False)
    temperature = body.get("temperature", 0.7)
    max_tokens = body.get("max_tokens", 1024)

    # Check balance
    balance_result = await db.execute(select(Balance).where(Balance.user_id == user.id))
    balance = balance_result.scalar_one_or_none()
    if not balance or balance.cred_balance <= 0:
        raise HTTPException(status_code=402, detail="Insufficient balance")

    # Resolve model - if "auto", use intelligent routing
    resolved_model = model_id
    resolved_provider = None
    route_reason = "直接指定"
    complexity_score = 5.0
    route_confidence = 1.0

    if model_id == "auto":
        # Intelligent routing: score complexity and pick best model
        prompt_text = " ".join([m.get("content", "") for m in messages if m.get("content")])
        complexity_score = _score_complexity(prompt_text)

        route_reason, resolved_model, resolved_provider = await _route_by_complexity(
            complexity_score, db
        )
        route_confidence = min(1.0, complexity_score / 10.0)
    else:
        # Resolve specific model
        model_result = await db.execute(select(Model).where(Model.id == model_id))
        model_obj = model_result.scalar_one_or_none()
        if not model_obj:
            raise HTTPException(status_code=404, detail=f"Model '{model_id}' not found")
        resolved_provider = model_obj.provider_id

    # Generate request ID
    request_id = f"chatcmpl-{uuid.uuid4().hex[:24]}"

    # Simulate token counts based on input
    prompt_tokens = sum(len(m.get("content", "")) // 4 for m in messages)
    completion_tokens = min(max_tokens, 256)

    # Calculate cost
    model_result = await db.execute(select(Model).where(Model.id == resolved_model))
    model_obj = model_result.scalar_one_or_none()
    input_price = model_obj.input_price if model_obj else 1.0
    output_price = model_obj.output_price if model_obj else 2.0
    cost_cred = (prompt_tokens / 1_000_000) * input_price + (completion_tokens / 1_000_000) * output_price

    # Log request
    log_entry = RequestLog(
        request_id=request_id,
        user_id=user.id,
        model_id=resolved_model,
        provider_id=resolved_provider,
        method="POST",
        endpoint="/v1/chat/completions",
        status_code=200,
        latency_ms=int(time.time() * 1000) % 500 + 100,
        prompt_tokens=prompt_tokens,
        completion_tokens=completion_tokens,
        complexity_score=round(complexity_score, 2),
        route_decision={"reason": route_reason, "model": resolved_model, "provider": resolved_provider},
    )
    db.add(log_entry)

    # Record transaction
    txn = Transaction(
        user_id=user.id,
        type="usage",
        request_id=request_id,
        model_id=resolved_model,
        provider_id=resolved_provider,
        prompt_tokens=prompt_tokens,
        completion_tokens=completion_tokens,
        total_tokens=prompt_tokens + completion_tokens,
        amount=-cost_cred,
        currency="CRED",
        route_reason=route_reason,
        route_confidence=route_confidence,
    )
    db.add(txn)

    # Deduct balance
    if balance:
        balance.cred_balance = round(balance.cred_balance - cost_cred, 6)

    if stream:
        return StreamingResponse(
            _stream_response(request_id, resolved_model, messages, completion_tokens),
            media_type="text/event-stream",
            headers={
                "X-Request-ID": request_id,
                "X-Router-Decision": json.dumps({"complexity_score": complexity_score, "model": resolved_model, "reason": route_reason}),
            },
        )

    # Non-streaming response
    response_content = _generate_mock_response(messages, resolved_model)
    return {
        "id": request_id,
        "object": "chat.completion",
        "created": int(time.time()),
        "model": resolved_model,
        "choices": [{
            "index": 0,
            "message": {
                "role": "assistant",
                "content": response_content,
            },
            "finish_reason": "stop",
        }],
        "usage": {
            "prompt_tokens": prompt_tokens,
            "completion_tokens": completion_tokens,
            "total_tokens": prompt_tokens + completion_tokens,
        },
        "router_decision": {
            "complexity_score": complexity_score,
            "route_reason": route_reason,
            "confidence": route_confidence,
            "cost_cred": round(cost_cred, 6),
        },
    }


async def _route_by_complexity(score: float, db: AsyncSession):
    """Route request based on complexity score."""
    if score < 4:
        return "简单查询，路由至自建 DeepSeek-V4", "deepseek-v4-self", "self-hosted"
    elif score < 7:
        return "中等复杂度，路由至 DeepSeek-V3", "deepseek-v3", "deepseek"
    elif score < 9:
        return "较高复杂度，路由至 GPT-4o Mini", "gpt-4o-mini", "openai"
    else:
        return "高复杂度，路由至 GPT-4o", "gpt-4o", "openai"


def _score_complexity(prompt: str) -> float:
    """Mock complexity scoring. In production, this calls Judge Agent (Qwen2.5-7B)."""
    length = len(prompt)
    code_keywords = ["def ", "function", "class ", "import ", "```", "async", "await", "SELECT", "WHERE"]
    reasoning_keywords = ["explain", "analyze", "compare", "summarize", "为什么", "分析", "对比"]
    creative_keywords = ["story", "poem", "write a", "creative", "故事", "诗歌", "创作"]

    score = 1.0

    # Length factor: longer prompts tend to be more complex
    if length > 2000:
        score += 3
    elif length > 1000:
        score += 2
    elif length > 500:
        score += 1

    # Keyword signals
    code_count = sum(1 for kw in code_keywords if kw.lower() in prompt.lower())
    reasoning_count = sum(1 for kw in reasoning_keywords if kw.lower() in prompt.lower())

    score += min(code_count, 3) * 1.5
    score += min(reasoning_count, 2) * 1.0

    return min(10.0, max(1.0, score))


async def _stream_response(request_id: str, model: str, messages: list, total_tokens: int):
    """Generate SSE stream response."""
    response_text = _generate_mock_response(messages, model)
    words = response_text.split()
    chunk_size = max(3, len(words) // 15)

    for i in range(0, len(words), chunk_size):
        chunk = " ".join(words[i:i+chunk_size])
        chunk_data = {
            "id": request_id,
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": model,
            "choices": [{
                "index": 0,
                "delta": {"content": chunk + " "},
                "finish_reason": None,
            }],
        }
        yield f"data: {json.dumps(chunk_data)}\n\n"
        import asyncio
        await asyncio.sleep(0.05)

    # Final chunk with finish_reason
    final = {
        "id": request_id,
        "object": "chat.completion.chunk",
        "created": int(time.time()),
        "model": model,
        "choices": [{
            "index": 0,
            "delta": {},
            "finish_reason": "stop",
        }],
    }
    yield f"data: {json.dumps(final)}\n\n"
    yield "data: [DONE]\n\n"


def _generate_mock_response(messages: list, model: str) -> str:
    """Generate a mock AI response (demo mode)."""
    last_msg = messages[-1]["content"] if messages else "Hello"

    responses = [
        f"您好！这是来自 MaaS-Router 的演示回复。\n\n您正在使用模型 **{model}**。在实际部署中，这里会返回真实的 AI 推理结果。\n\n您的问题：「{last_msg[:100]}」\n\nMaaS-Router 通过智能路由为您自动选择了最优模型，帮助降低 40-60% 的推理成本。",
    ]

    import random
    return random.choice(responses)


# ============================================
# API Key Management
# ============================================

@router.get("/keys")
async def list_api_keys(
    user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    """List user's API keys."""
    result = await db.execute(
        select(ApiKey).where(ApiKey.user_id == user.id).order_by(ApiKey.created_at.desc())
    )
    keys = result.scalars().all()

    return {
        "object": "list",
        "data": [{
            "id": k.id,
            "name": k.name,
            "prefix": k.key_prefix,
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
        "id": api_key.id,
        "name": api_key.name,
        "key": full_key,  # Only shown once!
        "prefix": prefix,
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
    result = await db.execute(
        select(ApiKey).where(ApiKey.id == key_id, ApiKey.user_id == user.id)
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
    result = await db.execute(select(Balance).where(Balance.user_id == user.id))
    balance = result.scalar_one_or_none()

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
    result = await db.execute(
        select(
            func.count(Transaction.id).label("total_requests"),
            func.sum(Transaction.total_tokens).label("total_tokens"),
            func.sum(Transaction.amount).label("total_cost"),
        ).where(
            Transaction.user_id == user.id,
            Transaction.type == "usage",
        )
    )
    row = result.one()

    return {
        "total_requests": row.total_requests or 0,
        "total_tokens": int(row.total_tokens or 0),
        "total_cost_cred": round(abs(float(row.total_cost or 0)), 6),
    }


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
    result = await db.execute(
        select(RequestLog)
        .where(RequestLog.user_id == user.id)
        .order_by(RequestLog.created_at.desc())
        .limit(limit)
    )
    logs = result.scalars().all()

    return {
        "object": "list",
        "data": [{
            "request_id": log.request_id,
            "model_id": log.model_id,
            "provider_id": log.provider_id,
            "complexity_score": log.complexity_score,
            "route_decision": log.route_decision,
            "prompt_tokens": log.prompt_tokens,
            "completion_tokens": log.completion_tokens,
            "latency_ms": log.latency_ms,
            "created_at": log.created_at.isoformat() if log.created_at else None,
        } for log in logs]
    }
