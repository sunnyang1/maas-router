"""
Tests for the FastAPI endpoints
"""

import pytest
from fastapi.testclient import TestClient
from unittest.mock import Mock, patch, AsyncMock

from main import app
from judge.agent import JudgeAgentFactory, JudgeConfig, JudgeAgent
from judge.scorer import ScoreResult, ComplexityLevel


@pytest.fixture
def client():
    """Create test client."""
    return TestClient(app)


@pytest.fixture(autouse=True)
def reset_factory():
    """Reset factory before each test."""
    JudgeAgentFactory.reset()
    yield
    JudgeAgentFactory.reset()


class TestRootEndpoint:
    """Tests for root endpoint."""

    def test_root(self, client):
        response = client.get("/")
        assert response.status_code == 200
        data = response.json()
        assert "service" in data
        assert "MaaS-Router Judge Agent" in data["service"]


class TestHealthEndpoint:
    """Tests for health check endpoint."""

    def test_health_get(self, client):
        with patch.object(JudgeAgent, "health_check", new_callable=AsyncMock) as mock_health:
            mock_health.return_value = {
                "status": "healthy",
                "components": {"scorer": "healthy"},
                "timestamp": 1234567890
            }
            response = client.get("/health")
            assert response.status_code == 200
            data = response.json()
            assert data["status"] == "healthy"

    def test_health_post(self, client):
        with patch.object(JudgeAgent, "health_check", new_callable=AsyncMock) as mock_health:
            mock_health.return_value = {
                "status": "healthy",
                "components": {"scorer": "healthy"},
                "timestamp": 1234567890
            }
            response = client.post("/health")
            assert response.status_code == 200
            data = response.json()
            assert "components" in data


class TestScoreEndpoint:
    """Tests for score endpoint."""

    def test_score_simple_query(self, client):
        with patch.object(JudgeAgent, "score", new_callable=AsyncMock) as mock_score:
            mock_score.return_value = ScoreResult(
                score=2,
                level=ComplexityLevel.SIMPLE,
                confidence=0.9,
                reasoning="Simple greeting",
                model_used="rule-based",
                latency_ms=10.0
            )

            response = client.post("/score", json={
                "query": "Hello",
                "use_llm": False,
                "include_routing": True
            })

            assert response.status_code == 200
            data = response.json()
            assert data["score"] == 2
            assert data["level"] == "simple"
            assert "routing" in data

    def test_score_complex_query(self, client):
        with patch.object(JudgeAgent, "score", new_callable=AsyncMock) as mock_score:
            mock_score.return_value = ScoreResult(
                score=8,
                level=ComplexityLevel.COMPLEX,
                confidence=0.85,
                reasoning="Code generation task",
                model_used="Qwen2.5-7B-Instruct",
                latency_ms=150.0
            )

            response = client.post("/score", json={
                "query": "Write a Python function to calculate fibonacci",
                "use_llm": True,
                "include_routing": True
            })

            assert response.status_code == 200
            data = response.json()
            assert data["score"] == 8
            assert data["level"] == "complex"

    def test_score_without_routing(self, client):
        with patch.object(JudgeAgent, "score", new_callable=AsyncMock) as mock_score:
            mock_score.return_value = ScoreResult(
                score=5,
                level=ComplexityLevel.NORMAL,
                confidence=0.8,
                reasoning="Normal task",
                latency_ms=50.0
            )

            response = client.post("/score", json={
                "query": "Translate this text",
                "include_routing": False
            })

            assert response.status_code == 200
            data = response.json()
            assert data["routing"] is None

    def test_score_empty_query(self, client):
        response = client.post("/score", json={"query": ""})
        assert response.status_code == 422  # Validation error

    def test_score_query_too_long(self, client):
        long_query = "a" * 10001
        response = client.post("/score", json={"query": long_query})
        assert response.status_code == 422  # Validation error


class TestModelsEndpoint:
    """Tests for models endpoint."""

    def test_get_models(self, client):
        response = client.get("/models")
        assert response.status_code == 200
        data = response.json()
        assert "models" in data
        assert len(data["models"]) == 4

        # Check structure
        model = data["models"][0]
        assert "tier" in model
        assert "score_range" in model
        assert "models" in model
        assert "description" in model
        assert "use_cases" in model


class TestBatchScoreEndpoint:
    """Tests for batch score endpoint."""

    def test_batch_score(self, client):
        with patch.object(JudgeAgent, "score", new_callable=AsyncMock) as mock_score:
            mock_score.side_effect = [
                ScoreResult(score=2, level=ComplexityLevel.SIMPLE, confidence=0.9, latency_ms=10.0),
                ScoreResult(score=5, level=ComplexityLevel.NORMAL, confidence=0.8, latency_ms=20.0),
                ScoreResult(score=8, level=ComplexityLevel.COMPLEX, confidence=0.85, latency_ms=30.0),
            ]

            response = client.post("/score/batch", json={
                "queries": ["Hello", "Translate this", "Write code"],
                "use_llm": False
            })

            assert response.status_code == 200
            data = response.json()
            assert len(data["results"]) == 3
            assert "total_latency_ms" in data

    def test_batch_score_empty_list(self, client):
        response = client.post("/score/batch", json={
            "queries": [],
            "use_llm": False
        })
        assert response.status_code == 422  # Validation error

    def test_batch_score_too_many(self, client):
        response = client.post("/score/batch", json={
            "queries": ["query"] * 101,
            "use_llm": False
        })
        assert response.status_code == 422  # Validation error
