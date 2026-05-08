"""
Tests for the agent module
"""

import pytest
import os
from unittest.mock import Mock, patch, AsyncMock

from judge.agent import (
    JudgeAgent,
    JudgeConfig,
    LLMClient,
    JudgeAgentFactory
)
from judge.scorer import ComplexityLevel


class TestJudgeConfig:
    """Tests for JudgeConfig."""

    def test_default_config(self):
        config = JudgeConfig()
        assert config.model_name == "Qwen2.5-7B-Instruct"
        assert config.api_url == "http://localhost:8001/v1/chat/completions"
        assert config.timeout == 30.0

    def test_from_yaml(self, tmp_path):
        yaml_content = """
judge:
  model:
    name: "Test-Model"
    api_url: "http://test:8000/v1/chat/completions"
    timeout: 60
  scoring:
    min_score: 1
    max_score: 10
"""
        config_file = tmp_path / "test_config.yaml"
        config_file.write_text(yaml_content)

        config = JudgeConfig.from_yaml(str(config_file))
        assert config.model_name == "Test-Model"
        assert config.api_url == "http://test:8000/v1/chat/completions"
        assert config.timeout == 60

    def test_env_var_resolution(self):
        os.environ["TEST_API_URL"] = "http://env-test:8000"
        config = JudgeConfig._resolve_env_vars("${TEST_API_URL}")
        assert config == "http://env-test:8000"
        del os.environ["TEST_API_URL"]

    def test_env_var_with_default(self):
        config = JudgeConfig._resolve_env_vars("${NONEXISTENT_VAR:http://default:8000}")
        assert config == "http://default:8000"


class TestLLMClient:
    """Tests for LLMClient."""

    @pytest.fixture
    def client(self):
        return LLMClient(
            api_url="http://test:8000/v1/chat/completions",
            api_key="test-key"
        )

    def test_initialization(self, client):
        assert client.api_url == "http://test:8000/v1/chat/completions"
        assert client.api_key == "test-key"
        assert client.headers["Authorization"] == "Bearer test-key"

    @pytest.mark.asyncio
    async def test_chat_completion(self, client):
        mock_response = Mock()
        mock_response.json.return_value = {
            "choices": [{"message": {"content": "7"}}]
        }
        mock_response.raise_for_status = Mock()

        with patch("httpx.AsyncClient.post", new_callable=AsyncMock) as mock_post:
            mock_post.return_value = mock_response
            result = await client.chat_completion(
                messages=[{"role": "user", "content": "test"}]
            )
            assert result["choices"][0]["message"]["content"] == "7"


class TestJudgeAgent:
    """Tests for JudgeAgent."""

    @pytest.fixture
    def agent(self):
        config = JudgeConfig()
        return JudgeAgent(config)

    @pytest.mark.asyncio
    async def test_score_rule_based(self, agent):
        result = await agent.score("Hello", use_llm=False)
        assert 1 <= result.score <= 10
        assert result.level is not None
        assert result.model_used == "rule-based"

    @pytest.mark.asyncio
    async def test_score_with_llm(self, agent):
        with patch.object(agent.llm_client, "chat_completion", new_callable=AsyncMock) as mock_llm:
            mock_llm.return_value = {
                "choices": [{"message": {"content": "7"}}]
            }
            result = await agent.score("Write a Python function", use_llm=True)
            assert result.score == 7
            assert result.model_used == "Qwen2.5-7B-Instruct"

    @pytest.mark.asyncio
    async def test_score_llm_failure_fallback(self, agent):
        with patch.object(agent.llm_client, "chat_completion", new_callable=AsyncMock) as mock_llm:
            mock_llm.side_effect = Exception("LLM error")
            result = await agent.score("Hello", use_llm=True)
            # Should fallback to rule-based
            assert result.model_used == "rule-based"

    @pytest.mark.asyncio
    async def test_health_check(self, agent):
        with patch.object(agent.llm_client, "chat_completion", new_callable=AsyncMock) as mock_llm:
            mock_llm.return_value = {"choices": [{"message": {"content": "Hi"}}]}
            health = await agent.health_check()
            assert health["status"] in ["healthy", "degraded"]
            assert "components" in health

    def test_get_supported_models(self, agent):
        models = agent.get_supported_models()
        assert len(models) == 4
        tiers = [m["tier"] for m in models]
        assert "lightweight" in tiers
        assert "premium" in tiers

    def test_get_routing_recommendation(self, agent):
        from judge.scorer import ScoreResult
        result = ScoreResult(
            score=8,
            level=ComplexityLevel.COMPLEX,
            confidence=0.9,
            reasoning="Test"
        )
        routing = agent.get_routing_recommendation(result)
        assert routing["recommended_tier"] == "advanced"
        assert "recommended_models" in routing

    def test_extract_content_openai_format(self, agent):
        response = {"choices": [{"message": {"content": "7"}}]}
        assert agent._extract_content(response) == "7"

    def test_extract_content_direct(self, agent):
        response = {"content": "8"}
        assert agent._extract_content(response) == "8"

    def test_extract_content_empty(self, agent):
        assert agent._extract_content({}) is None


class TestJudgeAgentFactory:
    """Tests for JudgeAgentFactory."""

    def setup_method(self):
        JudgeAgentFactory.reset()

    def teardown_method(self):
        JudgeAgentFactory.reset()

    def test_singleton(self):
        agent1 = JudgeAgentFactory.get_agent()
        agent2 = JudgeAgentFactory.get_agent()
        assert agent1 is agent2

    def test_reset(self):
        agent1 = JudgeAgentFactory.get_agent()
        JudgeAgentFactory.reset()
        agent2 = JudgeAgentFactory.get_agent()
        assert agent1 is not agent2
