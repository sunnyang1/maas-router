# MaaS-Router API 文档

本文档描述 MaaS-Router 的所有 API 接口，包括网关 API 和管理 API。

---

## 目录

- [认证方式](#认证方式)
- [网关 API](#网关-api)
- [管理 API](#管理-api)
- [错误码定义](#错误码定义)
- [请求/响应示例](#请求响应示例)

---

## 认证方式

MaaS-Router 支持两种认证方式：API Key 认证和 JWT 认证。

### API Key 认证 (网关 API)

用于调用 AI 推理接口，在请求头中携带 API Key。

```http
Authorization: Bearer {your-api-key}
```

**API Key 格式：**
- 前缀：`mr-`
- 示例：`mr-sk-abc123def456ghi789`

### JWT 认证 (管理 API)

用于访问管理后台接口，需要先登录获取 JWT Token。

```http
Authorization: Bearer {jwt-token}
```

**获取 JWT Token：**

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "your_password"
  }'
```

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 86400,
    "token_type": "Bearer"
  }
}
```

---

## 网关 API

网关 API 提供与 OpenAI API 兼容的接口，用于 AI 模型调用。

**Base URL:** `http://localhost:8080/v1`

### 1. Chat Completions

创建聊天完成请求，支持流式响应。

```http
POST /chat/completions
```

**请求头：**

| 头部 | 必填 | 说明 |
|------|------|------|
| Authorization | 是 | Bearer {api-key} |
| Content-Type | 是 | application/json |

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| model | string | 是 | 模型 ID 或 `auto` |
| messages | array | 是 | 消息列表 |
| temperature | float | 否 | 采样温度 (0-2)，默认 1 |
| max_tokens | integer | 否 | 最大生成 token 数 |
| stream | boolean | 否 | 是否流式响应，默认 false |
| top_p | float | 否 | 核采样参数 |
| frequency_penalty | float | 否 | 频率惩罚 (-2-2) |
| presence_penalty | float | 否 | 存在惩罚 (-2-2) |
| user | string | 否 | 用户标识 |

**请求示例：**

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer mr-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "auto",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "Hello, how are you?"}
    ],
    "temperature": 0.7,
    "max_tokens": 150
  }'
```

**响应示例 (非流式)：**

```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "deepseek-v4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! I'm doing well, thank you for asking. How can I assist you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 25,
    "completion_tokens": 15,
    "total_tokens": 40
  },
  "routing_info": {
    "provider": "deepseek-v4",
    "complexity_score": 0.25,
    "cost_usd": 0.00008
  }
}
```

**响应字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 请求唯一标识 |
| object | string | 对象类型 |
| created | integer | 创建时间戳 |
| model | string | 实际使用的模型 |
| choices | array | 生成结果列表 |
| usage | object | Token 使用量统计 |
| routing_info | object | 路由信息 (MaaS-Router 特有) |

### 2. 流式响应

设置 `stream: true` 启用 Server-Sent Events (SSE) 流式响应。

**请求：**

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer mr-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "auto",
    "messages": [{"role": "user", "content": "Count to 5"}],
    "stream": true
  }'
```

**响应格式 (SSE)：**

```
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"deepseek-v4","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"deepseek-v4","choices":[{"index":0,"delta":{"content":"1"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"deepseek-v4","choices":[{"index":0,"delta":{"content":", 2"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"deepseek-v4","choices":[{"index":0,"delta":{"content":", 3, 4, 5"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"deepseek-v4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

### 3. 模型列表

获取可用的模型列表。

```http
GET /models
```

**请求示例：**

```bash
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer mr-your-api-key"
```

**响应示例：**

```json
{
  "object": "list",
  "data": [
    {
      "id": "auto",
      "object": "model",
      "created": 1677610602,
      "owned_by": "maas-router",
      "description": "自动路由，根据请求复杂度选择最优模型"
    },
    {
      "id": "deepseek-v4",
      "object": "model",
      "created": 1677610602,
      "owned_by": "deepseek",
      "description": "DeepSeek-V4 自建集群"
    },
    {
      "id": "gpt-4",
      "object": "model",
      "created": 1677610602,
      "owned_by": "openai",
      "description": "OpenAI GPT-4"
    },
    {
      "id": "gpt-3.5-turbo",
      "object": "model",
      "created": 1677610602,
      "owned_by": "openai",
      "description": "OpenAI GPT-3.5 Turbo"
    },
    {
      "id": "claude-3-opus",
      "object": "model",
      "created": 1677610602,
      "owned_by": "anthropic",
      "description": "Anthropic Claude 3 Opus"
    }
  ]
}
```

### 4. Embeddings

创建文本嵌入向量。

```http
POST /embeddings
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| model | string | 是 | 嵌入模型 ID |
| input | string/array | 是 | 输入文本 |
| encoding_format | string | 否 | 编码格式，默认 float |
| dimensions | integer | 否 | 输出维度 |

