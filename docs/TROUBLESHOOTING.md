# 故障排查

> 常见问题与解决方案。找不到你的问题？请提交 Issue。

---

## 目录

- [环境搭建问题](#环境搭建问题)
- [数据库问题](#数据库问题)
- [API 问题](#api-问题)
- [前端问题](#前端问题)
- [Docker 问题](#docker-问题)

---

## 环境搭建问题

### `make seed` 报错 "ModuleNotFoundError: No module named 'app'"

**原因**：Python 找不到项目模块。

**解决**：

```bash
cd backend
pip install -e .  # 以开发模式安装
# 或者设置 PYTHONPATH
export PYTHONPATH="${PYTHONPATH}:$(pwd)"
python -m app.scripts.seed
```

### pip install 报错 "asyncpg" 安装失败

**原因**：缺少 PostgreSQL 开发库。

**解决**：

```bash
# macOS
brew install postgresql

# Ubuntu/Debian
sudo apt-get install libpq-dev python3-dev

# 然后重新安装
pip install asyncpg
```

### 端口被占用

**现象**：`uvicorn` 启动时报 `Address already in use`

**解决**：

```bash
# 查找占用端口的进程
lsof -i :8001
lsof -i :8005
lsof -i :5173

# 终止进程
kill -9 <PID>
```

---

## 数据库问题

### 连接数据库失败

**现象**：`could not connect to server: Connection refused`

**排查步骤**：

```bash
# 1. 确认 PostgreSQL 容器在运行
docker compose ps postgres
# 期望看到 State: Up

# 2. 如果未运行，启动它
docker compose up -d postgres

# 3. 等待健康检查通过（约 5 秒）
docker compose ps postgres
# 期望看到 "(healthy)"

# 4. 测试连接
docker compose exec postgres psql -U maas -d maas_router -c "SELECT 1"
```

### 数据库表不存在

**现象**：`relation "users" does not exist`

**解决**：

```bash
# 运行种子数据（会自动创建表）
make seed

# 或手动运行迁移
cd backend && alembic upgrade head
```

### Alembic 迁移冲突

**现象**：`Multiple heads are present`

**解决**：

```bash
cd backend

# 查看分支
alembic branches

# 合并分支
alembic merge <head1> <head2> -m "merge branches"
```

---

## API 问题

### 401 Unauthorized

**常见原因**：

1. **未携带 Authorization Header**

```bash
# 错误
curl http://localhost:8001/v1/models

# 正确
curl -H "Authorization: Bearer <token>" http://localhost:8001/v1/models
```

2. **API Key 格式错误**

API Key 必须以 `sk-mr-` 开头。检查是否复制完整。

3. **JWT Token 过期**

默认 60 分钟过期。重新登录获取新 Token。

### 402 Insufficient Balance

**原因**：CRED 余额不足。

**解决**：通过管理后台为账户充值，或使用种子数据中的 demo 账户。

### 404 Model Not Found

```bash
# 查看可用模型列表
curl -H "Authorization: Bearer <token>" http://localhost:8001/v1/models

# 或使用 auto 模式
curl -X POST http://localhost:8001/v1/chat/completions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"model": "auto", "messages": [{"role": "user", "content": "Hello"}]}'
```

---

## 前端问题

### 管理平台白屏

**排查步骤**：

```bash
# 1. 确认 Admin Server 在运行
curl http://localhost:8005/health

# 2. 检查前端控制台错误（F12 → Console）

# 3. 确认 Vite 代理配置正确
# admin-platform/vite.config.ts 中 proxy 指向正确端口

# 4. 清除缓存并重启
cd admin-platform
rm -rf node_modules/.vite
npm run dev
```

### API 请求报 CORS 错误

**现象**：浏览器控制台显示 `Access-Control-Allow-Origin` 错误。

**解决**：

1. 确认前端请求的端口与后端 CORS 配置一致
2. 检查 `CORS_ORIGINS` 环境变量是否包含前端地址
3. 开发环境默认允许 `localhost:5173` 和 `localhost:3000`

### 页面数据显示为空

**排查步骤**：

```bash
# 1. 确认种子数据已初始化
docker compose exec api-server python -m app.scripts.seed

# 2. 确认 Admin Server API 返回数据
curl -H "Authorization: Bearer <token>" http://localhost:8005/api/admin/v1/dashboard/overview

# 3. 检查浏览器 Network 面板中的 API 请求状态
```

---

## Docker 问题

### 容器启动后立即退出

**排查**：

```bash
# 查看容器日志
docker compose logs api-server --tail=50

# 常见原因
# - 数据库未就绪（等待 postgres healthy）
# - 环境变量缺失
# - 端口冲突
```

### Docker 构建失败

```bash
# 清理缓存后重试
docker compose build --no-cache

# 检查 Dockerfile 路径是否正确
docker compose config
```

### postgres 容器健康检查失败

```bash
# 查看详细日志
docker compose logs postgres

# 常见原因
# - 端口冲突（5432 被占用）
# - 数据卷权限问题
# - 磁盘空间不足

# 重置数据库（危险，会删除数据）
docker compose down -v
docker compose up -d postgres
```

### 修改代码后不生效

**原因**：Docker 使用了 volume 挂载，但某些更改需要重启。

**解决**：

```bash
# API/Admin Server（代码挂载到容器，改 py 文件自动生效）
# 前端（Vite HMR 自动生效）

# 如果修改了依赖或 Dockerfile：
docker compose up -d --build

# 如果修改了环境变量：
docker compose down && docker compose up -d
```

---

## 提交 Issue 模板

如果你遇到的问题不在以上列表中，提交 Issue 时请包含以下信息：

```markdown
### 问题描述
<!-- 清晰描述问题现象 -->

### 复现步骤
1. 执行什么命令/操作
2. 观察到什么结果
3. 期望什么结果

### 环境信息
- OS: macOS 14 / Ubuntu 22.04
- Python 版本: 3.12.x
- Node 版本: 20.x
- Docker 版本: 24.x.x
- 分支/commit: main / abc1234

### 日志
<!-- 粘贴相关错误日志 -->
```
