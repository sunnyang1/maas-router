# ============================================
# MaaS-Router Admin Platform (React + Vite)
# 日常开发用 Dockerfile — Vite HMR + 非 root 运行
# ============================================

# ---- Stage 1: Dependencies (层缓存友好) ----
FROM node:22-alpine AS deps

WORKDIR /deps

# 先复制包管理文件，充分利用 Docker 层缓存
COPY package.json package-lock.json* ./

# 安装依赖（npm ci 比 npm install 更快更可靠）
RUN npm ci --prefer-offline --no-audit --no-fund

# ---- Stage 2: Dev Runtime ----
FROM node:22-alpine AS dev

WORKDIR /app

# 从 deps 阶段复制 node_modules
COPY --from=deps /deps/node_modules ./node_modules
COPY --from=deps /deps/package.json /deps/package-lock.json* ./

# 创建非 root 用户
RUN addgroup -g 1000 appuser && \
    adduser -u 1000 -G appuser -D -h /app appuser

# 复制应用代码（docker-compose 会通过 volume mount 覆盖此目录）
COPY --chown=appuser:appuser . .

# 确保 node_modules 归属于 appuser
RUN chown -R appuser:appuser node_modules

# 切换到非 root
USER appuser

EXPOSE 5173

# 健康检查 — Vite dev server 无专用 /health，TCP 探测端口
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -q --spider http://localhost:5173 || exit 1

# 默认启动 dev server（docker-compose 的 command 会覆盖 CMD）
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
