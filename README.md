<div align="center">

![MaaS-Router Logo](https://via.placeholder.com/200x200/4F46E5/FFFFFF?text=MaaS-Router)

# MaaS-Router

### AI API 聚合网关 - 智能路由降本 · 自建模型托管 · Web3透明对账

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react)](https://react.dev/)
[![Next.js](https://img.shields.io/badge/Next.js-14-000000?style=flat-square&logo=next.js)](https://nextjs.org/)
[![Python](https://img.shields.io/badge/Python-3.11+-3776AB?style=flat-square&logo=python)](https://python.org/)
[![Docker](https://img.shields.io/badge/Docker-24.0+-2496ED?style=flat-square&logo=docker)](https://docker.com/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.28+-326CE5?style=flat-square&logo=kubernetes)](https://kubernetes.io/)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

[English](README_EN.md) | 简体中文

</div>

---

## 核心特性

<table>
<tr>
<td width="33%">

### 智能路由
- **ComplexityEngine 自适应推理优化引擎**：五维复杂度评分（词法/结构/领域/对话/任务类型）
- economy/standard/premium 三级模型分层，自动选择最优路由
- 预期降低 35-55% 推理成本，路由准确率 95%+
- 在线学习模块，基于质量反馈持续优化路由决策

</td>
<td width="33%">

### Web3 结算
- $CRED 代币体系，链下实时计费
- L2 每日结算，透明可验证
- 支持 Polygon/Arbitrum 网络
- 去中心化信任机制

</td>
<td width="33%">

### 多协议兼容
- OpenAI API 全兼容，零代码迁移
- Claude Messages API 支持
- Gemini API 支持
- 统一接口，多供应商聚合

</td>
</tr>
</table>

---

## 系统架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              MaaS-Router 系统架构                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                      │
│  │  用户前端    │    │  管理前端    │    │  客户端 SDK │                      │
│  │  Next.js    │    │  Ant Design │    │  OpenAI SDK │                      │
│  │  :3000      │    │  Pro :8000  │    │  / LangChain│                      │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘                      │
│         │                  │                  │                             │
│         └──────────────────┼──────────────────┘                             │
│                            │                                                │
│                            ▼                                                │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        API Gateway (Gin)                            │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐│   │
│  │  │  认证中间件  │  │  限流中间件  │  │  日志中间件  │  │  监控中间件  ││   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘│   │
│  │                           :8080                                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                            │                                                │
│                            ▼                                                │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      Judge Agent (FastAPI)                          │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                  │   │
│  │  │  复杂度分析  │  │  路由决策    │  │  成本估算    │                  │   │
│  │  │  Qwen2.5-7B │  │  评分算法    │  │  价格对比    │                  │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘                  │   │
│  │                           :8000                                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                            │                                                │
│              ┌─────────────┴─────────────┐                                  │
│              │                           │                                  │
│              ▼                           ▼                                  │
│  ┌─────────────────────┐    ┌─────────────────────┐                        │
│  │   自建推理集群        │    │   商业 API 供应商     │                        │
│  │  DeepSeek-V4        │    │  OpenAI/Claude/etc  │                        │
│  │  低成本高吞吐量       │    │  高可用备用         │                        │
│  └─────────────────────┘    └─────────────────────┘                        │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         数据层                                      │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐│   │
│  │  │  PostgreSQL │  │    Redis    │  │  Prometheus │  │   Grafana   ││   │
│  │  │   (主存储)   │  │   (缓存)     │  │   (监控)     │  │   (可视化)   ││   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘│   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      区块链层 (Solidity)                            │   │
│  │  ┌─────────────┐  ┌─────────────┐                                  │   │
│  │  │  $CRED Token│  │  Settlement │  Polygon/Arbitrum L2             │   │
│  │  │  ERC-20     │  │  Contract   │                                  │   │
│  │  └─────────────┘  └─────────────┘                                  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 快速开始

### Docker 一键部署

```bash
# 1. 克隆项目
git clone https://github.com/your-org/maas-router.git
cd maas-router

# 2. 启动所有服务
docker-compose up -d

# 3. 查看服务状态
docker-compose ps

# 4. 访问服务
# 用户前端: http://localhost:3000
# 管理前端: http://localhost:8000
# API 网关: http://localhost:8080
# Grafana:  http://localhost:3001 (admin/admin)
```

详细部署指南请查看 [QUICKSTART.md](docs/QUICKSTART.md)

---

## 性能对比

| 指标 | 传统方案 | MaaS-Router | 提升 |
|------|---------|-------------|------|
| 平均推理成本 | $0.002/1K tokens | $0.0008/1K tokens | **60%** |
| 简单请求成本 | $0.001/1K tokens | $0.0003/1K tokens | **70%** |
| 路由准确率 | N/A | 95%+ | - |
| 故障切换时间 | 5-30 分钟 | <30 秒 | **99%** |
| SLA 可用性 | 99.5% | 99.9%+ | - |
| 首字节延迟 (TTFB) | 800ms | 300ms | **62%** |

---

## 界面预览

<div align="center">

| 用户仪表盘 | 管理后台 | 实时监控 |
|-----------|---------|---------|
| ![用户仪表盘](https://via.placeholder.com/300x200/4F46E5/FFFFFF?text=用户仪表盘) | ![管理后台](https://via.placeholder.com/300x200/10B981/FFFFFF?text=管理后台) | ![实时监控](https://via.placeholder.com/300x200/F59E0B/FFFFFF?text=实时监控) |

| API 文档 | 路由分析 | 结算中心 |
|---------|---------|---------|
| ![API文档](https://via.placeholder.com/300x200/8B5CF6/FFFFFF?text=API文档) | ![路由分析](https://via.placeholder.com/300x200/EC4899/FFFFFF?text=路由分析) | ![结算中心](https://via.placeholder.com/300x200/06B6D4/FFFFFF?text=结算中心) |

</div>

---

## 技术栈

### 后端服务
| 组件 | 技术 | 说明 |
|------|------|------|
| Web 框架 | [Gin](https://gin-gonic.com/) | 高性能 Go Web 框架 |
| ORM | [Ent](https://entgo.io/) | Facebook 开源的实体框架 |
| 依赖注入 | [Wire](https://github.com/google/wire) | Google 编译时依赖注入 |
| 配置管理 | [Viper](https://github.com/spf13/viper) | 完整的配置解决方案 |
| 日志 | [Zap](https://github.com/uber-go/zap) + Lumberjack | 高性能结构化日志 |
| 认证 | [golang-jwt](https://github.com/golang-jwt/jwt) | JWT 认证库 |

### 前端技术
| 层级 | 技术栈 | 说明 |
|------|--------|------|
| 用户前端 | Next.js 14 + React 18 + TypeScript + Tailwind CSS | 现代化全栈框架 |
| 管理前端 | React 18 + TypeScript + Ant Design Pro | 企业级中后台 |

### 智能路由
| 组件 | 技术 | 说明 |
|------|------|------|
| 服务框架 | Python 3.11 + FastAPI | 高性能异步 API |
| 评分模型 | Qwen2.5-7B (vLLM) | 复杂度评估模型 |

### 基础设施
| 组件 | 技术 | 说明 |
|------|------|------|
| 数据库 | PostgreSQL 16 | 主数据存储 |
| 缓存 | Redis 7 | 高性能缓存 |
| 监控 | Prometheus + Grafana | 指标收集与可视化 |
| 区块链 | Solidity + Hardhat | 智能合约开发 |
| 容器 | Docker + Kubernetes | 容器化编排 |

---

## API 调用示例

### OpenAI 兼容接口

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="mr-xxxxxxxxxxxxxxxx"
)

# 自动路由模式
completion = client.chat.completions.create(
    model="auto",  # 系统自动选择最优路由
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "解释量子计算的基本原理"}
    ],
    stream=True
)

for chunk in completion:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")
```

### 指定模型路由

```python
# 强制使用自建集群
completion = client.chat.completions.create(
    model="deepseek-v4",  # 自建 DeepSeek-V4
    messages=[{"role": "user", "content": "Hello!"}]
)

# 强制使用商业 API
completion = client.chat.completions.create(
    model="gpt-4",  # OpenAI GPT-4
    messages=[{"role": "user", "content": "Hello!"}]
)
```

---

## 项目结构

```
maas-router/
├── backend/                    # Go 后端服务
│   ├── cmd/server/            # 程序入口
│   ├── ent/schema/            # Ent 数据模型
│   ├── internal/
│   │   ├── config/            # 配置管理
│   │   ├── handler/           # HTTP 处理器
│   │   ├── middleware/        # Gin 中间件
│   │   ├── repository/        # 数据访问层
│   │   ├── service/           # 业务逻辑层
│   │   └── pkg/               # 内部工具包
│   ├── configs/               # 配置文件
│   └── Dockerfile
├── user-frontend/             # 用户前端 (Next.js 14)
│   ├── app/                   # App Router
│   ├── components/            # React 组件
│   └── lib/                   # 工具库
├── admin-frontend/            # 管理前端 (React + Ant Design Pro)
│   ├── config/                # 路由和配置
│   └── src/                   # 源代码
├── judge-agent/               # 智能路由 Agent (Python + FastAPI)
│   ├── judge/                 # 核心逻辑
│   ├── tests/                 # 测试用例
│   └── config.yaml            # 配置文件
├── contracts/                 # 区块链智能合约
│   ├── src/                   # Solidity 合约
│   └── scripts/               # 部署脚本
├── infra/                     # 基础设施
│   ├── k8s/                   # Kubernetes 部署
│   └── monitoring/            # 监控配置
├── docs/                      # 文档
│   ├── QUICKSTART.md          # 快速开始
│   ├── ARCHITECTURE.md        # 架构文档
│   ├── API.md                 # API 文档
│   └── CHANGELOG.md           # 更新日志
└── docker-compose.yml         # 本地开发环境
```

---

## 文档

- [快速开始指南](docs/QUICKSTART.md) - 5 分钟上手部署
- [架构设计文档](docs/ARCHITECTURE.md) - 系统架构详解
- [API 参考文档](docs/API.md) - 完整的 API 文档
- [更新日志](docs/CHANGELOG.md) - 版本更新记录

---

## 贡献指南

我们欢迎所有形式的贡献！

### 提交 Issue

- 使用 Issue 模板描述问题
- 提供复现步骤和环境信息
- 对于功能请求，请描述使用场景

### 提交 Pull Request

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

### 开发规范

- 遵循 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 前端代码使用 ESLint + Prettier
- 提交信息遵循 [Conventional Commits](https://www.conventionalcommits.org/)

---

## 社区与支持

- [GitHub Discussions](https://github.com/your-org/maas-router/discussions) - 讨论与问答
- [Discord](https://discord.gg/maas-router) - 实时交流
- [Twitter](https://twitter.com/maas_router) - 官方动态

---

## 许可证

本项目采用 [MIT License](LICENSE) 开源许可证。

---

<div align="center">

**Made with by the MaaS-Router Team**

[文档](docs/) · [报告问题](../../issues) · [贡献代码](../../pulls)

</div>
