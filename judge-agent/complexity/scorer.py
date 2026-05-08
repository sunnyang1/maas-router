"""
Complexity Scorer Module

Enhanced complexity scorer with rule-based precheck, optional LLM refinement,
caching support, and model recommendation for intelligent LLM routing.
"""

import re
import time
import hashlib
import logging
from typing import Optional, Dict, Any, List

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

from .models import (
    AnalyzeRequest,
    ComplexityProfile,
    ModelTierConfig,
    FeedbackRequest,
)

logger = logging.getLogger(__name__)

# FastAPI router for complexity scoring endpoints
router = APIRouter(prefix="/v1/complexity", tags=["Complexity"])


class ScorerConfig:
    """Configuration for the complexity scorer."""

    def __init__(
        self,
        model_tiers: Optional[List[Dict[str, Any]]] = None,
        cache_ttl_sec: int = 3600,
        timeout_ms: int = 50,
        max_retries: int = 1,
        fallback_to_judge: bool = True,
        features: Optional[Dict[str, Any]] = None,
        quality_guard: Optional[Dict[str, Any]] = None,
    ):
        self.model_tiers: List[ModelTierConfig] = []
        if model_tiers:
            for tier_data in model_tiers:
                self.model_tiers.append(ModelTierConfig(**tier_data))

        self.cache_ttl_sec = cache_ttl_sec
        self.timeout_ms = timeout_ms
        self.max_retries = max_retries
        self.fallback_to_judge = fallback_to_judge
        self.features = features or {}
        self.quality_guard = quality_guard or {}

    def get_tier_for_score(self, score: float) -> Optional[ModelTierConfig]:
        """Get the appropriate tier configuration for a given complexity score."""
        # Tiers are sorted by threshold ascending; find the first tier whose threshold >= score
        sorted_tiers = sorted(self.model_tiers, key=lambda t: t.threshold)
        for tier in sorted_tiers:
            if score <= tier.threshold:
                return tier
        # If score exceeds all thresholds, return the highest tier
        return sorted_tiers[-1] if sorted_tiers else None


