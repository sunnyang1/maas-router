"""
Anthropic (Claude) provider adapter.

Anthropic uses a different API format than OpenAI — this adapter
translates between the two.
"""
import httpx
from app.providers.base import BaseProvider, NormalizedRequest, ProviderResponse


class AnthropicProvider(BaseProvider):
    provider_id = "anthropic"

    def __init__(
        self,
        api_key: str = "",
        base_url: str = "https://api.anthropic.com/v1",
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
                    "x-api-key": self.api_key,
                    "anthropic-version": "2023-06-01",
                    "Content-Type": "application/json",
                },
            )
        return self._client

    async def chat_completion(self, request: NormalizedRequest) -> ProviderResponse:
        if not self.api_key:
            return self._mock_response(request)

        client = await self._get_client()
        try:
            # Convert OpenAI-style messages to Anthropic format
            system_msg = ""
            messages = []
            for m in request.messages:
                if m.role == "system":
                    system_msg = m.content
                else:
                    messages.append({"role": m.role, "content": m.content})

            payload = {
                "model": request.model,
                "messages": messages,
                "max_tokens": request.max_tokens,
                "temperature": request.temperature,
                "top_p": request.top_p,
            }
            if system_msg:
                payload["system"] = system_msg
            if request.stop:
                payload["stop_sequences"] = request.stop

            response = await client.post("/messages", json=payload)
            response.raise_for_status()
            data = response.json()

            content = data["content"][0]["text"] if data.get("content") else ""
            usage = data.get("usage", {})
            return ProviderResponse(
                content=content,
                model=data["model"],
                usage={
                    "prompt_tokens": usage.get("input_tokens", 0),
                    "completion_tokens": usage.get("output_tokens", 0),
                    "total_tokens": usage.get("input_tokens", 0) + usage.get("output_tokens", 0),
                },
                finish_reason=data.get("stop_reason", "stop"),
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
                f"[Anthropic Demo] MaaS-Router routed to {request.model}.\n\n"
                f"Prompt: \"{last_msg[:100]}...\"\n\n"
                f"Configure ANTHROPIC_API_KEY for real responses."
            ),
            model=request.model,
            usage={
                "prompt_tokens": prompt_tokens,
                "completion_tokens": completion_tokens,
                "total_tokens": prompt_tokens + completion_tokens,
            },
            finish_reason="end_turn",
        )
