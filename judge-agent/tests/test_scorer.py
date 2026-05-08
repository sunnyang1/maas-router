"""
Tests for the scorer module
"""

import pytest
from judge.scorer import (
    ComplexityScorer,
    ScoringConfig,
    ComplexityLevel,
    PromptBuilder,
    ScoreResult
)


class TestScoringConfig:
    """Tests for ScoringConfig."""

    def test_default_config(self):
        config = ScoringConfig()
        assert config.min_score == 1
        assert config.max_score == 10
        assert "simple" in config.thresholds
        assert "advanced" in config.thresholds

    def test_get_level_simple(self):
        config = ScoringConfig()
        assert config.get_level(1) == ComplexityLevel.SIMPLE
        assert config.get_level(3) == ComplexityLevel.SIMPLE

    def test_get_level_normal(self):
        config = ScoringConfig()
        assert config.get_level(4) == ComplexityLevel.NORMAL
        assert config.get_level(6) == ComplexityLevel.NORMAL

    def test_get_level_complex(self):
        config = ScoringConfig()
        assert config.get_level(7) == ComplexityLevel.COMPLEX
        assert config.get_level(8) == ComplexityLevel.COMPLEX

    def test_get_level_advanced(self):
        config = ScoringConfig()
        assert config.get_level(9) == ComplexityLevel.ADVANCED
        assert config.get_level(10) == ComplexityLevel.ADVANCED


class TestComplexityScorer:
    """Tests for ComplexityScorer."""

    @pytest.fixture
    def scorer(self):
        return ComplexityScorer()

    def test_rule_based_score_greeting(self, scorer):
        result = scorer.rule_based_score("Hello")
        assert result.score <= 3
        assert result.level == ComplexityLevel.SIMPLE
        assert result.model_used == "rule-based"

    def test_rule_based_score_translation(self, scorer):
        result = scorer.rule_based_score("请翻译这段话到英文")
        assert 4 <= result.score <= 6
        assert result.level == ComplexityLevel.NORMAL

    def test_rule_based_score_code(self, scorer):
        result = scorer.rule_based_score("请帮我写一个Python函数")
        assert result.score >= 7
        assert result.level == ComplexityLevel.COMPLEX

    def test_rule_based_score_research(self, scorer):
        result = scorer.rule_based_score("请分析量子计算的发展趋势")
        assert result.score >= 7

    def test_parse_llm_response_direct_number(self, scorer):
        assert scorer.parse_llm_response("7") == 7
        assert scorer.parse_llm_response("  5  ") == 5

    def test_parse_llm_response_json(self, scorer):
        assert scorer.parse_llm_response('{"score": 8}') == 8
        assert scorer.parse_llm_response('{"complexity": 6}') == 6

    def test_parse_llm_response_text(self, scorer):
        assert scorer.parse_llm_response("The score is 7") == 7
        assert scorer.parse_llm_response("复杂度评分：8分") == 8

    def test_parse_llm_response_invalid(self, scorer):
        assert scorer.parse_llm_response("") is None
        assert scorer.parse_llm_response("no number here") is None

    def test_validate_score_clamping(self, scorer):
        assert scorer.validate_score(0) == 1
        assert scorer.validate_score(15) == 10
        assert scorer.validate_score(5) == 5


class TestPromptBuilder:
    """Tests for PromptBuilder."""

    @pytest.fixture
    def builder(self):
        return PromptBuilder()

    def test_build_messages(self, builder):
        messages = builder.build_messages("Test query")
        assert len(messages) == 2
        assert messages[0]["role"] == "system"
        assert messages[1]["role"] == "user"
        assert "Test query" in messages[1]["content"]

    def test_build_prompt(self, builder):
        prompt = builder.build_prompt("Test query")
        assert "Test query" in prompt
        assert "复杂度评估" in prompt

    def test_custom_prompts(self):
        builder = PromptBuilder(
            system_prompt="Custom system",
            user_template="Query: {query}"
        )
        messages = builder.build_messages("test")
        assert messages[0]["content"] == "Custom system"
        assert "Query: test" in messages[1]["content"]
