"""
Pydantic schemas for chat completion requests and responses (OpenAI-compatible).
"""
from typing import Optional
from pydantic import BaseModel, Field


class ChatMessage(BaseModel):
    """A single message in a chat conversation."""
    role: str = Field(
        ...,
        description="Role of the message sender",
        pattern=r"^(system|user|assistant|function|tool)$",
    )
    content: str = Field(..., description="Message content")
    name: Optional[str] = Field(None, description="Optional name of the sender")


class ChatCompletionRequest(BaseModel):
    """OpenAI-compatible chat completion request body."""
    model: str = Field(
        default="gpt-4o-mini",
        description="Model ID to use. Use 'auto' for intelligent routing.",
    )
    messages: list[ChatMessage] = Field(
        ...,
        min_length=1,
        description="A list of messages comprising the conversation so far.",
    )
    stream: bool = Field(
        default=False,
        description="If true, the response will be streamed as SSE.",
    )
    temperature: Optional[float] = Field(
        default=0.7,
        ge=0.0,
        le=2.0,
        description="Sampling temperature.",
    )
    max_tokens: Optional[int] = Field(
        default=1024,
        ge=1,
        le=128000,
        description="Maximum number of tokens to generate.",
    )
    top_p: Optional[float] = Field(
        default=1.0,
        ge=0.0,
        le=1.0,
        description="Nucleus sampling parameter.",
    )
    n: Optional[int] = Field(
        default=1,
        ge=1,
        le=10,
        description="Number of chat completion choices to generate.",
    )
    stop: Optional[list[str]] = Field(
        default=None,
        description="Sequences where the API will stop generating.",
    )
    user: Optional[str] = Field(
        default=None,
        description="A unique identifier representing your end-user.",
    )


class ChatCompletionChoice(BaseModel):
    index: int
    message: ChatMessage
    finish_reason: Optional[str] = "stop"


class ChatCompletionUsage(BaseModel):
    prompt_tokens: int
    completion_tokens: int
    total_tokens: int


class RouterDecision(BaseModel):
    complexity_score: float
    route_reason: str
    confidence: float
    cost_cred: float


class ChatCompletionResponse(BaseModel):
    id: str
    object: str = "chat.completion"
    created: int
    model: str
    choices: list[ChatCompletionChoice]
    usage: Optional[ChatCompletionUsage] = None
    router_decision: Optional[RouterDecision] = None
