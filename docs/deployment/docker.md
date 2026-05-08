# MaaS-Router Docker 部署指南

本文档介绍如何使用 Docker 和 Docker Compose 部署 MaaS-Router 项目。

## 目录

- [环境要求](#环境要求)
- [单容器部署](#单容器部署)
- [Docker Compose 部署](#docker-compose-部署)
- [环境变量说明](#环境变量说明)
- [数据持久化](#数据持久化)
- [日志管理](#日志管理)
- [常见问题](#常见问题)

## 环境要求

- Docker 20.10+
- Docker Compose 2.0+
- 至少 4GB 可用内存
- 至少 20GB 可用磁盘空间

## 单容器部署

### 1. 构建后端服务镜像

```bash
# 进入后端目录
cd backend

# 构建镜像
docker build -t maas-router/backend:v1.0.0 .
```

### 2. 运行后端服务

```bash
# 运行单容器
docker run -d \
  --name maas-backend \
  -p 8080:8080 \
  -e MAAS_ROUTER_DATABASE_HOST=your-db-host \
  -e MAAS_ROUTER_DATABASE_PORT=5432 \
  -e MAAS_ROUTER_DATABASE_USER=maas_user \
  -e MAAS_ROUTER_DATABASE_PASSWORD=your-secure-password \
  -e MAAS_ROUTER_DATABASE_DATABASE=maas_router \
  -e MAAS_ROUTER_REDIS_HOST=your-redis-host \
  -e MAAS_ROUTER_REDIS_PORT=6379 \
  -e MAAS_ROUTER_JWT_SECRET=your-super-secret-jwt-key \
  -v /path/to/logs:/app/logs \
  --restart unless-stopped \
  maas-router/backend:v1.0.0
```

### 3. 构建 Judge Agent 镜像

```bash
cd judge-agent

# 构建镜像
docker build -t maas-router/judge-agent:v1.0.0 .
```

### 4. 运行 Judge Agent

```bash
docker run -d \
  --name maas-judge-agent \
  -p 8000:8000 \
  -e JUDGE_MODEL_API_URL=http://your-llm-api/v1/chat/completions \
  -e JUDGE_MODEL_NAME=Qwen/Qwen2.5-7B-Instruct \
  -e LOG_LEVEL=info \
  --restart unless-stopped \
  maas-router/judge-agent:v1.0.0
```

## Docker Compose 部署

### 1. 完整部署（推荐）

```bash
# 克隆项目
git clone https://github.com/your-org/maas-router.git
cd maas-router

# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f backend
```

### 2. 仅启动核心服务

```bash
# 启动数据库、Redis 和后端服务
docker-compose up -d postgres redis backend
```

### 3. 包含 Judge Agent 的部署

```bash
# 使用 with-judge profile 启动
docker-compose --profile with-judge up -d
```

### 4. 自定义配置

创建 `.env` 文件：

```env
# 数据库配置
POSTGRES_USER=maas_user
POSTGRES_PASSWORD=your-secure-password
POSTGRES_DB=maas_router

# Redis 配置
REDIS_PASSWORD=your-redis-password

# JWT 配置
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production

# 日志级别
LOG_LEVEL=info

# Judge Agent 配置
JUDGE_MODEL_API_URL=http://host.docker.internal:8001/v1/chat/completions
JUDGE_MODEL_NAME=Qwen/Qwen2.5-7B-Instruct
```

## 环境变量说明

### 后端服务环境变量

| 变量名 | 说明 | 默认值 | 必需 |
|--------|------|--------|------|
| `MAAS_ROUTER_DATABASE_HOST` | PostgreSQL 主机地址 | localhost | 是 |
| `MAAS_ROUTER_DATABASE_PORT` | PostgreSQL 端口 | 5432 | 是 |
| `MAAS_ROUTER_DATABASE_USER` | 数据库用户名 | maas_user | 是 |
| `MAAS_ROUTER_DATABASE_PASSWORD` | 数据库密码 | - | 是 |
| `MAAS_ROUTER_DATABASE_DATABASE` | 数据库名称 | maas_router | 是 |
| `MAAS_ROUTER_REDIS_HOST` | Redis 主机地址 | localhost | 是 |
| `MAAS_ROUTER_REDIS_PORT` | Redis 端口 | 6379 | 是 |
| `MAAS_ROUTER_REDIS_PASSWORD` | Redis 密码 | - | 否 |
| `MAAS_ROUTER_JWT_SECRET` | JWT 签名密钥 | - | 是 |
| `MAAS_ROUTER_JUDGE_AGENT_URL` | Judge Agent 服务地址 | http://judge-agent:8000 | 是 |
| `MAAS_ROUTER_LOG_LEVEL` | 日志级别 | info | 否 |

### Judge Agent 环境变量

| 变量名 | 说明 | 默认值 | 必需 |
|--------|------|--------|------|
| `JUDGE_MODEL_API_URL` | LLM API 地址 | - | 是 |
| `JUDGE_MODEL_NAME` | 模型名称 | Qwen/Qwen2.5-7B-Instruct | 是 |
| `LOG_LEVEL` | 日志级别 | info | 否 |

## 数据持久化

### 1. 数据卷配置

Docker Compose 中已配置以下数据卷：

```yaml
volumes:
  postgres_data:    # PostgreSQL 数据
  redis_data:       # Redis 数据
  prometheus_data:  # Prometheus 数据
  grafana_data:     # Grafana 数据
```

### 2. 备份数据

```bash
# 备份 PostgreSQL 数据
docker exec maas-postgres pg_dump -U maas_user maas_router > backup_$(date +%Y%m%d).sql

# 备份 Redis 数据
docker exec maas-redis redis-cli BGSAVE
docker cp maas-redis:/data/dump.rdb ./redis_backup_$(date +%Y%m%d).rdb
```

### 3. 恢复数据

```bash
# 恢复 PostgreSQL 数据
docker exec -i maas-postgres psql -U maas_user -d maas_router < backup_20240101.sql

# 恢复 Redis 数据
docker cp redis_backup_20240101.rdb maas-redis:/data/dump.rdb
docker restart maas-redis
```

## 日志管理

### 1. 查看日志

```bash
# 查看所有服务日志
docker-compose logs

# 查看特定服务日志
docker-compose logs -f backend

# 查看最近 100 行日志
docker-compose logs --tail=100 backend
```

### 2. 日志轮转

后端服务内置日志轮转功能，配置如下：

```yaml
log:
  level: "info"
  file_path: "logs/maas-router.log"
  max_size: 100      # 单个日志文件最大 100MB
  max_backups: 10    # 保留 10 个备份
  max_age: 30        # 保留 30 天
  compress: true     # 压缩旧日志
  json_format: false # 文本格式
```

### 3. 集中式日志收集

使用 Docker 日志驱动：

```yaml
services:
  backend:
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "10"
```

或使用 Fluentd：

```yaml
services:
  backend:
    logging:
      driver: "fluentd"
      options:
        fluentd-address: localhost:24224
        tag: docker.maas-router
```

## 常见问题

### 1. 端口冲突

```bash
# 检查端口占用
sudo lsof -i :8080

# 修改 docker-compose.yml 中的端口映射
ports:
  - "8081:8080"  # 将主机端口改为 8081
```

### 2. 数据库连接失败

```bash
# 检查数据库健康状态
docker-compose ps
docker-compose logs postgres

# 手动初始化数据库
docker-compose exec postgres psql -U maas_user -d maas_router -f /docker-entrypoint-initdb.d/init.sql
```

### 3. 内存不足

```bash
# 查看容器内存使用
docker stats

# 限制容器内存
docker run -m 512m --memory-swap 1g maas-router/backend:v1.0.0
```

### 4. 时区问题

```bash
# 在 docker-compose.yml 中设置时区
services:
  backend:
    environment:
      - TZ=Asia/Shanghai
    volumes:
      - /etc/localtime:/etc/localtime:ro
```

### 5. 网络问题

```bash
# 检查网络配置
docker network ls
docker network inspect maas-router_maas-network

# 重新创建网络
docker-compose down
docker network rm maas-router_maas-network
docker-compose up -d
```

## 生产环境建议

1. **使用外部数据库和 Redis**：生产环境建议使用托管的数据库和缓存服务
2. **配置 HTTPS**：使用反向代理（如 Nginx、Traefik）配置 SSL/TLS
3. **资源限制**：为每个服务设置 CPU 和内存限制
4. **健康检查**：配置健康检查和自动重启
5. **监控告警**：集成 Prometheus 和 Grafana 进行监控

## 升级指南

```bash
# 1. 拉取最新代码
git pull origin main

# 2. 重新构建镜像
docker-compose build

# 3. 停止并移除旧容器
docker-compose down

# 4. 启动新容器
docker-compose up -d

# 5. 验证升级
docker-compose ps
docker-compose logs -f backend
```
