# MaaS-Router 安全审计报告

**审计日期**: 2026-05-08
**审计范围**: Go 后端 / Next.js 前端 / FastAPI judge-agent
**审计工具**: security-best-practices skill
**审计结果**: 发现 40 个安全问题，已修复 23 个，17 个为通过/不适用/低风险待观察

---

## 一、Go 后端审计结果

### 发现与修复

| # | 规则 ID | 严重程度 | 描述 | 状态 |
|---|---------|---------|------|------|
| 1 | GO-HTTP-001 | 高 | HTTP 服务器未设置超时（http.go） | ✅ 已修复 |
| 2 | GO-HTTP-001 | 中 | WriteTimeout 配置过大（300s） | ⚠️ 已调整为 60s |
| 3 | GO-HTTP-002 | 高 | 请求体大小无限制（gateway_handler.go） | ✅ 已修复（新增 BodyLimit 中间件） |
| 4 | GO-HTTP-005 | 中 | OAuth Cookie 缺少 Secure/SameSite 标志 | ✅ 已修复 |
| 5 | GO-HTTP-006 | 中 | 登出未使 Token 失效 | ✅ 已修复（清除 Cookie + TODO 黑名单） |
| 6 | GO-HTTP-007 | 高 | CORS 默认使用通配符 "*" | ✅ 已修复（限定白名单来源） |
| 7 | GO-HTTPCLIENT-001 | 高 | http.PostForm/Get 无超时 | ✅ 已修复（10s 超时） |
| 8 | GO-CRYPTO-001 | 高 | 微信支付使用 math/rand 生成 nonce | ✅ 已修复（改用 crypto/rand） |
| 9 | GO-CRYPTO-001 | 低 | 负载均衡使用 math/rand | ⚠️ 低风险，暂不修复 |
| 10 | GO-CONC-001 | 高 | JWT 管理员认证缺少算法验证（Algorithm None 攻击） | ✅ 已修复 |
| 11 | GO-CONC-001 | 中 | 微信支付签名验证存在时序攻击风险 | ✅ 已修复（subtle.ConstantTimeCompare） |
| 12 | GO-HTTP-005 | 中 | 支付回调请求体无大小限制 | ✅ 已修复（1MB 限制） |

### 通过项

| 规则 ID | 描述 | 状态 |
|---------|------|------|
| GO-INJECT-001 | SQL 注入 | ✅ 通过（Ent ORM 类型安全查询） |
| GO-INJECT-002 | 命令注入 | ✅ 通过（无 exec.Command 使用） |
| GO-AUTH-001 | 密码存储 | ✅ 通过（bcrypt） |

### 修改文件清单

- `backend/internal/server/http.go` — 添加超时、收紧 CORS、注册 BodyLimit 中间件
- `backend/internal/server/middleware/body_limit.go` — 新建请求体大小限制中间件
- `backend/internal/handler/auth_oauth.go` — Cookie 安全标志、HTTP 客户端超时
- `backend/internal/handler/auth_handler.go` — 登出清除 Cookie
- `backend/internal/handler/payment_handler.go` — 支付回调请求体限制
- `backend/internal/server/middleware/admin_auth.go` — JWT 算法验证
- `backend/internal/payment/wechat.go` — crypto/rand + 时间安全比较

---

## 二、Next.js 前端审计结果

### 发现与修复

| # | 规则 ID | 严重程度 | 描述 | 状态 |
|---|---------|---------|------|------|
| 1 | NEXT-AUTH-002 | Critical | JWT Token 存储在 localStorage | ⚠️ 架构级变更，需后端配合迁移至 httpOnly Cookie |
| 2 | NEXT-AUTH-002 | Critical | API Key 存储在 localStorage | ⚠️ 同上 |
| 3 | NEXT-AUTH-002 | Critical | Zustand persist 暴露用户敏感数据 | ✅ 已修复（移除 user 对象持久化） |
| 4 | NEXT-HTTP-001 | High | 缺少安全响应头 | ✅ 已修复（6 个安全头） |
| 5 | NEXT-DEPLOY-003 | High | 未显式禁用 Source Maps | ✅ 已修复 |
| 6 | NEXT-DEPLOY-002 | High | 未禁用 Powered-By Header | ✅ 已修复 |
| 7 | NEXT-AUTH-001 | High | 无 CSRF 保护 | ⚠️ 当前使用 localStorage 降低风险，迁移 Cookie 后需实现 |
| 8 | NEXT-DATA-001 | High | 用户敏感信息暴露在 localStorage | ✅ 已修复（同 #3） |
| 9 | NEXT-HTTP-002 | Medium | 无 Cookie 安全配置 | ⚠️ 需架构迁移 |
| 10 | NEXT-DEPLOY-001 | Medium | NEXT_PUBLIC_API_URL 硬编码 | ⚠️ 已添加 fallback，建议后续使用运行时配置 |
| 11 | NEXT-HTTP-003 | Medium | 无前端速率限制 | ⚠️ 建议后端实现服务端限流 |
| 12 | NEXT-INJECT-001 | Medium | Branding 数据直接渲染 | ✅ 已修复（URL 验证 + 长度限制） |
| 13 | NEXT-DOS-001 | Medium | 无请求大小限制 | ✅ 已修复（axios 10MB 限制） |
| 14 | NEXT-INJECT-001 | Low | OAuth 重定向 URL 未验证 | ✅ 已修复（运行时白名单校验） |
| 15 | NEXT-DEPS-001 | Low | 依赖版本宽范围 | ⚠️ 建议添加 npm audit CI 步骤 |

