"""
Background logging worker.

Records request logs asynchronously using a separate DB session.
"""
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import async_session_factory
from app.models.routing import RequestLog
from app.repositories.request_log_repo import RequestLogRepository


async def record_request_log_async(
    request_id: str,
    user_id: str,
    model_id: str | None = None,
    provider_id: str | None = None,
    method: str = "POST",
    endpoint: str = "/v1/chat/completions",
    status_code: int = 200,
    latency_ms: int = 0,
    prompt_tokens: int = 0,
    completion_tokens: int = 0,
    complexity_score: float | None = None,
    route_decision: dict | None = None,
    error_code: str | None = None,
    error_message: str | None = None,
) -> RequestLog | None:
    """Record a request log entry asynchronously."""
    async with async_session_factory() as session:
        try:
            repo = RequestLogRepository(session)
            log_entry = RequestLog(
                request_id=request_id,
                user_id=user_id,
                model_id=model_id,
                provider_id=provider_id,
                method=method,
                endpoint=endpoint,
                status_code=status_code,
                latency_ms=latency_ms,
                prompt_tokens=prompt_tokens,
                completion_tokens=completion_tokens,
                complexity_score=complexity_score,
                route_decision=route_decision,
                error_code=error_code,
                error_message=error_message,
            )
            await repo.create(log_entry)
            await session.commit()
            return log_entry
        except Exception:
            await session.rollback()
            return None
