# MaaS-Router 快速开始指南

本指南将帮助您在 5 分钟内完成 MaaS-Router 的部署和配置。

---

## 目录

- [环境要求](#环境要求)
- [Docker 一键部署](#docker-一键部署)
- [初始配置向导](#初始配置向导)
- [验证安装](#验证安装)
- [常见问题](#常见问题)

---

## 环境要求

### 最低配置

| 组件 | 要求 |
|------|------|
| Docker | 24.0+ |
| Docker Compose | 2.20+ |
| CPU | 4 核 |
| 内存 | 8 GB |
| 磁盘 | 50 GB |

### 推荐配置

| 组件 | 要求 |
|------|------|
| Docker | 24.0+ |
| Docker Compose | 2.20+ |
| CPU | 8 核 |
| 内存 | 16 GB |
| 磁盘 | 100 GB SSD |
| GPU | NVIDIA GPU (用于 Judge Agent) |

### 支持的操作系统

- Ubuntu 20.04/22.04/24.04 LTS
- CentOS 7/8
- Debian 11/12
- macOS 12+ (Apple Silicon/Intel)
- Windows 10/11 (WSL2)

---

## Docker 一键部署

### 1. 克隆项目

```bash
# 克隆代码仓库
git clone https://github.com/your-org/maas-router.git
cd maas-router

# 查看项目结构
ls -la
```

### 2. 配置环境变量

```bash
# 复制环境变量模板
cp .env.example .env

# 编辑 .env 文件
nano .env
```

**最小化配置示例：**

```env
# 数据库配置
POSTGRES_USER=maas_user
POSTGRES_PASSWORD=your_secure_password
POSTGRES_DB=maas_router

# JWT 密钥 (生产环境请使用强密码)
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production

# API 密钥 (用于管理员账号)
ADMIN_API_KEY=mr-admin-xxxxxxxxxxxxxxxx
```

### 3. 启动基础服务

```bash
# 启动数据库和缓存
docker-compose up -d postgres redis

# 等待数据库就绪 (约 10-20 秒)
docker-compose logs -f postgres
# 当看到 "database system is ready to accept connections" 时按 Ctrl+C
```

### 4. 启动核心服务

```bash
# 启动后端 API 网关
docker-compose up -d backend

# 启动前端服务
docker-compose up -d user-frontend admin-frontend

# 启动监控服务 (可选)
docker-compose up -d prometheus grafana
```

### 5. 启动智能路由 Agent (可选)

> 注意：Judge Agent 需要 GPU 支持，如果没有 GPU 可以跳过此步骤，系统将使用默认路由策略。

```bash
# 启动 Judge Agent (需要配置外部模型服务)
docker-compose --profile with-judge up -d judge-agent
```

### 6. 查看服务状态

```bash
# 查看所有服务状态
docker-compose ps

# 预期输出：
# NAME                    IMAGE                   STATUS          PORTS
# maas-postgres          postgres:16-alpine      Up 2 minutes    0.0.0.0:5432->5432/tcp
# maas-redis             redis:7-alpine          Up 2 minutes    0.0.0.0:6379->6379/tcp
# maas-backend           maas-router-backend     Up 1 minute     0.0.0.0:8080->8080/tcp
# maas-user-frontend     maas-router-user-fe     Up 1 minute     0.0.0.0:3000->3000/tcp
# maas-admin-frontend    maas-router-admin-fe    Up 1 minute     0.0.0.0:8000->8000/tcp
# maas-prometheus        prom/prometheus         Up 1 minute     0.0.0.0:9090->9090/tcp
# maas-grafana           grafana/grafana         Up 1 minute     0.0.0.0:3001->3000/tcp
```

---

## 初始配置向导

### 1. 访问管理后台

打开浏览器访问 http://localhost:8000

### 2. 创建管理员账号

```bash
# 使用命令行创建管理员
docker-compose exec backend go run ./cmd/cli admin create \
  --email admin@example.com \
  --password your_admin_password \
  --name "系统管理员"
```

### 3. 配置 API 供应商

登录管理后台后，进入 "供应商管理" 页面：

1. **添加自建集群 (DeepSeek-V4)**
   - 名称：DeepSeek-V4 自建集群
   - 类型：自建
   - API 地址：http://your-deepseek-endpoint:8000/v1
   - API 密钥：your-api-key
   - 模型列表：deepseek-v4

2. **添加商业供应商 (OpenAI)**
   - 名称：OpenAI
   - 类型：商业
   - API 地址：https://api.openai.com/v1
   - API 密钥：sk-xxxxxxxx
   - 模型列表：gpt-4, gpt-4-turbo, gpt-3.5-turbo

### 4. 配置路由规则

进入 "路由规则" 页面，配置默认路由策略：

```yaml
# 简单请求路由 (评分 < 0.4)
simple_requests:
  target: deepseek-v4
  max_tokens: 2048
  cost_priority: high

# 中等复杂度请求 (0.4 <= 评分 < 0.7)
medium_requests:
  target: deepseek-v4
  fallback: gpt-3.5-turbo
  max_tokens: 4096

# 复杂请求 (评分 >= 0.7)
complex_requests:
  target: gpt-4
  max_tokens: 8192
  quality_priority: high
```

### 5. 创建用户 API Key

1. 进入 "用户管理" 页面
2. 创建新用户或选择现有用户
3. 点击 "生成 API Key"
4. 复制生成的 API Key (格式：`mr-xxxxxxxxxxxxxxxx`)

---

## 验证安装

### 1. 检查服务健康状态

```bash
# 检查后端健康端点
curl http://localhost:8080/health

# 预期响应：
{
  "status": "healthy",
  "version": "v1.0.0",
  "services": {
    "database": "connected",
    "redis": "connected",
    "judge_agent": "connected"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 2. 测试 API 调用

```bash
# 使用 curl 测试 (替换为你的 API Key)
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer mr-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "auto",
    "messages": [
      {"role": "user", "content": "Hello, MaaS-Router!"}
    ]
  }'
```

### 3. 使用 Python SDK 测试

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="mr-your-api-key"
)

# 测试简单请求
response = client.chat.completions.create(
    model="auto",
    messages=[{"role": "user", "content": "Hello!"}]
)
print(response.choices[0].message.content)

# 测试流式响应
stream = client.chat.completions.create(
    model="auto",
    messages=[{"role": "user", "content": "讲个笑话"}],
    stream=True
)
for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")
```

### 4. 验证监控数据

1. 访问 http://localhost:3001 (Grafana)
2. 使用默认账号登录：admin/admin
3. 查看 "MaaS-Router Dashboard" 仪表盘
4. 确认能看到请求量、延迟、成本等指标

---

## 常见问题

### Q1: 数据库连接失败

**现象：**
```
error: failed to connect to database
```

**解决方案：**
```bash
# 1. 检查 PostgreSQL 容器状态
docker-compose ps postgres

# 2. 查看数据库日志
docker-compose logs postgres

# 3. 重启数据库服务
docker-compose restart postgres

# 4. 等待数据库完全启动后重启后端
docker-compose restart backend
```

### Q2: Judge Agent 无法连接

**现象：**
```
warning: judge agent unavailable, using default routing
```

**解决方案：**
```bash
# 1. 检查 Judge Agent 是否启动
docker-compose ps judge-agent

# 2. 如果没有启动，使用 with-judge profile 启动
docker-compose --profile with-judge up -d judge-agent

# 3. 或者修改配置使用外部 Judge Agent
# 编辑 backend/configs/config.yaml
judge_agent:
  url: "http://your-judge-agent:8000"
```

### Q3: 前端无法访问后端 API

**现象：**
页面显示 "Network Error" 或 "无法连接到服务器"

**解决方案：**
```bash
# 1. 检查后端服务状态
docker-compose logs backend

# 2. 检查 CORS 配置
curl -I http://localhost:8080/health \
  -H "Origin: http://localhost:3000"

# 3. 确认环境变量配置正确
cat .env | grep API_URL

# 4. 重启前端服务
docker-compose restart user-frontend admin-frontend
```

### Q4: API 调用返回 401 Unauthorized

**现象：**
```json
{
  "error": "Unauthorized",
  "message": "Invalid API key"
}
```

**解决方案：**
```bash
# 1. 确认 API Key 格式正确 (mr- 开头)
# 2. 检查请求头格式
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer mr-your-actual-api-key"

# 3. 在管理后台重新生成 API Key
# 4. 检查 API Key 是否已过期
```

### Q5: 模型路由不正确

**现象：**
简单请求被路由到昂贵的商业 API

**解决方案：**
```bash
# 1. 检查 Judge Agent 日志
docker-compose logs judge-agent

# 2. 查看路由决策日志
docker-compose logs backend | grep "routing_decision"

# 3. 调整路由阈值
# 在管理后台 "路由规则" 页面修改评分阈值

# 4. 手动测试 Judge Agent
curl http://localhost:8000/analyze \
  -H "Content-Type: application/json" \
  -d '{"content": "Hello, how are you?"}'
```

### Q6: 如何查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f backend
docker-compose logs -f judge-agent

# 查看最近 100 行日志
docker-compose logs --tail=100 backend
```

### Q7: 如何更新到最新版本

```bash
# 1. 拉取最新代码
git pull origin main

# 2. 重新构建镜像
docker-compose build --no-cache

# 3. 重启服务
docker-compose up -d

# 4. 执行数据库迁移 (如有需要)
docker-compose exec backend go run ./cmd/cli migrate up
```

### Q8: 如何完全重置环境

```bash
# 1. 停止所有服务
docker-compose down

# 2. 删除数据卷 (警告：将丢失所有数据)
docker-compose down -v

# 3. 重新启动
docker-compose up -d

# 4. 重新执行初始配置向导
```

---

## 下一步

- [架构设计文档](ARCHITECTURE.md) - 深入了解系统架构
- [API 参考文档](API.md) - 查看完整的 API 文档
- [更新日志](CHANGELOG.md) - 了解最新功能和修复

---

## 获取帮助

如果以上解决方案无法解决您的问题，请：

1. 查看 [GitHub Issues](https://github.com/your-org/maas-router/issues) 是否有类似问题
2. 创建新的 Issue，包含：
   - 问题描述
   - 复现步骤
   - 环境信息 (OS, Docker 版本等)
   - 相关日志

---

<div align="center">

**[返回首页](../README.md)**

</div>
