"""
Redis caching utilities.

Provides a simple async cache decorator for repository methods.
Uses cache-aside pattern with configurable TTL per data type.
"""
import json
from functools import wraps
from typing import Any, Callable, Optional

from app.core.redis import redis_client


# Default TTLs in seconds
DEFAULT_TTL = 60
MODEL_LIST_TTL = 60
ROUTING_RULE_TTL = 300
DASHBOARD_TTL = 60


def cached(
    ttl: int = DEFAULT_TTL,
    prefix: str = "cache",
    skip_args: Optional[list[int]] = None,
):
    """
    Async Redis cache decorator.

    Args:
        ttl: Cache time-to-live in seconds
        prefix: Key prefix for namespacing
        skip_args: Indices of args to exclude from cache key (e.g., db session)
    """
    def decorator(func):
        @wraps(func)
        async def wrapper(*args, **kwargs):
            # Build cache key from function name and args
            key_parts = [prefix, func.__name__]
            skip_set = set(skip_args or [])

            for i, arg in enumerate(args):
                if i not in skip_set:
                    key_parts.append(str(arg))
            for k, v in sorted(kwargs.items()):
                key_parts.append(f"{k}={v}")

            cache_key = ":".join(key_parts)[:200]  # Limit key length

            # Try cache
            try:
                cached_data = await redis_client.get(cache_key)
                if cached_data:
                    return json.loads(cached_data)
            except Exception:
                pass  # Redis unavailable, fall through to DB

            # Cache miss - call the function
            result = await func(*args, **kwargs)

            # Store in cache
            try:
                await redis_client.setex(
                    cache_key,
                    ttl,
                    json.dumps(result, default=str),
                )
            except Exception:
                pass  # Redis unavailable, skip caching

            return result
        return wrapper
    return decorator
