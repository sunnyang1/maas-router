"""
Admin Server - MaaS-Router 管理后台 API
"""
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.core.config import get_settings
from app.core.database import init_db
from app.admin_server.dashboard import router as dashboard_router
from app.admin_server.users import router as users_router
from app.admin_server.models_admin import router as models_router
from app.admin_server.billing_admin import router as billing_router
from app.admin_server.monitoring import router as monitoring_router
from app.admin_server.settings import router as settings_router
from app.admin_server.auth_admin import router as auth_router

settings = get_settings()
settings.validate_security()


@asynccontextmanager
async def lifespan(app: FastAPI):
    await init_db()
    yield


app = FastAPI(
    title="MaaS-Router Admin API",
    description="MaaS-Router 内部管理后台 API",
    version="1.0.0",
    lifespan=lifespan,
)

# CORS - explicit origins, no wildcard
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.get_cors_origins(),
    allow_credentials=True,
    allow_methods=["GET", "POST", "PUT", "DELETE"],
    allow_headers=["Authorization", "Content-Type"],
)

# Mount admin routes under /api/admin/v1
app.include_router(auth_router, prefix="/api/admin/v1")
app.include_router(dashboard_router, prefix="/api/admin/v1")
app.include_router(users_router, prefix="/api/admin/v1")
app.include_router(models_router, prefix="/api/admin/v1")
app.include_router(billing_router, prefix="/api/admin/v1")
app.include_router(monitoring_router, prefix="/api/admin/v1")
app.include_router(settings_router, prefix="/api/admin/v1")


@app.get("/")
async def root():
    return {
        "name": "MaaS-Router Admin API",
        "version": "1.0.0",
        "docs": "/docs",
        "health": "/health",
        "api_prefix": "/api/admin/v1",
    }


@app.get("/health")
async def health():
    return {"status": "ok", "service": "admin-server"}
