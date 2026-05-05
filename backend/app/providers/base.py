"""
Abstract base provider and shared data structures.

Each AI provider (OpenAI, DeepSeek, etc.) implements the BaseProvider
interface, enabling the router to treat all providers uniformly.
"""
from abc import ABC, abstractmethod
from dataclasses import dataclass, field


@dataclass
class NormalizedRequest:
    """Provider-agnostic chat completion request."""
    model: str
    messages: list  # list of ChatMessage
    temperature: float = 0.7
    max_tokens: int = 1024
    stream: bool = False
    top_p: float = 1.0
    stop: list[str] | None = None


@dataclass
class ProviderResponse:
    """Normalized response from any provider."""
    content: str
    model: str
    usage: dict = field(default_factory=lambda: {
        "prompt_tokens": 0,
        "completion_tokens": 0,
        "total_tokens": 0,
    })
    finish_reason: str = "stop"


class BaseProvider(ABC):
    """
    Abstract base class for AI provider adapters.

    Each concrete provider handles the HTTP protocol details
    for its specific API, while presenting a uniform interface
    to the rest of the system.
    """

    @abstractmethod
    async def chat_completion(self, request: NormalizedRequest) -> ProviderResponse:
        """
        Send a chat completion request to the provider.

        Args:
            request: Normalized request with model, messages, and parameters.

        Returns:
            ProviderResponse with content and usage statistics.
        """
        ...

    @abstractmethod
    async def health_check(self) -> bool:
        """
        Check if the provider is currently reachable.

        Returns:
            True if the provider responds successfully.
        """
        ...

    @property
    @abstractmethod
    def provider_id(self) -> str:
        """Unique identifier for this provider (e.g., 'openai', 'deepseek')."""
        ...
