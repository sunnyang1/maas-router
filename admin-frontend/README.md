# MaaS-Router Admin Frontend

MaaS-Router 管理前端 - 基于 React 18 + TypeScript + Ant Design Pro 构建的 LLM 路由管理系统。

## 功能特性

- **管理仪表盘** - 实时监控系统运行状态、请求趋势、延迟分布
- **用户管理** - 用户列表、详情查看、状态管理、配额控制
- **API Key 管理** - 创建、撤销、权限控制 API 访问密钥
- **供应商管理** - 配置 OpenAI、Anthropic、Azure 等 LLM 供应商
- **模型管理** - 管理支持的模型、配置定价策略和能力参数
- **计费管理** - 账单记录、用户充值、消费统计
- **路由规则** - 智能路由配置、负载均衡策略
- **监控告警** - 实时监控、告警规则配置
- **系统设置** - 系统参数配置、运行状态查看

## 技术栈

- React 18
- TypeScript 5
- Ant Design 5
- Ant Design Pro Components
- UmiJS Max
- ECharts
- Day.js

## 项目结构

```
maas-router/admin-frontend/
├── config/
│   ├── config.ts          # UmiJS 配置文件
│   └── routes.ts          # 路由配置
├── src/
│   ├── app.tsx            # 应用入口
│   ├── models/
│   │   └── user.ts        # 用户数据模型
│   ├── services/
│   │   └── api.ts         # API 服务
│   ├── pages/
│   │   ├── dashboard/     # 仪表盘
│   │   ├── users/         # 用户管理
│   │   ├── api-keys/      # API Key 管理
│   │   ├── providers/     # 供应商管理
│   │   ├── models/        # 模型管理
│   │   ├── billing/       # 计费管理
│   │   ├── routing/       # 路由规则
│   │   ├── monitoring/    # 监控告警
│   │   ├── system/        # 系统设置
│   │   ├── User/Login/    # 登录页面
│   │   ├── Welcome/       # 欢迎页面
│   │   └── 404.tsx        # 404 页面
│   └── access.ts          # 权限控制
├── package.json
├── tsconfig.json
└── README.md
```

## 快速开始

### 环境要求

- Node.js >= 18.0.0
- npm >= 9.0.0

### 安装依赖

```bash
cd /data/user/work/maas-router/admin-frontend
npm install
```

### 启动开发服务器

```bash
npm run dev
```

默认访问地址：http://localhost:8000

### 构建生产版本

```bash
npm run build
```

## 配置说明

### 代理配置

在 `config/config.ts` 中配置后端 API 代理：

```typescript
proxy: {
  '/api': {
    target: 'http://localhost:8080',
    changeOrigin: true,
    pathRewrite: { '^/api': '' },
  },
},
```

### 路由配置

路由定义在 `config/routes.ts` 中，支持嵌套路由和权限控制。

## API 接口

所有 API 接口定义在 `src/services/api.ts` 中，包括：

- 认证相关：登录、登出、获取当前用户
- 用户管理：CRUD 操作、状态管理
- API Key：创建、撤销、列表查询
- 供应商：CRUD、连接测试
- 模型：CRUD、定价配置
- 计费：记录查询、充值
- 路由规则：CRUD、优先级调整
- 监控告警：规则管理、指标查询
- 系统设置：配置管理、状态查询

## 开发规范

### 代码风格

- 使用 TypeScript 严格模式
- 组件使用函数式组件 + Hooks
- 使用 ProComponents 的表格和表单组件
- API 请求统一封装在 services 目录

### 命名规范

- 组件文件：PascalCase (例如：UserList.tsx)
- 工具函数：camelCase (例如：formatDate.ts)
- 类型定义：PascalCase + 后缀 (例如：UserInfo)

## 浏览器支持

- Chrome >= 90
- Firefox >= 88
- Safari >= 14
- Edge >= 90

## License

MIT
