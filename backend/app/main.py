"""
API Server - MaaS-Router 用户端 API
OpenAI-compatible endpoints
"""
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.core.config import get_settings
from app.core.database import init_db
from app.core.logging_config import setup_logging
from app.api_server.router import router as api_router
from app.middleware.rate_limit import RateLimitMiddleware

settings = get_settings()
settings.validate_security()
setup_logging(settings.environment)


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    await init_db()
    # Initialize provider registry
    from app.providers.registry import get_provider_registry
    get_provider_registry()
    yield
    # Shutdown


app = FastAPI(
    title="MaaS-Router API",
    description="AI 推理聚合网关 - 用户端 API（OpenAI 兼容）",
    version="1.0.0",
    lifespan=lifespan,
)

# Rate limiting (applied before CORS)
app.add_middleware(RateLimitMiddleware)

# CORS - explicit origins, no wildcard
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.get_cors_origins(),
    allow_credentials=True,
    allow_methods=["GET", "POST", "PUT", "DELETE"],
    allow_headers=["Authorization", "Content-Type", "X-Idempotency-Key"],
)

# Include API routes
app.include_router(api_router)


@app.get("/health")
async def health():
    return {"status": "ok", "service": "api-server"}


@app.get("/health/ready")
async def readiness():
    """Readiness probe: checks DB + Redis connectivity."""
    import time as _time
    start = _time.time()
    results = {"database": False, "redis": False}

    try:
        from app.core.database import async_session_factory
        async with async_session_factory() as session:
            await session.execute(
                __import__("sqlalchemy").text("SELECT 1")
            )
        results["database"] = True
    except Exception:
        pass

    try:
        from app.core.redis import redis_client
        await redis_client.ping()
        results["redis"] = True
    except Exception:
        pass

    latency = round((_time.time() - start) * 1000, 1)
    all_ok = all(results.values())

    return {
        "status": "ok" if all_ok else "degraded",
        "checks": results,
        "latency_ms": latency,
    }


@app.get("/health/live")
async def liveness():
    """Liveness probe: minimal check."""
    return {"status": "alive"}
