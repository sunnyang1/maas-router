"""
Complexity Scorer Module

Provides scoring logic for evaluating query complexity using LLM-based assessment.
"""

import re
import json
import logging
from enum import Enum
from dataclasses import dataclass
from typing import Optional, Dict, Any, List
from pydantic import BaseModel, Field

logger = logging.getLogger(__name__)


class ComplexityLevel(str, Enum):
    """Complexity level categories based on score ranges."""
    SIMPLE = "simple"      # 1-3: 简单问答、格式化任务
    NORMAL = "normal"      # 4-6: 常规对话、翻译、摘要
    COMPLEX = "complex"    # 7-8: 复杂推理、代码生成
    ADVANCED = "advanced"  # 9-10: 高级创作、深度分析


class ScoreResult(BaseModel):
    """Result of complexity scoring."""
    score: int = Field(..., ge=1, le=10, description="Complexity score from 1 to 10")
    level: ComplexityLevel = Field(..., description="Complexity level category")
    confidence: float = Field(..., ge=0.0, le=1.0, description="Confidence score")
    reasoning: Optional[str] = Field(None, description="Reasoning behind the score")
    model_used: Optional[str] = Field(None, description="Model used for scoring")
    latency_ms: Optional[float] = Field(None, description="Scoring latency in milliseconds")

    class Config:
        json_schema_extra = {
            "example": {
                "score": 7,
                "level": "complex",
                "confidence": 0.92,
                "reasoning": "Query involves multi-step reasoning",
                "model_used": "Qwen2.5-7B-Instruct",
                "latency_ms": 150.5
            }
        }


@dataclass
class ScoringConfig:
    """Configuration for complexity scoring."""
    min_score: int = 1
    max_score: int = 10
    thresholds: Dict[str, Dict[str, Any]] = None

    def __post_init__(self):
        if self.thresholds is None:
            self.thresholds = {
                "simple": {"min": 1, "max": 3, "description": "简单问答、格式化任务"},
                "normal": {"min": 4, "max": 6, "description": "常规对话、翻译、摘要"},
                "complex": {"min": 7, "max": 8, "description": "复杂推理、代码生成"},
                "advanced": {"min": 9, "max": 10, "description": "高级创作、深度分析"}
            }

    def get_level(self, score: int) -> ComplexityLevel:
        """Get complexity level based on score."""
        for level_name, threshold in self.thresholds.items():
            if threshold["min"] <= score <= threshold["max"]:
                return ComplexityLevel(level_name)
        # Default to normal if out of range
        return ComplexityLevel.NORMAL


