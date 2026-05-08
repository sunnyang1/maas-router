# 部署指南

> 从开发到生产，MaaS-Router 的完整部署方案。

---

## 目录

1. [部署架构](#1-部署架构)
2. [Docker Compose 部署（推荐）](#2-docker-compose-部署推荐)
3. [生产环境部署](#3-生产环境部署)
4. [环境变量配置](#4-环境变量配置)
5. [CI/CD 集成](#5-cicd-集成)
6. [监控与告警](#6-监控与告警)
7. [备份与恢复](#7-备份与恢复)

---

## 1. 部署架构

```
                      Internet
                         │
                  ┌──────▼──────┐
                  │   Nginx      │  (反向代理 + SSL)
                  │   :443       │
                  └──┬───────┬──┘
                     │       │
          ┌──────────▼──┐ ┌──▼──────────┐
          │ API Server  │ │Admin Server │  (可独立扩缩)
          │   :8001     │ │   :8005     │
          └──────┬──────┘ └──────┬──────┘
                 │               │
          ┌──────▼───────────────▼──────┐
          │       PostgreSQL 16         │
          └──────────────┬──────────────┘
                         │
          ┌──────────────▼──────────────┐
          │         Redis 7             │
          └─────────────────────────────┘
```

**推荐部署方式**：Docker Compose（开发/小规模）→ Kubernetes（大规模生产）

---

## 2. Docker Compose 部署（推荐）

### 2.1 快速部署

```bash
# 1. 克隆项目
git clone <repo-url> && cd maas-router

# 2. 配置环境变量（重要！）
cp .env.example .env
# 编辑 .env，修改 JWT_SECRET_KEY 和数据库密码

# 3. 构建并启动
docker compose up -d --build

# 4. 初始化数据
docker compose exec api-server python -m app.scripts.seed

# 5. 验证
curl http://localhost:8001/health
curl http://localhost:8005/health
```

### 2.2 服务列表

| 服务 | 容器名 | 端口 | 健康检查 |
|------|--------|------|----------|
| PostgreSQL | `maas-postgres` | 5432 | `pg_isready` |
| Redis | `maas-redis` | 6379 | `redis-cli ping` |
| API Server | `maas-api-server` | 8001 | `/health` |
| Admin Server | `maas-admin-server` | 8005 | `/health` |
| Admin Platform | `maas-admin-platform` | 5173 | — |

### 2.3 常用运维命令

```bash
# 查看所有服务状态
docker compose ps

# 查看日志
docker compose logs -f api-server
docker compose logs -f --tail=100 admin-server

# 重启单个服务
docker compose restart api-server

# 更新并重新部署
git pull
docker compose up -d --build

# 停止所有服务
docker compose down

# 清理数据（危险！）
docker compose down -v
```

---

## 3. 生产环境部署

### 3.1 安全清单

部署到生产环境前，**必须**完成以下检查：

- [ ] **更换 JWT Secret**：`openssl rand -hex 32` 生成，替换 `.env` 中的 `JWT_SECRET_KEY`
- [ ] **更换数据库密码**：不使用默认密码 `maas_dev_2026`
- [ ] **关闭 Debug 模式**：`.env` 中设置 `DEBUG=false`
- [ ] **配置 CORS**：`CORS_ORIGINS` 只包含生产域名
- [ ] **启用 HTTPS**：通过 Nginx 或 CDN 配置 SSL 证书
- [ ] **配置防火墙**：只开放 80/443 端口，数据库端口仅内网访问
- [ ] **备份数据库**：配置自动备份策略

### 3.2 Nginx 反向代理配置

```nginx
# /etc/nginx/sites-available/maas-router
server {
    listen 443 ssl;
    server_name api.your-domain.com;

    ssl_certificate     /etc/ssl/certs/your-domain.crt;
    ssl_certificate_key /etc/ssl/private/your-domain.key;

    # API Server
    location /v1/ {
        proxy_pass http://127.0.0.1:8001;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE 支持（Chat Completion 流式响应）
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 300s;
    }

    # Admin Server
    location /api/admin/ {
        proxy_pass http://127.0.0.1:8005;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# HTTP → HTTPS 重定向
server {
    listen 80;
    server_name api.your-domain.com;
    return 301 https://$host$request_uri;
}
```

### 3.3 生产环境 docker-compose 覆盖

创建 `docker-compose.prod.yml`：

```yaml
version: "3.9"

services:
  postgres:
    restart: always
    environment:
      POSTGRES_PASSWORD: ${DB_PASSWORD}  # 从环境变量读取

  api-server:
    restart: always
    environment:
      ENVIRONMENT: production
      DEBUG: "false"
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G

  admin-server:
    restart: always
    environment:
      ENVIRONMENT: production
      DEBUG: "false"

  admin-platform:
    restart: always
    command: npm run build && npm run preview
```

启动：

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

---

## 4. 环境变量配置

详见 [配置参考](CONFIGURATION.md)。生产环境关键变量：

```bash
# 环境
ENVIRONMENT=production          # development | staging | production
DEBUG=false

# 安全（必须更换！）
JWT_SECRET_KEY=<openssl rand -hex 32>
DATABASE_URL=postgresql+asyncpg://user:strong-password@postgres:5432/maas_router
REDIS_URL=redis://:password@redis:6379/0

# CORS（生产域名）
CORS_ORIGINS=https://your-domain.com,https://admin.your-domain.com
```

---

## 5. CI/CD 集成

### 5.1 GitHub Actions（推荐）

项目已包含 CI 配置（`.github/workflows/ci.yml`），参考 [Git 工作流](GIT_WORKFLOW.md) 第 8 节。

### 5.2 部署流水线

```
Push to main
    │
    ▼
GitHub Actions
    ├── Lint (black, flake8, prettier)
    ├── Test (pytest)
    ├── Build (docker compose build)
    └── Deploy
         ├── SSH to server
         ├── docker compose pull
         └── docker compose up -d
```

---

## 6. 监控与告警

### 6.1 健康检查端点

```bash
# API Server
curl http://localhost:8001/health
# → {"status": "ok", "service": "api-server"}

# Admin Server
curl http://localhost:8005/health
# → {"status": "ok", "service": "admin-server"}
```

### 6.2 关键指标

| 指标 | 来源 | 告警阈值建议 |
|------|------|------------|
| API 响应时间 | RequestLog.latency_ms | P95 > 2000ms |
| 错误率 | RequestLog.status_code | > 5% |
| 数据库连接池 | 数据库监控 | 活跃连接 > 80% |
| Redis 内存 | Redis INFO | 使用率 > 80% |
| 磁盘空间 | 系统监控 | 使用率 > 85% |

### 6.3 日志收集

Docker Compose 环境：

```bash
# 实时查看所有日志
docker compose logs -f

# 导出日志到文件
docker compose logs api-server > api-server.log

# 配置日志驱动（docker-compose.yml）
services:
  api-server:
    logging:
      driver: "json-file"
      options:
        max-size: "50m"
        max-file: "10"
```

---

## 7. 备份与恢复

### 7.1 数据库备份

```bash
# Docker 环境备份
docker compose exec postgres pg_dump -U maas maas_router > backup_$(date +%Y%m%d).sql

# 压缩备份
docker compose exec postgres pg_dump -U maas maas_router | gzip > backup_$(date +%Y%m%d).sql.gz
```

### 7.2 数据库恢复

```bash
# Docker 环境恢复
docker compose exec -T postgres psql -U maas maas_router < backup_20260505.sql

# 压缩备份恢复
gunzip -c backup_20260505.sql.gz | docker compose exec -T postgres psql -U maas maas_router
```

### 7.3 定时备份脚本

```bash
#!/bin/bash
# /etc/cron.daily/maas-backup
BACKUP_DIR=/data/backups/maas-router
mkdir -p $BACKUP_DIR
cd /path/to/maas-router
docker compose exec -T postgres pg_dump -U maas maas_router | gzip > $BACKUP_DIR/maas_$(date +%Y%m%d).sql.gz
# 保留最近 30 天
find $BACKUP_DIR -name "*.sql.gz" -mtime +30 -delete
```
