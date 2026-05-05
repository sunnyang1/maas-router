"""
Self-hosted provider adapter.

Compatible with vLLM, Ollama, and other OpenAI-compatible self-hosted
inference servers. Endpoint is configurable via environment.
"""
import httpx
from app.core.config import get_settings
from app.providers.base import BaseProvider, NormalizedRequest, ProviderResponse


class SelfHostedProvider(BaseProvider):
    provider_id = "self-hosted"

    def __init__(self, base_url: str | None = None):
        settings = get_settings()
        self.base_url = (base_url or
            getattr(settings, "self_hosted_url", "http://localhost:8000/v1"))
        self._client: httpx.AsyncClient | None = None

    async def _get_client(self) -> httpx.AsyncClient:
        if self._client is None:
            self._client = httpx.AsyncClient(
                base_url=self.base_url,
                timeout=120.0,
                headers={"Content-Type": "application/json"},
            )
        return self._client

    async def chat_completion(self, request: NormalizedRequest) -> ProviderResponse:
        """
        Send a chat completion request to the self-hosted inference server.
        Falls back to a mock response if the server is unreachable.
        """
        client = await self._get_client()
        try:
            payload = {
                "model": request.model,
                "messages": [
                    {"role": m.role, "content": m.content}
                    for m in request.messages
                ],
                "temperature": request.temperature,
                "max_tokens": request.max_tokens,
                "top_p": request.top_p,
            }
            if request.stop:
                payload["stop"] = request.stop

            response = await client.post("/chat/completions", json=payload)
            response.raise_for_status()
            data = response.json()

            choice = data["choices"][0]
            usage = data.get("usage", {})
            return ProviderResponse(
                content=choice["message"]["content"],
                model=data["model"],
                usage={
                    "prompt_tokens": usage.get("prompt_tokens", 0),
                    "completion_tokens": usage.get("completion_tokens", 0),
                    "total_tokens": usage.get("total_tokens", 0),
                },
                finish_reason=choice.get("finish_reason", "stop"),
            )
        except Exception:
            return self._mock_response(request)

    async def health_check(self) -> bool:
        try:
            client = await self._get_client()
            r = await client.get("/models")
            return r.status_code == 200
        except Exception:
            return False

    def _mock_response(self, request: NormalizedRequest) -> ProviderResponse:
        last_msg = request.messages[-1].content if request.messages else ""
        prompt_tokens = len(last_msg) // 4
        completion_tokens = 128

        return ProviderResponse(
            content=(
                f"[Self-Hosted Demo] MaaS-Router routed to {request.model}.\n\n"
                f"Prompt: \"{last_msg[:100]}...\"\n\n"
                f"The self-hosted server at {self.base_url} is not reachable.\n"
                f"Start it and try again."
            ),
            model=request.model,
            usage={
                "prompt_tokens": prompt_tokens,
                "completion_tokens": completion_tokens,
                "total_tokens": prompt_tokens + completion_tokens,
            },
            finish_reason="stop",
        )
