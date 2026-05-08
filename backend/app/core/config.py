"""
MaaS-Router Core Configuration
"""
import sys
from pydantic_settings import BaseSettings
from functools import lru_cache


class Settings(BaseSettings):
    # Environment
    environment: str = "development"  # development | staging | production

    # Service
    service_name: str = "api-server"
    service_port: int = 8001
    debug: bool = True

    # Database
    database_url: str = "postgresql+asyncpg://maas:maas_dev_2026@localhost:5432/maas_router"
    database_url_read: str | None = None  # optional read replica
    db_pool_size: int = 20
    db_max_overflow: int = 10
    db_pool_recycle: int = 3600
    db_pool_timeout: int = 10

    # Redis
    redis_url: str = "redis://localhost:6379/0"

    # JWT
    jwt_secret_key: str = "maas-router-dev-secret-key-change-in-production"
    jwt_algorithm: str = "HS256"
    jwt_expire_minutes: int = 60

    # API Key
    api_key_prefix: str = "sk-mr-"
    api_key_length: int = 48

    # CORS
    cors_origins: str = "http://localhost:5173,http://localhost:3000"

    # Rate Limits
    rate_limit_rpm_free: int = 100
    rate_limit_tpm_free: int = 10000
    rate_limit_rpm_pro: int = 1000
    rate_limit_tpm_pro: int = 100000

    # Routing
    default_complexity_threshold: int = 5
    router_cache_ttl: int = 3600

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"

    def get_cors_origins(self) -> list[str]:
        """Parse comma-separated CORS origins string into a list."""
        return [origin.strip() for origin in self.cors_origins.split(",") if origin.strip()]

    def validate_security(self):
        """Validate security-critical settings. Call at startup."""
        default_secret = "maas-router-dev-secret-key-change-in-production"
        if self.environment in ("staging", "production"):
            if self.jwt_secret_key == default_secret:
                print(
                    "CRITICAL: JWT_SECRET_KEY is using default value. "
                    "Generate a secure key with: openssl rand -hex 32",
                    file=sys.stderr,
                )
                sys.exit(1)
            if self.cors_origins == "http://localhost:5173,http://localhost:3000":
                print(
                    "WARNING: CORS_ORIGINS is using default localhost values. "
                    "Update for production.",
                    file=sys.stderr,
                )


@lru_cache()
def get_settings() -> Settings:
    return Settings()
