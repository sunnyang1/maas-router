"""
OpenAI provider adapter.

Handles chat completions via the OpenAI API.
"""
import json
import httpx
from app.providers.base import BaseProvider, NormalizedRequest, ProviderResponse


class OpenAIProvider(BaseProvider):
    provider_id = "openai"

    def __init__(
        self,
        api_key: str = "",
        base_url: str = "https://api.openai.com/v1",
    ):
        self.api_key = api_key
        self.base_url = base_url
        self._client: httpx.AsyncClient | None = None

    async def _get_client(self) -> httpx.AsyncClient:
        if self._client is None:
            self._client = httpx.AsyncClient(
                base_url=self.base_url,
                timeout=60.0,
                headers={
                    "Authorization": f"Bearer {self.api_key}",
                    "Content-Type": "application/json",
                },
            )
        return self._client

    async def chat_completion(self, request: NormalizedRequest) -> ProviderResponse:
        """
        Send a real chat completion request to the OpenAI API.

        Falls back to a mock response if no API key is configured.
        """
        if not self.api_key:
            return self._mock_response(request)

        client = await self._get_client()
        try:
            payload = {
                "model": request.model.replace("gpt-", ""),  # Strip prefix if needed
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
        """Generate a placeholder response when no API key is set."""
        last_msg = request.messages[-1].content if request.messages else ""
        prompt_tokens = len(last_msg) // 4
        completion_tokens = 128

        return ProviderResponse(
            content=(
                f"[OpenAI Demo] MaaS-Router routed your request to {request.model}.\n\n"
                f"Your prompt ({prompt_tokens} tokens): \"{last_msg[:100]}...\"\n\n"
                f"Configure OPENAI_API_KEY in .env for real responses."
            ),
            model=request.model,
            usage={
                "prompt_tokens": prompt_tokens,
                "completion_tokens": completion_tokens,
                "total_tokens": prompt_tokens + completion_tokens,
            },
            finish_reason="stop",
        )
