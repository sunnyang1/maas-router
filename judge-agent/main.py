"""
MaaS-Router Judge Agent - FastAPI Application

This module provides the FastAPI application for the Judge Agent service,
which evaluates query complexity for intelligent LLM routing.
"""

import os
import time
import logging
from contextlib import asynccontextmanager
from typing import Optional, List, Dict, Any

import yaml
from fastapi import FastAPI, HTTPException, status, Depends, Security, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.security import APIKeyHeader
from fastapi.responses import JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware
from pydantic import BaseModel, Field

from judge.agent import JudgeAgent, JudgeAgentFactory, JudgeConfig
from judge.scorer import ScoreResult, ComplexityLevel
from complexity.scorer import router as complexity_router

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Global configuration
CONFIG_PATH = os.getenv("JUDGE_CONFIG_PATH", "config.yaml")
APP_CONFIG: Dict[str, Any] = {}


# ============== Security Middlewares ==============

class RequestSizeLimitMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request, call_next):
        content_length = request.headers.get("content-length")
        if content_length:
            try:
                if int(content_length) > 10_000_000:  # 10MB
                    return JSONResponse(
                        status_code=413,
                        content={"error": "Request body too large", "max_size": "10MB"}
                    )
            except ValueError:
                pass
        return await call_next(request)


class SecurityHeadersMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request, call_next):
        response = await call_next(request)
        response.headers["X-Content-Type-Options"] = "nosniff"
        response.headers["X-Frame-Options"] = "DENY"
        response.headers["X-XSS-Protection"] = "1; mode=block"
        response.headers["Referrer-Policy"] = "strict-origin-when-cross-origin"
        response.headers["Permissions-Policy"] = "camera=(), microphone=(), geolocation=()"
        response.headers["Strict-Transport-Security"] = "max-age=31536000; includeSubDomains"
        return response


# ============== API Key Authentication ==============

JUDGE_API_KEY = os.getenv("JUDGE_API_KEY", "")
api_key_header = APIKeyHeader(name="X-API-Key", auto_error=False)


async def verify_api_key(request: Request, key: Optional[str] = Security(api_key_header)):
    # Skip auth for health check endpoints
    if request.url.path in ("/health", "/v1/complexity/health"):
        return None
    # If no API key is configured, allow all (for development)
    if not JUDGE_API_KEY:
        return None
    if not key or key != JUDGE_API_KEY:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid or missing API key"
        )
    return key


# ============== Application Config ==============

