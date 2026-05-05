# MaaS-Router 系统架构

> 最后更新：2026-05-05 | 版本：1.0.0

---

## 目录

1. [架构总览](#1-架构总览)
2. [核心模块](#2-核心模块)
3. [路由引擎](#3-路由引擎)
4. [认证体系](#4-认证体系)
5. [计费模型](#5-计费模型)
6. [设计决策记录](#6-设计决策记录)

---

## 1. 架构总览

MaaS-Router 采用 **前后端分离 + 双服务** 架构：

```
                          ┌─────────────────────┐
                          │   用户 / 第三方应用    │
                          └──────────┬──────────┘
                                     │ HTTPS
                          ┌──────────▼──────────┐
                          │   Nginx / LB (WIP)   │
                          └──────┬───────┬──────┘
                                 │       │
                    ┌────────────▼──┐ ┌──▼────────────┐
                    │  API Server   │ │ Admin Server   │
                    │  (FastAPI)    │ │ (FastAPI)      │
                    │  :8001        │ │ :8005          │
                    │               │ │                │
                    │ 用户端 API     │ │ 管理后台 API    │
                    │ OpenAI 兼容    │ │ CRUD + 统计    │
                    └───────┬───────┘ └──────┬─────────┘
                            │               │
                   ┌────────▼───────┐       │
                   │   PostgreSQL 16│◄──────┘
                   │   (主数据库)    │
                   └────────────────┘
                            │
                   ┌────────▼───────┐
                   │    Redis 7     │
                   │  (缓存/限流)    │
                   └────────────────┘
                                     │
                          ┌──────────▼──────────┐
                          │   Admin Platform     │
                          │   (React + Vite)     │
                          │   :5173              │
                          └─────────────────────┘
```

### 为什么是双服务架构？

| 考量 | 决策 |
|------|------|
| **安全隔离** | 用户端 API 和管理后台 API 暴露不同的攻击面，分开部署可独立加固 |
| **独立扩缩容** | 用户请求量波动大时只扩 API Server，管理后台保持稳定 |
| **部署灵活** | 管理后台可部署在内网，用户端 API 暴露公网 |
| **故障隔离** | 一个服务故障不影响另一个 |

---

## 2. 核心模块

### 2.1 后端模块划分

```
backend/app/
│
├── core/                    # 核心基础设施
│   ├── config.py            # Pydantic Settings，所有配置项
│   ├── database.py          # 异步 SQLAlchemy 引擎 + 会话工厂
│   ├── security.py          # 密码哈希、JWT、API Key、认证依赖
│   └── redis.py             # Redis 连接 + 缓存/限流工具
│
├── models/                  # 数据模型（11 张表）
│   ├── user.py              # 用户
│   ├── team.py              # 团队 + 成员
│   ├── api_key.py           # API 密钥
│   ├── provider.py          # AI 供应商 + 模型
│   ├── billing.py           # 余额 + 交易记录
│   └── routing.py           # 路由规则 + 请求日志 + 审计日志
│
├── api_server/              # 用户端 API（14 个端点）
│   └── router.py            # /v1/models, /v1/chat/completions, /v1/keys, etc.
│
├── admin_server/            # 管理后台 API（39 个端点）
│   ├── auth_admin.py        # 登录/登出
│   ├── dashboard.py         # 统计概览、趋势图
│   ├── users.py             # 用户 CRUD
│   ├── models_admin.py      # 供应商、模型、路由规则管理
│   ├── billing_admin.py     # 计费管理
│   ├── monitoring.py        # 运维监控、告警
│   ├── settings.py          # 系统设置、审计日志
│   └── documents.py         # 文档生成（报表导出）
│
├── services/                # 业务服务
│   └── document_service.py  # PDF/Excel/Word/PPTX 文档生成引擎
│
├── schemas/                 # Pydantic 请求/响应模式
│   └── chat.py              # Chat Completion 请求/响应模型
│
└── scripts/
    └── seed.py              # 种子数据（8 家供应商 + 15 个模型）
```

### 2.2 前端模块划分

```
admin-platform/src/
│
├── components/
│   └── layout/
│       └── AdminLayout.tsx   # 全局布局（侧边栏 + 顶栏 + 内容区）
│
├── pages/
│   ├── LoginPage.tsx         # 登录页
│   ├── dashboard/            # 仪表盘（统计卡片 + 趋势图 + 最近请求）
│   ├── users/                # 用户管理（列表 + 搜索 + 新建 + 启用/禁用）
│   ├── models/               # 模型管理（供应商/模型/路由规则 三 Tab）
│   ├── billing/              # 计费管理（收入 + CRED + 交易记录）
│   ├── monitoring/           # 运维监控（健康 + 指标 + 告警 + 日志）
│   └── settings/             # 系统设置 + 审计日志
│
└── services/
    └── api.ts                # 统一 API 调用层（auth, users, models, billing, ...）
```

---

## 3. 路由引擎

### 3.1 复杂度评分算法

智能路由的核心是 **请求复杂度评分**（1-10 分）：

```
评分维度                 权重         信号来源
─────────────────────────────────────────────────
Prompt 长度              30%         字符数分段 (< 500 / 1000 / 2000)
代码关键词密度            30%         识别 "def ", "function", "SELECT" 等
推理关键词密度            20%         识别 "explain", "分析", "对比" 等
创意关键词密度            10%         识别 "story", "诗歌", "创作" 等
─────────────────────────────────────────────────
```

> **注意**：当前为 Demo 模式，使用规则评分。生产环境计划接入 Judge Agent（Qwen2.5-7B）进行语义级复杂度判断。

### 3.2 路由决策矩阵

| 复杂度 | 路由目标 | 适用场景 | 相对成本 |
|--------|----------|----------|----------|
| 1-3 | DeepSeek-V4 自建 | 简单问答、翻译 | 1x（基准） |
| 4-6 | DeepSeek-V3 | 中等推理、摘要 | 2x |
| 7-8 | GPT-4o Mini | 代码生成、分析 | 5x |
| 9-10 | GPT-4o | 复杂推理、创作 | 15x |

### 3.3 路由响应头

每个 Chat Completion 响应都包含路由决策信息：

```json
{
  "router_decision": {
    "complexity_score": 3.2,
    "route_reason": "简单查询，路由至自建 DeepSeek-V4",
    "confidence": 0.32,
    "cost_cred": 0.000256
  }
}
```

---

## 4. 认证体系

### 4.1 双模式认证

```
请求进入
    │
    ├── Header 前缀是 "sk-mr-" ？
    │   ├── 是 → API Key 认证
    │   │        ├── SHA-256 哈希比对
    │   │        ├── 检查状态（active/revoked）
    │   │        └── 更新 last_used_at
    │   │
    │   └── 否 → JWT Token 认证
    │            ├── 解码 JWT
    │            ├── 验证签名 + 过期时间
    │            ├── 查找用户
    │            └── 检查用户状态
```

### 4.2 API Key 设计

```
格式: sk-mr-<8位hex>-<48位hex>
示例: sk-mr-a1b2c3d4-e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6
      └── 前缀 ─┘ └────── 随机部分 ──────────────────────────────┘
```

- 前缀 `sk-mr-` 用于快速识别认证模式
- 密钥哈希（SHA-256）存储，原始密钥仅在创建时返回一次
- 支持按 Key 设置独立速率限制

---

## 5. 计费模型

### 5.1 CRED 虚拟货币

```
 用户充值 ($USD)
      │
      ▼
  CRED 余额（按供应商定价换算）
      │
      ▼
  每次 API 调用扣费:
  cost = (prompt_tokens / 1M) × input_price + (completion_tokens / 1M) × output_price
      │
      ▼
  Transaction 记录（type: usage, amount: -0.000256）
```

### 5.2 交易类型

| Type | 说明 | amount 符号 |
|------|------|------------|
| `topup` | 用户充值 | +正数 |
| `usage` | API 调用消费 | -负数 |
| `refund` | 退款（故障补偿等） | +正数 |
| `freeze` | 冻结（预扣，未实现） | -负数 |

---

## 6. 设计决策记录

### ADR-001: 选择 FastAPI 而非 Django

**决策**：使用 FastAPI 作为后端框架。

**理由**：
- 原生异步支持（async/await），适合高并发 API 网关场景
- 自动生成 OpenAPI 文档，减少文档维护成本
- Pydantic 集成提供强类型校验
- 比 Django REST Framework 更轻量，更适合微服务

### ADR-002: 双服务架构而非单服务

**决策**：API Server 和 Admin Server 分开部署。

**理由**：见 [架构总览](#1-架构总览) 中的分析。

### ADR-003: 选择 SQLAlchemy 2.0 Async 而非 Tortoise ORM

**决策**：使用 SQLAlchemy 2.0 异步模式。

**理由**：
- 生态成熟，社区活跃
- 支持复杂查询（子查询、CTE、窗口函数）
- Alembic 迁移工具成熟
- 团队有 SQLAlchemy 使用经验

### ADR-004: 复杂度评分采用规则引擎 + 未来升级 LLM Judge

**决策**：当前使用规则引擎评分，未来升级为 LLM Judge。

**理由**：
- 规则引擎零延迟，适合 Demo 阶段快速验证
- LLM Judge 调用的成本和延迟在当前阶段不划算
- 架构已预留 Judge Agent 接口，升级时改动最小
