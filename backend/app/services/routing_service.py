"""
Intelligent model routing service.

Routes chat requests to the optimal model/provider based on
configurable rules stored in the database, cached in Redis.
"""
from sqlalchemy.ext.asyncio import AsyncSession

from app.repositories.model_repo import ModelRepository
from app.repositories.routing_rule_repo import RoutingRuleRepository


class RoutingDecision:
    """Result of a routing decision."""

    def __init__(
        self,
        reason: str,
        model_id: str,
        provider_id: str,
        complexity_score: float,
        confidence: float,
    ):
        self.reason = reason
        self.model_id = model_id
        self.provider_id = provider_id
        self.complexity_score = complexity_score
        self.confidence = confidence

    def to_dict(self) -> dict:
        return {
            "reason": self.reason,
            "model_id": self.model_id,
            "provider_id": self.provider_id,
            "complexity_score": self.complexity_score,
            "confidence": self.confidence,
        }


class RoutingService:
    """Routes requests to the best model/provider."""

    def __init__(self, session: AsyncSession):
        self.model_repo = ModelRepository(session)
        self.rule_repo = RoutingRuleRepository(session)

    async def route(
        self,
        model_id: str,
        prompt: str,
        user_plan: str = "free",
    ) -> RoutingDecision:
        """
        Route a chat request to the best model/provider.

        If model_id is "auto", uses complexity-based intelligent routing.
        Otherwise resolves the specific model.
        """
        if model_id == "auto":
            return await self._auto_route(prompt, user_plan)
        else:
            return await self._specific_route(model_id)

    async def _auto_route(
        self, prompt: str, user_plan: str
    ) -> RoutingDecision:
        """Intelligent routing based on prompt complexity."""
        complexity = self._score_complexity(prompt)

        # Try DB rules first, fall back to hardcoded routing
        rules = await self.rule_repo.get_active_rules()
        for rule in rules:
            cond = rule.condition or {}
            action = rule.action or {}
            min_score = cond.get("complexity_min", 0)
            max_score = cond.get("complexity_max", 10)
            plan_ok = cond.get("plan") in (None, user_plan)
            if min_score <= complexity <= max_score and plan_ok:
                return RoutingDecision(
                    reason=rule.description or f"规则匹配: {rule.name}",
                    model_id=action.get("model_id", "gpt-4o-mini"),
                    provider_id=action.get("provider_id", "openai"),
                    complexity_score=complexity,
                    confidence=min(1.0, complexity / 10.0),
                )

        # Default routing based on complexity tiers
        if complexity < 4:
            reason, model, provider = "简单查询，路由至自建 DeepSeek-V4", "deepseek-v4-self", "self-hosted"
        elif complexity < 7:
            reason, model, provider = "中等复杂度，路由至 DeepSeek-V3", "deepseek-v3", "deepseek"
        elif complexity < 9:
            reason, model, provider = "较高复杂度，路由至 GPT-4o Mini", "gpt-4o-mini", "openai"
        else:
            reason, model, provider = "高复杂度，路由至 GPT-4o", "gpt-4o", "openai"

        return RoutingDecision(
            reason=reason,
            model_id=model,
            provider_id=provider,
            complexity_score=complexity,
            confidence=min(1.0, complexity / 10.0),
        )

    async def _specific_route(self, model_id: str) -> RoutingDecision:
        """Resolve a specific model ID."""
        result = await self.model_repo.get_with_provider(model_id)
        if not result:
            raise ValueError(f"Model '{model_id}' not found")
        model, provider = result
        return RoutingDecision(
            reason="直接指定",
            model_id=model.id,
            provider_id=provider.id,
            complexity_score=5.0,
            confidence=1.0,
        )

    def _score_complexity(self, prompt: str) -> float:
        """Score prompt complexity on a scale of 1.0-10.0."""
        length = len(prompt)
        code_keywords = [
            "def ", "function", "class ", "import ", "```",
            "async", "await", "SELECT", "WHERE",
        ]
        reasoning_keywords = [
            "explain", "analyze", "compare", "summarize",
            "为什么", "分析", "对比",
        ]

        score = 1.0

        # Length factor
        if length > 2000:
            score += 3
        elif length > 1000:
            score += 2
        elif length > 500:
            score += 1

        # Keyword signals
        code_count = sum(1 for kw in code_keywords if kw.lower() in prompt.lower())
        reasoning_count = sum(
            1 for kw in reasoning_keywords if kw.lower() in prompt.lower()
        )

        score += min(code_count, 3) * 1.5
        score += min(reasoning_count, 2) * 1.0

        return min(10.0, max(1.0, score))
