# ============================================
# MaaS-Router Backend (FastAPI)
# 日常开发用 Dockerfile — 支持热重载 + 非 root 运行
# ============================================

# ---- Stage 1: Builder ----
FROM python:3.12-slim AS builder

WORKDIR /build

# 安装构建依赖（编译 asyncpg / bcrypt 等需要 gcc + libpq-dev）
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libpq-dev \
    && rm -rf /var/lib/apt/lists/*

# 创建隔离的 virtualenv，便于复制到运行时镜像
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# 先复制依赖文件（充分利用 Docker 层缓存）
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# ---- Stage 2: Runtime ----
FROM python:3.12-slim AS runtime

WORKDIR /app

# 只装运行时必需的库（不需要 gcc）
RUN apt-get update && apt-get install -y --no-install-recommends \
    libpq-dev \
    curl \
    && rm -rf /var/lib/apt/lists/*

# 从 builder 复制 virtualenv
COPY --from=builder /opt/venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# 创建非 root 用户
RUN groupadd -r -g 1000 appuser && \
    useradd -r -u 1000 -g appuser -d /app appuser

# 复制应用代码（docker-compose 会通过 volume mount 覆盖此目录）
COPY --chown=appuser:appuser . .

# 切换到非 root
USER appuser

EXPOSE 8000 8001 8005

# 健康检查 — 两个服务入口 (main.py / admin_main.py) 都有 /health
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -sf http://localhost:${SERVICE_PORT:-8001}/health || exit 1

# 默认启动 API Server（docker-compose 的 command 会覆盖 CMD）
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8001"]
