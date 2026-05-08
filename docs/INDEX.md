# MaaS-Router 技术文档

欢迎来到 MaaS-Router 技术文档中心。本文档体系按角色和场景组织，帮助不同背景的团队成员快速找到需要的信息。

---

## 按角色导航

### 🆕 新加入的开发者

1. [README.md](README.md) — 项目概览，30 秒了解项目
2. [开发指南](DEVELOPMENT.md) — 搭建本地开发环境
3. [架构设计](ARCHITECTURE.md) — 理解系统如何工作
4. [Git 工作流](GIT_WORKFLOW.md) — 团队协作规范

### 🔧 后端开发者

1. [架构设计](ARCHITECTURE.md) — 后端模块划分与设计决策
2. [API 参考](API_REFERENCE.md) — 所有 API 端点文档
3. [数据库设计](DATABASE.md) — 表结构与关系
4. [配置参考](CONFIGURATION.md) — 所有配置项说明
5. [故障排查](TROUBLESHOOTING.md) — 常见后端问题

### 🎨 前端开发者

1. [开发指南 — 前端部分](DEVELOPMENT.md#前端开发) — 前端环境与项目结构
2. [API 参考 — Admin Server](API_REFERENCE.md#admin-server-api) — 管理后台 API 接口
3. [部署指南](DEPLOYMENT.md) — 前端构建与部署

### 🏗️ 架构师 / Tech Lead

1. [架构设计](ARCHITECTURE.md) — 核心架构决策与权衡
2. [数据库设计](DATABASE.md) — 数据模型设计
3. [PRD 文档](../PRD/) — 产品需求与路线图

### 🚀 DevOps / SRE

1. [部署指南](DEPLOYMENT.md) — 生产环境部署方案
2. [配置参考](CONFIGURATION.md) — 环境变量与安全配置
3. [故障排查](TROUBLESHOOTING.md) — 运维常见问题
4. `docker-compose.yml` — 容器编排配置（见项目根目录）

---

## 按场景导航

| 场景 | 推荐文档 |
|------|---------|
| 搭建本地开发环境 | [开发指南](DEVELOPMENT.md) |
| 理解用户端 API 怎么用 | [API 参考 — API Server](API_REFERENCE.md#api-server-用户端) |
| 理解管理后台 API 怎么用 | [API 参考 — Admin Server](API_REFERENCE.md#admin-server-api) |
| 新增一个数据库表 | [开发指南 — 数据库变更](DEVELOPMENT.md#数据库变更) |
| 部署到生产环境 | [部署指南](DEPLOYMENT.md) |
| 排查连接不上数据库 | [故障排查 — 数据库](TROUBLESHOOTING.md#数据库问题) |
| 理解智能路由如何工作 | [架构设计 — 路由引擎](ARCHITECTURE.md#3-路由引擎) |
| 添加新的 AI 供应商 | [开发指南 — 扩展指南](DEVELOPMENT.md#扩展指南) |
| 设置 CI/CD | [部署指南 — CI/CD](DEPLOYMENT.md#cicd-集成) |
| 贡献代码 | [贡献指南](CONTRIBUTING.md) |

---

## 文档规范

本文档体系遵循以下规范：

- **Divio 文档系统**：区分教程（tutorial）、操作指南（how-to）、参考（reference）、解释（explanation）
- **逐段可独立阅读**：每个文档自成一体，同时包含清晰的交叉引用
- **代码示例可运行**：所有示例代码经过验证
- **版本同步**：文档版本与代码版本保持同步

## 文档反馈

发现文档错误或有改进建议？请通过以下方式反馈：

- 提交 Issue：描述问题所在文档和具体错误
- 提交 PR：直接修改文档（参考 [贡献指南](CONTRIBUTING.md)）
