# MaaS-Router API 参考

> 版本：1.0.0 | 基础 URL：`http://localhost:8001` (API) / `http://localhost:8005` (Admin)

---

## 目录

- [API Server（用户端）](#api-server用户端)
  - [模型列表](#1-获取模型列表)
  - [Chat Completion](#2-chat-completion)
  - [API Key 管理](#3-api-key-管理)
  - [余额查询](#4-余额查询)
  - [用量统计](#5-用量统计)
  - [路由决策记录](#6-路由决策记录)
- [Admin Server（管理后台）](#admin-server-api)
  - [认证](#认证)
  - [仪表盘](#仪表盘)
  - [用户管理](#用户管理)
  - [模型管理](#模型管理)
  - [计费管理](#计费管理)
  - [运维监控](#运维监控)
  - [系统设置](#系统设置)
- [通用说明](#通用说明)

---

## API Server（用户端）

**Base URL**: `http://localhost:8001`

所有用户端 API 遵循 OpenAI 兼容格式。认证方式支持 API Key 和 JWT Token 两种。

### 认证

```
Authorization: Bearer <your-api-key>
# 或
Authorization: Bearer <your-jwt-token>
```

API Key 格式：`sk-mr-<8位hex>-<48位hex>`

---

### 1. 获取模型列表

```http
GET /v1/models
```

列出所有可用的 AI 模型（仅返回状态为 `active` 的模型）。

**响应示例**：

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4o",
      "object": "model",
      "created": 1714608000,
      "owned_by": "OpenAI",
      "provider": { "id": "openai", "name": "OpenAI" },
      "context_window": 128000,
      "pricing": { "input": 5.0, "output": 15.0 },
      "tags": ["chat", "reasoning", "multimodal"],
      "features": ["vision", "function_calling", "json_mode"],
      "is_recommended": true
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 模型 ID，用于 Chat Completion |
| `owned_by` | string | 供应商名称 |
| `context_window` | int | 上下文窗口大小（tokens） |
| `pricing.input` | float | 输入价格（CRED / 百万 tokens） |
| `pricing.output` | float | 输出价格（CRED / 百万 tokens） |
| `tags` | string[] | 模型标签 |
| `features` | string[] | 支持的特性列表 |
| `is_recommended` | bool | 是否推荐模型 |

---

### 1.1 获取单个模型

```http
GET /v1/models/{model_id}
```

**路径参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `model_id` | string | 模型 ID，如 `gpt-4o`、`deepseek-v3` |

**错误响应**：

```json
{
  "detail": "Model 'unknown-model' not found"
}
```
> HTTP 状态码：`404`

---

### 2. Chat Completion

```http
POST /v1/chat/completions
```

OpenAI 兼容的 Chat Completion 接口。支持 `auto` 模式自动路由。

**请求体**：

```json
{
  "model": "auto",
  "messages": [
    { "role": "system", "content": "你是一个有帮助的助手" },
    { "role": "user", "content": "解释一下什么是 MaaS 架构" }
  ],
  "temperature": 0.7,
  "max_tokens": 1024,
  "stream": false
}
```

| 参数 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `model` | string | ✅ | — | 模型 ID，或 `"auto"` 启用自动路由 |
| `messages` | array | ✅ | — | 消息列表（role + content） |
| `temperature` | float | ❌ | 0.7 | 采样温度（0-2） |
| `max_tokens` | int | ❌ | 1024 | 最大生成 token 数 |
| `stream` | bool | ❌ | false | 是否流式输出（SSE） |

**非流式响应**：

```json
{
  "id": "chatcmpl-a1b2c3d4e5f6g7h8i9j0k1l2",
  "object": "chat.completion",
  "created": 1714608000,
  "model": "deepseek-v4-self",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "MaaS（Model as a Service）架构是..."
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 45,
    "completion_tokens": 128,
    "total_tokens": 173
  },
  "router_decision": {
    "complexity_score": 3.2,
    "route_reason": "简单查询，路由至自建 DeepSeek-V4",
    "confidence": 0.32,
    "cost_cred": 0.000256
  }
}
```

**流式响应（SSE）**：

设置 `"stream": true` 后，响应为 Server-Sent Events 流：

```
data: {"id":"chatcmpl-...","object":"chat.completion.chunk","choices":[{"delta":{"content":"MaaS "},"index":0}]}

data: {"id":"chatcmpl-...","object":"chat.completion.chunk","choices":[{"delta":{"content":"架构 "},"index":0}]}

data: [DONE]
```

流式响应包含以下自定义 headers：

| Header | 说明 |
|--------|------|
| `X-Request-ID` | 请求唯一标识 |
| `X-Router-Decision` | JSON 格式的路由决策详情 |

**错误响应**：

| HTTP 状态码 | detail | 说明 |
|------------|--------|------|
| `401` | Missing credentials | 未提供认证信息 |
| `401` | Invalid API key | API Key 无效或已撤销 |
| `402` | Insufficient balance | CRED 余额不足 |
| `404` | Model 'xxx' not found | 指定的模型不存在 |

---

### 3. API Key 管理

#### 3.1 列出 API Keys

```http
GET /v1/keys
```

**响应**：

```json
{
  "object": "list",
  "data": [{
    "id": "key_uuid",
    "name": "My App Key",
    "prefix": "sk-mr-a1b2c3d4",
    "status": "active",
    "last_used_at": "2026-05-05T12:00:00Z",
    "created_at": "2026-05-01T08:00:00Z",
    "rate_limit_rpm": 100,
    "rate_limit_tpm": 10000
  }]
}
```

#### 3.2 创建 API Key

```http
POST /v1/keys
```

**请求体**：

```json
{
  "name": "My New Key"
}
```

**响应**：

```json
{
  "id": "key_uuid",
  "name": "My New Key",
  "key": "sk-mr-a1b2c3d4-e5f6g7h8...",
  "prefix": "sk-mr-a1b2c3d4",
  "status": "active",
  "created_at": "2026-05-05T12:00:00Z"
}
```

> ⚠️ **重要**：`key` 字段仅在创建时返回一次，之后不可获取。请妥善保存。

#### 3.3 撤销 API Key

```http
DELETE /v1/keys/{key_id}
```

**响应**：

```json
{
  "id": "key_uuid",
  "status": "revoked"
}
```

---

### 4. 余额查询

```http
GET /v1/balance
```

**响应**：

```json
{
  "cred_balance": 100.5,
  "usd_balance": 10.05,
  "frozen_cred": 0
}
```

| 字段 | 说明 |
|------|------|
| `cred_balance` | 可用 CRED 余额 |
| `usd_balance` | 等值美元余额 |
| `frozen_cred` | 冻结中的 CRED（预留，当前始终为 0） |

---

### 5. 用量统计

```http
GET /v1/usage/summary
```

**响应**：

```json
{
  "total_requests": 1523,
  "total_tokens": 456789,
  "total_cost_cred": 1.234567
}
```

---

### 6. 路由决策记录

```http
GET /v1/router/decisions?limit=20
```

**查询参数**：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `limit` | int | 20 | 返回条数 |

**响应**：

```json
{
  "object": "list",
  "data": [{
    "request_id": "chatcmpl-xxx",
    "model_id": "gpt-4o",
    "provider_id": "openai",
    "complexity_score": 8.5,
    "route_decision": {
      "reason": "高复杂度，路由至 GPT-4o",
      "model": "gpt-4o",
      "provider": "openai"
    },
    "prompt_tokens": 150,
    "completion_tokens": 200,
    "latency_ms": 350,
    "created_at": "2026-05-05T12:00:00Z"
  }]
}
```

---

### 健康检查

```http
GET /health
```

**响应**：

```json
{
  "status": "ok",
  "service": "api-server"
}
```

---

## Admin Server API

**Base URL**: `http://localhost:8005`
**API Prefix**: `/api/admin/v1`

### 认证

管理后台使用 JWT Token 认证。

```
Authorization: Bearer <jwt-token>
```

#### 登录

```http
POST /api/admin/v1/auth/login
```

**请求体**：

```json
{
  "email": "admin@maas-router.com",
  "password": "admin123"
}
```

**响应**：

```json
{
  "access_token": "eyJhbGciOiJI...",
  "token_type": "bearer",
  "user": {
    "id": "user_uuid",
    "email": "admin@maas-router.com",
    "display_name": "Admin",
    "role": "admin"
  }
}
```

**错误**：`401` — 邮箱或密码错误

---

### 仪表盘

#### 概览统计

```http
GET /api/admin/v1/dashboard/overview
```

**响应**：

```json
{
  "total_users": 42,
  "active_today": 15,
  "today_revenue": 1250.50,
  "monthly_revenue": 35800.00,
  "active_api_keys": 87,
  "today_requests": 3456
}
```

#### 趋势数据

```http
GET /api/admin/v1/dashboard/trends?days=7
```

用于折线图展示。返回指定天数内每日的请求量、token 消耗、费用。

#### 模型分布

```http
GET /api/admin/v1/dashboard/model-distribution
```

返回各模型的使用次数分布（饼图数据）。

#### 最近请求

```http
GET /api/admin/v1/dashboard/recent-requests?limit=10
```

返回最近的请求日志，含路由决策信息。

---

### 用户管理

| 方法 | 端点 | 说明 |
|------|------|------|
| `GET` | `/users` | 用户列表（支持搜索、分页） |
| `GET` | `/users/{id}` | 用户详情 |
| `POST` | `/users` | 创建用户 |
| `PUT` | `/users/{id}` | 更新用户 |
| `PUT` | `/users/{id}/status` | 启用/禁用用户 |
| `DELETE` | `/users/{id}` | 删除用户 |

### 模型管理

| 方法 | 端点 | 说明 |
|------|------|------|
| `GET` | `/models/providers` | 供应商列表 |
| `POST` | `/models/providers` | 添加供应商 |
| `PUT` | `/models/providers/{id}` | 更新供应商 |
| `GET` | `/models/list` | 模型列表 |
| `POST` | `/models/list` | 添加模型 |
| `PUT` | `/models/list/{id}` | 更新模型 |
| `GET` | `/models/routing-rules` | 路由规则列表 |
| `POST` | `/models/routing-rules` | 添加路由规则 |
| `PUT` | `/models/routing-rules/{id}` | 更新路由规则 |

### 计费管理

| 方法 | 端点 | 说明 |
|------|------|------|
| `GET` | `/billing/overview` | 计费概览 |
| `GET` | `/billing/transactions` | 交易记录（支持筛选） |
| `POST` | `/billing/topup` | 用户充值 |
| `GET` | `/billing/balances` | 用户余额列表 |

### 运维监控

| 方法 | 端点 | 说明 |
|------|------|------|
| `GET` | `/monitoring/health` | 服务健康状态 |
| `GET` | `/monitoring/metrics` | 实时指标 |
| `GET` | `/monitoring/alerts` | 告警列表 |
| `GET` | `/monitoring/error-logs` | 故障日志 |

### 系统设置

| 方法 | 端点 | 说明 |
|------|------|------|
| `GET` | `/settings/rate-limits` | 速率限制配置 |
| `PUT` | `/settings/rate-limits` | 更新速率限制 |
| `GET` | `/settings/pricing` | 定价配置 |
| `PUT` | `/settings/pricing` | 更新定价 |
| `GET` | `/settings/audit-logs` | 审计日志 |

### 文档生成

| 方法 | 端点 | 说明 |
|------|------|------|
| `POST` | `/documents/generate/billing` | 生成计费报表 (PDF/Excel) |
| `POST` | `/documents/generate/user-report` | 生成用户报告 (PDF/Word) |
| `POST` | `/documents/generate/ops-daily` | 生成运维日报 (PDF) |
| `POST` | `/documents/generate/audit` | 生成审计报告 (Word) |
| `POST` | `/documents/generate/data-export` | 数据导出 (Excel/CSV) |
| `GET` | `/documents/list` | 已生成文档列表 |
| `GET` | `/documents/download/{filename}` | 下载文档 |

---

## 通用说明

### HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| `200` | 成功 |
| `201` | 创建成功 |
| `400` | 请求参数错误 |
| `401` | 未认证或认证失败 |
| `402` | 余额不足 |
| `403` | 无权限 |
| `404` | 资源不存在 |
| `429` | 请求频率超限 |
| `500` | 服务器内部错误 |

### 认证错误响应格式

```json
{
  "detail": "Invalid API key"
}
```

### 日期时间格式

所有日期时间使用 ISO 8601 格式：`2026-05-05T12:00:00Z`

### 分页

当前版本暂不支持分页参数。列表接口返回全部数据（数据规模较小）。

---

> **OpenAPI 规范文件**：启动服务后访问 `http://localhost:8001/docs` 或 `http://localhost:8005/docs` 获取交互式 Swagger UI。
