"""
Idempotency support using Redis.

Prevents duplicate processing of the same request by caching
responses keyed by X-Idempotency-Key header with a configurable TTL.
"""
import json
import hashlib
from typing import Optional

from app.core.redis import redis_client


IDEMPOTENCY_TTL_SECONDS = 86400  # 24 hours
IDEMPOTENCY_PREFIX = "idem"


def _make_key(key: str) -> str:
    """Build a Redis key for the idempotency store."""
    hashed = hashlib.sha256(key.encode()).hexdigest()[:32]
    return f"{IDEMPOTENCY_PREFIX}:{hashed}"


async def check_idempotency(key: str) -> Optional[dict]:
    """
    Check if a request with this idempotency key was already processed.

    Returns the cached response dict if found, None otherwise.
    """
    redis_key = _make_key(key)
    cached = await redis_client.get(redis_key)
    if cached:
        return json.loads(cached)
    return None


async def mark_idempotency(key: str, response: dict) -> None:
    """
    Cache a response for the given idempotency key.

    The cached response will be returned for subsequent requests
    with the same key within the TTL window.
    """
    redis_key = _make_key(key)
    await redis_client.setex(
        redis_key,
        IDEMPOTENCY_TTL_SECONDS,
        json.dumps(response, default=str),
    )


async def acquire_idempotency_lock(key: str) -> bool:
    """
    Acquire a distributed lock for idempotent operations.

    Returns True if the lock was acquired, False if already held.
    Uses Redis SETNX for atomicity.
    """
    lock_key = f"{IDEMPOTENCY_PREFIX}:lock:{_make_key(key)}"
    acquired = await redis_client.setnx(lock_key, "1")
    if acquired:
        await redis_client.expire(lock_key, 30)  # 30 second lock timeout
    return bool(acquired)


async def release_idempotency_lock(key: str) -> None:
    """Release a distributed lock."""
    lock_key = f"{IDEMPOTENCY_PREFIX}:lock:{_make_key(key)}"
    await redis_client.delete(lock_key)
