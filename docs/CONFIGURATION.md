# 配置参考

> MaaS-Router 完整配置项说明。所有配置通过环境变量注入。

---

## 目录

1. [配置方式](#1-配置方式)
2. [完整配置列表](#2-完整配置列表)
3. [环境配置模板](#3-环境配置模板)

---

## 1. 配置方式

MaaS-Router 使用 **Pydantic Settings** 管理配置，支持以下优先级（从高到低）：

1. 环境变量（最高优先级）
2. `.env` 文件
3. 代码默认值（最低优先级）

配置类定义在 `backend/app/core/config.py`：

```python
class Settings(BaseSettings):
    environment: str = "development"

    class Config:
        env_file = ".env"
```

---

## 2. 完整配置列表

### 环境

| 变量 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `ENVIRONMENT` | string | `development` | `development` / `staging` / `production` |

开发环境使用宽松的安全策略，生产环境会进行安全校验。

### 服务

| 变量 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `SERVICE_NAME` | string | `api-server` | 服务标识（`api-server` / `admin-server`） |
| `SERVICE_PORT` | int | `8001` | 服务监听端口 |
| `DEBUG` | bool | `true` | 是否开启调试模式 |

### 数据库

| 变量 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `DATABASE_URL` | string | `postgresql+asyncpg://maas:maas_dev_2026@localhost:5432/maas_router` | 主数据库连接 |
| `DATABASE_URL_READ` | string | `null` | 只读副本连接（可选） |
| `DB_POOL_SIZE` | int | `20` | 连接池大小 |
| `DB_MAX_OVERFLOW` | int | `10` | 最大溢出连接数 |
| `DB_POOL_RECYCLE` | int | `3600` | 连接回收时间（秒） |
| `DB_POOL_TIMEOUT` | int | `10` | 获取连接超时（秒） |

**生产环境建议**：

```bash
DB_POOL_SIZE=50
DB_MAX_OVERFLOW=20
DB_POOL_RECYCLE=1800
```

### Redis

| 变量 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `REDIS_URL` | string | `redis://localhost:6379/0` | Redis 连接地址 |

### JWT 认证

| 变量 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `JWT_SECRET_KEY` | string | `maas-router-dev-secret-key-change-in-production` | JWT 签名密钥 |
| `JWT_ALGORITHM` | string | `HS256` | 签名算法 |
| `JWT_EXPIRE_MINUTES` | int | `60` | Token 过期时间（分钟） |

> ⚠️ **生产环境必须更换 JWT_SECRET_KEY**。生成方式：`openssl rand -hex 32`

### API Key

| 变量 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `API_KEY_PREFIX` | string | `sk-mr-` | API Key 前缀 |
| `API_KEY_LENGTH` | int | `48` | 密钥随机部分长度 |

### CORS

| 变量 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `CORS_ORIGINS` | string | `http://localhost:5173,http://localhost:3000` | 允许的跨域来源（逗号分隔） |

**生产环境示例**：

```bash
CORS_ORIGINS=https://app.your-domain.com,https://admin.your-domain.com
```

### 速率限制

| 变量 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `RATE_LIMIT_RPM_FREE` | int | `100` | 免费用户每分钟请求数 |
| `RATE_LIMIT_TPM_FREE` | int | `10000` | 免费用户每分钟 Token 数 |
| `RATE_LIMIT_RPM_PRO` | int | `1000` | Pro 用户每分钟请求数 |
| `RATE_LIMIT_TPM_PRO` | int | `100000` | Pro 用户每分钟 Token 数 |

### 路由

| 变量 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `DEFAULT_COMPLEXITY_THRESHOLD` | int | `5` | 默认复杂度阈值 |
| `ROUTER_CACHE_TTL` | int | `3600` | 路由缓存过期时间（秒） |

---

## 3. 环境配置模板

### 开发环境（.env）

```bash
# backend/.env
ENVIRONMENT=development
DEBUG=true

# 数据库
DATABASE_URL=postgresql+asyncpg://maas:maas_dev_2026@localhost:5432/maas_router
REDIS_URL=redis://localhost:6379/0

# 安全（开发环境可用默认值）
JWT_SECRET_KEY=maas-router-dev-secret-key-change-in-production
JWT_EXPIRE_MINUTES=1440

# CORS
CORS_ORIGINS=http://localhost:5173,http://localhost:3000

# 服务端口
API_SERVER_PORT=8001
ADMIN_SERVER_PORT=8005
```

### 预发布环境（.env.staging）

```bash
ENVIRONMENT=staging
DEBUG=false

DATABASE_URL=postgresql+asyncpg://maas:staging-password@staging-db:5432/maas_router
REDIS_URL=redis://:staging-redis-pass@staging-redis:6379/0

JWT_SECRET_KEY=<openssl rand -hex 32>
JWT_EXPIRE_MINUTES=120

CORS_ORIGINS=https://staging.your-domain.com
```

### 生产环境（.env.production）

```bash
ENVIRONMENT=production
DEBUG=false

DATABASE_URL=postgresql+asyncpg://maas:<strong-password>@prod-db:5432/maas_router
REDIS_URL=redis://:<strong-password>@prod-redis:6379/0

JWT_SECRET_KEY=<openssl rand -hex 32>
JWT_ALGORITHM=HS256
JWT_EXPIRE_MINUTES=60

API_KEY_PREFIX=sk-mr-

CORS_ORIGINS=https://api.your-domain.com,https://admin.your-domain.com

DB_POOL_SIZE=50
DB_MAX_OVERFLOW=20

RATE_LIMIT_RPM_FREE=60
RATE_LIMIT_TPM_FREE=5000
RATE_LIMIT_RPM_PRO=500
RATE_LIMIT_TPM_PRO=50000
```

---

## 安全校验

项目启动时自动执行安全校验（`config.validate_security()`）：

- 如果 `ENVIRONMENT` 为 `staging` 或 `production`，且 `JWT_SECRET_KEY` 使用默认值 → **启动失败**
- 如果 `ENVIRONMENT` 为 `staging` 或 `production`，且 `CORS_ORIGINS` 包含 localhost → **打印警告**
