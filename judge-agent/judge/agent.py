"""
Judge Agent Module

Core agent implementation for complexity-based routing decisions.
"""

import os
import time
import logging
from typing import Optional, Dict, Any, List
from dataclasses import dataclass, field
from urllib.parse import urlparse

import httpx
import yaml

from .scorer import (
    ComplexityScorer,
    ScoreResult,
    ScoringConfig,
    PromptBuilder
)

logger = logging.getLogger(__name__)


@dataclass
class JudgeConfig:
    """Configuration for Judge Agent."""
    # Model configuration
    model_name: str = "Qwen2.5-7B-Instruct"
    api_url: str = "http://localhost:8001/v1/chat/completions"
    api_key: Optional[str] = None
    timeout: float = 30.0
    max_tokens: int = 256
    temperature: float = 0.1

    # Scoring configuration
    scoring_config: ScoringConfig = field(default_factory=ScoringConfig)

    # Prompt configuration
    system_prompt: Optional[str] = None
    user_template: Optional[str] = None

    @classmethod
    def from_yaml(cls, config_path: str) -> "JudgeConfig":
        """Load configuration from YAML file."""
        with open(config_path, 'r', encoding='utf-8') as f:
            config_data = yaml.safe_load(f)

        judge_config = config_data.get('judge', {})
        model_config = judge_config.get('model', {})
        scoring_config_data = judge_config.get('scoring', {})
        prompts_config = judge_config.get('prompts', {})

        # Resolve environment variables
        api_url = cls._resolve_env_vars(model_config.get('api_url', 'http://localhost:8001/v1/chat/completions'))
        api_key = cls._resolve_env_vars(model_config.get('api_key', '')) or None

        # Build scoring config
        scoring_cfg = ScoringConfig(
            min_score=scoring_config_data.get('min_score', 1),
            max_score=scoring_config_data.get('max_score', 10),
            thresholds=scoring_config_data.get('thresholds')
        )

        return cls(
            model_name=model_config.get('name', 'Qwen2.5-7B-Instruct'),
            api_url=api_url,
            api_key=api_key if api_key else None,
            timeout=model_config.get('timeout', 30.0),
            max_tokens=model_config.get('max_tokens', 256),
            temperature=model_config.get('temperature', 0.1),
            scoring_config=scoring_cfg,
            system_prompt=prompts_config.get('system_prompt'),
            user_template=prompts_config.get('user_prompt_template')
        )

    @staticmethod
    def _resolve_env_vars(value: str) -> str:
        """Resolve environment variables in config values."""
        if not isinstance(value, str):
            return value

        import re
        pattern = r'\$\{([^}]+)\}'
        matches = re.findall(pattern, value)

        for match in matches:
            if ':' in match:
                env_var, default = match.split(':', 1)
                env_value = os.getenv(env_var, default)
            else:
                env_value = os.getenv(match, '')
            value = value.replace(f'${{{match}}}', env_value)

        return value


class LLMClient:
    """HTTP client for LLM API calls."""

    BLOCKED_HOSTS = {"169.254.169.254", "metadata.google.internal", "metadata.azure.com"}

    def __init__(
        self,
        api_url: str,
        api_key: Optional[str] = None,
        timeout: float = 30.0
    ):
        parsed = urlparse(api_url)
        if parsed.scheme not in ("http", "https"):
            raise ValueError(f"Unsupported URL scheme: {parsed.scheme}")
        if parsed.hostname in self.BLOCKED_HOSTS:
            raise ValueError(f"Blocked host: {parsed.hostname}")
        self.api_url = api_url
        self.api_key = api_key
        self.timeout = timeout
        self.headers = {
            "Content-Type": "application/json"
        }
        if api_key:
            self.headers["Authorization"] = f"Bearer {api_key}"

    async def chat_completion(
        self,
        messages: List[Dict[str, str]],
        max_tokens: int = 256,
        temperature: float = 0.1,
        model: Optional[str] = None
    ) -> Dict[str, Any]:
        """Send chat completion request to LLM API."""
        payload = {
            "messages": messages,
            "max_tokens": max_tokens,
            "temperature": temperature,
            "stream": False
        }
        if model:
            payload["model"] = model

        async with httpx.AsyncClient(timeout=self.timeout) as client:
            response = await client.post(
                self.api_url,
                headers=self.headers,
                json=payload
            )
            response.raise_for_status()
            return response.json()

    async def completion(
        self,
        prompt: str,
        max_tokens: int = 256,
        temperature: float = 0.1,
        model: Optional[str] = None
    ) -> Dict[str, Any]:
        """Send completion request to LLM API (non-chat)."""
        payload = {
            "prompt": prompt,
            "max_tokens": max_tokens,
            "temperature": temperature,
            "stream": False
        }
        if model:
            payload["model"] = model

        async with httpx.AsyncClient(timeout=self.timeout) as client:
            response = await client.post(
                self.api_url,
                headers=self.headers,
                json=payload
            )
            response.raise_for_status()
            return response.json()


