# MaaS-Router — AI 推理聚合网关

> 多供应商 AI 模型统一接入与智能路由平台，帮助团队降低 40-60% 的推理成本。

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Python 3.12+](https://img.shields.io/badge/Python-3.12+-blue.svg)](https://www.python.org/)
[![FastAPI](https://img.shields.io/badge/FastAPI-0.115-009688.svg)](https://fastapi.tiangolo.com/)
[![React 19](https://img.shields.io/badge/React-19-61DAFB.svg)](https://react.dev/)

---

## 这是什么？

MaaS-Router 是一个 **AI 推理聚合网关**。当你对接多个 AI 模型供应商（OpenAI、DeepSeek、自建模型等）时，MaaS-Router 提供：

- **统一 API 入口** — OpenAI 兼容接口，一次接入即可使用所有供应商的模型
- **智能路由** — 根据请求复杂度自动选择最优模型，简单查询走便宜模型，复杂推理走强模型
- **统一计费** — CRED 虚拟货币体系，跨供应商统一计量和扣费
- **管理后台** — 用户管理、模型管理、计费监控、运维告警一站完成

## 快速开始

**前置条件**：Docker & Docker Compose 已安装

```bash
# 1. 克隆项目
git clone <repo-url> && cd maas-router

# 2. 启动基础设施（PostgreSQL + Redis）
docker compose up -d postgres redis

# 3. 初始化种子数据
make seed

# 4. 启动全部服务
docker compose up -d

# 5. 验证
curl http://localhost:8001/health  # API Server
curl http://localhost:8005/health  # Admin Server
open http://localhost:5173          # 管理平台
```

**演示账号**：admin@maas-router.com / admin123

## 项目结构

```
maas-router/
├── backend/                        # Python FastAPI 后端
│   ├── app/
│   │   ├── core/                   # 配置、数据库、安全、Redis
│   │   ├── models/                 # 11 个 SQLAlchemy 数据模型
│   │   ├── api_server/             # 用户端 API（OpenAI 兼容，14 个端点）
│   │   ├── admin_server/           # 管理后台 API（39 个端点）
│   │   ├── services/               # 文档生成等服务
│   │   ├── schemas/                # Pydantic 请求/响应模型
│   │   └── scripts/                # 种子数据脚本
│   ├── migrations/                 # Alembic 数据库迁移
│   ├── requirements.txt
│   └── Dockerfile
│
├── admin-platform/                 # React + TypeScript 管理前端
│   ├── src/
│   │   ├── components/layout/      # AdminLayout（侧边栏导航）
│   │   ├── pages/                  # 仪表盘、用户、模型、计费、监控、设置
│   │   └── services/               # API 调用层
│   ├── package.json
│   └── Dockerfile
│
├── docs/                           # 📚 技术文档
│   ├── INDEX.md                    # 文档导航
│   ├── ARCHITECTURE.md             # 系统架构
│   ├── API_REFERENCE.md            # API 参考
│   ├── DATABASE.md                 # 数据库设计
│   ├── DEVELOPMENT.md              # 开发指南
│   ├── DEPLOYMENT.md               # 部署指南
│   ├── CONFIGURATION.md            # 配置参考
│   └── TROUBLESHOOTING.md          # 故障排查
│
├── PRD/                            # 产品需求文档
├── GIT_WORKFLOW.md                 # Git 工作流规范
├── CONTRIBUTING.md                 # 贡献指南
├── docker-compose.yml
└── Makefile
```

## 技术栈

| 层级 | 技术 | 版本 |
|------|------|------|
| API 服务 | Python + FastAPI | 3.12 / 0.115 |
| 数据库 ORM | SQLAlchemy (async) | 2.0 |
| 数据库 | PostgreSQL | 16 |
| 缓存 | Redis | 7 |
| 认证 | JWT + API Key + bcrypt | — |
| 管理前端 | React + TypeScript + Tailwind | 19 / 5.6 / 3.4 |
| 图表 | Recharts | 2.15 |
| 构建工具 | Vite | 6 |
| 部署 | Docker + Docker Compose | — |

## 数据概览

| 类别 | 数量 |
|------|------|
| 数据库表 | 11 张 |
| API Server 端点 | 14 个 |
| Admin Server 端点 | 39 个 |
| 管理平台页面 | 7 个 |
| 种子供应商 | 8 家 |
| 种子模型 | 15 个 |

## 文档导航

| 文档 | 适合人群 | 说明 |
|------|----------|------|
| [文档索引](INDEX.md) | 所有人 | 文档导航和阅读路线 |
| [架构设计](ARCHITECTURE.md) | 架构师、Tech Lead | 系统架构与设计决策 |
| [API 参考](API_REFERENCE.md) | 前后端开发者 | 完整 API 文档与示例 |
| [数据库设计](DATABASE.md) | 后端开发者、DBA | 表结构与关系说明 |
| [开发指南](DEVELOPMENT.md) | 新加入的开发者 | 环境搭建与开发流程 |
| [部署指南](DEPLOYMENT.md) | DevOps、SRE | 生产环境部署方案 |
| [配置参考](CONFIGURATION.md) | 运维、开发者 | 完整配置项说明 |
| [故障排查](TROUBLESHOOTING.md) | 所有人 | 常见问题与解决方案 |
| [Git 工作流](GIT_WORKFLOW.md) | 全部开发者 | 分支策略与 Commit 规范 |
| [贡献指南](CONTRIBUTING.md) | 外部贡献者 | 如何参与贡献 |

## 核心概念

### 智能路由

MaaS-Router 的核心能力是**基于复杂度的智能路由**：

```
用户请求 "auto"
     │
     ▼
复杂度评分（Judge Agent）
     │
     ├── 简单 (< 4) → DeepSeek-V4 自建  (成本最低)
     ├── 中等 (4-7) → DeepSeek-V3      (性价比)
     ├── 较高 (7-9) → GPT-4o Mini      (能力适中)
     └── 高 (> 9)   → GPT-4o           (最强能力)
```

### 认证方式

- **用户端 API**：支持 API Key（`sk-mr-...`）和 JWT Token 双模式认证
- **管理后台**：JWT Token（登录获取），支持角色权限控制

### 计费模型

- 虚拟货币 **CRED**，按 token 消耗扣费
- 支持充值（topup）、消费（usage）、冻结（freeze）等交易类型
- 不同模型不同定价，路由决策中可见预估费用

## 环境变量速查

关键配置项通过环境变量注入，详见 [配置参考](CONFIGURATION.md)：

```bash
# 数据库
DATABASE_URL=postgresql+asyncpg://maas:password@localhost:5432/maas_router

# Redis
REDIS_URL=redis://localhost:6379/0

# JWT（生产环境务必更换）
JWT_SECRET_KEY=openssl rand -hex 32

# 服务端口
SERVICE_NAME=api-server
SERVICE_PORT=8001
```

## 许可

MIT © MaaS-Router Team
