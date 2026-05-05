"""
Provider registry — dynamic provider management and discovery.

Maintains a global singleton registry of all available AI providers,
enabling the router to quickly look up and health-check providers.
"""
from typing import Optional
from app.providers.base import BaseProvider


class ProviderRegistry:
    """
    Central registry for AI provider adapters.

    Providers are registered once at startup and looked up by provider_id
    during request routing.
    """

    def __init__(self):
        self._providers: dict[str, BaseProvider] = {}

    def register(self, provider: BaseProvider) -> None:
        """Register a provider adapter."""
        self._providers[provider.provider_id] = provider

    def get(self, provider_id: str) -> Optional[BaseProvider]:
        """Get a provider by its unique ID."""
        return self._providers.get(provider_id)

    def get_healthy_providers(self) -> list[str]:
        """
        Return list of provider IDs whose health checks pass.

        Note: This runs health checks synchronously. For production,
        consider caching results and refreshing on a schedule.
        """
        healthy = []
        import asyncio
        loop = asyncio.get_event_loop()
        for pid, provider in self._providers.items():
            try:
                ok = loop.run_until_complete(provider.health_check())
            except Exception:
                ok = False
            if ok:
                healthy.append(pid)
        return healthy

    async def health_check_all(self) -> dict[str, bool]:
        """Run health checks on all registered providers and return results."""
        results = {}
        for pid, provider in self._providers.items():
            try:
                results[pid] = await provider.health_check()
            except Exception:
                results[pid] = False
        return results

    def list_all(self) -> list[str]:
        """Return all registered provider IDs."""
        return list(self._providers.keys())

    def unregister(self, provider_id: str) -> None:
        """Remove a provider from the registry."""
        self._providers.pop(provider_id, None)


# Global singleton — lazy-initialized on first use
_provider_registry: Optional[ProviderRegistry] = None


def get_provider_registry() -> ProviderRegistry:
    """
    Get or create the global provider registry singleton.

    Registers all built-in providers on first call.
    """
    global _provider_registry
    if _provider_registry is None:
        _provider_registry = ProviderRegistry()
        from app.providers.openai import OpenAIProvider
        from app.providers.deepseek import DeepSeekProvider
        from app.providers.anthropic import AnthropicProvider
        from app.providers.self_hosted import SelfHostedProvider

        _provider_registry.register(OpenAIProvider())
        _provider_registry.register(DeepSeekProvider())
        _provider_registry.register(AnthropicProvider())
        _provider_registry.register(SelfHostedProvider())
    return _provider_registry


def reset_provider_registry() -> None:
    """Reset the global provider registry (useful for testing)."""
    global _provider_registry
    _provider_registry = None
