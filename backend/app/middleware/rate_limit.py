"""
Redis-based rate limiting middleware.

Enforces per-user and per-API-key rate limits using
a sliding window algorithm backed by Redis.
"""
import time
from fastapi import Request, HTTPException
from starlette.middleware.base import BaseHTTPMiddleware

from app.core.redis import redis_client
from app.core.config import get_settings

settings = get_settings()

# Rate limit window in seconds
RATE_WINDOW = 60  # 1 minute


class RateLimitMiddleware(BaseHTTPMiddleware):
    """
    Enforces rate limits per user/API key.

    Uses a Redis sorted set sliding window for each user.
    Falls back gracefully if Redis is unavailable.
    """

    async def dispatch(self, request: Request, call_next):
        # Skip rate limiting for non-API routes
        if not request.url.path.startswith("/v1/"):
            return await call_next(request)

        # Get user from request state (set by auth middleware)
        user = getattr(request.state, "user", None)
        if user is None:
            return await call_next(request)

        user_id = user.id if hasattr(user, "id") else str(user)
        rpm_limit, tpm_limit = self._get_limits(user)

        # Check RPM
        allowed = await self._check_rate_limit(
            f"rate:rpm:{user_id}", rpm_limit
        )
        if not allowed:
            raise HTTPException(
                status_code=429,
                detail=f"Rate limit exceeded: {rpm_limit} requests per minute",
            )

        response = await call_next(request)

        # Add rate limit headers
        remaining = await self._get_remaining(f"rate:rpm:{user_id}", rpm_limit)
        response.headers["X-RateLimit-Limit"] = str(rpm_limit)
        response.headers["X-RateLimit-Remaining"] = str(remaining)

        return response

    def _get_limits(self, user) -> tuple[int, int]:
        """Determine rate limits based on user plan."""
        plan = getattr(user, "plan_id", "free") or "free"
        if plan == "pro":
            return settings.rate_limit_rpm_pro, settings.rate_limit_tpm_pro
        elif plan == "enterprise":
            return 10_000, 1_000_000  # effectively unlimited
        return settings.rate_limit_rpm_free, settings.rate_limit_tpm_free

    async def _check_rate_limit(self, key: str, limit: int) -> bool:
        """
        Sliding window rate limit check using Redis sorted sets.

        Returns True if allowed, False if rate limit exceeded.
        """
        now = time.time()
        window_start = now - RATE_WINDOW

        try:
            # Remove expired entries
            await redis_client.zremrangebyscore(key, 0, window_start)

            # Count current window
            count = await redis_client.zcard(key)

            if count >= limit:
                return False

            # Add current request with score = timestamp
            await redis_client.zadd(key, {str(now): now})

            # Set expiry on the key
            await redis_client.expire(key, RATE_WINDOW * 2)

            return True
        except Exception:
            # Redis unavailable - allow request (fail open)
            return True

    async def _get_remaining(self, key: str, limit: int) -> int:
        """Get remaining requests in current window."""
        try:
            count = await redis_client.zcard(key)
            return max(0, limit - (count or 0))
        except Exception:
            return limit
