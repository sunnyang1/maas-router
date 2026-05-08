"""
Provider adapter layer for MaaS-Router.

Provides a uniform interface for all AI model providers (OpenAI, DeepSeek,
Anthropic, self-hosted) through the BaseProvider abstract class.

The ProviderRegistry singleton manages provider lifecycle, and the factory
module creates adapters from database configuration.
"""
from app.providers.base import BaseProvider, NormalizedRequest, ProviderResponse
from app.providers.openai import OpenAIProvider
from app.providers.deepseek import DeepSeekProvider
from app.providers.anthropic import AnthropicProvider
from app.providers.self_hosted import SelfHostedProvider
from app.providers.registry import (
    ProviderRegistry,
    get_provider_registry,
    reset_provider_registry,
)
from app.providers.factory import create_provider_from_db, load_providers_from_db

__all__ = [
    "BaseProvider",
    "NormalizedRequest",
    "ProviderResponse",
    "OpenAIProvider",
    "DeepSeekProvider",
    "AnthropicProvider",
    "SelfHostedProvider",
    "ProviderRegistry",
    "get_provider_registry",
    "reset_provider_registry",
    "create_provider_from_db",
    "load_providers_from_db",
]
