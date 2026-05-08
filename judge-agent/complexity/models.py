"""
Complexity Analysis Data Models

Pydantic models for complexity analysis requests, responses, and configuration.
"""

from typing import Literal, Optional, List
from pydantic import BaseModel, Field


class Message(BaseModel):
    """Generic message structure for complexity analysis."""
    role: Literal["user", "assistant", "system"] = Field(..., description="Message role")
    content: str = Field(..., min_length=1, max_length=50000, description="Message content")


class AnalyzeRequest(BaseModel):
    """Request model for complexity analysis."""
    model: str = Field(..., min_length=1, max_length=200, pattern=r'^[a-zA-Z0-9._-]+$', description="Model name")
    messages: List[Message] = Field(..., min_length=1, max_length=100, description="Conversation messages")
    system: Optional[str] = Field(None, max_length=10000, description="System prompt")
    max_tokens: int = Field(0, ge=0, le=128000, description="Maximum tokens")
    stream: bool = Field(False, description="Streaming mode")

    class Config:
        json_schema_extra = {
            "example": {
                "model": "claude-sonnet-4-20250514",
                "messages": [
                    {"role": "user", "content": "Please explain quantum computing"}
                ],
                "system": "You are a helpful assistant.",
                "max_tokens": 4096,
                "stream": False
            }
        }


class ComplexityProfile(BaseModel):
    """Result of complexity analysis with routing recommendations."""
    score: float = Field(..., ge=0.0, le=1.0, description="Overall complexity score (0-1)")
    level: str = Field(..., description="Complexity level: simple, normal, complex, advanced")
    confidence: float = Field(..., ge=0.0, le=1.0, description="Confidence of the analysis")
    lexical_score: float = Field(0.0, ge=0.0, le=1.0, description="Lexical complexity score")
    structural_score: float = Field(0.0, ge=0.0, le=1.0, description="Structural complexity score")
    domain_score: float = Field(0.0, ge=0.0, le=1.0, description="Domain-specific complexity score")
    conversational_score: float = Field(0.0, ge=0.0, le=1.0, description="Conversational complexity score")
    task_type_score: float = Field(0.0, ge=0.0, le=1.0, description="Task type complexity score")
    recommended_tier: str = Field("", description="Recommended routing tier: economy, standard, premium")
    recommended_model: str = Field("", description="Recommended model for this complexity level")
    fallback_model: str = Field("", description="Fallback model if recommended is unavailable")
    estimated_cost: float = Field(0.0, ge=0.0, description="Estimated cost for the request")
    cost_saving_ratio: float = Field(0.0, ge=0.0, le=1.0, description="Potential cost saving ratio")
    quality_risk: str = Field("low", description="Quality risk level: low, medium, high")
    needs_upgrade: bool = Field(False, description="Whether the request needs a model upgrade")

    class Config:
        json_schema_extra = {
            "example": {
                "score": 0.35,
                "level": "simple",
                "confidence": 0.88,
                "lexical_score": 0.3,
                "structural_score": 0.2,
                "domain_score": 0.4,
                "conversational_score": 0.5,
                "task_type_score": 0.3,
                "recommended_tier": "economy",
                "recommended_model": "claude-haiku-4-20250514",
                "fallback_model": "deepseek-v4-flash",
                "estimated_cost": 0.001,
                "cost_saving_ratio": 0.7,
                "quality_risk": "low",
                "needs_upgrade": False
            }
        }


class FeedbackRequest(BaseModel):
    """Request model for quality feedback recording."""
    request_id: str = Field(..., description="Original request ID for tracking")
    quality_score: float = Field(..., ge=0.0, le=1.0, description="Quality score (0-1) from user feedback")

    class Config:
        json_schema_extra = {
            "example": {
                "request_id": "req_abc123",
                "quality_score": 0.85
            }
        }


class ModelTierConfig(BaseModel):
    """Configuration for a model routing tier."""
    tier: str = Field(..., description="Tier name: economy, standard, premium")
    models: List[str] = Field(..., min_length=1, description="Models in this tier")
    threshold: float = Field(..., ge=0.0, le=1.0, description="Complexity threshold for this tier")
    cost_per_token: float = Field(..., ge=0.0, description="Cost per token for this tier")
    fallback_model: str = Field("", description="Fallback model if primary is unavailable")

    class Config:
        json_schema_extra = {
            "example": {
                "tier": "economy",
                "models": ["claude-haiku-4-20250514", "deepseek-v4-flash", "gpt-4.1-mini"],
                "threshold": 0.4,
                "cost_per_token": 0.0000001,
                "fallback_model": "claude-haiku-4-20250514"
            }
        }
