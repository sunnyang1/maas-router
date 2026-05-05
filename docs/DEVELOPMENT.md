# 开发指南

> 从零搭建本地开发环境，适合新加入团队的开发者。

**预计阅读时间**：10 分钟 | **搭建时间**：约 15 分钟

---

## 目录

1. [环境要求](#1-环境要求)
2. [快速搭建（Docker 方式）](#2-快速搭建docker-方式)
3. [本地开发（非 Docker）](#3-本地开发非-docker)
4. [项目结构详解](#4-项目结构详解)
5. [常用开发命令](#5-常用开发命令)
6. [开发工作流](#6-开发工作流)
7. [扩展指南](#7-扩展指南)

---

## 1. 环境要求

| 工具 | 最低版本 | 用途 |
|------|----------|------|
| Python | 3.12+ | 后端运行环境 |
| Node.js | 20+ | 前端构建 |
| Docker | 24+ | 容器化部署和基础设施 |
| PostgreSQL | 16 | 主数据库 |
| Redis | 7 | 缓存和限流 |

---

## 2. 快速搭建（Docker 方式）

推荐使用 Docker Compose 一键启动全部服务：

```bash
# 1. 克隆项目
git clone <repo-url>
cd maas-router

# 2. 启动基础设施（数据库 + 缓存）
docker compose up -d postgres redis

# 3. 安装依赖
make install

# 4. 初始化数据
make seed

# 5. 启动全部服务
docker compose up -d

# 6. 验证服务
curl http://localhost:8001/health   # API Server
curl http://localhost:8005/health   # Admin Server
open http://localhost:5173           # 管理平台
```

> **提示**：首次启动需要拉取 Docker 镜像，可能需要几分钟。

---

## 3. 本地开发（非 Docker）

当你需要频繁修改代码时，建议本地运行后端和前端，只通过 Docker 运行数据库和 Redis：

```bash
# 1. 启动基础设施
docker compose up -d postgres redis

# 2. 安装后端依赖
cd backend
python -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate
pip install -r requirements.txt

# 3. 初始化种子数据
python -m app.scripts.seed

# 4. 启动 API Server（热重载）
uvicorn app.main:app --host 0.0.0.0 --port 8001 --reload

# 5. 新建终端，启动 Admin Server（热重载）
uvicorn app.admin_main:app --host 0.0.0.0 --port 8005 --reload

# 6. 新建终端，启动前端
cd admin-platform
npm install
npm run dev
```

或使用 Makefile 快捷命令：

```bash
make dev-api        # → API Server :8001
make dev-admin      # → Admin Server :8005
make dev-frontend   # → 管理平台 :5173
```

---

## 4. 项目结构详解

### 4.1 后端目录

```
backend/
├── app/
│   ├── core/                # 🔧 核心基础设施
│   │   ├── config.py        #    所有配置项（Pydantic Settings）
│   │   ├── database.py      #    异步数据库引擎 + 会话管理
│   │   ├── security.py      #    密码、JWT、API Key、认证依赖
│   │   └── redis.py         #    Redis 连接管理
│   │
│   ├── models/              # 📊 数据模型
│   │   ├── user.py          #    User
│   │   ├── team.py          #    Team, TeamMember
│   │   ├── api_key.py       #    ApiKey
│   │   ├── provider.py      #    Provider, Model
│   │   ├── billing.py       #    Balance, Transaction
│   │   └── routing.py       #    RoutingRule, AuditLog, RequestLog
│   │
│   ├── api_server/          # 🌐 用户端 API（OpenAI 兼容）
│   │   └── router.py        #    /v1/models, /v1/chat/completions...
│   │
│   ├── admin_server/        # 🔐 管理后台 API
│   │   ├── auth_admin.py    #    登录/登出
│   │   ├── dashboard.py     #    统计概览
│   │   ├── users.py         #    用户管理
│   │   ├── models_admin.py  #    模型管理
│   │   ├── billing_admin.py #    计费管理
│   │   ├── monitoring.py    #    运维监控
│   │   ├── settings.py      #    系统设置
│   │   └── documents.py     #    文档生成
│   │
│   ├── services/            # 🏗️ 业务服务
│   │   └── document_service.py
│   │
│   ├── schemas/             # 📋 请求/响应 Schema
│   │   └── chat.py
│   │
│   └── scripts/             # 🛠️ 工具脚本
│       └── seed.py          #    种子数据
│
├── migrations/              # Alembic 迁移
├── main.py                  # API Server 入口
├── admin_main.py            # Admin Server 入口
├── requirements.txt
└── Dockerfile
```

### 4.2 前端目录

```
admin-platform/src/
├── App.tsx                   # 路由配置
├── main.tsx                  # 入口文件
├── components/
│   └── layout/
│       └── AdminLayout.tsx   # 全局布局
├── pages/
│   ├── LoginPage.tsx
│   ├── dashboard/            # 仪表盘
│   ├── users/                # 用户管理
│   ├── models/               # 模型管理
│   ├── billing/              # 计费管理
│   ├── monitoring/           # 运维监控
│   └── settings/             # 系统设置
└── services/
    └── api.ts                # 统一 API 调用
```

---

## 5. 常用开发命令

```bash
# ── Docker ──
docker compose up -d             # 启动全部服务
docker compose down              # 停止全部服务
docker compose logs -f api-server # 查看 API Server 日志
docker compose restart api-server # 重启 API Server

# ── 后端 ──
make seed                        # 初始化数据库种子数据
make dev-api                     # 启动 API Server (热重载)
make dev-admin                   # 启动 Admin Server (热重载)

# ── 前端 ──
make dev-frontend                # 启动管理平台 (热重载)
cd admin-platform && npm run build  # 生产构建

# ── 数据库 ──
alembic revision --autogenerate -m "描述"  # 生成迁移
alembic upgrade head                        # 执行迁移
alembic downgrade -1                        # 回滚

# ── 工具 ──
make clean                       # 清理临时文件
make install                     # 安装全部依赖
```

---

## 6. 开发工作流

### 6.1 日常开发节奏

```bash
# 每天开始
git checkout main
git pull origin main
git checkout -b feat/my-feature

# 开发过程（频繁提交）
git add backend/app/models/new_model.py
git commit -m "feat(db): add NewModel table"

# 准备 PR
git fetch origin main
git rebase origin/main
git push --force-with-lease origin feat/my-feature
gh pr create --title "feat: add new model" --base main
```

详细流程见 [Git 工作流规范](GIT_WORKFLOW.md)。

### 6.2 数据库变更

当需要修改数据库结构时：

```bash
# 1. 修改模型文件（如 models/user.py）
# 2. 生成迁移文件
cd backend
alembic revision --autogenerate -m "add phone field to users"

# 3. 检查生成的迁移文件是否正确
# 4. 执行迁移
alembic upgrade head

# 5. 如需回滚
alembic downgrade -1
```

### 6.3 添加新的 API 端点

以后端为例：

```python
# 在 admin_server/xxx.py 中添加
router = APIRouter(tags=["YourFeature"])

@router.get("/your-feature")
async def your_endpoint(db: AsyncSession = Depends(get_db)):
    # 业务逻辑
    return {"data": "result"}

# 在 admin_main.py 中注册
from app.admin_server.xxx import router as your_router
app.include_router(your_router, prefix="/api/admin/v1")
```

### 6.4 前端添加新页面

```tsx
// 1. 在 src/pages/ 下创建新页面组件
// 2. 在 App.tsx 中添加路由
// 3. 在 AdminLayout.tsx 的侧边栏中添加导航项
// 4. 如需 API 调用，在 src/services/api.ts 中添加方法
```

---

## 7. 扩展指南

### 7.1 添加新的 AI 供应商

1. 在 `provider.py` 的种子数据中添加供应商信息
2. 在 `models` 表中注册该供应商的模型
3. 在 `routing_rules` 中配置路由规则
4. 重新运行 `make seed` 或通过管理后台 UI 添加

### 7.2 添加新的计费模式

1. 在 `Billing` 模型中添加新的计费字段
2. 在 `billing_admin.py` 中添加新的计费端点
3. 前端 `BillingPage.tsx` 中添加对应的 UI
4. 运行迁移 `alembic revision --autogenerate`

### 7.3 接入真实 AI 推理

当前 Demo 模式使用模拟响应。接入真实推理需修改 `api_server/router.py`：

```python
# 当前 (Demo):
response_content = _generate_mock_response(messages, resolved_model)

# 改为 (Production):
response_content = await call_provider_api(
    provider=resolved_provider,
    model=resolved_model,
    messages=messages,
    temperature=temperature,
    max_tokens=max_tokens,
)
```

---

## 故障排查

遇到问题？查看 [故障排查指南](TROUBLESHOOTING.md)。