**请求示例：**

```bash
curl -X POST http://localhost:8080/v1/embeddings \
  -H "Authorization: Bearer mr-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": "The quick brown fox jumps over the lazy dog"
  }'
```

**响应示例：**

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [0.0023064255, -0.009327292, ...],
      "index": 0
    }
  ],
  "model": "text-embedding-3-small",
  "usage": {
    "prompt_tokens": 9,
    "total_tokens": 9
  }
}
```

### 5. 图像生成

生成图像 (如果配置的供应商支持)。

```http
POST /images/generations
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| prompt | string | 是 | 图像描述 |
| model | string | 否 | 模型 ID |
| n | integer | 否 | 生成数量，默认 1 |
| size | string | 否 | 图像尺寸 |
| quality | string | 否 | 图像质量 |
| response_format | string | 否 | 响应格式 |

---

## 管理 API

管理 API 用于系统管理和监控，需要 JWT 认证。

**Base URL:** `http://localhost:8080/api/v1`

### 认证相关

#### 用户注册

```http
POST /auth/register
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| email | string | 是 | 邮箱地址 |
| password | string | 是 | 密码 (至少 8 位) |
| name | string | 是 | 用户名称 |

**请求示例：**

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "securepassword123",
    "name": "张三"
  }'
```

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": "usr-abc123",
    "email": "user@example.com",
    "name": "张三",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

#### 用户登录

```http
POST /auth/login
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| email | string | 是 | 邮箱地址 |
| password | string | 是 | 密码 |

#### 刷新 Token

```http
POST /auth/refresh
```

**请求头：**

```http
Authorization: Bearer {refresh-token}
```

### 用户管理

#### 获取用户信息

```http
GET /user/profile
```

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": "usr-abc123",
    "email": "user@example.com",
    "name": "张三",
    "role": "user",
    "balance": 100.50,
    "cred_balance": "500.00",
    "created_at": "2024-01-15T10:30:00Z",
    "last_login_at": "2024-01-20T08:15:00Z"
  }
}
```

#### 更新用户信息

```http
PUT /user/profile
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 用户名称 |
| avatar | string | 否 | 头像 URL |

### API Key 管理

#### 获取 API Key 列表

```http
GET /keys
```

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "keys": [
      {
        "id": "key-abc123",
        "name": "Production Key",
        "key_preview": "mr-sk-...xyz789",
        "created_at": "2024-01-15T10:30:00Z",
        "last_used_at": "2024-01-20T08:15:00Z",
        "is_active": true,
        "rate_limit": 1000
      }
    ]
  }
}
```

#### 创建 API Key

```http
POST /keys
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | Key 名称 |
| rate_limit | integer | 否 | 每分钟请求限制 |

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "key-def456",
    "name": "Development Key",
    "key": "mr-sk-newkey123456789",
    "created_at": "2024-01-20T10:30:00Z",
    "is_active": true
  }
}
```

**注意：** API Key 仅在创建时返回完整值，请妥善保存。

#### 删除 API Key

```http
DELETE /keys/{key-id}
```

### 使用记录

#### 获取使用统计