def load_app_config(config_path: str) -> Dict[str, Any]:
    """Load application configuration from YAML file."""
    try:
        with open(config_path, 'r', encoding='utf-8') as f:
            config = yaml.safe_load(f)
        logger.info(f"Configuration loaded from {config_path}")
        return config
    except FileNotFoundError:
        logger.warning(f"Config file not found: {config_path}, using defaults")
        return {
            "app": {
                "name": "maas-router-judge-agent",
                "version": "1.0.0"
            }
        }
    except Exception as e:
        logger.error(f"Error loading config: {e}")
        return {}


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan manager."""
    # Startup
    logger.info("Starting up Judge Agent...")
    global APP_CONFIG
    APP_CONFIG = load_app_config(CONFIG_PATH)

    # Initialize the agent
    try:
        agent = JudgeAgentFactory.get_agent(CONFIG_PATH)
        logger.info("Judge Agent initialized successfully")
    except Exception as e:
        logger.error(f"Failed to initialize Judge Agent: {e}")
        # Continue with default config
        agent = JudgeAgent(JudgeConfig())
        JudgeAgentFactory._instance = agent

    yield

    # Shutdown
    logger.info("Shutting down Judge Agent...")


# Create FastAPI application
app = FastAPI(
    title="MaaS-Router Judge Agent",
    description="""
    智能路由Judge Agent服务，用于评估查询复杂度并进行模型路由。

    ## 功能

    * **复杂度评分** (`/score`): 对用户查询进行1-10分的复杂度评分
    * **健康检查** (`/health`): 服务健康状态检查
    * **模型列表** (`/models`): 获取支持的模型路由配置

    ## 评分标准

    | 分数范围 | 级别 | 描述 |
    |---------|------|------|
    | 1-3 | simple | 简单问答、格式化任务 |
    | 4-6 | normal | 常规对话、翻译、摘要 |
    | 7-8 | complex | 复杂推理、代码生成 |
    | 9-10 | advanced | 高级创作、深度分析 |
    """,
    version=APP_CONFIG.get("app", {}).get("version", "1.0.0"),
    lifespan=lifespan
)

# Add middlewares (order matters: last added = first executed)
# RequestSizeLimitMiddleware runs first (outermost)
app.add_middleware(RequestSizeLimitMiddleware)
# SecurityHeadersMiddleware runs second
app.add_middleware(SecurityHeadersMiddleware)

# CORS configuration with restricted origins
_allowed_origins_str = os.getenv("CORS_ORIGINS", "http://localhost:3000,http://localhost:8000")
ALLOWED_ORIGINS = [origin.strip() for origin in _allowed_origins_str.split(",") if origin.strip()]

app.add_middleware(
    CORSMiddleware,
    allow_origins=ALLOWED_ORIGINS,
    allow_credentials=True,
    allow_methods=["GET", "POST", "OPTIONS"],
    allow_headers=["Content-Type", "Authorization", "X-API-Key"],
)

# Register complexity analysis router
app.include_router(complexity_router, tags=["complexity"])


# ============== Pydantic Models ==============

class ScoreRequest(BaseModel):
    """Request model for complexity scoring."""
    query: str = Field(
        ...,
        min_length=1,
        max_length=10000,
        description="用户查询内容",
        examples=["请帮我写一个Python函数来计算斐波那契数列"]
    )
    use_llm: bool = Field(
        True,
        description="是否使用LLM进行评分（否则使用规则评分）"
    )
    include_routing: bool = Field(
        True,
        description="是否包含路由建议"
    )

    class Config:
        json_schema_extra = {
            "example": {
                "query": "请帮我写一个Python函数来计算斐波那契数列",
                "use_llm": True,
                "include_routing": True
            }
        }


class ScoreResponse(BaseModel):
    """Response model for complexity scoring."""
    score: int = Field(..., ge=1, le=10, description="复杂度评分 (1-10)")
    level: str = Field(..., description="复杂度级别")
    confidence: float = Field(..., ge=0.0, le=1.0, description="置信度")
    reasoning: Optional[str] = Field(None, description="评分理由")
    model_used: Optional[str] = Field(None, description="使用的评分模型")
    latency_ms: float = Field(..., description="评分耗时(毫秒)")
    routing: Optional[Dict[str, Any]] = Field(None, description="路由建议")

    class Config:
        json_schema_extra = {
            "example": {
                "score": 7,
                "level": "complex",
                "confidence": 0.92,
                "reasoning": "Query involves code generation task",
                "model_used": "Qwen2.5-7B-Instruct",
                "latency_ms": 150.5,
                "routing": {
                    "recommended_tier": "advanced",
                    "recommended_models": ["qwen2.5-14b", "llama3.1-70b"],
                    "latency_estimate_ms": 800
                }
            }
        }


class HealthResponse(BaseModel):
    """Response model for health check."""
    status: str = Field(..., description="服务状态")
    version: str = Field(..., description="服务版本")
    components: Dict[str, str] = Field(..., description="组件状态")
    timestamp: float = Field(..., description="检查时间戳")


class ModelInfo(BaseModel):
    """Model information for routing tiers."""
    tier: str = Field(..., description="模型层级")
    score_range: List[int] = Field(..., description="适用的分数范围")
    models: List[str] = Field(..., description="推荐的模型列表")
    description: str = Field(..., description="层级描述")
    use_cases: List[str] = Field(..., description="适用场景")


class ModelsResponse(BaseModel):
    """Response model for models endpoint."""
    models: List[ModelInfo] = Field(..., description="模型路由配置列表")


class ErrorResponse(BaseModel):
    """Error response model."""
    error: str = Field(..., description="错误信息")
    detail: Optional[str] = Field(None, description="详细错误信息")


# ============== API Endpoints ==============

@app.get("/", tags=["Root"])
async def root(api_key: str = Depends(verify_api_key)):
    """Root endpoint with service information."""
    return {
        "service": "MaaS-Router Judge Agent",
        "version": APP_CONFIG.get("app", {}).get("version", "1.0.0"),
        "docs": "/docs",
        "health": "/health"
    }


@app.post(
    "/score",
    response_model=ScoreResponse,
    tags=["Scoring"],
    responses={
        200: {"description": "Successfully scored the query"},
        400: {"model": ErrorResponse, "description": "Invalid request"},
        500: {"model": ErrorResponse, "description": "Internal server error"}
    }
)
async def score_query(request: ScoreRequest, api_key: str = Depends(verify_api_key)):
    """
    对查询进行复杂度评分。

    根据查询内容返回1-10的复杂度评分，用于智能路由决策。
    """
    try:
        agent = JudgeAgentFactory.get_agent()
        result = await agent.score(request.query, use_llm=request.use_llm)

        response = ScoreResponse(
            score=result.score,
            level=result.level.value,
            confidence=result.confidence,
            reasoning=result.reasoning,
            model_used=result.model_used,
            latency_ms=result.latency_ms or 0.0
        )

        if request.include_routing:
            response.routing = agent.get_routing_recommendation(result)

        return response

    except Exception as e:
        logger.error(f"Error scoring query: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to score query"
        )


@app.get(
    "/health",
    response_model=HealthResponse,
    tags=["Health"]
)
async def health_check():
    """
    健康检查端点。

    检查服务及其依赖组件的健康状态。
    """
    try:
        agent = JudgeAgentFactory.get_agent()
        health = await agent.health_check()

        return HealthResponse(
            status=health.get("status", "unknown"),
            version=APP_CONFIG.get("app", {}).get("version", "1.0.0"),
            components=health.get("components", {}),
            timestamp=health.get("timestamp", time.time())
        )

    except Exception as e:
        logger.error(f"Health check failed: {e}")
        return HealthResponse(
            status="unhealthy",
            version=APP_CONFIG.get("app", {}).get("version", "1.0.0"),
            components={"agent": "unhealthy"},
            timestamp=time.time()
        )


@app.post(
    "/health",
    response_model=HealthResponse,
    tags=["Health"]
)
async def health_check_post():
    """
    健康检查端点 (POST方法)。

    与GET /health相同，用于兼容性支持。
    """
    return await health_check()


@app.get(
    "/models",
    response_model=ModelsResponse,
    tags=["Models"]
)
async def get_models(api_key: str = Depends(verify_api_key)):
    """
    获取支持的模型路由配置。

    返回基于复杂度分数的模型路由层级配置。
    """
    try:
        agent = JudgeAgentFactory.get_agent()
        models = agent.get_supported_models()

        model_infos = [
            ModelInfo(
                tier=m["tier"],
                score_range=m["score_range"],
                models=m["models"],
                description=m["description"],
                use_cases=m["use_cases"]
            )
            for m in models
        ]

        return ModelsResponse(models=model_infos)

    except Exception as e:
        logger.error(f"Error getting models: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to get models"
        )


# ============== Batch Scoring Endpoint ==============

class BatchScoreRequest(BaseModel):
    """Request model for batch complexity scoring."""
    queries: List[str] = Field(
        ...,
        min_length=1,
        max_length=100,
        description="查询列表",
        examples=[["你好", "请解释量子计算", "写一个快速排序算法"]]
    )
    use_llm: bool = Field(True, description="是否使用LLM进行评分")


class BatchScoreResponse(BaseModel):
    """Response model for batch scoring."""
    results: List[ScoreResponse] = Field(..., description="评分结果列表")
    total_latency_ms: float = Field(..., description="总耗时(毫秒)")


@app.post(
    "/score/batch",
    response_model=BatchScoreResponse,
    tags=["Scoring"]
)
async def score_batch(request: BatchScoreRequest, api_key: str = Depends(verify_api_key)):
    """
    批量复杂度评分。

    对多个查询进行批量评分，提高效率。
    """
    start_time = time.time()
    results = []

    try:
        agent = JudgeAgentFactory.get_agent()

        for query in request.queries:
            result = await agent.score(query, use_llm=request.use_llm)
            results.append(ScoreResponse(
                score=result.score,
                level=result.level.value,
                confidence=result.confidence,
                reasoning=result.reasoning,
                model_used=result.model_used,
                latency_ms=result.latency_ms or 0.0,
                routing=agent.get_routing_recommendation(result)
            ))

        total_latency_ms = (time.time() - start_time) * 1000

        return BatchScoreResponse(
            results=results,
            total_latency_ms=total_latency_ms
        )

    except Exception as e:
        logger.error(f"Error in batch scoring: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to process batch"
        )


# ============== Error Handlers ==============

@app.exception_handler(Exception)
async def generic_exception_handler(request, exc):
    """Handle generic exceptions."""
    import uuid
    request_id = str(uuid.uuid4())
    logger.error(f"Unhandled exception [request_id={request_id}]: {exc}")
    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content={"error": "Internal server error", "request_id": request_id}
    )


# ============== Main Entry Point ==============

if __name__ == "__main__":
    import uvicorn

    # Load config for server settings
    config = load_app_config(CONFIG_PATH)
    app_config = config.get("app", {})

    host = app_config.get("host", "0.0.0.0")
    port = app_config.get("port", 8000)
    debug = app_config.get("debug", False)

    logger.info(f"Starting Judge Agent on {host}:{port}")

    uvicorn.run(
        "main:app",
        host=host,
        port=port,
        reload=debug,
        log_level="info"
    )