class JudgeAgent:
    """
    Judge Agent for complexity-based routing.

    This agent evaluates the complexity of user queries and assigns scores
    that can be used to route requests to appropriate model tiers.
    """

    def __init__(self, config: JudgeConfig):
        self.config = config
        self.scorer = ComplexityScorer(config.scoring_config)
        self.prompt_builder = PromptBuilder(
            system_prompt=config.system_prompt,
            user_template=config.user_template
        )
        self.llm_client = LLMClient(
            api_url=config.api_url,
            api_key=config.api_key,
            timeout=config.timeout
        )
        self._health_status = {"status": "unknown", "last_check": None}

    @classmethod
    def from_yaml(cls, config_path: str) -> "JudgeAgent":
        """Create JudgeAgent from YAML configuration file."""
        config = JudgeConfig.from_yaml(config_path)
        return cls(config)

    async def score(self, query: str, use_llm: bool = True) -> ScoreResult:
        """
        Score the complexity of a query.

        Args:
            query: The user query to evaluate
            use_llm: Whether to use LLM-based scoring (True) or rule-based only (False)

        Returns:
            ScoreResult containing the complexity score and metadata
        """
        start_time = time.time()
        llm_score: Optional[int] = None

        # Always compute rule-based score as fallback
        rule_score = self.scorer.rule_based_score(query)

        if use_llm:
            try:
                llm_score = await self._llm_score(query)
            except Exception as e:
                logger.warning(f"LLM scoring failed, using rule-based fallback: {e}")

        latency_ms = (time.time() - start_time) * 1000

        return self.scorer.merge_scores(
            llm_score=llm_score,
            rule_score=rule_score,
            model_name=self.config.model_name if llm_score else "rule-based",
            latency_ms=latency_ms
        )

    async def _llm_score(self, query: str) -> Optional[int]:
        """Get complexity score from LLM."""
        messages = self.prompt_builder.build_messages(query)

        response = await self.llm_client.chat_completion(
            messages=messages,
            max_tokens=self.config.max_tokens,
            temperature=self.config.temperature,
            model=self.config.model_name
        )

        # Extract content from response
        content = self._extract_content(response)
        if content:
            return self.scorer.parse_llm_response(content)

        return None

    def _extract_content(self, response: Dict[str, Any]) -> Optional[str]:
        """Extract text content from LLM API response."""
        try:
            # OpenAI-style response format
            if 'choices' in response and len(response['choices']) > 0:
                choice = response['choices'][0]
                if 'message' in choice:
                    return choice['message'].get('content', '')
                elif 'text' in choice:
                    return choice['text']

            # Direct content field
            if 'content' in response:
                return response['content']

            # Text field
            if 'text' in response:
                return response['text']

            # Response field
            if 'response' in response:
                return response['response']

        except Exception as e:
            logger.error(f"Error extracting content from response: {e}")

        return None

    async def health_check(self) -> Dict[str, Any]:
        """Perform health check on the agent and dependencies."""
        status = {
            "status": "healthy",
            "components": {
                "scorer": "healthy",
                "llm_client": "unknown"
            },
            "timestamp": time.time()
        }

        # Check LLM connectivity
        try:
            test_messages = [{"role": "user", "content": "Hi"}]
            response = await self.llm_client.chat_completion(
                messages=test_messages,
                max_tokens=10,
                temperature=0.0
            )
            if response:
                status["components"]["llm_client"] = "healthy"
            else:
                status["components"]["llm_client"] = "degraded"
                status["status"] = "degraded"
        except Exception as e:
            logger.warning(f"LLM health check failed: {e}")
            status["components"]["llm_client"] = "unhealthy"
            status["status"] = "degraded"
            status["llm_error"] = "LLM connection failed"

        self._health_status = status
        return status

    def get_supported_models(self) -> List[Dict[str, Any]]:
        """Get list of supported models for routing based on complexity tiers."""
        return [
            {
                "tier": "lightweight",
                "score_range": [1, 3],
                "models": ["qwen2.5-1.5b", "llama3.2-1b"],
                "description": "轻量级模型，适用于简单任务",
                "use_cases": ["简单问答", "格式化任务", "问候", "简单事实查询"]
            },
            {
                "tier": "standard",
                "score_range": [4, 6],
                "models": ["qwen2.5-7b", "llama3.1-8b"],
                "description": "标准模型，适用于一般任务",
                "use_cases": ["常规对话", "翻译", "摘要", "一般性解释"]
            },
            {
                "tier": "advanced",
                "score_range": [7, 8],
                "models": ["qwen2.5-14b", "llama3.1-70b"],
                "description": "高级模型，适用于复杂任务",
                "use_cases": ["复杂推理", "代码生成", "数学推理", "逻辑分析"]
            },
            {
                "tier": "premium",
                "score_range": [9, 10],
                "models": ["qwen2.5-72b", "gpt-4o"],
                "description": "顶级模型，适用于高难度任务",
                "use_cases": ["高级创作", "深度分析", "复杂系统设计", "研究论文"]
            }
        ]

    def get_routing_recommendation(self, score_result: ScoreResult) -> Dict[str, Any]:
        """Get routing recommendation based on complexity score."""
        models = self.get_supported_models()

        for tier in models:
            min_score, max_score = tier["score_range"]
            if min_score <= score_result.score <= max_score:
                return {
                    "recommended_tier": tier["tier"],
                    "recommended_models": tier["models"],
                    "confidence": score_result.confidence,
                    "reasoning": score_result.reasoning,
                    "latency_estimate_ms": self._estimate_latency(tier["tier"])
                }

        # Default to standard tier
        return {
            "recommended_tier": "standard",
            "recommended_models": ["qwen2.5-7b"],
            "confidence": score_result.confidence,
            "reasoning": "Default routing (score out of expected range)",
            "latency_estimate_ms": 500
        }

    def _estimate_latency(self, tier: str) -> int:
        """Estimate latency for a given tier (in milliseconds)."""
        latency_map = {
            "lightweight": 100,
            "standard": 300,
            "advanced": 800,
            "premium": 1500
        }
        return latency_map.get(tier, 500)


class JudgeAgentFactory:
    """Factory for creating JudgeAgent instances."""

    _instance: Optional[JudgeAgent] = None

    @classmethod
    def get_agent(cls, config_path: Optional[str] = None) -> JudgeAgent:
        """Get or create singleton JudgeAgent instance."""
        if cls._instance is None:
            if config_path:
                cls._instance = JudgeAgent.from_yaml(config_path)
            else:
                # Try default config locations
                default_paths = [
                    "config.yaml",
                    "/data/user/work/maas-router/judge-agent/config.yaml",
                    "/app/config.yaml"
                ]
                for path in default_paths:
                    if os.path.exists(path):
                        cls._instance = JudgeAgent.from_yaml(path)
                        break
                else:
                    # Create with default config
                    cls._instance = JudgeAgent(JudgeConfig())

        return cls._instance

    @classmethod
    def reset(cls):
        """Reset the singleton instance (useful for testing)."""
        cls._instance = None
