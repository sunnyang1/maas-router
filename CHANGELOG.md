# 变更日志

本项目遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

---

## [1.0.0] - 2026-05-05

### 首次发布 🎉

**MaaS-Router v1.0.0** — AI 推理聚合网关，多供应商 AI 模型统一接入与智能路由平台。

### 新增

#### 后端 API Server（14 个端点）
- `GET /v1/models` — 模型列表（OpenAI 兼容）
- `GET /v1/models/{id}` — 单个模型详情
- `POST /v1/chat/completions` — Chat Completion（支持 auto 路由 + 流式输出）
- `GET/POST/DELETE /v1/keys` — API Key 管理
- `GET /v1/balance` — 余额查询
- `GET /v1/usage/summary` — 用量统计
- `GET /v1/router/decisions` — 路由决策记录

#### 后端 Admin Server（39 个端点）
- `POST /auth/login` — 登录认证
- `GET /dashboard/overview` — 仪表盘概览统计
- `GET /dashboard/trends` — 趋势数据
- `GET /dashboard/model-distribution` — 模型使用分布
- `GET /dashboard/recent-requests` — 最近请求日志
- `CRUD /users` — 用户管理
- `CRUD /models/providers` — 供应商管理
- `CRUD /models/list` — 模型管理
- `CRUD /models/routing-rules` — 路由规则管理
- `GET/POST /billing/*` — 计费管理
- `GET /monitoring/*` — 运维监控
- `GET/PUT /settings/*` — 系统设置
- `POST /documents/generate/*` — 文档自动生成（PDF/Excel/Word/PPTX）

#### 管理平台前端（7 个页面）
- 登录页
- 仪表盘 — 统计卡片 + 趋势图 + 模型分布 + 最近请求
- 用户管理 — 列表 + 搜索 + 创建 + 状态管理
- 模型管理 — 供应商/模型/路由规则三 Tab
- 计费管理 — 收入概览 + CRED 供应 + 交易记录
- 运维监控 — 服务健康 + 实时指标 + 告警 + 日志
- 系统设置 — 速率/路由/定价/结算 + 审计日志

#### 基础设施
- PostgreSQL 16 数据库
- Redis 7 缓存
- Docker Compose 一键部署
- Alembic 数据库迁移
- Pre-commit hooks（black, flake8, prettier）
- 8 家种子供应商 + 15 个种子模型
- 文档自动化引擎（PDF/Excel/Word/PPTX 生成）

#### 核心能力
- **智能路由**：基于复杂度的自动模型选择
- **双模式认证**：JWT + API Key
- **CRED 虚拟货币**：统一计费体系
- **OpenAI 兼容**：可直接替换 OpenAI SDK 端点
- **流式响应**：SSE 格式的流式 Chat Completion

---

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)。
