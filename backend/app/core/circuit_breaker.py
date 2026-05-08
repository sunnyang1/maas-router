"""
Circuit breaker pattern for provider resilience.

Protects the system from cascading failures when a provider
becomes unresponsive by automatically opening the circuit
after a threshold of consecutive failures.
"""
import time
from dataclasses import dataclass, field
from enum import Enum


class CircuitState(str, Enum):
    CLOSED = "closed"       # Normal operation — requests pass through
    OPEN = "open"            # Failing — requests are rejected immediately
    HALF_OPEN = "half_open" # Testing — limited requests to check recovery


class CircuitBreakerOpenError(Exception):
    """Raised when a request is rejected because the circuit is open."""

    def __init__(self, provider_id: str):
        self.provider_id = provider_id
        super().__init__(
            f"Circuit breaker is OPEN for provider '{provider_id}'"
        )


@dataclass
class CircuitBreaker:
    """
    Circuit breaker for a single provider.

    States:
    - CLOSED → OPEN: After `failure_threshold` consecutive failures
    - OPEN → HALF_OPEN: After `recovery_timeout` seconds
    - HALF_OPEN → CLOSED: After `half_open_max` consecutive successes
    - HALF_OPEN → OPEN: On any failure
    """

    provider_id: str
    failure_threshold: int = 5
    recovery_timeout: float = 30.0
    half_open_max: int = 3

    failure_count: int = 0
    last_failure_time: float = 0.0
    state: CircuitState = CircuitState.CLOSED
    half_open_successes: int = 0

    async def call(self, func, *args, **kwargs):
        """
        Execute a function with circuit breaker protection.

        Args:
            func: Async callable to execute.
            *args, **kwargs: Arguments passed to func.

        Returns:
            The return value of func.

        Raises:
            CircuitBreakerOpenError: If the circuit is open.
            Exception: Any exception raised by func.
        """
        if self.state == CircuitState.OPEN:
            if time.time() - self.last_failure_time > self.recovery_timeout:
                self.state = CircuitState.HALF_OPEN
                self.half_open_successes = 0
            else:
                raise CircuitBreakerOpenError(self.provider_id)

        try:
            result = await func(*args, **kwargs)
            self._on_success()
            return result
        except Exception:
            self._on_failure()
            raise

    def _on_success(self):
        """Record a successful call."""
        if self.state == CircuitState.HALF_OPEN:
            self.half_open_successes += 1
            if self.half_open_successes >= self.half_open_max:
                self.state = CircuitState.CLOSED
                self.failure_count = 0
        else:
            self.failure_count = 0

    def _on_failure(self):
        """Record a failed call."""
        self.failure_count += 1
        self.last_failure_time = time.time()
        if self.failure_count >= self.failure_threshold:
            self.state = CircuitState.OPEN

    def reset(self):
        """Force reset the circuit to CLOSED state."""
        self.state = CircuitState.CLOSED
        self.failure_count = 0
        self.half_open_successes = 0

    def to_dict(self) -> dict:
        """Serialize circuit breaker state for monitoring."""
        return {
            "provider_id": self.provider_id,
            "state": self.state.value,
            "failure_count": self.failure_count,
            "last_failure_time": self.last_failure_time,
        }


class CircuitBreakerRegistry:
    """Manages circuit breakers for multiple providers."""

    def __init__(self):
        self._breakers: dict[str, CircuitBreaker] = {}

    def get_or_create(
        self,
        provider_id: str,
        failure_threshold: int = 5,
        recovery_timeout: float = 30.0,
    ) -> CircuitBreaker:
        """Get an existing circuit breaker or create a new one."""
        if provider_id not in self._breakers:
            self._breakers[provider_id] = CircuitBreaker(
                provider_id=provider_id,
                failure_threshold=failure_threshold,
                recovery_timeout=recovery_timeout,
            )
        return self._breakers[provider_id]

    def list_all(self) -> list[dict]:
        """List all circuit breaker states."""
        return [b.to_dict() for b in self._breakers.values()]

    def reset_all(self):
        """Reset all circuit breakers to CLOSED."""
        for breaker in self._breakers.values():
            breaker.reset()


# Global registry singleton
circuit_breaker_registry = CircuitBreakerRegistry()
