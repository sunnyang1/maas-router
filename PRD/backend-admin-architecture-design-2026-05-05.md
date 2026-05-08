# MaaS-Router 后端 & 管理平台架构设计

**日期**：2026-05-05
**版本**：v1.0
**状态**：初稿

---

## 目录

1. [系统总览](#1-系统总览)
2. [后端微服务架构](#2-后端微服务架构)
3. [API 网关层设计](#3-api-网关层设计)
4. [智能路由引擎设计](#4-智能路由引擎设计)
5. [计费与结算系统](#5-计费与结算系统)
6. [数据架构设计](#6-数据架构设计)
7. [管理平台设计](#7-管理平台设计)
8. [技术栈选型](#8-技术栈选型)
9. [部署架构](#9-部署架构)
10. [安全设计](#10-安全设计)

---

## 1. 系统总览

### 1.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          MaaS-Router 全栈架构                                │
└─────────────────────────────────────────────────────────────────────────────┘

                          ┌──────────────────────┐
                          │   用户端 / 客户端      │
                          │  ┌────────┐ ┌──────┐ │
                          │  │Web 前端 │ │ SDK  │ │
                          │  └────────┘ └──────┘ │
                          └──────────┬───────────┘
                                     │ HTTPS / WSS
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           负载均衡 (Nginx / ALB)                              │
│                    限流 / 认证 / TLS Termination                              │
└────────────────────────────────────┬────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         API 网关层 (Kong / 自研)                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐     │
│  │ 认证鉴权  │ │ 速率限制  │ │ 请求路由  │ │ 日志记录  │ │ 指标采集      │     │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────────┘     │
└────────────────────────────────────┬────────────────────────────────────────┘
                                     │
          ┌──────────────────────────┼──────────────────────────┐
          │                          │                          │
          ▼                          ▼                          ▼
┌─────────────────┐    ┌─────────────────────┐    ┌─────────────────────────┐
│ 统一 API 服务    │    │   智能路由引擎        │    │   管理平台服务           │
│ (api-server)    │    │   (router-engine)    │    │   (admin-server)        │
│                 │    │                      │    │                          │
│ • /v1/chat      │    │ • Judge Agent        │    │ • 用户管理 CRUD          │
│ • /v1/models    │◄──▶│ • 复杂度评分           │    │ • API Key 管理           │
│ • /v1/embeddings│    │ • 路由决策             │    │ • 模型/供应商管理         │
│ • SSE Stream    │    │ • Failover 切换        │    │ • 计费财务管理           │
│ • Key 验证      │    │ • 缓存决策             │    │ • 运维监控面板           │
└────────┬────────┘    └──────────┬──────────┘    └────────────┬────────────┘
         │                        │                             │
         │                        │                             │
         ▼                        ▼                             ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              服务网格 / 消息队列                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐ │
│  │  Kafka   │  │  Redis   │  │  gRPC    │  │  NATS    │  │  RabbitMQ    │ │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
         │                        │                             │
         ▼                        ▼                             ▼
┌─────────────────┐    ┌─────────────────────┐    ┌─────────────────────────┐
│   推理执行层      │    │   计费结算服务        │    │   数据持久层             │
│ (inference-exec) │    │   (billing-svc)      │    │                         │
│                 │    │                      │    │  ┌───────────────────┐  │
│ • 自建 DS-V4    │    │ • 实时 Token 计费     │    │  │  PostgreSQL       │  │
│ • DeepSeek API  │    │ • $CRED 余额管理      │    │  │  (主数据库)        │  │
│ • OpenAI API    │    │ • 链上结算            │    │  └───────────────────┘  │
│ • Azure OpenAI  │    │ • 财务报表            │    │  ┌───────────────────┐  │
│ • Anthropic     │    │                      │    │  │  Redis             │  │
│ • Google AI     │    │                      │    │  │  (缓存/实时计数)    │  │
└─────────────────┘    └──────────────────────┘    │  └───────────────────┘  │
                                                    │  ┌───────────────────┐  │
                                                    │  │  ClickHouse       │  │
                                                    │  │  (分析/时序)       │  │
                                                    │  └───────────────────┘  │
                                                    └─────────────────────────┘
```

### 1.2 微服务划分

| 服务名称 | 职责 | 端口 | 语言 |
|---------|------|------|------|
| **api-gateway** | 统一入口，认证鉴权，限流，日志 | 443/80 | Go/Rust |
| **api-server** | OpenAI 兼容 API 实现，SSE 流式 | 8001 | Python (FastAPI) |
| **router-engine** | Judge Agent 复杂度评分，路由决策 | 8002 | Python |
| **inference-exec** | 模型推理执行，供应商适配 | 8003 | Python |
| **billing-svc** | 实时计费，$CRED 管理，财务报表 | 8004 | Go |
| **admin-server** | 管理平台 API，运营后台 | 8005 | Go |
| **auth-svc** | 统一认证服务，JWT/Session | 8006 | Go |
| **notification-svc** | 邮件/短信/Slack 通知 | 8007 | Go |

---

## 2. 后端微服务架构

### 2.1 api-gateway（API 网关）

```
┌──────────────────────────────────────────────────────────────┐
│                        API Gateway                            │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐ │
│  │ 认证中间件 │──▶│ 限流中间件 │──▶│ 路由中间件 │──▶│ 日志中间件 │ │
│  └──────────┘   └──────────┘   └──────────┘   └──────────┘ │
│                                                               │
│  认证策略：                                                    │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ • Bearer Token (API Key)  — 优先，用户 API 调用           │ │
│  │ • JWT Token              — 管理平台登录态                │ │
│  │ • mTLS                   — 服务间通信                    │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
│  限流策略：                                                    │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ Free 用户:     100 RPM  /  10K TPM                       │ │
│  │ Pro 用户:     1000 RPM  / 100K TPM                       │ │
│  │ Enterprise:   10000 RPM / 1M TPM (可配置)                 │ │
│  │ 超限返回:     429 Too Many Requests                      │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

### 2.2 api-server（统一 API 服务）

**OpenAI-Compatible Endpoints：**

| 端点 | 方法 | 说明 | 流式 |
|------|------|------|------|
| `/v1/chat/completions` | POST | 聊天补全 | ✅ SSE |
| `/v1/completions` | POST | 文本补全 | ✅ SSE |
| `/v1/embeddings` | POST | 文本嵌入 | ❌ |
| `/v1/models` | GET | 模型列表 | ❌ |
| `/v1/models/{model}` | GET | 模型详情 | ❌ |

**MaaS-Router 独有 Endpoints：**

| 端点 | 方法 | 说明 |
|------|------|------|
| `/v1/router/decision` | GET | 查看最近请求的路由决策 |
| `/v1/router/decision/{request_id}` | GET | 查看特定请求的路由决策 |
| `/v1/router/rules` | GET/POST/PUT/DELETE | 自定义路由规则管理 |
| `/v1/usage/summary` | GET | 用量概览（当日/周/月） |
| `/v1/usage/details` | GET | 用量明细列表 |
| `/v1/keys` | GET/POST/DELETE | API Key 管理 |
| `/v1/balance` | GET | $CRED 余额查询 |

**请求处理流程：**

```
Client Request
     │
     ▼
[Gateway: API Key 验证]
     │
     ▼
[api-server: 请求解析与参数验证]
     │
     ▼
[api-server: 用户余额检查] ──→ 余额不足 → 返回 402 Payment Required
     │
     ▼
[router-engine: 复杂度评分] ← REST/gRPC 调用
     │
     ▼
[router-engine: 路由决策] ──→ 返回目标 Provider + Model
     │
     ▼
[api-server: 转发请求至目标 Provider]
     │
     ├──→ Stream Mode: SSE 逐块转发
     │
     └──→ Normal Mode: 等待完整响应
     │
     ▼
[billing-svc: Token 计数 + 费用扣减] ← 异步事件
     │
     ▼
[api-server: 返回响应至客户端]
```

---

## 3. 智能路由引擎设计

### 3.1 Judge Agent 架构

```
┌──────────────────────────────────────────────────────────────┐
│                     Judge Agent 路由引擎                      │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  输入请求                                                     │
│     │                                                        │
│     ▼                                                        │
│  ┌────────────────────────────────────────────────────────┐ │
│  │               特征提取 (Feature Extractor)              │ │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │ │
│  │  │Prompt长度 │ │Token数   │ │语言检测   │ │意图分类   │  │ │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘  │ │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │ │
│  │  │格式复杂度 │ │推理需求   │ │领域标签   │ │历史模式   │  │ │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘  │ │
│  └────────────────────────────────────────────────────────┘ │
│     │                                                        │
│     ▼                                                        │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              复杂度评分模型 (Complexity Scorer)          │ │
│  │                                                         │ │
│  │   Qwen2.5-7B / Llama-3.1-8B (轻量微调)                  │ │
│  │   输出: float 1-10 (复杂度分数)                          │ │
│  │   延迟目标: < 50ms (微秒级)                              │ │
│  │                                                         │ │
│  └────────────────────────────────────────────────────────┘ │
│     │                                                        │
│     ▼                                                        │
│  ┌────────────────────────────────────────────────────────┐ │
│  │                  路由决策引擎 (Router)                   │ │
│  │                                                         │ │
│  │  ┌─────────────────────────────────────────────────┐   │ │
│  │  │ 决策矩阵：                                        │   │ │
│  │  │                                                  │   │ │
│  │  │ 复杂度 1-3  →  自建层 (缓存优先)  $0.3/1M        │   │ │
│  │  │ 复杂度 4-6  →  自建层 (推理)      $0.5/1M        │   │ │
│  │  │ 复杂度 7-8  →  DeepSeek API      $1.0/1M        │   │ │
│  │  │ 复杂度 9-10 →  GPT-4/Claude      $15/1M         │   │ │
│  │  └─────────────────────────────────────────────────┘   │ │
│  │                                                         │ │
│  │  Fallback 链（故障时按序尝试）：                          │ │
│  │  自建层 → DeepSeek API → Azure OpenAI → OpenAI          │ │
│  │                                                         │ │
│  └────────────────────────────────────────────────────────┘ │
│     │                                                        │
│     ▼                                                        │
│  输出: { provider, model, route_reason, confidence }         │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

### 3.2 路由规则配置模型

```yaml
# 用户自定义路由规则示例
rules:
  - name: "代码生成走 DeepSeek-Coder"
    condition:
      type: "tag"
      value: "code"
    action:
      provider: "self-hosted"
      model: "deepseek-coder"
    priority: 100

  - name: "高复杂度走 GPT-4"
    condition:
      type: "complexity_range"
      min: 8
      max: 10
    action:
      provider: "openai"
      model: "gpt-4o"
    priority: 90

  - name: "简单任务走缓存"
    condition:
      type: "complexity_range"
      max: 3
    action:
      cache_first: true
      provider: "self-hosted"
    priority: 80
```

---

## 4. 计费与结算系统

### 4.1 双层计费架构

```
┌──────────────────────────────────────────────────────────────┐
│                      MaaS-Router 计费架构                     │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                    链下层 (实时)                         │ │
│  │                                                         │ │
│  │  ┌──────────┐     ┌──────────┐     ┌──────────┐       │ │
│  │  │请求完成事件│────▶│Token计数  │────▶│费用计算   │       │ │
│  │  │(Kafka)   │     │(Redis)   │     │(billing) │       │ │
│  │  └──────────┘     └──────────┘     └─────┬────┘       │ │
│  │                                          │             │ │
│  │                                          ▼             │ │
│  │                                    ┌──────────┐       │ │
│  │                                    │余额扣减   │       │ │
│  │                                    │(Redis +  │       │ │
│  │                                    │PostgreSQL│       │ │
│  │                                    │双写)     │       │ │
│  │                                    └──────────┘       │ │
│  └─────────────────────────────────────────────────────────┘ │
│                            │                                  │
│                            ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                    链上层 (每日)                         │ │
│  │                                                         │ │
│  │  ┌──────────┐     ┌──────────┐     ┌──────────┐       │ │
│  │  │每日汇总   │────▶│Merkle树  │────▶│L2 结算   │       │ │
│  │  │(UTC 0:00)│     │生成      │     │(Polygon)│       │ │
│  │  └──────────┘     └──────────┘     └──────────┘       │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

### 4.2 定价模型

| 服务 | 计费方式 | 单价 |
|------|---------|------|
| 路由层服务费 | 按 token 抽成 | 总费用的 3-5% |
| 自建层（DS-V4）| $0.50-0.80/1M tokens | DeepSeek 直连的 50-80% |
| 商业 API 转发 | 成本价 + 3% | 按供应商实时价格 |
| $CRED 持有者折扣 | 7 折优惠 | 持有 ≥ 1000 $CRED |

---

## 5. 数据架构设计

### 5.1 核心数据模型 (PostgreSQL)

```sql
-- ============================================
-- 用户与账户
-- ============================================

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    display_name    VARCHAR(100),
    avatar_url      TEXT,
    
    -- 账户状态
    status          VARCHAR(20) DEFAULT 'active',  -- active, suspended, deleted
    email_verified  BOOLEAN DEFAULT FALSE,
    
    -- 订阅
    plan_id         VARCHAR(20) DEFAULT 'free',    -- free, pro, enterprise
    
    -- 元数据
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    last_login_at   TIMESTAMPTZ
);

-- ============================================
-- 团队
-- ============================================

CREATE TABLE teams (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    owner_id        UUID REFERENCES users(id),
    status          VARCHAR(20) DEFAULT 'active',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE team_members (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID REFERENCES teams(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES users(id) ON DELETE CASCADE,
    role            VARCHAR(20) DEFAULT 'member',  -- owner, admin, member, viewer
    joined_at       TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(team_id, user_id)
);

-- ============================================
-- API 密钥
-- ============================================

CREATE TABLE api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id) ON DELETE CASCADE,
    team_id         UUID REFERENCES teams(id),
    name            VARCHAR(100) NOT NULL,
    key_hash        VARCHAR(255) UNIQUE NOT NULL,
    key_prefix      VARCHAR(20) NOT NULL,          -- "sk-mr-"
    status          VARCHAR(20) DEFAULT 'active',  -- active, revoked
    rate_limit_rpm  INTEGER DEFAULT 100,
    rate_limit_tpm  INTEGER DEFAULT 10000,
    last_used_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 模型与供应商
-- ============================================

CREATE TABLE providers (
    id              VARCHAR(50) PRIMARY KEY,       -- "openai", "deepseek", "self-hosted"
    name            VARCHAR(100) NOT NULL,
    logo_url        TEXT,
    description     TEXT,
    api_base_url    TEXT,
    status          VARCHAR(20) DEFAULT 'active',  -- active, degraded, offline
    config          JSONB DEFAULT '{}',            -- 供应商特定配置
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE models (
    id              VARCHAR(100) PRIMARY KEY,      -- "gpt-4o", "deepseek-v4"
    provider_id     VARCHAR(50) REFERENCES providers(id),
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    tags            TEXT[],
    context_window  INTEGER,
    input_price     DECIMAL(10,6),                -- $/1M tokens
    output_price    DECIMAL(10,6),
    features        TEXT[],
    status          VARCHAR(20) DEFAULT 'active',
    popularity      INTEGER DEFAULT 0,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 计费与交易
-- ============================================

CREATE TABLE balances (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id) UNIQUE,
    team_id         UUID REFERENCES teams(id),
    cred_balance    DECIMAL(20,6) DEFAULT 0,       -- $CRED 余额
    usd_balance     DECIMAL(20,6) DEFAULT 0,       -- USD 余额（法币通道）
    frozen_cred     DECIMAL(20,6) DEFAULT 0,       -- 冻结额度
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE transactions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id),
    team_id         UUID REFERENCES teams(id),
    api_key_id      UUID REFERENCES api_keys(id),
    
    -- 交易类型
    type            VARCHAR(20) NOT NULL,          -- usage, topup, refund, bonus
    
    -- 用量详情
    request_id      VARCHAR(100),
    model_id        VARCHAR(100),
    provider_id     VARCHAR(50),
    prompt_tokens   INTEGER DEFAULT 0,
    completion_tokens INTEGER DEFAULT 0,
    total_tokens    INTEGER DEFAULT 0,
    
    -- 费用
    amount          DECIMAL(20,6) NOT NULL,         -- 正数为充值，负数为消费
    currency        VARCHAR(10) DEFAULT 'CRED',
    unit_price      DECIMAL(10,6),
    
    -- 路由决策
    route_reason    TEXT,
    route_confidence DECIMAL(5,4),
    
    -- 状态
    status          VARCHAR(20) DEFAULT 'completed', -- pending, completed, failed
    
    -- 元数据
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 请求日志（ClickHouse 存储，此为 PG 热数据缓存）
-- ============================================

CREATE TABLE request_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id      VARCHAR(100) UNIQUE,
    user_id         UUID,
    api_key_id      UUID,
    model_id        VARCHAR(100),
    provider_id     VARCHAR(50),
    
    -- 请求详情
    method          VARCHAR(10),
    endpoint        VARCHAR(200),
    status_code     INTEGER,
    
    -- 性能
    latency_ms      INTEGER,
    first_token_ms  INTEGER,
    
    -- Token
    prompt_tokens   INTEGER DEFAULT 0,
    completion_tokens INTEGER DEFAULT 0,
    
    -- 路由
    complexity_score DECIMAL(5,4),
    route_decision  JSONB,
    
    -- 错误
    error_code      VARCHAR(50),
    error_message   TEXT,
    
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_request_logs_user_id ON request_logs(user_id, created_at DESC);
CREATE INDEX idx_request_logs_created ON request_logs(created_at DESC);

-- ============================================
-- 路由规则
-- ============================================

CREATE TABLE routing_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id),
    team_id         UUID REFERENCES teams(id),
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    priority        INTEGER DEFAULT 0,
    condition       JSONB NOT NULL,               -- 匹配条件
    action          JSONB NOT NULL,               -- 路由动作
    status          VARCHAR(20) DEFAULT 'active',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 系统配置
-- ============================================

CREATE TABLE system_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key             VARCHAR(100) UNIQUE NOT NULL,
    value           JSONB NOT NULL,
    description     TEXT,
    updated_by      UUID REFERENCES users(id),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 操作审计日志
-- ============================================

CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID,
    action          VARCHAR(100) NOT NULL,
    resource_type   VARCHAR(50),
    resource_id     VARCHAR(100),
    old_value       JSONB,
    new_value       JSONB,
    ip_address      INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user ON audit_logs(user_id, created_at DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
```

### 5.2 Redis 数据结构

```
# 实时速率限制
ratelimit:{api_key}:minute     → counter (60s TTL)
ratelimit:{api_key}:token      → counter (60s TTL)

# 实时余额（热数据）
balance:{user_id}:cred         → DECIMAL
balance:{team_id}:cred         → DECIMAL

# 活跃请求跟踪
request:{request_id}           → Hash { status, model, tokens, created_at }

# 会话管理
session:{session_id}           → Hash { user_id, role, created_at }

# 缓存
cache:response:{hash}          → String (响应内容, TTL 1h)
cache:embedding:{hash}         → String (嵌入向量, TTL 24h)

# 供应商健康状态
provider:health:{provider_id}  → Hash { status, last_check, failures }

# 排行榜（Sorted Set）
ranking:models:popularity      → ZSET (每日更新)
ranking:models:usage           → ZSET (实时)
```

---

## 6. 管理平台设计

### 6.1 管理平台总览

管理平台是**内部运营后台**，独立于用户端的前端 SPA，采用相同设计语言（React + Tailwind）。

```
┌──────────────────────────────────────────────────────────────┐
│                     Admin Dashboard                          │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ 顶部导航：Logo | 概览 | 用户 | 模型 | 计费 | 运维 | 系统 │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                               │
│  ┌──────────┬──────────┬──────────┬──────────┐              │
│  │ 总用户数   │ 日活用户  │ 今日收入   │ 活跃API  │              │
│  │  12,483   │   847    │ $4,230   │  3,201   │              │
│  └──────────┴──────────┴──────────┴──────────┘              │
│                                                               │
│  ┌─────────────────────────┐ ┌────────────────────────────┐ │
│  │ 请求量趋势（7天）         │ │ Token 用量分布（饼图）       │ │
│  │ 📈 折线图                │ │ 🍩 环形图                   │ │
│  └─────────────────────────┘ └────────────────────────────┘ │
│                                                               │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ 最近异常请求 / 系统告警                                   │ │
│  │ ┌────┬──────────┬──────────┬──────────┬──────────────┐ │ │
│  │ │ ⚠️ │ 429 激增  │ 5分钟前   │ api-key  │ 限流触发      │ │ │
│  │ │ 🔴 │ DS-V4 离线│ 10分钟前  │ provider │ 已自动切换     │ │ │
│  │ └────┴──────────┴──────────┴──────────┴──────────────┘ │ │
│  └────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

### 6.2 管理平台功能模块

#### 模块 1：概览仪表盘

| 指标卡片 | 图表 | 告警 |
|---------|------|------|
| 总用户数、日/月活 | 请求量趋势（折线图） | 异常请求预警 |
| 今日收入、本月收入 | Token 用量分布（环形图） | 供应商健康告警 |
| 活跃 API Key 数 | 路由分布（自建 vs 商业） | 余额不足预警 |
| P50/P99 延迟 | Top 模型/用户排行 | 限流触发记录 |

#### 模块 2：用户管理

```
┌──────────────────────────────────────────────────────────────┐
│  用户管理                                                     │
├──────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 🔍 搜索：邮箱/用户名...    [筛选: 全部 ▾]  [+ 新建用户]   │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
│  ┌──────┬────────────┬──────────┬──────┬────────┬──────────┐ │
│  │ ID   │ 用户        │ 邮箱      │ 套餐  │ 余额    │ 操作     │ │
│  ├──────┼────────────┼──────────┼──────┼────────┼──────────┤ │
│  │ 1001 │ 张三        │ z@ai.com │ Pro  │ 500 CRED│ [详情]  │ │
│  │ 1002 │ Acme Corp  │ a@ac.com │ Ent  │ 50K CRED│ [详情]  │ │
│  └──────┴────────────┴──────────┴──────┴────────┴──────────┘ │
│                                                               │
│  用户详情弹窗：                                                │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ Tab: 基本信息 | API Keys | 用量统计 | 交易记录 | 操作日志 │ │
│  │                                                         │ │
│  │ 基本信息：                                               │ │
│  │   用户名 / 邮箱 / 注册时间 / 最后登录                     │ │
│  │   套餐方案 / 升级历史                                    │ │
│  │   所属团队 / 角色                                        │ │
│  │                                                         │ │
│  │ 操作：                                                   │ │
│  │   [禁用账户] [重置密码] [调整额度] [变更套餐] [删除]      │ │
│  └─────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

#### 模块 3：模型与供应商管理

```
┌──────────────────────────────────────────────────────────────┐
│  模型管理                                                     │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  供应商列表：                                                  │
│  ┌────────┬──────────┬──────┬──────────┬──────────────────┐ │
│  │ 名称    │ 接入状态   │ 模型数 │ API Base  │ 操作             │ │
│  ├────────┼──────────┼──────┼──────────┼──────────────────┤ │
│  │ OpenAI │ ✅ 在线   │ 2    │ api.openai│ [配置] [禁用]    │ │
│  │ DeepSk │ ✅ 在线   │ 2    │ api.ds   │ [配置] [禁用]    │ │
│  │ Self   │ ✅ 在线   │ 1    │ internal │ [配置] [扩容]    │ │
│  └────────┴──────────┴──────┴──────────┴──────────────────┘ │
│                                                               │
│  添加供应商：                                                  │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 供应商名称: [________]  API Base URL: [________]         │ │
│  │ API Key:     [________]  状态: [active ▾]               │ │
│  │ 模型列表:    导入或手动添加...                            │ │
│  │                   [保存]                                 │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
│  路由规则管理：                                                │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 默认路由策略：                                            │ │
│  │   复杂度 < 5  → 自建 DS-V4                               │ │
│  │   复杂度 ≥ 5  → 按供应商优先级分配                         │ │
│  │                                                         │ │
│  │ 自定义规则列表：                                          │ │
│  │ ┌─────┬──────────────┬──────────┬──────┬──────────────┐ │ │
│  │ │ 优先级│ 规则名称       │ 条件      │ 目标  │ 操作         │ │ │
│  │ ├─────┼──────────────┼──────────┼──────┼──────────────┤ │ │
│  │ │ 100  │ 代码优先      │ tag=code │ DS-C │ [编辑][删除] │ │ │
│  │ └─────┴──────────────┴──────────┴──────┴──────────────┘ │ │
│  │                                  [+ 添加规则]            │ │
│  └─────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

#### 模块 4：计费与财务管理

```
┌──────────────────────────────────────────────────────────────┐
│  计费管理                                                     │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  收入概览：                                                    │
│  ┌──────────┬──────────┬──────────┬──────────┐              │
│  │ 今日收入   │ 本月收入   │ 上月收入   │ 同比增长  │              │
│  │ $4,230   │ $89,450  │ $72,100  │ +24.1%  │              │
│  └──────────┴──────────┴──────────┴──────────┘              │
│                                                               │
│  交易列表：                                                    │
│  ┌──────┬──────────┬──────────┬──────────┬──────────────┐   │
│  │ 时间  │ 用户      │ 类型      │ 金额      │ 状态          │   │
│  ├──────┼──────────┼──────────┼──────────┼──────────────┤   │
│  │ 14:32│ zhangsan │ usage    │ -0.05    │ completed    │   │
│  │ 14:30│ acme_corp│ topup    │ +1000    │ completed    │   │
│  └──────┴──────────┴──────────┴──────────┴──────────────┘   │
│                                                               │
│  财务报表导出：                                                │
│    [本月报表 PDF] [本月报表 CSV] [本月报表 Excel]             │
│                                                               │
│  $CRED 管理：                                                  │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 总发行量: 10,000,000 CRED    流通量: 2,345,000 CRED      │ │
│  │ 准备金率: 100% ( $2,345,000 USD )                       │ │
│  │ [铸造 $CRED] [销毁 $CRED] [准备金证明]                   │ │
│  └─────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

#### 模块 5：运维监控

```
┌──────────────────────────────────────────────────────────────┐
│  运维监控                                                     │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  服务健康状态：                                                │
│  ┌──────────┬──────────┬──────────┬──────────┬──────────┐   │
│  │ API GW   │ Router   │ DS-V4    │ DeepSeek │ OpenAI   │   │
│  │ 🟢 99.9% │ 🟢 99.8% │ 🟢 99.5% │ 🟢 99.9% │ 🟢 99.9% │   │
│  └──────────┴──────────┴──────────┴──────────┴──────────┘   │
│                                                               │
│  实时指标：                                                    │
│  ┌──────────────┬──────────────┬────────────────────────┐    │
│  │ QPS: 847     │ P50: 120ms  │ P99: 450ms             │    │
│  │ 错误率: 0.1% │ 缓存命中: 42%│ 自建占比: 62%          │    │
│  └──────────────┴──────────────┴────────────────────────┘    │
│                                                               │
│  故障切换日志：                                                │
│  ┌──────┬──────────┬──────────┬──────────┬──────────────┐   │
│  │ 时间  │ 故障源     │ 切换至    │ 恢复时间  │ 影响请求       │   │
│  ├──────┼──────────┼──────────┼──────────┼──────────────┤   │
│  │ 14:00│ DS-V4    │ DeepSeek │ 14:00:28 │ 156          │   │
│  └──────┴──────────┴──────────┴──────────┴──────────────┘   │
│                                                               │
│  告警规则：                                                    │
│  ┌────────────────┬──────────┬──────────┬──────────────────┐ │
│  │ 规则             │ 条件      │ 通知方式   │ 状态              │ │
│  ├────────────────┼──────────┼──────────┼──────────────────┤ │
│  │ 错误率 > 1%     │ P0       │ 电话+短信 │ ✅ 启用           │ │
│  │ 延迟 P99 > 2s  │ P1       │ Slack    │ ✅ 启用           │ │
│  │ 余额 < 10%      │ P2       │ 邮件      │ ✅ 启用           │ │
│  └────────────────┴──────────┴──────────┴──────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

#### 模块 6：系统配置

```
┌──────────────────────────────────────────────────────────────┐
│  系统配置                                                     │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  全局参数：                                                    │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 默认速率限制: [100 RPM] [10000 TPM]                      │ │
│  │ 免费额度:     [100000 tokens/月]                         │ │
│  │ 路由策略:     [智能路由 ▾]  复杂度阈值: [5]              │ │
│  │ 缓存TTL:     [3600 秒]     缓存策略: [精确匹配 ▾]       │ │
│  │ 结算时间:     [UTC 00:00]  L2 网络: [Polygon ▾]         │ │
│  │                          [保存]                          │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
│  管理员账户：                                                  │
│  ┌──────┬────────────┬──────────┬──────────┬──────────────┐  │
│  │ 账号  │ 角色         │ 最后登录   │ 状态      │ 操作         │  │
│  ├──────┼────────────┼──────────┼──────────┼──────────────┤  │
│  │ admin│ 超级管理员   │ 14:32    │ ✅       │ [编辑][禁用]  │  │
│  │ ops  │ 运维管理员   │ 10:15    │ ✅       │ [编辑][禁用]  │  │
│  └──────┴────────────┴──────────┴──────────┴──────────────┘  │
│                                     [+ 添加管理员]            │
└──────────────────────────────────────────────────────────────┘
```

### 6.3 管理平台 RBAC 权限

| 角色 | 概览 | 用户管理 | 模型管理 | 计费 | 运维 | 系统配置 |
|------|------|---------|---------|------|------|---------|
| 超级管理员 | ✅ | ✅ CRUD | ✅ CRUD | ✅ | ✅ | ✅ |
| 运营管理员 | ✅ | ✅ 查看 | ✅ 查看 | ✅ 查看 | ❌ | ❌ |
| 财务管理员 | ✅ | ❌ | ❌ | ✅ CRUD | ❌ | ❌ |
| 运维管理员 | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ |
| 只读 | ✅ | ✅ 查看 | ✅ 查看 | ✅ 查看 | ✅ 查看 | ❌ |

---

## 7. 技术栈选型

### 7.1 后端技术栈

| 层级 | 技术 | 理由 |
|------|------|------|
| **API 服务** | Python (FastAPI) | 与 AI/ML 生态无缝集成，异步高性能，自动 OpenAPI 文档 |
| **路由引擎** | Python (FastAPI + vLLM) | Judge Agent 推理需要，与 ML 模型深度集成 |
| **计费服务** | Go (Gin/Fiber) | 高并发、低延迟，适合金融级事务处理 |
| **管理 API** | Go (Gin/Fiber) | 高性能 CRUD，ORM 成熟（GORM） |
| **认证服务** | Go | JWT 高性能验证，可独立扩缩 |
| **数据库** | PostgreSQL 16 + TimescaleDB | 关系型主库，时序扩展 |
| **缓存** | Redis 7 (Cluster) | 实时计数、会话、缓存、限流 |
| **分析** | ClickHouse | TB 级请求日志存储与实时分析 |
| **消息队列** | Kafka / Redpanda | 请求事件流、异步计费、日志收集 |
| **服务网格** | gRPC + Envoy/Istio | 服务间高性能通信、流量治理 |
| **容器编排** | Kubernetes | 自动扩缩容、滚动更新、健康检查 |
| **可观测性** | OpenTelemetry + Grafana + Prometheus | 分布式追踪、指标、日志 |
| **CI/CD** | GitHub Actions + ArgoCD | GitOps 部署工作流 |

### 7.2 管理平台前端技术栈

| 层级 | 技术 | 理由 |
|------|------|------|
| **框架** | React 19 + TypeScript 6 | 与用户前端技术栈一致 |
| **构建** | Vite 8 | 与用户前端一致 |
| **样式** | Tailwind CSS 3 + shadcn/ui | 快速 UI 开发，与设计语言一致 |
| **路由** | React Router 7 | SPA 路由 |
| **状态管理** | Zustand / TanStack Query | 轻量状态 + 服务端缓存 |
| **表格** | TanStack Table | 高性能表格（用户列表、交易记录） |
| **图表** | Recharts / ECharts | Dashboard 图表 |
| **表单** | React Hook Form + Zod | 类型安全表单验证 |

---

## 8. 部署架构

### 8.1 Kubernetes 集群拓扑

```
┌──────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                        │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌─────────────────────┐    ┌─────────────────────┐          │
│  │   Ingress Controller │    │   Cert Manager       │          │
│  │   (Nginx/Traefik)   │    │   (TLS 自动签发)      │          │
│  └──────────┬──────────┘    └─────────────────────┘          │
│             │                                                 │
│  ┌──────────┴──────────────────────────────────────────┐     │
│  │                     服务层                            │     │
│  │                                                      │     │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐ │     │
│  │  │api-gw    │ │api-svr   │ │router    │ │admin   │ │     │
│  │  │(3 pods)  │ │(3 pods)  │ │(2 pods)  │ │(2 pods)│ │     │
│  │  └──────────┘ └──────────┘ └──────────┘ └────────┘ │     │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐            │     │
│  │  │inference │ │billing   │ │auth      │            │     │
│  │  │(2 pods)  │ │(2 pods)  │ │(2 pods)  │            │     │
│  │  │+ GPU     │ │          │ │          │            │     │
│  │  └──────────┘ └──────────┘ └──────────┘            │     │
│  └──────────────────────────────────────────────────────┘     │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐     │
│  │                     数据层                            │     │
│  │                                                      │     │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐            │     │
│  │  │PostgreSQL│ │Redis     │ │ClickHouse│            │     │
│  │  │(HA)      │ │(Cluster) │ │(Cluster) │            │     │
│  │  └──────────┘ └──────────┘ └──────────┘            │     │
│  └──────────────────────────────────────────────────────┘     │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

### 8.2 环境划分

| 环境 | 用途 | 数据库 | 配置 |
|------|------|--------|------|
| **dev** | 日常开发 | SQLite / 本地 PG | 低资源，热重载 |
| **staging** | 集成测试 | PG + Redis（单机） | 模拟生产 |
| **prod** | 生产环境 | PG HA + Redis Cluster + ClickHouse Cluster | 高可用，多副本 |

---

## 9. 安全设计

### 9.1 API Key 管理

```
┌──────────────────────────────────────────────────────────────┐
│                    API Key 生命周期                            │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  1. 创建：sha256(prefix + random_bytes) → 只存储 hash         │
│     Key 格式: sk-mr-{prefix}-{random_secret}                  │
│                                                               │
│  2. 验证：每次请求提取 prefix，查 DB 获取 hash，对比验证        │
│                                                               │
│  3. 轮换：支持平滑轮换（新旧 Key 共存 24h）                     │
│                                                               │
│  4. 吊销：立即生效，已建立 SSE 连接不中断（宽限期 60s）         │
│                                                               │
│  5. 展示：仅创建时展示完整 Key，之后只显示前缀                  │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

### 9.2 安全措施清单

| 层级 | 措施 | 说明 |
|------|------|------|
| 传输 | TLS 1.3 | 全链路 HTTPS/WSS |
| 认证 | API Key + JWT 双轨 | API 调用用 Key，管理后台用 JWT |
| 授权 | RBAC | 角色 + 资源 + 操作 三维权限控制 |
| 数据 | AES-256-GCM 加密 | 敏感字段（密钥、余额）落盘加密 |
| 注入防护 | 参数化查询 | 100% 使用 ORM/参数化查询 |
| 限流 | Token Bucket (Redis) | 多维度限流（用户/Key/IP） |
| 审计 | 全量操作日志 | 不可篡改，ClickHouse 存储 |
| CSP | 内容安全策略头 | 管理平台 XSS 防护 |
| 依赖 | Dependabot + Snyk | 自动扫描已知漏洞 |

---

## 10. API 详细设计

### 10.1 用户端 API

详见 `api-server` 接口列表（第 2.2 节）。补充关键数据结构：

```typescript
// POST /v1/chat/completions 请求体
{
  "model": "auto",                    // 模型 ID 或 "auto" 启用智能路由
  "messages": [
    { "role": "user", "content": "..." }
  ],
  "stream": true,
  "temperature": 0.7,
  "max_tokens": 1024,
  
  // MaaS-Router 扩展字段
  "router": {
    "strategy": "cost_optimized",    // cost_optimized | performance | balanced
    "preferred_providers": [],       // 偏好供应商
    "max_cost_per_request": 0.01,    // 单次请求最大费用 (CRED)
    "complexity_hint": null          // 用户主动提示复杂度 (1-10)
  }
}

// POST /v1/chat/completions 响应（非流式）
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1714900000,
  "model": "deepseek-v4",            // 实际使用的模型
  "choices": [...],
  "usage": {
    "prompt_tokens": 150,
    "completion_tokens": 300,
    "total_tokens": 450
  },
  
  // MaaS-Router 扩展字段
  "router_decision": {
    "complexity_score": 4.2,
    "route_reason": "简单对话，路由至自建 DeepSeek-V4 集群",
    "confidence": 0.92,
    "cost_cred": 0.000225,
    "saved_vs_direct": "50%"
  }
}
```

### 10.2 管理平台 API（admin-server）

```
Admin API Base: /api/admin/v1

认证: Bearer {admin_jwt_token}

# ===== 仪表盘 =====
GET    /dashboard/overview              # 概览统计数据
GET    /dashboard/trends                # 趋势数据（可指定时间范围）
GET    /dashboard/alerts                # 当前活跃告警

# ===== 用户管理 =====
GET    /users                           # 用户列表（分页、搜索、筛选）
GET    /users/:id                       # 用户详情
POST   /users                           # 创建用户
PUT    /users/:id                       # 更新用户
DELETE /users/:id                       # 删除用户（软删除）
PUT    /users/:id/status                # 启用/禁用用户
PUT    /users/:id/plan                  # 变更套餐
GET    /users/:id/api-keys              # 用户 API Keys
GET    /users/:id/transactions          # 用户交易记录
GET    /users/:id/request-logs          # 用户请求日志
GET    /users/:id/audit-logs            # 用户操作审计

# ===== 团队管理 =====
GET    /teams                           # 团队列表
GET    /teams/:id                       # 团队详情（含成员）
POST   /teams                           # 创建团队
PUT    /teams/:id                       # 编辑团队
DELETE /teams/:id                       # 删除团队
POST   /teams/:id/members               # 添加成员
DELETE /teams/:id/members/:userId       # 移除成员
PUT    /teams/:id/members/:userId/role  # 修改成员角色

# ===== 模型管理 =====
GET    /providers                       # 供应商列表
POST   /providers                       # 添加供应商
PUT    /providers/:id                   # 编辑供应商
DELETE /providers/:id                   # 删除供应商
PUT    /providers/:id/status            # 启用/禁用供应商

GET    /models                          # 模型列表
POST   /models                          # 添加模型
PUT    /models/:id                      # 编辑模型（含定价）
DELETE /models/:id                      # 删除模型
PUT    /models/:id/status               # 启用/禁用模型

GET    /routing-rules                   # 路由规则列表
POST   /routing-rules                   # 创建路由规则
PUT    /routing-rules/:id               # 更新路由规则
DELETE /routing-rules/:id               # 删除路由规则

# ===== 计费管理 =====
GET    /billing/overview                # 收入概览
GET    /billing/transactions            # 交易列表（分页、筛选）
GET    /billing/transactions/:id        # 交易详情
POST   /billing/adjust                  # 调整余额（增减）
POST   /billing/refund/:transactionId   # 退款

GET    /cred/supply                     # $CRED 发行/流通数据
POST   /cred/mint                       # 铸造 $CRED
POST   /cred/burn                       # 销毁 $CRED
GET    /cred/reserve-proof              # 准备金证明

GET    /billing/reports                 # 生成财务报表
GET    /billing/reports/download        # 下载报表 (PDF/CSV/Excel)

# ===== 运维监控 =====
GET    /monitoring/services             # 服务健康状态
GET    /monitoring/metrics              # 实时指标
GET    /monitoring/failover-logs        # 故障切换日志

GET    /alerts/rules                    # 告警规则列表
POST   /alerts/rules                    # 创建告警规则
PUT    /alerts/rules/:id                # 更新告警规则
DELETE /alerts/rules/:id                # 删除告警规则

# ===== 系统配置 =====
GET    /system/configs                  # 系统配置列表
PUT    /system/configs/:key             # 更新系统配置

GET    /system/admins                   # 管理员列表
POST   /system/admins                   # 添加管理员
PUT    /system/admins/:id               # 编辑管理员
DELETE /system/admins/:id               # 删除管理员

# ===== 审计日志 =====
GET    /audit-logs                      # 审计日志列表（分页、搜索）
```

### 10.3 服务间 gRPC 接口

```protobuf
// router_engine.proto
service RouterEngine {
  rpc ScoreComplexity(ComplexityRequest) returns (ComplexityResponse);
  rpc RouteDecision(RouteRequest) returns (RouteResponse);
  rpc HealthCheck(Empty) returns (HealthStatus);
}

// billing.proto
service BillingService {
  rpc RecordUsage(UsageEvent) returns (BillingResponse);
  rpc CheckBalance(BalanceRequest) returns (BalanceResponse);
  rpc ReserveCredits(ReserveRequest) returns (ReserveResponse);
}

// inference.proto
service InferenceService {
  rpc ChatCompletion(ChatRequest) returns (stream ChatChunk);
  rpc Embedding(EmbeddingRequest) returns (EmbeddingResponse);
}
```

---

## 11. 项目目录结构

```
maas-router/
├── backend/
│   ├── cmd/                          # 各服务入口
│   │   ├── api-gateway/
│   │   ├── api-server/
│   │   ├── router-engine/
│   │   ├── inference-exec/
│   │   ├── billing-svc/
│   │   ├── admin-server/
│   │   └── auth-svc/
│   │
│   ├── internal/                     # 内部包（不对外）
│   │   ├── api-gateway/
│   │   │   ├── middleware/
│   │   │   └── handler/
│   │   ├── api-server/
│   │   │   ├── handler/
│   │   │   ├── service/
│   │   │   └── model/
│   │   ├── router-engine/
│   │   │   ├── scorer/
│   │   │   ├── decision/
│   │   │   └── rules/
│   │   ├── billing/
│   │   │   ├── ledger/
│   │   │   ├── settlement/
│   │   │   └── report/
│   │   └── admin/
│   │       ├── handler/
│   │       ├── service/
│   │       └── repository/
│   │
│   ├── pkg/                          # 共享库
│   │   ├── auth/
│   │   ├── db/
│   │   ├── cache/
│   │   ├── queue/
│   │   ├── logging/
│   │   ├── metrics/
│   │   └── proto/                    # gRPC Proto 定义
│   │
│   ├── migrations/                   # 数据库迁移
│   ├── deploy/                       # K8s 部署文件
│   │   ├── k8s/
│   │   │   ├── api-gateway.yaml
│   │   │   ├── api-server.yaml
│   │   │   └── ...
│   │   └── docker/
│   │
│   └── config/                       # 配置文件
│       ├── dev.yaml
│       ├── staging.yaml
│       └── prod.yaml
│
├── admin-platform/                   # 管理平台前端
│   ├── src/
│   │   ├── components/               # 共享组件
│   │   │   ├── Layout/
│   │   │   ├── DataTable/
│   │   │   ├── ChartCard/
│   │   │   └── ...
│   │   ├── pages/
│   │   │   ├── Dashboard/
│   │   │   ├── Users/
│   │   │   ├── Teams/
│   │   │   ├── Models/
│   │   │   ├── Billing/
│   │   │   ├── Monitoring/
│   │   │   ├── Settings/
│   │   │   └── AuditLogs/
│   │   ├── hooks/
│   │   ├── services/                 # API 调用
│   │   ├── stores/                   # Zustand stores
│   │   ├── types/
│   │   └── utils/
│   ├── package.json
│   ├── vite.config.ts
│   └── tailwind.config.ts
│
├── user-frontend/                    # (现有) 用户端前端
│   └── ai-api-hub/                   # 已有的 React SPA
│
├── docker-compose.yml                # 本地开发环境
├── Makefile                          # 常用命令
└── README.md
```

---

## 12. 开发阶段规划

### Phase 1：基础后端 + 管理平台 MVP（对应 PRD Phase 1-2）

| 交付物 | 优先级 | 说明 |
|--------|--------|------|
| api-gateway 基础版 | P0 | API Key 认证、基础限流、日志 |
| api-server OpenAI 兼容 | P0 | `/v1/chat/completions` + SSE 流式 |
| 2 家商业 API 适配器 | P0 | DeepSeek + OpenAI 接入 |
| 基础路由（规则引擎） | P0 | 基于模型名/Token 数的硬编码路由 |
| billing-svc 基础版 | P0 | 实时 Token 计数 + 余额扣减 |
| admin-server 基础版 | P0 | 用户管理 + API Key 管理 + 仪表盘 |
| admin-platform 基础版 | P0 | 概览 + 用户管理 + 模型管理页面 |
| PostgreSQL + Redis | P0 | 主库 + 缓存/计数 |

### Phase 2：智能路由 + 完整管理平台（对应 PRD Phase 3-4）

| 交付物 | 优先级 | 说明 |
|--------|--------|------|
| Judge Agent 路由引擎 | P1 | Qwen2.5-7B 复杂度评分 + 路由决策 |
| 路由决策日志 | P1 | 全量路由决策记录与可视化 |
| 自定义路由规则 | P1 | 用户/管理员可配置路由规则 |
| 故障自动切换 | P1 | 供应商健康检查 + 自动 failover |
| $CRED 链下积分系统 | P1 | 积分余额、充值、消费 |
| 财务报表 | P1 | CSV/PDF/Excel 导出 |
| 管理平台完整功能 | P1 | 所有模块功能完善 |
| ClickHouse 接入 | P1 | 请求日志时序存储 |

---

## 附录 A：与现有前端的接口对齐

现有用户前端（ai-api-hub）的 `Docs.tsx` 中预定义了 API：

```typescript
// 现有前端假设的 API
GET  https://api.aihub.com/v1/models
POST https://api.aihub.com/v1/chat/completions
```

后端设计完全兼容此接口，并扩展：

| 现有前端期望 | 后端实现 | 状态 |
|-------------|---------|------|
| `GET /v1/models` | api-server 从 models 表读取 | ✅ 兼容 |
| `POST /v1/chat/completions` | api-server 完整实现 + 路由 | ✅ 兼容 |
| API Key 管理（/keys 页面） | admin-server `/users/:id/api-keys` | ✅ 新增 |
| Dashboard 数据（Dashboard 页面） | admin-server `/dashboard/*` | ✅ 需要后端 |
| 定价数据（Pricing 页面） | admin-server `/models` 定价字段 | ✅ 需要后端 |
| Chat 页面（在线体验） | api-server `/v1/chat/completions` | ✅ 兼容 |

---

## 附录 B：关键决策记录

| 决策 | 选择 | 理由 |
|------|------|------|
| API 服务语言 | Python (FastAPI) | AI/ML 生态优势，异步性能足够 |
| 计费服务语言 | Go | 金融级事务，高并发，低延迟 |
| 消息队列 | Kafka / Redpanda | 高吞吐、持久化、生态成熟 |
| 分析数据库 | ClickHouse | TB 级时序数据实时分析 |
| 前端组件库 | shadcn/ui | 与现有设计语言一致，可控性强 |
| 部署方式 | Kubernetes | 微服务编排标准，自动扩缩容 |

---

> **本文档由架构设计团队 AI 协作生成**
> **版本**：v1.0
> **日期**：2026-05-05
> **状态**：初稿，待团队评审