```http
GET /usage
```

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 开始日期 (YYYY-MM-DD) |
| end_date | string | 否 | 结束日期 (YYYY-MM-DD) |
| group_by | string | 否 | 分组方式 (day/model) |

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_requests": 15234,
    "total_tokens": 4567890,
    "total_cost_usd": 12.34,
    "period": {
      "start": "2024-01-01",
      "end": "2024-01-20"
    },
    "breakdown": [
      {
        "date": "2024-01-20",
        "requests": 1234,
        "input_tokens": 56789,
        "output_tokens": 23456,
        "cost_usd": 1.23,
        "model_breakdown": {
          "deepseek-v4": {"requests": 800, "cost": 0.45},
          "gpt-4": {"requests": 300, "cost": 0.65},
          "gpt-3.5-turbo": {"requests": 134, "cost": 0.13}
        }
      }
    ]
  }
}
```

#### 获取请求日志

```http
GET /usage/logs
```

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | integer | 否 | 页码，默认 1 |
| page_size | integer | 否 | 每页数量，默认 20 |
| model | string | 否 | 按模型筛选 |

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 15234,
    "page": 1,
    "page_size": 20,
    "logs": [
      {
        "id": "req-abc123",
        "timestamp": "2024-01-20T10:30:00Z",
        "model": "deepseek-v4",
        "input_tokens": 150,
        "output_tokens": 80,
        "cost_usd": 0.0023,
        "latency_ms": 450,
        "status": "success",
        "routing_info": {
          "complexity_score": 0.32,
          "provider": "deepseek-v4"
        }
      }
    ]
  }
}
```

### 管理员接口

#### 获取仪表盘统计

```http
GET /admin/dashboard/stats
```

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "overview": {
      "total_users": 1234,
      "total_requests_today": 56789,
      "total_tokens_today": 12345678,
      "revenue_today_usd": 123.45,
      "active_providers": 5
    },
    "routing_stats": {
      "deepseek-v4": {"requests": 40000, "percentage": 70.5},
      "gpt-4": {"requests": 10000, "percentage": 17.6},
      "gpt-3.5-turbo": {"requests": 6789, "percentage": 11.9}
    },
    "cost_savings": {
      "estimated_cost_without_router": 345.67,
      "actual_cost": 123.45,
      "savings_percentage": 64.3
    },
    "system_health": {
      "status": "healthy",
      "avg_latency_ms": 320,
      "error_rate": 0.001
    }
  }
}
```

#### 用户管理

```http
GET /admin/users
POST /admin/users
GET /admin/users/{user-id}
PUT /admin/users/{user-id}
DELETE /admin/users/{user-id}
```

#### 供应商管理

```http
GET /admin/providers
POST /admin/providers
PUT /admin/providers/{provider-id}
DELETE /admin/providers/{provider-id}
```

**创建供应商请求示例：**

```bash
curl -X POST http://localhost:8080/api/v1/admin/providers \
  -H "Authorization: Bearer {jwt-token}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "OpenAI",
    "type": "commercial",
    "base_url": "https://api.openai.com/v1",
    "api_key": "sk-xxxxxxxx",
    "models": ["gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"],
    "priority": 1,
    "is_active": true,
    "rate_limit": 10000
  }'
```

#### 路由规则管理

```http
GET /admin/router-rules
PUT /admin/router-rules
```

**更新路由规则：**

```bash
curl -X PUT http://localhost:8080/api/v1/admin/router-rules \
  -H "Authorization: Bearer {jwt-token}" \
  -H "Content-Type: application/json" \
  -d '{
    "simple_threshold": 0.3,
    "medium_threshold": 0.6,
    "complex_threshold": 0.8,
    "default_provider": "deepseek-v4",
    "fallback_provider": "gpt-3.5-turbo",
    "cost_optimization": true,
    "quality_optimization": false
  }'
```

### 运维监控

#### 获取系统指标

```http
GET /admin/ops/metrics
```

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "requests_per_minute": 1234,
    "avg_response_time_ms": 320,
    "p95_response_time_ms": 850,
    "p99_response_time_ms": 1500,
    "error_rate": 0.001,
    "active_connections": 456,
    "queue_depth": 12
  }
}
```

#### 获取服务健康状态

