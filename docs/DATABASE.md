# 数据库设计

> 最后更新：2026-05-05 | 数据库：PostgreSQL 16

---

## 目录

1. [ER 图](#1-er-图)
2. [表结构详解](#2-表结构详解)
3. [索引策略](#3-索引策略)
4. [迁移管理](#4-迁移管理)

---

## 1. ER 图

```
┌──────────┐       ┌──────────┐       ┌──────────┐
│   users   │──1:N──│ api_keys │       │  teams   │
└────┬─────┘       └──────────┘       └────┬─────┘
     │                                     │
     │ 1:1                                 │ N:M
     │                                     │
┌────▼─────┐                          ┌────▼─────┐
│ balances │                          │team_members│
└──────────┘                          └──────────┘

┌──────────┐       ┌──────────┐
│providers │──1:N──│  models  │
└──────────┘       └────┬─────┘
                         │
                    ┌────▼──────┐
                    │routing_rules│
                    └────────────┘

┌──────────┐
│transactions│──FK── users, models
└──────────┘

┌──────────┐
│request_logs│──FK── users, models, providers
└──────────┘

┌──────────┐
│audit_logs│──FK── users
└──────────┘
```

共 **11 张表**，分为 5 个功能域：

| 功能域 | 表 |
|--------|---|
| 用户与团队 | `users`, `teams`, `team_members` |
| 认证 | `api_keys` |
| 模型与路由 | `providers`, `models`, `routing_rules` |
| 计费 | `balances`, `transactions` |
| 日志 | `request_logs`, `audit_logs` |

---

## 2. 表结构详解

### 2.1 users — 用户表

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | VARCHAR(36) | PK, UUID | 用户唯一标识 |
| `email` | VARCHAR(255) | UNIQUE, NOT NULL, INDEX | 邮箱 |
| `password_hash` | VARCHAR(255) | NOT NULL | bcrypt 哈希 |
| `display_name` | VARCHAR(100) | NULL | 显示名称 |
| `avatar_url` | TEXT | NULL | 头像 URL |
| `status` | VARCHAR(20) | DEFAULT 'active' | active / suspended / deleted |
| `email_verified` | BOOLEAN | DEFAULT false | 邮箱是否验证 |
| `plan_id` | VARCHAR(20) | DEFAULT 'free' | free / pro / enterprise |
| `created_at` | TIMESTAMPTZ | | 创建时间 |
| `updated_at` | TIMESTAMPTZ | | 更新时间 |
| `last_login_at` | TIMESTAMPTZ | NULL | 最后登录时间 |

**关联关系**：
- `api_keys`: 1:N（一个用户多个 API Key）
- `balance`: 1:1（一个用户一个余额账户）

---

### 2.2 api_keys — API 密钥表

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | VARCHAR(36) | PK, UUID | 密钥 ID |
| `user_id` | VARCHAR(36) | FK → users.id, NOT NULL | 所属用户 |
| `name` | VARCHAR(100) | NOT NULL | 密钥名称 |
| `key_hash` | VARCHAR(64) | NOT NULL, UNIQUE | SHA-256 哈希（不存明文） |
| `key_prefix` | VARCHAR(32) | NOT NULL | 密钥前缀（用于识别） |
| `status` | VARCHAR(20) | DEFAULT 'active' | active / revoked |
| `rate_limit_rpm` | INT | | 每分钟请求限制 |
| `rate_limit_tpm` | INT | | 每分钟 Token 限制 |
| `last_used_at` | TIMESTAMPTZ | NULL | 最后使用时间 |
| `created_at` | TIMESTAMPTZ | | 创建时间 |

> ⚠️ `key_hash` 存储 SHA-256 哈希，原始密钥仅在创建时返回。无法从哈希反推原始密钥。

---

### 2.3 providers — AI 供应商表

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | VARCHAR(50) | PK | 供应商 ID（如 `openai`, `deepseek`） |
| `name` | VARCHAR(100) | NOT NULL | 供应商显示名称 |
| `api_base_url` | VARCHAR(500) | NULL | API 端点 URL |
| `api_key_encrypted` | TEXT | NULL | 加密的 API Key |
| `status` | VARCHAR(20) | DEFAULT 'active' | active / suspended |
| `priority` | INT | DEFAULT 0 | 优先级（越高越优先） |
| `created_at` | TIMESTAMPTZ | | 创建时间 |

**种子数据（8 家供应商）**：OpenAI, DeepSeek, Anthropic, Google, Meta, Mistral, Cohere, Self-Hosted

---

### 2.4 models — 模型表

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | VARCHAR(100) | PK | 模型 ID（如 `gpt-4o`） |
| `provider_id` | VARCHAR(50) | FK → providers.id | 所属供应商 |
| `display_name` | VARCHAR(200) | NOT NULL | 显示名称 |
| `context_window` | INT | | 上下文窗口大小 |
| `input_price` | FLOAT | | 输入价格（CRED/百万 token） |
| `output_price` | FLOAT | | 输出价格（CRED/百万 token） |
| `status` | VARCHAR(20) | DEFAULT 'active' | active / suspended |
| `tags` | JSON | | 标签列表 |
| `features` | JSON | | 特性列表 |
| `is_recommended` | BOOLEAN | DEFAULT false | 是否推荐 |
| `popularity` | INT | DEFAULT 0 | 热度排序 |
| `created_at` | TIMESTAMPTZ | | 创建时间 |

**种子数据（15 个模型）**：GPT-4o, GPT-4o Mini, GPT-4 Turbo, Claude 3.5 Sonnet, Claude 3 Haiku, DeepSeek-V3, DeepSeek-V4 Self, Gemini 1.5 Pro, Gemini 1.5 Flash, Llama 3.1 405B, Llama 3.1 70B, Mistral Large, Mistral Small, Command R+, Qwen2.5-72B

---

### 2.5 routing_rules — 路由规则表

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | VARCHAR(36) | PK, UUID | 规则 ID |
| `name` | VARCHAR(200) | NOT NULL | 规则名称 |
| `priority` | INT | | 优先级 |
| `condition` | JSON | | 匹配条件 |
| `target_model_id` | VARCHAR(100) | FK → models.id | 目标模型 |
| `status` | VARCHAR(20) | DEFAULT 'active' | active / disabled |
| `created_at` | TIMESTAMPTZ | | 创建时间 |

---

### 2.6 balances — 余额表

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | VARCHAR(36) | PK, UUID | 账户 ID |
| `user_id` | VARCHAR(36) | FK → users.id, UNIQUE | 所属用户（1:1） |
| `cred_balance` | FLOAT | DEFAULT 0.0 | CRED 余额 |
| `usd_balance` | FLOAT | DEFAULT 0.0 | 美元等价余额 |
| `frozen_cred` | FLOAT | DEFAULT 0.0 | 冻结中的 CRED |
| `updated_at` | TIMESTAMPTZ | | 最后更新 |

---

### 2.7 transactions — 交易记录表

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | VARCHAR(36) | PK, UUID | 交易 ID |
| `user_id` | VARCHAR(36) | FK → users.id | 用户 |
| `type` | VARCHAR(20) | NOT NULL | topup / usage / refund / freeze |
| `status` | VARCHAR(20) | DEFAULT 'completed' | completed / pending / failed |
| `request_id` | VARCHAR(64) | NULL | 关联的请求 ID |
| `model_id` | VARCHAR(100) | NULL | 使用的模型 |
| `provider_id` | VARCHAR(50) | NULL | 供应商 |
| `amount` | FLOAT | NOT NULL | 金额（正=收入, 负=支出） |
| `currency` | VARCHAR(10) | DEFAULT 'CRED' | 货币类型 |
| `prompt_tokens` | INT | | 输入 token 数 |
| `completion_tokens` | INT | | 输出 token 数 |
| `total_tokens` | INT | | 总 token 数 |
| `route_reason` | VARCHAR(500) | NULL | 路由原因 |
| `route_confidence` | FLOAT | NULL | 路由置信度 |
| `created_at` | TIMESTAMPTZ | | 创建时间 |

---

### 2.8 request_logs — 请求日志表

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | VARCHAR(36) | PK, UUID | 日志 ID |
| `request_id` | VARCHAR(64) | | 请求唯一标识 |
| `user_id` | VARCHAR(36) | FK → users.id | 用户 |
| `model_id` | VARCHAR(100) | | 使用的模型 |
| `provider_id` | VARCHAR(50) | | 供应商 |
| `method` | VARCHAR(10) | | HTTP 方法 |
| `endpoint` | VARCHAR(500) | | 请求路径 |
| `status_code` | INT | | HTTP 状态码 |
| `latency_ms` | INT | | 响应延迟（毫秒） |
| `prompt_tokens` | INT | | 输入 tokens |
| `completion_tokens` | INT | | 输出 tokens |
| `complexity_score` | FLOAT | NULL | 复杂度评分 |
| `route_decision` | JSON | NULL | 路由决策详情 |
| `error_code` | VARCHAR(50) | NULL | 错误码 |
| `error_message` | TEXT | NULL | 错误信息 |
| `created_at` | TIMESTAMPTZ | | 创建时间 |

---

### 2.9 audit_logs — 审计日志表

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | VARCHAR(36) | PK, UUID | 日志 ID |
| `user_id` | VARCHAR(36) | FK → users.id, NULL | 操作人 |
| `action` | VARCHAR(100) | NOT NULL | 操作类型 |
| `resource_type` | VARCHAR(50) | | 资源类型 |
| `resource_id` | VARCHAR(100) | NULL | 资源 ID |
| `detail` | JSON | NULL | 操作详情 |
| `ip_address` | VARCHAR(45) | NULL | IP 地址 |
| `created_at` | TIMESTAMPTZ | | 操作时间 |

---

### 2.10 teams — 团队表

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | VARCHAR(36) | PK, UUID | 团队 ID |
| `name` | VARCHAR(200) | NOT NULL | 团队名称 |
| `owner_id` | VARCHAR(36) | FK → users.id | 团队所有者 |
| `created_at` | TIMESTAMPTZ | | 创建时间 |

### 2.11 team_members — 团队成员表

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | VARCHAR(36) | PK, UUID | 记录 ID |
| `team_id` | VARCHAR(36) | FK → teams.id | 团队 |
| `user_id` | VARCHAR(36) | FK → users.id | 用户 |
| `role` | VARCHAR(20) | DEFAULT 'member' | owner / admin / member |

---

## 3. 索引策略

### 现有索引

| 表 | 索引列 | 类型 | 用途 |
|----|--------|------|------|
| `users` | `email` | UNIQUE B-tree | 登录查询 |
| `api_keys` | `key_hash` | UNIQUE B-tree | API Key 验证 |
| `api_keys` | `user_id` | B-tree（FK 自动） | 用户关联 |
| `models` | `provider_id` | B-tree（FK 自动） | 供应商关联 |

### 建议添加的索引

以下索引在生产环境数据量增长后建议添加：

```sql
-- 加速交易记录按时间查询
CREATE INDEX idx_transactions_user_created ON transactions(user_id, created_at DESC);

-- 加速请求日志统计查询
CREATE INDEX idx_request_logs_created ON request_logs(created_at DESC);
CREATE INDEX idx_request_logs_user_created ON request_logs(user_id, created_at DESC);

-- 加速审计日志查询
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
```

---

## 4. 迁移管理

### Alembic 配置

项目使用 Alembic 进行数据库迁移管理：

```bash
backend/
└── migrations/
    ├── alembic.ini          # 迁移配置
    ├── env.py               # 迁移环境
    └── versions/            # 迁移版本文件
```

### 常用命令

```bash
cd backend

# 自动生成迁移（基于模型变更）
alembic revision --autogenerate -m "add new column"

# 执行迁移
alembic upgrade head

# 回滚一个版本
alembic downgrade -1

# 查看迁移历史
alembic history

# 查看当前版本
alembic current
```

### 迁移规范

- **每次迁移只做一件事**：一个表变更 = 一个迁移文件
- **迁移必须可回滚**：`upgrade()` 和 `downgrade()` 都要实现
- **禁止在迁移中写业务逻辑**：迁移只做 DDL
- **生产环境先 staging 验证**：迁移在 staging 环境执行通过后才能应用到 production
