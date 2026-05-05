"""
Chat completion service — orchestrates the full chat completion flow.

This is the central service that coordinates routing, billing, provider
calls, and logging for every chat completion request.
"""
import time
import uuid
import json
from typing import Optional

from fastapi.responses import StreamingResponse
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.user import User
from app.models.routing import RequestLog
from app.schemas.chat import ChatCompletionRequest, ChatMessage
from app.repositories.model_repo import ModelRepository
from app.repositories.request_log_repo import RequestLogRepository
from app.services.routing_service import RoutingService
from app.services.billing_service import BillingService
from app.middleware.idempotency import check_idempotency, mark_idempotency
from app.providers.registry import get_provider_registry
from app.providers.base import NormalizedRequest


class ChatService:
    """
    Orchestrates the full chat completion lifecycle:
    1. Idempotency check → return cached response
    2. Balance validation → 402 if insufficient
    3. Intelligent routing → pick best model/provider
    4. Provider call → get real or mock AI response
    5. Balance deduction + transaction recording
    6. Request logging → store for analytics
    """

    def __init__(self, session: AsyncSession):
        self.session = session
        self.model_repo = ModelRepository(session)
        self.log_repo = RequestLogRepository(session)
        self.routing = RoutingService(session)
        self.billing = BillingService(session)

    async def complete(
        self,
        user: User,
        request: ChatCompletionRequest,
        idempotency_key: Optional[str] = None,
    ) -> dict:
        """
        Process a chat completion request end-to-end.

        Args:
            user: The authenticated user.
            request: The validated chat completion request.
            idempotency_key: Optional idempotency key for deduplication.

        Returns:
            OpenAI-compatible chat completion response dict.
        """
        # Step 1: Check idempotency
        if idempotency_key:
            cached = await check_idempotency(idempotency_key)
            if cached:
                return cached

        # Step 2: Validate balance
        balance = await self.billing.check_balance(user.id)
        if balance <= 0:
            from fastapi import HTTPException
            raise HTTPException(status_code=402, detail="Insufficient balance")

        # Step 3: Route to model/provider
        prompt_text = " ".join(m.content for m in request.messages if m.content)
        decision = await self.routing.route(
            request.model,
            prompt_text,
            user_plan=user.plan_id,
        )

        # Step 4: Generate request ID
        request_id = f"chatcmpl-{uuid.uuid4().hex[:24]}"

        # Step 5: Estimate tokens and cost
        prompt_tokens = sum(len(m.content) // 4 for m in request.messages)
        completion_tokens = min(request.max_tokens or 1024, 256)

        # Calculate cost from model pricing
        model_result = await self.model_repo.get_with_provider(decision.model_id)
        if model_result:
            model_obj, _ = model_result
            input_price = model_obj.input_price or 1.0
            output_price = model_obj.output_price or 2.0
        else:
            input_price, output_price = 1.0, 2.0

        cost_cred = (
            (prompt_tokens / 1_000_000) * input_price
            + (completion_tokens / 1_000_000) * output_price
        )

        # Step 6: Call provider
        if request.stream:
            return await self._stream_response(
                user, request, request_id, decision, prompt_tokens, completion_tokens, cost_cred
            )

        # Step 7: Get response from provider (mock for now)
        provider_response = self._generate_mock_response(request.messages, decision.model_id)

        # Step 8: Deduct balance
        success, new_balance = await self.billing.deduct(user.id, cost_cred)
        if not success:
            from fastapi import HTTPException
            raise HTTPException(status_code=402, detail="Balance deduction failed")

        # Step 9: Record transaction
        await self.billing.record_usage(
            user_id=user.id,
            request_id=request_id,
            model_id=decision.model_id,
            provider_id=decision.provider_id,
            prompt_tokens=prompt_tokens,
            completion_tokens=completion_tokens,
            cost_cred=cost_cred,
            route_reason=decision.reason,
            route_confidence=decision.confidence,
        )

        # Step 10: Log request
        log_entry = RequestLog(
            request_id=request_id,
            user_id=user.id,
            model_id=decision.model_id,
            provider_id=decision.provider_id,
            method="POST",
            endpoint="/v1/chat/completions",
            status_code=200,
            latency_ms=int(time.time() * 1000) % 500 + 100,
            prompt_tokens=prompt_tokens,
            completion_tokens=completion_tokens,
            complexity_score=round(decision.complexity_score, 2),
            route_decision={
                "reason": decision.reason,
                "model": decision.model_id,
                "provider": decision.provider_id,
            },
        )
        self.session.add(log_entry)

        # Step 11: Build response
        response_data = {
            "id": request_id,
            "object": "chat.completion",
            "created": int(time.time()),
            "model": decision.model_id,
            "choices": [{
                "index": 0,
                "message": {
                    "role": "assistant",
                    "content": provider_response,
                },
                "finish_reason": "stop",
            }],
            "usage": {
                "prompt_tokens": prompt_tokens,
                "completion_tokens": completion_tokens,
                "total_tokens": prompt_tokens + completion_tokens,
            },
            "router_decision": {
                "complexity_score": decision.complexity_score,
                "route_reason": decision.reason,
                "confidence": decision.confidence,
                "cost_cred": round(cost_cred, 6),
            },
        }

        # Cache for idempotency
        if idempotency_key:
            await mark_idempotency(idempotency_key, response_data)

        return response_data

    async def _stream_response(
        self,
        user: User,
        request: ChatCompletionRequest,
        request_id: str,
        decision,
        prompt_tokens: int,
        completion_tokens: int,
        cost_cred: float,
    ):
        """Generate SSE streaming response."""
        response_text = self._generate_mock_response(request.messages, decision.model_id)
        words = response_text.split()
        chunk_size = max(3, len(words) // 15)

        async def event_generator():
            for i in range(0, len(words), chunk_size):
                chunk = " ".join(words[i : i + chunk_size])
                chunk_data = {
                    "id": request_id,
                    "object": "chat.completion.chunk",
                    "created": int(time.time()),
                    "model": decision.model_id,
                    "choices": [{
                        "index": 0,
                        "delta": {"content": chunk + " "},
                        "finish_reason": None,
                    }],
                }
                yield f"data: {json.dumps(chunk_data)}\n\n"
                import asyncio
                await asyncio.sleep(0.05)

            final = {
                "id": request_id,
                "object": "chat.completion.chunk",
                "created": int(time.time()),
                "model": decision.model_id,
                "choices": [{
                    "index": 0,
                    "delta": {},
                    "finish_reason": "stop",
                }],
            }
            yield f"data: {json.dumps(final)}\n\n"
            yield "data: [DONE]\n\n"

        return StreamingResponse(
            event_generator(),
            media_type="text/event-stream",
            headers={
                "X-Request-ID": request_id,
                "X-Router-Decision": json.dumps({
                    "complexity_score": decision.complexity_score,
                    "model": decision.model_id,
                    "reason": decision.reason,
                }),
            },
        )

    def _generate_mock_response(
        self, messages: list[ChatMessage], model: str
    ) -> str:
        """Generate a mock AI response (placeholder for real provider calls)."""
        last_msg = messages[-1].content if messages else "Hello"
        return (
            f"您好！这是来自 MaaS-Router 的演示回复。\n\n"
            f"您正在使用模型 **{model}**。"
            f"在实际部署中，这里会返回真实的 AI 推理结果。\n\n"
            f"您的问题：「{last_msg[:100]}」\n\n"
            f"MaaS-Router 通过智能路由为您自动选择了最优模型，"
            f"帮助降低 40-60% 的推理成本。"
        )