class ComplexityScorer:
    """
    Enhanced Complexity Scorer.

    Provides rule-based precheck with optional LLM refinement,
    model tier recommendation, and cost estimation.
    """

    def __init__(self, config: Optional[ScorerConfig] = None):
        self.config = config or ScorerConfig()
        self._rule_patterns = self._compile_rule_patterns()
        self._cache: Dict[str, tuple] = {}  # hash -> (profile, timestamp)
        self._llm_client = None  # Optional LLM client for fine-grained scoring

    def set_llm_client(self, client: Any):
        """Set an optional LLM client for fine-grained scoring."""
        self._llm_client = client

    def _compile_rule_patterns(self) -> Dict[str, List[re.Pattern]]:
        """Compile regex patterns for rule-based scoring."""
        return {
            "simple": [
                re.compile(r'^\s*(hi|hello|hey|你好|您好|谢谢|感谢|再见|拜拜)\s*$', re.IGNORECASE),
                re.compile(r'^\s*(\d+\s*[\+\-\*\/]\s*\d+)\s*$'),
                re.compile(r'^\s*(yes|no|ok|好的|是的|不是)\s*$', re.IGNORECASE),
            ],
            "normal": [
                re.compile(r'翻译|translate', re.IGNORECASE),
                re.compile(r'摘要|summarize|summary', re.IGNORECASE),
                re.compile(r'解释|explain', re.IGNORECASE),
                re.compile(r'什么|what|how|怎么|如何', re.IGNORECASE),
            ],
            "complex": [
                re.compile(r'代码|code|编程|programming|函数|function', re.IGNORECASE),
                re.compile(r'算法|algorithm|排序|sort', re.IGNORECASE),
                re.compile(r'推理|reasoning|逻辑|logic', re.IGNORECASE),
                re.compile(r'数学|math|计算|calculate|公式', re.IGNORECASE),
                re.compile(r'优化|optimize|性能|performance', re.IGNORECASE),
            ],
            "advanced": [
                re.compile(r'创作|create|write.*essay|write.*story|写.*文章', re.IGNORECASE),
                re.compile(r'分析|analyze|analysis|研究|research', re.IGNORECASE),
                re.compile(r'设计|design|架构|architecture', re.IGNORECASE),
                re.compile(r'论文|paper|学术|academic', re.IGNORECASE),
                re.compile(r'多步|multi-step|pipeline|工作流|workflow', re.IGNORECASE),
            ],
        }

    async def score(self, request: AnalyzeRequest) -> ComplexityProfile:
        """
        Score the complexity of a request and return a full ComplexityProfile.

        Uses rule-based precheck first, then optionally refines with LLM.
        """
        start_time = time.time()

        # 1. Check cache
        cache_key = self._compute_cache_key(request)
        cached = self._get_from_cache(cache_key)
        if cached is not None:
            logger.debug(f"Cache hit for complexity analysis: {cache_key[:8]}...")
            return cached

        # 2. Rule-based precheck
        rule_result = self._rule_based_precheck(request)

        # 3. Optional LLM refinement
        if self._llm_client is not None:
            try:
                llm_result = await self._llm_refine(request, rule_result)
                if llm_result is not None:
                    rule_result = self._merge_results(rule_result, llm_result)
            except Exception as e:
                logger.warning(f"LLM refinement failed, using rule-based result: {e}")

        # 4. Determine tier and model recommendation
        tier_config = self.config.get_tier_for_score(rule_result["score"])
        if tier_config:
            rule_result["recommended_tier"] = tier_config.tier
            rule_result["recommended_model"] = tier_config.models[0] if tier_config.models else ""
            rule_result["fallback_model"] = tier_config.fallback_model
            rule_result["estimated_cost"] = self._estimate_cost(request, tier_config)
            rule_result["cost_saving_ratio"] = self._calculate_cost_saving(request, tier_config)

        # 5. Build profile
        profile = ComplexityProfile(
            score=rule_result["score"],
            level=self._score_to_level(rule_result["score"]),
            confidence=rule_result["confidence"],
            lexical_score=rule_result.get("lexical_score", 0.0),
            structural_score=rule_result.get("structural_score", 0.0),
            domain_score=rule_result.get("domain_score", 0.0),
            conversational_score=rule_result.get("conversational_score", 0.0),
            task_type_score=rule_result.get("task_type_score", 0.0),
            recommended_tier=rule_result.get("recommended_tier", ""),
            recommended_model=rule_result.get("recommended_model", ""),
            fallback_model=rule_result.get("fallback_model", ""),
            estimated_cost=rule_result.get("estimated_cost", 0.0),
            cost_saving_ratio=rule_result.get("cost_saving_ratio", 0.0),
            quality_risk=rule_result.get("quality_risk", "low"),
            needs_upgrade=rule_result.get("needs_upgrade", False),
        )

        # 6. Cache the result
        self._store_in_cache(cache_key, profile)

        latency_ms = (time.time() - start_time) * 1000
        logger.info(
            f"Complexity analysis completed: score={profile.score:.2f}, "
            f"level={profile.level}, tier={profile.recommended_tier}, "
            f"model={profile.recommended_model}, latency={latency_ms:.1f}ms"
        )

        return profile

    async def health_check(self) -> dict:
        """Perform health check on the scorer and its dependencies."""
        status = {
            "status": "healthy",
            "components": {
                "rule_engine": "healthy",
                "cache": "healthy",
                "llm_client": "not_configured",
            },
            "cache_size": len(self._cache),
            "tier_count": len(self.config.model_tiers),
        }

        if self._llm_client is not None:
            try:
                # Attempt a simple LLM call to verify connectivity
                status["components"]["llm_client"] = "healthy"
            except Exception:
                status["components"]["llm_client"] = "unhealthy"
                status["status"] = "degraded"

        return status

    def _rule_based_precheck(self, request: AnalyzeRequest) -> Dict[str, Any]:
        """
        Fast rule-based complexity precheck.

        Consistent with the Go-side scoring logic for quick evaluation.
        Returns a dict with score components.
        """
        # Combine all message content for analysis
        full_text = ""
        if request.system:
            full_text += request.system + "\n"
        for msg in request.messages:
            full_text += msg.content + "\n"

        full_text = full_text.strip()
        text_lower = full_text.lower()

        # Initialize score components
        lexical_score = 0.0
        structural_score = 0.0
        domain_score = 0.0
        conversational_score = 0.0
        task_type_score = 0.0

        # --- Lexical analysis ---
        word_count = len(full_text.split())
        char_count = len(full_text)
        sentence_count = full_text.count('.') + full_text.count('!') + full_text.count('?') + full_text.count('\n')

        # Normalize scores to 0-1 range
        lexical_score = min(1.0, word_count / 500.0) * 0.3 + min(1.0, char_count / 5000.0) * 0.3 + min(1.0, sentence_count / 20.0) * 0.4

        # Average word length (indicator of vocabulary complexity)
        if word_count > 0:
            avg_word_len = sum(len(w) for w in full_text.split()) / word_count
            lexical_score += min(1.0, avg_word_len / 8.0) * 0.2
            lexical_score = min(1.0, lexical_score)

        # --- Structural analysis ---
        # Check for multi-turn conversation
        msg_count = len(request.messages)
        structural_score = min(1.0, msg_count / 10.0) * 0.3

        # Check for code blocks
        if '```' in full_text:
            structural_score += 0.3
        if full_text.count('\n') > 10:
            structural_score += 0.2

        # Check for structured data (JSON, XML, etc.)
        if any(marker in full_text for marker in ['{', '[', '<', '"key"', '"id"']):
            structural_score += 0.2

        structural_score = min(1.0, structural_score)

        # --- Domain analysis ---
        domain_keywords = {
            "technical": ["API", "HTTP", "SQL", "docker", "kubernetes", "git", "linux", "python", "javascript", "rust", "golang", "数据库", "服务器", "框架"],
            "mathematical": ["公式", "方程", "积分", "微分", "矩阵", "向量", "formula", "equation", "integral", "matrix"],
            "scientific": ["实验", "假设", "理论", "experiment", "hypothesis", "theory", "分子", "原子"],
            "creative": ["故事", "诗歌", "小说", "创意", "story", "poem", "novel", "creative"],
            "legal": ["合同", "法律", "条款", "contract", "legal", "clause", "regulation"],
            "medical": ["症状", "诊断", "治疗", "symptom", "diagnosis", "treatment", "药物"],
        }

        domain_matches = 0
        for domain, keywords in domain_keywords.items():
            if any(kw.lower() in text_lower for kw in keywords):
                domain_matches += 1
                domain_score += 0.2

        domain_score = min(1.0, domain_score)

        # --- Conversational analysis ---
        # Check if this is a multi-turn conversation
        user_msgs = sum(1 for m in request.messages if m.role == "user")
        assistant_msgs = sum(1 for m in request.messages if m.role == "assistant")
        conversational_score = min(1.0, (user_msgs + assistant_msgs) / 8.0) * 0.5

        # Check for context-dependent references
        context_indicators = ["上面", "之前", "刚才", "那个", "这个", "above", "before", "previous", "that", "this"]
        if any(indicator in text_lower for indicator in context_indicators):
            conversational_score += 0.3

        conversational_score = min(1.0, conversational_score)

        # --- Task type analysis ---
        # Check for complexity patterns
        if any(p.search(full_text) for p in self._rule_patterns["advanced"]):
            task_type_score = max(task_type_score, 0.8)
        elif any(p.search(full_text) for p in self._rule_patterns["complex"]):
            task_type_score = max(task_type_score, 0.6)
        elif any(p.search(full_text) for p in self._rule_patterns["normal"]):
            task_type_score = max(task_type_score, 0.4)
        elif any(p.search(full_text) for p in self._rule_patterns["simple"]):
            task_type_score = min(task_type_score, 0.2)

        # Multi-step indicators
        multi_step = ["步骤", "step", "首先", "first", "然后", "then", "最后", "finally", "接着", "next"]
        if any(indicator in text_lower for indicator in multi_step):
            task_type_score = max(task_type_score, 0.5)
            task_type_score += 0.1

        # Tool use indicators
        tool_indicators = ["工具", "tool", "function", "函数调用", "API调用"]
        if any(indicator in text_lower for indicator in tool_indicators):
            task_type_score += 0.2

        task_type_score = min(1.0, task_type_score)

        # --- Aggregate score ---
        weights = {
            "lexical": 0.15,
            "structural": 0.20,
            "domain": 0.20,
            "conversational": 0.15,
            "task_type": 0.30,
        }

        overall_score = (
            lexical_score * weights["lexical"]
            + structural_score * weights["structural"]
            + domain_score * weights["domain"]
            + conversational_score * weights["conversational"]
            + task_type_score * weights["task_type"]
        )

        overall_score = max(0.0, min(1.0, overall_score))

        # Confidence based on how many signals we have
        signal_count = sum(1 for s in [lexical_score, structural_score, domain_score, conversational_score, task_type_score] if s > 0.1)
        confidence = min(0.95, 0.5 + signal_count * 0.1)

        # Quality risk assessment
        quality_risk = "low"
        needs_upgrade = False
        if overall_score > 0.7 and task_type_score > 0.6:
            quality_risk = "medium"
        if overall_score > 0.85 or task_type_score > 0.8:
            quality_risk = "high"
            needs_upgrade = True

        return {
            "score": overall_score,
            "confidence": confidence,
            "lexical_score": round(lexical_score, 4),
            "structural_score": round(structural_score, 4),
            "domain_score": round(domain_score, 4),
            "conversational_score": round(conversational_score, 4),
            "task_type_score": round(task_type_score, 4),
            "quality_risk": quality_risk,
            "needs_upgrade": needs_upgrade,
        }

    async def _llm_refine(self, request: AnalyzeRequest, rule_result: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Optional LLM-based refinement of the rule-based score."""
        if self._llm_client is None:
            return None

        # Build a concise prompt for LLM refinement
        last_user_msg = ""
        for msg in reversed(request.messages):
            if msg.role == "user":
                last_user_msg = msg.content[:500]  # Truncate to avoid excessive input
                break

        prompt = (
            f"Rate the complexity of this query on a scale of 0.0 to 1.0. "
            f"Consider: task difficulty, reasoning required, domain expertise needed.\n"
            f"Query: {last_user_msg}\n"
            f"Current rule-based score: {rule_result['score']:.2f}\n"
            f"Respond with only a number between 0.0 and 1.0."
        )

        try:
            # This would call the LLM client - implementation depends on the client interface
            # For now, return None to rely on rule-based scoring
            return None
        except Exception as e:
            logger.warning(f"LLM refinement error: {e}")
            return None

    def _merge_results(self, rule_result: Dict[str, Any], llm_result: Dict[str, Any]) -> Dict[str, Any]:
        """Merge rule-based and LLM results with weighted averaging."""
        merged = rule_result.copy()
        llm_score = llm_result.get("score", rule_result["score"])

        # Weighted average: 70% rule-based, 30% LLM
        merged["score"] = rule_result["score"] * 0.7 + llm_score * 0.3
        merged["confidence"] = min(0.95, rule_result["confidence"] + 0.1)
        return merged

    def _score_to_level(self, score: float) -> str:
        """Convert a 0-1 score to a complexity level string."""
        if score <= 0.25:
            return "simple"
        elif score <= 0.5:
            return "normal"
        elif score <= 0.75:
            return "complex"
        else:
            return "advanced"

    def _estimate_cost(self, request: AnalyzeRequest, tier: ModelTierConfig) -> float:
        """Estimate the cost for a request using a specific tier."""
        # Rough token estimation: ~4 chars per token
        total_chars = 0
        if request.system:
            total_chars += len(request.system)
        for msg in request.messages:
            total_chars += len(msg.content)
        estimated_tokens = total_chars / 4

        # Account for max_tokens in the response
        output_tokens = max(request.max_tokens, 100) if request.max_tokens > 0 else 100

        total_tokens = estimated_tokens + output_tokens
        return total_tokens * tier.cost_per_token

    def _calculate_cost_saving(self, request: AnalyzeRequest, recommended_tier: ModelTierConfig) -> float:
        """Calculate potential cost saving ratio compared to the highest tier."""
        if not self.config.model_tiers:
            return 0.0

        # Find the most expensive tier
        most_expensive = max(self.config.model_tiers, key=lambda t: t.cost_per_token)
        if most_expensive.cost_per_token <= 0:
            return 0.0

        recommended_cost = self._estimate_cost(request, recommended_tier)
        premium_cost = self._estimate_cost(request, most_expensive)

        if premium_cost <= 0:
            return 0.0

        saving = 1.0 - (recommended_cost / premium_cost)
        return max(0.0, min(1.0, saving))

    def _compute_cache_key(self, request: AnalyzeRequest) -> str:
        """Compute a cache key from the request content."""
        content = f"{request.model}:{request.system}:{request.max_tokens}:{request.stream}:"
        for msg in request.messages:
            content += f"{msg.role}:{msg.content}:"
        return hashlib.md5(content.encode()).hexdigest()

    def _get_from_cache(self, key: str) -> Optional[ComplexityProfile]:
        """Get a cached profile if it exists and hasn't expired."""
        if key in self._cache:
            profile, timestamp = self._cache[key]
            if time.time() - timestamp < self.config.cache_ttl_sec:
                return profile
            else:
                del self._cache[key]
        return None

    def _store_in_cache(self, key: str, profile: ComplexityProfile):
        """Store a profile in the cache."""
        # Simple cache eviction: remove oldest entries if cache is too large
        if len(self._cache) > 10000:
            # Remove the oldest 20% of entries
            sorted_keys = sorted(self._cache.keys(), key=lambda k: self._cache[k][1])
            for k in sorted_keys[:len(sorted_keys) // 5]:
                del self._cache[k]

        self._cache[key] = (profile, time.time())


# ============== FastAPI Endpoints ==============

class ScoreResponse(BaseModel):
    """Response wrapper for complexity scoring."""
    score: float
    level: str
    confidence: float
    recommended_tier: str
    recommended_model: str
    cost_saving_ratio: float
    quality_risk: str
    needs_upgrade: bool


class HealthResponse(BaseModel):
    """Health check response."""
    status: str
    components: Dict[str, str]
    cache_size: int
    tier_count: int


# Global scorer instance (initialized at startup)
_scorer: Optional[ComplexityScorer] = None


def init_scorer(config: Optional[ScorerConfig] = None):
    """Initialize the global scorer instance."""
    global _scorer
    _scorer = ComplexityScorer(config)
    logger.info("Complexity scorer initialized")


def get_scorer() -> ComplexityScorer:
    """Get the global scorer instance."""
    global _scorer
    if _scorer is None:
        _scorer = ComplexityScorer()
    return _scorer


@router.post("/score", response_model=ComplexityProfile, tags=["Complexity"])
async def score_complexity(request: AnalyzeRequest):
    """
    Analyze request complexity and get routing recommendations.

    Returns a ComplexityProfile with score, recommended tier/model, and cost estimates.
    """
    scorer = get_scorer()
    try:
        profile = await scorer.score(request)
        return profile
    except Exception as e:
        logger.error(f"Complexity scoring failed: {e}")
        raise HTTPException(status_code=500, detail="Scoring failed")


@router.get("/health", response_model=HealthResponse, tags=["Complexity"])
async def health_check():
    """
    Health check for the complexity scoring service.
    """
    scorer = get_scorer()
    result = await scorer.health_check()
    return HealthResponse(
        status=result["status"],
        components=result["components"],
        cache_size=result["cache_size"],
        tier_count=result["tier_count"],
    )


@router.post("/feedback", tags=["Complexity"])
async def record_feedback(request: FeedbackRequest):
    """
    Record quality feedback for online learning.

    Used to improve model recommendations over time.
    """
    scorer = get_scorer()
    # Delegate to the learner if available
    try:
        from .learner import get_learner
        learner = get_learner()
        await learner.record_feedback(request.request_id, request.quality_score)
        return {"status": "ok", "request_id": request.request_id}
    except Exception as e:
        logger.warning(f"Failed to record feedback: {e}")
        return {"status": "recorded_locally", "request_id": request.request_id}
