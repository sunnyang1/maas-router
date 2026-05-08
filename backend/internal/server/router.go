package server

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

	"maas-router/internal/server/middleware"
	"maas-router/internal/server/routes"
)

// RouterConfig 路由注册配置
type RouterConfig struct {
	// Handler 聚合所有业务 handler
	Handlers routes.HandlerGroup
	// JWT 认证配置
	JWTSecret string
	// JWT 发行者
	JWTIssuer string
	// Admin API Key
	AdminAPIKey string
	// API Key 查询函数
	APIKeyLookupFunc middleware.APIKeyLookupFunc
	// Redis 客户端（用于限流器）
	RedisClient *redis.Client
	// 是否启用简单模式（跳过计费）
	SimpleMode bool
	// 是否启用限流
	EnableRateLimit bool
	// 限流窗口大小（秒）
	RateLimitWindowSeconds int
	// 限流窗口内最大请求数
	RateLimitMaxRequests int
	// 限流故障模式: "fail_open" 或 "fail_close"
	RateLimitFailMode string
}

// RegisterRoutes 注册所有路由
// 这是路由注册的统一入口，负责：
//  1. 创建不同认证级别的路由组
//  2. 挂载对应的认证中间件
//  3. 调用各路由模块的注册函数
//
// 路由分组策略（参考 sub2api 三层认证体系）：
//   - 公开路由: 无需认证（健康检查、认证接口）
//   - JWT 路由: 需要 JWT 认证（用户管理、Key 管理、使用记录）
//   - API Key 路由: 需要 API Key 认证（网关代理，核心路由）
//   - Admin 路由: 需要管理员认证（管理后台）
func RegisterRoutes(engine *gin.Engine, config RouterConfig) {
	h := config.Handlers

	// ========== 公开路由（无需认证） ==========
	public := engine.Group("")
	{
		// 通用路由（健康检查等）
		routes.RegisterCommonRoutes(public, h)

		// 认证路由（注册、登录等）
		routes.RegisterAuthRoutes(public, h)
	}

	// ========== JWT 认证路由 ==========
	jwtConfig := middleware.JWTAuthConfig{
		Secret: config.JWTSecret,
		Issuer: config.JWTIssuer,
	}
	jwtGroup := engine.Group("")
	jwtGroup.Use(middleware.JWTAuth(jwtConfig))
	{
		// 用户路由
		routes.RegisterUserRoutes(jwtGroup, h)

		// API Key 管理路由
		routes.RegisterKeyRoutes(jwtGroup, h)

		// 使用记录路由
		routes.RegisterUsageRoutes(jwtGroup, h)
	}

	// ========== API Key 认证路由（核心网关路由） ==========
	apiKeyConfig := middleware.APIKeyAuthConfig{
		LookupFunc:  config.APIKeyLookupFunc,
		SimpleMode:  config.SimpleMode,
		SkipBilling: false,
	}

	apiKeyGroup := engine.Group("")
	apiKeyGroup.Use(middleware.APIKeyAuth(apiKeyConfig))

	// 可选：挂载限流中间件
	if config.EnableRateLimit && config.RedisClient != nil {
		rateLimitConfig := middleware.RateLimitConfig{
			RedisClient:    config.RedisClient,
			WindowSeconds:  config.RateLimitWindowSeconds,
			MaxRequests:    config.RateLimitMaxRequests,
			FailMode:       config.RateLimitFailMode,
			KeyPrefix:      "gateway",
		}
		apiKeyGroup.Use(middleware.RateLimiter(rateLimitConfig))
	}

	{
		// 网关路由（OpenAI/Claude/Gemini 兼容）
		routes.RegisterGatewayRoutes(apiKeyGroup, h)
	}

	// ========== 管理员认证路由 ==========
	adminConfig := middleware.AdminAuthConfig{
		AdminAPIKey: config.AdminAPIKey,
		JWTConfig:   jwtConfig,
	}
	adminGroup := engine.Group("")
	adminGroup.Use(middleware.AdminAuth(adminConfig))
	{
		// 管理员路由
		routes.RegisterAdminRoutes(adminGroup, h)
	}
}