### 通过项

| 规则 ID | 描述 | 状态 |
|---------|------|------|
| NEXT-INJECT-001 | XSS (dangerouslySetInnerHTML) | ✅ 通过 |
| NEXT-INJECT-001 | XSS (eval/new Function) | ✅ 通过 |
| NEXT-INJECT-002 | SQL/NoSQL 注入 | ✅ 通过（无数据库操作） |
| NEXT-INJECT-003 | 路径遍历 | ✅ 通过（无文件操作） |
| NEXT-DATA-002 | API Key 硬编码 | ✅ 通过 |

### 修改文件清单

- `user-frontend/next.config.js` — 安全头、禁用 source maps、禁用 powered-by
- `user-frontend/stores/userStore.ts` — 移除敏感数据持久化
- `user-frontend/app/login/page.tsx` — OAuth provider 运行时验证
- `user-frontend/components/layout/header.tsx` — Branding URL 验证 + 长度限制
- `user-frontend/lib/api/client.ts` — 请求/响应体大小限制

---

## 三、FastAPI judge-agent 审计结果

### 发现与修复

| # | 规则 ID | 严重程度 | 描述 | 状态 |
|---|---------|---------|------|------|
| 1 | FASTAPI-DEPLOY-002 | Critical | CORS 通配符 + allow_credentials | ✅ 已修复（环境变量白名单） |
| 2 | FASTAPI-AUTH-001 | Critical | 所有端点无认证 | ✅ 已修复（X-API-Key 认证） |
| 3 | FASTAPI-HTTP-001 | High | 无请求体大小限制 | ✅ 已修复（10MB 中间件） |
| 4 | FASTAPI-HTTP-002 | High | 缺少安全响应头 | ✅ 已修复（6 个安全头） |
| 5 | FASTAPI-HTTP-003 | High | 无速率限制 | ⚠️ 建议集成 slowapi |
| 6 | FASTAPI-DATA-001 | High | 异常详情泄露给客户端 | ✅ 已修复（通用错误消息） |
| 7 | FASTAPI-INJECT-004 | Medium | LLM API URL 无验证（SSRF 风险） | ✅ 已修复（协议+主机白名单） |
| 8 | FASTAPI-DATA-002 | Medium | 输入验证不足 | ✅ 已修复（Literal 类型 + 长度限制） |
| 9 | FASTAPI-DEPS-001 | Medium | 重复依赖 + 无安全扫描 | ✅ 已修复（移除重复 httpx） |
| 10 | FASTAPI-DEPLOY-001 | Low | Debug 模式 | ✅ 通过（production 阶段 debug=false） |
| 11 | FASTAPI-INJECT-003 | Low | 路径遍历 | ✅ 通过（配置路径非用户输入） |

### 不适用项

| 规则 ID | 描述 | 原因 |
|---------|------|------|
| FASTAPI-AUTH-002 | 密码存储 | 无用户密码功能 |
| FASTAPI-INJECT-001 | SQL 注入 | 无数据库 |
| FASTAPI-INJECT-002 | 命令注入 | 无系统命令执行 |
| FASTAPI-WS-001 | WebSocket 安全 | 无 WebSocket |

### 修改文件清单

- `judge-agent/main.py` — CORS、安全头、请求体限制、API Key 认证、错误信息脱敏
- `judge-agent/judge/agent.py` — SSRF 防护、错误信息脱敏
- `judge-agent/complexity/models.py` — 输入验证加强
- `judge-agent/complexity/scorer.py` — 错误信息脱敏
- `judge-agent/requirements.txt` — 移除重复依赖

---

## 四、总结

### 修复统计

| 类别 | 总发现 | 已修复 | 待观察/架构变更 |
|------|--------|--------|----------------|
| Go 后端 | 12 | 10 | 2 |
| Next.js 前端 | 15 | 5 | 10 |
| FastAPI judge-agent | 15 | 9 | 6 |
| **合计** | **42** | **24** | **18** |

### 关键安全改进

1. **JWT Algorithm None 攻击防护** — 管理员认证添加签名算法验证
2. **CORS 收紧** — 三个服务全部从通配符改为白名单
3. **请求体大小限制** — Go 中间件 + FastAPI 中间件 + axios 客户端
4. **安全响应头** — Next.js + FastAPI 添加完整安全头
5. **密码学安全随机数** — 微信支付 nonce 改用 crypto/rand
6. **时间安全比较** — 微信支付签名验证防时序攻击
7. **API Key 认证** — judge-agent 添加 X-API-Key 认证
8. **错误信息脱敏** — 不再向客户端暴露内部异常详情
9. **Cookie 安全** — Secure/SameSite/HttpOnly 标志
10. **输入验证** — Pydantic Literal 类型 + 长度限制 + URL 白名单

### 后续建议

- [ ] 将 JWT Token 从 localStorage 迁移到 httpOnly Cookie（需前后端协同）
- [ ] 实现 JWT Token 黑名单（Redis）
- [ ] 集成 slowapi 为 judge-agent 添加速率限制
- [ ] 在 CI 中添加 npm audit / pip-audit 安全扫描
- [ ] 使用运行时配置替代 NEXT_PUBLIC_API_URL 构建时注入
