"""
MaaS-Router Core Configuration
"""
from pydantic_settings import BaseSettings
from functools import lru_cache


class Settings(BaseSettings):
    # Service
    service_name: str = "api-server"
    service_port: int = 8001
    debug: bool = True

    # Database
    database_url: str = "postgresql+asyncpg://maas:maas_dev_2026@localhost:5432/maas_router"

    # Redis
    redis_url: str = "redis://localhost:6379/0"

    # JWT
    jwt_secret_key: str = "maas-router-dev-secret-key-change-in-production"
    jwt_algorithm: str = "HS256"
    jwt_expire_minutes: int = 60

    # API Key
    api_key_prefix: str = "sk-mr-"
    api_key_length: int = 48

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


@lru_cache()
def get_settings() -> Settings:
    return Settings()
