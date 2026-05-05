"""
Provider factory — creates provider adapters from database configuration.

This allows providers to be managed dynamically via the admin panel
rather than requiring code changes to add a new provider.
"""
from typing import Optional
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select

from app.models.provider import Provider
from app.providers.base import BaseProvider
from app.providers.openai import OpenAIProvider
from app.providers.deepseek import DeepSeekProvider
from app.providers.anthropic import AnthropicProvider
from app.providers.self_hosted import SelfHostedProvider


PROVIDER_CLASS_MAP = {
    "openai": OpenAIProvider,
    "deepseek": DeepSeekProvider,
    "anthropic": AnthropicProvider,
    "self-hosted": SelfHostedProvider,
}


async def create_provider_from_db(
    provider: Provider,
) -> Optional[BaseProvider]:
    """
    Create a provider adapter from a Provider database record.

    Args:
        provider: The Provider ORM model instance.

    Returns:
        A concrete BaseProvider instance, or None if the provider type is unknown.
    """
    provider_cls = PROVIDER_CLASS_MAP.get(provider.id)
    if provider_cls is None:
        return None

    config = provider.config or {}

    # Each provider may need different constructor args
    if provider.id == "openai":
        return provider_cls(
            api_key=config.get("api_key", ""),
            base_url=provider.api_base_url or "https://api.openai.com/v1",
        )
    elif provider.id == "deepseek":
        return provider_cls(
            api_key=config.get("api_key", ""),
            base_url=provider.api_base_url or "https://api.deepseek.com/v1",
        )
    elif provider.id == "anthropic":
        return provider_cls(
            api_key=config.get("api_key", ""),
            base_url=provider.api_base_url or "https://api.anthropic.com/v1",
        )
    elif provider.id == "self-hosted":
        return provider_cls(
            base_url=provider.api_base_url or "http://localhost:8000/v1",
        )
    return None


async def load_providers_from_db(
    session: AsyncSession,
    registry,
) -> int:
    """
    Load all active providers from the database and register them.

    Args:
        session: Async database session.
        registry: ProviderRegistry instance to populate.

    Returns:
        Number of providers loaded.
    """
    result = await session.execute(
        select(Provider).where(Provider.status == "active")
    )
    providers = result.scalars().all()

    count = 0
    for p in providers:
        adapter = await create_provider_from_db(p)
        if adapter:
            registry.register(adapter)
            count += 1

    return count
