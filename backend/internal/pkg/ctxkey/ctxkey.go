// Package ctxkey 定义了所有 Gin Context 中使用的键常量
// 使用自定义类型避免与其他包的 context key 冲突
package ctxkey

// contextKey 是自定义的 context key 类型，防止键冲突
type contextKey string

// 以下为所有 Context Key 常量

// ContextKeyUser 存储当前认证用户信息（*model.User）
const ContextKeyUser contextKey = "user"

// ContextKeyUserRole 存储当前用户角色（string: "admin"/"user"）
const ContextKeyUserRole contextKey = "user_role"

// ContextKeyUserID 存储当前用户 ID（string）
const ContextKeyUserID contextKey = "user_id"

// ContextKeyAPIKey 存储当前请求使用的 API Key 信息（*model.APIKey）
const ContextKeyAPIKey contextKey = "api_key"

// ContextKeyRequestID 存储请求唯一标识符（string）
const ContextKeyRequestID contextKey = "request_id"

// ContextKeyTokenVersion 存储用户的 Token 版本号，用于判断密码修改后旧 token 是否失效（int64）
const ContextKeyTokenVersion contextKey = "token_version"

// ContextKeySkipBilling 标记是否跳过计费检查（bool）
const ContextKeySkipBilling contextKey = "skip_billing"

// ContextKeyIsAdmin 标记当前请求是否具有管理员权限（bool）
const ContextKeyIsAdmin contextKey = "is_admin"

// ContextKeyClientIP 存储客户端真实 IP 地址（string）
const ContextKeyClientIP contextKey = "client_ip"