```http
GET /admin/ops/health
```

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "healthy",
    "services": {
      "database": {"status": "up", "latency_ms": 2},
      "redis": {"status": "up", "latency_ms": 1},
      "judge_agent": {"status": "up", "latency_ms": 45},
      "providers": {
        "deepseek-v4": {"status": "up", "latency_ms": 120},
        "openai": {"status": "up", "latency_ms": 280}
      }
    }
  }
}
```

---

## 错误码定义

### HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 401 | 未授权，认证失败 |
| 403 | 禁止访问，权限不足 |
| 404 | 资源不存在 |
| 429 | 请求过于频繁，触发限流 |
| 500 | 服务器内部错误 |
| 502 | 上游服务错误 |
| 503 | 服务暂时不可用 |

### 业务错误码

| 错误码 | 说明 | 处理建议 |
|--------|------|----------|
| 0 | 成功 | - |
| 10001 | 参数错误 | 检查请求参数 |
| 10002 | 认证失败 | 检查 API Key 或 Token |
| 10003 | 权限不足 | 确认账号权限 |
| 10004 | 资源不存在 | 检查资源 ID |
| 10005 | 资源已存在 | 更换唯一标识 |
| 20001 | 余额不足 | 充值账户 |
| 20002 | 超出配额 | 升级套餐或等待重置 |
| 20003 | 限流触发 | 降低请求频率 |
| 30001 | 模型不可用 | 更换模型或稍后重试 |
| 30002 | 供应商错误 | 系统自动切换供应商 |
| 30003 | 请求超时 | 检查网络或增加超时时间 |
| 30004 | 内容审核失败 | 修改请求内容 |
| 50001 | 内部错误 | 联系技术支持 |

### 错误响应格式

```json
{
  "error": {
    "code": 20001,
    "message": "Insufficient balance",
    "type": "billing_error",
    "details": {
      "current_balance": 0.50,
      "required_balance": 1.00,
      "currency": "USD"
    }
  }
}
```

---

## 请求/响应示例

### Python SDK 示例

```python
from openai import OpenAI
import os

# 初始化客户端
client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key=os.getenv("MAAS_API_KEY")
)

# 简单对话
def simple_chat():
    response = client.chat.completions.create(
        model="auto",
        messages=[
            {"role": "user", "content": "Hello!"}
        ]
    )
    print(response.choices[0].message.content)
    print(f"Used model: {response.model}")
    print(f"Cost: ${response.routing_info.cost_usd}")

# 流式对话
def stream_chat():
    stream = client.chat.completions.create(
        model="auto",
        messages=[
            {"role": "user", "content": "Write a poem about AI"}
        ],
        stream=True
    )
    
    for chunk in stream:
        if chunk.choices[0].delta.content:
            print(chunk.choices[0].delta.content, end="")

# 带上下文的对话
def context_chat():
    messages = [
        {"role": "system", "content": "You are a helpful coding assistant."},
        {"role": "user", "content": "How do I write a Python function?"},
        {"role": "assistant", "content": "Here's how to write a Python function..."},
        {"role": "user", "content": "Can you show me an example with parameters?"}
    ]
    
    response = client.chat.completions.create(
        model="auto",
        messages=messages,
        temperature=0.7,
        max_tokens=500
    )
    print(response.choices[0].message.content)

# 获取使用统计
def get_usage():
    import requests
    
    headers = {"Authorization": f"Bearer {os.getenv('MAAS_JWT_TOKEN')}"}
    response = requests.get(
        "http://localhost:8080/api/v1/usage",
        headers=headers,
        params={"start_date": "2024-01-01", "end_date": "2024-01-20"}
    )
    
    data = response.json()
    print(f"Total requests: {data['data']['total_requests']}")
    print(f"Total cost: ${data['data']['total_cost_usd']}")

if __name__ == "__main__":
    simple_chat()
```

### JavaScript/TypeScript 示例

```typescript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: process.env.MAAS_API_KEY,
});

// 异步对话
async function chatCompletion() {
  const response = await client.chat.completions.create({
    model: 'auto',
    messages: [
      { role: 'user', content: 'What is machine learning?' }
    ],
    temperature: 0.7,
  });

  console.log(response.choices[0].message.content);
  console.log('Model used:', response.model);
}