class ComplexityScorer:
    """
    Complexity Scorer using LLM-based assessment.

    This class provides methods to score query complexity using either:
    1. LLM-based scoring (primary method)
    2. Rule-based fallback scoring
    """

    def __init__(self, config: Optional[ScoringConfig] = None):
        self.config = config or ScoringConfig()
        self._rule_patterns = self._compile_rule_patterns()

    def _compile_rule_patterns(self) -> Dict[str, List[re.Pattern]]:
        """Compile regex patterns for rule-based scoring."""
        return {
            "simple": [
                re.compile(r'^\s*(hi|hello|hey|你好|您好)\s*$', re.IGNORECASE),
                re.compile(r'^\s*(谢谢|感谢|thank)\s*$', re.IGNORECASE),
                re.compile(r'^\s*(再见|拜拜|bye)\s*$', re.IGNORECASE),
                re.compile(r'^\s*(\d+\s*[\+\-\*\/]\s*\d+)\s*$'),  # Simple math
            ],
            "normal": [
                re.compile(r'翻译|translate', re.IGNORECASE),
                re.compile(r'摘要|summarize|summary', re.IGNORECASE),
                re.compile(r'解释|explain', re.IGNORECASE),
            ],
            "complex": [
                re.compile(r'代码|code|编程|programming', re.IGNORECASE),
                re.compile(r'算法|algorithm', re.IGNORECASE),
                re.compile(r'推理|reasoning|逻辑|logic', re.IGNORECASE),
                re.compile(r'数学|math|计算|calculate', re.IGNORECASE),
            ],
            "advanced": [
                re.compile(r'创作|create|write.*essay|write.*story', re.IGNORECASE),
                re.compile(r'分析|analyze|analysis', re.IGNORECASE),
                re.compile(r'设计|design|架构|architecture', re.IGNORECASE),
                re.compile(r'研究|research|论文|paper', re.IGNORECASE),
            ]
        }

    def rule_based_score(self, query: str) -> ScoreResult:
        """
        Calculate complexity score using rule-based heuristics.

        This is used as a fallback when LLM scoring is unavailable.
        """
        query_lower = query.lower()
        score = 5  # Default middle score
        confidence = 0.5
        reasoning_parts = []

        # Check query length
        word_count = len(query.split())
        char_count = len(query)

        if char_count < 20:
            score -= 2
            reasoning_parts.append("Very short query")
        elif char_count > 500:
            score += 2
            reasoning_parts.append("Long query")

        # Check for complexity indicators
        if any(pattern.search(query) for pattern in self._rule_patterns["advanced"]):
            score = max(score, 9)
            confidence = 0.7
            reasoning_parts.append("Contains advanced task keywords")
        elif any(pattern.search(query) for pattern in self._rule_patterns["complex"]):
            score = max(score, 7)
            confidence = 0.75
            reasoning_parts.append("Contains complex task keywords")
        elif any(pattern.search(query) for pattern in self._rule_patterns["normal"]):
            score = max(score, 4)
            confidence = 0.8
            reasoning_parts.append("Contains normal task keywords")
        elif any(pattern.search(query) for pattern in self._rule_patterns["simple"]):
            score = min(score, 3)
            confidence = 0.85
            reasoning_parts.append("Simple greeting or short query")

        # Check for multi-step indicators
        multi_step_indicators = ['步骤', 'step', '首先', 'first', '然后', 'then', '最后', 'finally']
        if any(indicator in query_lower for indicator in multi_step_indicators):
            score += 1
            reasoning_parts.append("Multi-step task indicated")

        # Check for code blocks or structured data
        if '```' in query or query.count('\n') > 10:
            score += 1
            reasoning_parts.append("Contains code or structured content")

        # Clamp score to valid range
        score = max(self.config.min_score, min(self.config.max_score, score))

        return ScoreResult(
            score=score,
            level=self.config.get_level(score),
            confidence=confidence,
            reasoning="; ".join(reasoning_parts) if reasoning_parts else "Rule-based heuristic scoring",
            model_used="rule-based"
        )

    def parse_llm_response(self, response: str) -> Optional[int]:
        """
        Parse LLM response to extract numeric score.

        Handles various response formats:
        - Direct number: "7"
        - JSON: {"score": 7}
        - Text with number: "The complexity score is 7"
        """
        if not response:
            return None

        response = response.strip()

        # Try direct integer parsing
        try:
            return int(response)
        except ValueError:
            pass

        # Try JSON parsing
        try:
            data = json.loads(response)
            if isinstance(data, dict):
                for key in ['score', 'complexity', 'rating', 'value']:
                    if key in data:
                        return int(data[key])
            elif isinstance(data, (int, float)):
                return int(data)
        except json.JSONDecodeError:
            pass

        # Try regex extraction
        patterns = [
            r'(?:score|rating|complexity).*?(\d+)',
            r'(\d+)(?:\s*分|/\s*10)',
            r'^[\s\S]*?(\d+)[\s\S]*$',
        ]

        for pattern in patterns:
            match = re.search(pattern, response, re.IGNORECASE)
            if match:
                try:
                    return int(match.group(1))
                except ValueError:
                    continue

        return None

    def validate_score(self, score: int) -> int:
        """Validate and clamp score to valid range."""
        if score < self.config.min_score:
            logger.warning(f"Score {score} below minimum, clamping to {self.config.min_score}")
            return self.config.min_score
        if score > self.config.max_score:
            logger.warning(f"Score {score} above maximum, clamping to {self.config.max_score}")
            return self.config.max_score
        return score

    def calculate_confidence(
        self,
        llm_score: Optional[int],
        rule_score: ScoreResult,
        parsing_success: bool
    ) -> float:
        """Calculate confidence score based on multiple factors."""
        confidence = 0.5

        if parsing_success and llm_score is not None:
            confidence += 0.3

            # Check agreement between LLM and rule-based
            if abs(llm_score - rule_score.score) <= 2:
                confidence += 0.2

        return min(1.0, confidence)

    def merge_scores(
        self,
        llm_score: Optional[int],
        rule_score: ScoreResult,
        model_name: str,
        latency_ms: float
    ) -> ScoreResult:
        """Merge LLM and rule-based scores with confidence weighting."""
        parsing_success = llm_score is not None

        if parsing_success:
            final_score = self.validate_score(llm_score)
            reasoning = f"LLM scored: {llm_score}"
        else:
            final_score = rule_score.score
            reasoning = f"Fallback to rule-based: {rule_score.reasoning}"

        confidence = self.calculate_confidence(llm_score, rule_score, parsing_success)

        return ScoreResult(
            score=final_score,
            level=self.config.get_level(final_score),
            confidence=confidence,
            reasoning=reasoning,
            model_used=model_name if parsing_success else "rule-based",
            latency_ms=latency_ms
        )


class PromptBuilder:
    """Builds prompts for LLM-based complexity scoring."""

    DEFAULT_SYSTEM_PROMPT = """你是一个专业的任务复杂度评估专家。你的任务是对用户输入的查询进行复杂度评分。

评分标准（1-10分）：
- 1-3分：简单问答、格式化任务（如：问候、简单事实查询、格式转换）
- 4-6分：常规对话、翻译、摘要（如：一般性对话、文本翻译、内容摘要）
- 7-8分：复杂推理、代码生成（如：数学推理、编程任务、逻辑分析）
- 9-10分：高级创作、深度分析（如：创意写作、深度研究报告、复杂系统设计）

你只需要输出一个1-10之间的整数分数，不要有任何其他解释。"""

    DEFAULT_USER_TEMPLATE = """请评估以下查询的复杂度：

查询内容：\"\"\"{query}\"\"\"

请只输出一个1-10之间的整数分数："""

    def __init__(
        self,
        system_prompt: Optional[str] = None,
        user_template: Optional[str] = None
    ):
        self.system_prompt = system_prompt or self.DEFAULT_SYSTEM_PROMPT
        self.user_template = user_template or self.DEFAULT_USER_TEMPLATE

    def build_messages(self, query: str) -> List[Dict[str, str]]:
        """Build message list for chat completion API."""
        return [
            {"role": "system", "content": self.system_prompt},
            {"role": "user", "content": self.user_template.format(query=query)}
        ]

    def build_prompt(self, query: str) -> str:
        """Build single prompt for completion API."""
        return f"{self.system_prompt}\n\n{self.user_template.format(query=query)}"
