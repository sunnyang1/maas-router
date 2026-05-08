# 更新日志

本文件记录 MaaS-Router 项目的所有重要变更。

---

## 版本号规范

本项目遵循 [语义化版本控制 2.0.0](https://semver.org/lang/zh-CN/) 规范。

版本号格式：`主版本号.次版本号.修订号`

- **主版本号 (MAJOR)**：不兼容的 API 修改
- **次版本号 (MINOR)**：向下兼容的功能新增
- **修订号 (PATCH)**：向下兼容的问题修复

### 预发布版本标识

- `alpha`：内部测试版本，不稳定
- `beta`：公开测试版本，功能基本完整
- `rc` (Release Candidate)：发布候选版本，即将正式发布

示例：`v1.0.0-beta.1`、`v1.1.0-rc.1`

---

## [v1.1.0] - 2026-05-08

### 新增 (Added)
- **ComplexityEngine 自适应推理优化引擎**：基于多信号特征提取的请求复杂度分析系统，支持词法、结构、领域、对话、任务类型五维评分
- **智能路由分级**：economy/standard/premium 三级模型分层，根据请求复杂度自动选择最优模型，预期降低 35-55% 推理成本
- **复杂度分析 API**：POST /v1/complexity/analyze、GET /v1/complexity/stats、POST /v1/complexity/feedback、GET /v1/complexity/tiers
- **在线学习模块**：基于质量反馈的自适应阈值调整，持续优化路由决策
- **Dashboard 智能路由概览**：新增智能路由次数、成本节省比例、路由准确率统计卡片
- **Dashboard 模型使用分布图**：新增 economy/standard/premium 使用分布柱状图
- **Playground 模型选择器**：支持 7 个模型的手动选择（Auto/DeepSeek-V4-Pro/Flash/Claude Sonnet 4/Opus 4/GPT-4.1/Mini）
- **Playground 复杂度分析面板**：实时展示请求复杂度评分、推荐模型、成本节省比例
- **Playground API Key 选择器**：顶部工具栏支持切换 API Key
- **Playground 动态费用计算**：基于模型 tier 的实时费用估算
- **API Keys 使用统计**：每个密钥展示本月调用次数、Token 消耗、费用
- **API Keys 权限配置**：创建密钥时支持模型白名单、IP 白名单、RPM 限制
- **API Keys 健康状态指示器**：绿/黄/红三级状态标签
- **API Keys 批量操作**：支持批量启用/禁用
- **生产环境配置模板**：新增 config.production.yaml 和 .env.example
- **配置验证机制**：生产环境下 JWT Secret 为空时拒绝启动

### 变更 (Changed)
- **统一 ORM**：移除 GORM 死依赖，全面使用 Ent ORM
- **CORS 默认配置**：AllowOrigins 从通配符改为 localhost 白名单，AllowCredentials 默认关闭
- **JWT Secret 默认值**：从硬编码占位符改为空字符串，强制生产环境配置
- **数据库密码默认值**：从硬编码改为空字符串，强制通过环境变量配置

### 修复 (Fixed)
- 修复 CORS AllowOrigins=["*"] 与 AllowCredentials=true 的规范矛盾
- 修复 Playground 费用计算硬编码问题
- 修复 API Keys 页面类型安全问题

### 安全 (Security)
- 新增 Config.Validate() 生产环境安全检查
- 新增 .env.example 环境变量模板
- 新增 config.production.yaml 生产配置模板
- .gitignore 新增 *.pem 和 config.production.yaml 规则

---

## [v1.0.0] - 2024-01-15

### 初始版本发布

MaaS-Router 首个正式版本，提供完整的 AI API 聚合网关功能。

### 新增功能

#### 核心功能
- **智能路由系统**：基于 Qwen2.5-7B 的请求复杂度评分和自动路由
- **OpenAI 兼容 API**：完全兼容 OpenAI API 格式，零代码迁移
- **多供应商支持**：支持自建 DeepSeek-V4 集群和商业 API (OpenAI、Anthropic 等)
- **故障自动切换**：主供应商故障时 <30 秒自动切换，保障 99.9%+ SLA
- **实时计费系统**：精确到 token 级别的实时计费和成本追踪

#### Web3 功能
- **$CRED 代币体系**：链下实时计费 + L2 每日结算
- **智能合约**：ERC-20 代币合约和结算合约
- **透明对账**：所有计费数据上链，可验证可追溯
- **多链支持**：支持 Polygon 和 Arbitrum L2 网络

#### 管理功能
- **用户管理**：完整的用户注册、认证、权限管理
- **API Key 管理**：支持多 Key、限流、有效期控制
- **供应商管理**：可视化配置多个 AI 供应商
- **路由规则管理**：灵活的路由策略配置
- **使用统计**：详细的调用日志和成本分析

#### 前端界面
- **用户前台** (Next.js 14)：现代化的用户仪表盘
- **管理后台** (Ant Design Pro)：功能完善的管理界面
- **实时监控**：Grafana 仪表盘，可视化系统指标

#### 运维功能
- **Docker 部署**：一键 Docker Compose 部署
- **Kubernetes 支持**：完整的 K8s 部署配置
- **监控告警**：Prometheus + Grafana 监控体系
- **日志收集**：结构化日志，支持集中式收集

### 技术栈

| 层级 | 技术 |
|------|------|
| 后端网关 | Go 1.22 + Gin + Ent + Wire |
| 智能路由 | Python 3.11 + FastAPI + Qwen2.5-7B |
| 用户前端 | Next.js 14 + React 18 + Tailwind CSS |
| 管理前端 | React 18 + Ant Design Pro |
| 数据库 | PostgreSQL 16 + Redis 7 |
| 区块链 | Solidity + Hardhat + Polygon |
| 监控 | Prometheus + Grafana |
| 部署 | Docker + Kubernetes |

### API 端点

#### 网关 API (OpenAI 兼容)
- `POST /v1/chat/completions` - 聊天完成
- `GET /v1/models` - 模型列表
- `POST /v1/embeddings` - 文本嵌入
- `POST /v1/images/generations` - 图像生成

#### 管理 API
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/register` - 用户注册
- `GET /api/v1/user/profile` - 用户信息
- `GET /api/v1/keys` - API Key 管理
- `GET /api/v1/usage` - 使用统计
- `GET /api/v1/admin/dashboard/stats` - 仪表盘数据
- `GET /api/v1/admin/providers` - 供应商管理
- `GET /api/v1/admin/router-rules` - 路由规则

### 性能指标

| 指标 | 数值 |
|------|------|
| 路由准确率 | 95%+ |
| 平均成本降低 | 40-60% |
| 故障切换时间 | <30 秒 |
| SLA 可用性 | 99.9%+ |
| 首字节延迟 (TTFB) | 300ms |
| Judge Agent 响应时间 | ~100ms |

### 文档

- [README.md](../README.md) - 项目介绍和快速开始
- [QUICKSTART.md](QUICKSTART.md) - 详细部署指南
- [ARCHITECTURE.md](ARCHITECTURE.md) - 系统架构设计
- [API.md](API.md) - 完整 API 文档

### 已知问题

1. Judge Agent 在极高并发下可能出现响应延迟
2. Web3 结算在 Gas 费用波动时可能需要手动调整
3. 部分旧版浏览器可能存在前端兼容性问题

### 后续计划

#### v1.2.0 (计划中)
- [ ] 多模态模型支持 (图像理解)
- [ ] 批量推理优化
- [ ] 更细粒度的权限控制
- [ ] 移动端 App

#### v2.0.0 (规划中)
- [ ] 分布式架构，支持多区域部署
- [ ] 模型微调服务集成
- [ ] 企业级 SSO 集成
- [ ] AI Agent 工作流编排

---

## 版本历史

| 版本 | 发布日期 | 说明 |
|------|----------|------|
| v1.1.0 | 2026-05-08 | ComplexityEngine 自适应推理优化引擎，智能路由分级，Dashboard/Playground 增强 |
| v1.0.0 | 2024-01-15 | 初始版本发布 |

---

## 如何升级

### 从 v0.x 升级到 v1.0.0

v1.0.0 是首个正式版本，与之前的开发版本存在不兼容的变更。

**升级步骤：**

1. **备份数据**
   ```bash
   # 备份数据库
   docker-compose exec postgres pg_dump -U maas_user maas_router > backup.sql
   ```

2. **拉取新版本代码**
   ```bash
   git fetch origin
   git checkout v1.0.0
   ```

3. **更新配置文件**
   ```bash
   # 对比配置文件变更
   diff backend/configs/config.yaml backend/configs/config.yaml.example
   
   # 手动合并配置
   nano backend/configs/config.yaml
   ```

4. **执行数据库迁移**
   ```bash
   docker-compose run --rm backend go run ./cmd/cli migrate up
   ```

5. **重启服务**
   ```bash
   docker-compose down
   docker-compose up -d
   ```

6. **验证升级**
   ```bash
   curl http://localhost:8080/health
   ```

### 小版本升级 (v1.0.x)

小版本升级通常只包含问题修复，可以直接升级：

```bash
# 拉取最新代码
git pull origin main

# 重新构建
docker-compose build --no-cache

# 重启服务
docker-compose up -d
```

---

## 贡献记录

感谢以下贡献者为 v1.0.0 版本做出的贡献：

- 核心开发团队
- 测试团队
- 文档贡献者
- 社区反馈者

---

## 安全公告

### 安全更新

| 版本 | 日期 | 安全修复 |
|------|------|----------|
| v1.1.0 | 2026-05-08 | 生产环境安全检查、CORS 修复、配置验证 |
| v1.0.0 | 2024-01-15 | 初始安全基线 |

### 报告安全问题

如发现安全漏洞，请通过以下方式报告：

- Email: security@maas-router.com
- 不要公开披露安全漏洞
- 我们将在 48 小时内响应

---

## 废弃功能

暂无

---

## 迁移指南

### 从其他网关迁移

#### 从 Kong 迁移

Kong 用户迁移到 MaaS-Router 的步骤：

1. 导出 Kong 配置
2. 转换 API Key 格式
3. 配置供应商信息
4. 更新客户端 base_url

#### 从自研网关迁移

1. 评估现有 API 兼容性
2. 设计迁移方案
3. 灰度切换流量
4. 监控迁移过程

---

<div align="center">

**[返回首页](../README.md)** · **[快速开始](QUICKSTART.md)** · **[API 文档](API.md)**

</div>