// 流式响应
async function streamCompletion() {
  const stream = await client.chat.completions.create({
    model: 'auto',
    messages: [
      { role: 'user', content: 'Tell me a story' }
    ],
    stream: true,
  });

  for await (const chunk of stream) {
    process.stdout.write(chunk.choices[0]?.delta?.content || '');
  }
}

// 使用 fetch 调用管理 API
async function getUsageStats() {
  const response = await fetch('http://localhost:8080/api/v1/usage', {
    headers: {
      'Authorization': `Bearer ${process.env.MAAS_JWT_TOKEN}`,
    },
  });

  const data = await response.json();
  console.log('Usage stats:', data.data);
}

chatCompletion();
```

### cURL 示例合集

```bash
#!/bin/bash

API_KEY="mr-your-api-key"
JWT_TOKEN="your-jwt-token"
BASE_URL="http://localhost:8080"

# 1. 测试连接
echo "=== Test Connection ==="
curl -s ${BASE_URL}/health | jq .

# 2. 获取模型列表
echo -e "\n=== List Models ==="
curl -s ${BASE_URL}/v1/models \
  -H "Authorization: Bearer ${API_KEY}" | jq .

# 3. 简单对话
echo -e "\n=== Simple Chat ==="
curl -s ${BASE_URL}/v1/chat/completions \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "auto",
    "messages": [{"role": "user", "content": "Hello!"}]
  }' | jq .

# 4. 流式对话 (使用 -N 禁用缓冲)
echo -e "\n=== Stream Chat ==="
curl -N ${BASE_URL}/v1/chat/completions \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "auto",
    "messages": [{"role": "user", "content": "Count 1 to 3"}],
    "stream": true
  }'

# 5. 获取用户信息
echo -e "\n=== User Profile ==="
curl -s ${BASE_URL}/api/v1/user/profile \
  -H "Authorization: Bearer ${JWT_TOKEN}" | jq .

# 6. 获取使用统计
echo -e "\n=== Usage Stats ==="
curl -s "${BASE_URL}/api/v1/usage?start_date=2024-01-01&end_date=2024-01-20" \
  -H "Authorization: Bearer ${JWT_TOKEN}" | jq .

# 7. 创建 API Key
echo -e "\n=== Create API Key ==="
curl -s ${BASE_URL}/api/v1/keys \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Key",
    "rate_limit": 100
  }' | jq .

# 8. 获取仪表盘统计 (管理员)
echo -e "\n=== Dashboard Stats ==="
curl -s ${BASE_URL}/api/v1/admin/dashboard/stats \
  -H "Authorization: Bearer ${JWT_TOKEN}" | jq .
```

### 错误处理示例

```python
from openai import OpenAI, APIError, RateLimitError, AuthenticationError

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="mr-your-api-key"
)

def safe_chat_completion():
    try:
        response = client.chat.completions.create(
            model="auto",
            messages=[{"role": "user", "content": "Hello!"}]
        )
        return response
        
    except AuthenticationError as e:
        print(f"Authentication failed: {e}")
        # 处理认证错误，可能需要刷新 token
        
    except RateLimitError as e:
        print(f"Rate limit exceeded: {e}")
        # 处理限流，可以重试或降低频率
        import time
        time.sleep(1)
        return safe_chat_completion()
        
    except APIError as e:
        print(f"API error: {e}")
        print(f"Error code: {e.code}")
        print(f"Error message: {e.message}")
        # 根据错误码处理
        if e.code == 20001:  # 余额不足
            print("Please recharge your account")
        elif e.code == 30001:  # 模型不可用
            print("Model temporarily unavailable, retrying...")
            time.sleep(5)
            return safe_chat_completion()
            
    except Exception as e:
        print(f"Unexpected error: {e}")
        raise
```

---

<div align="center">

**[返回首页](../README.md)** · **[快速开始](QUICKSTART.md)** · **[架构文档](ARCHITECTURE.md)**

</div>
