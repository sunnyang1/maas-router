// Package routes 提供 MaaS-Router 的路由注册功能
package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandlerGroup 聚合所有 handler，由上层注入
type HandlerGroup struct {
	// 通用 handler
	HealthCheck     gin.HandlerFunc
	SetupStatus     gin.HandlerFunc

	// 认证 handler
	Register        gin.HandlerFunc
	Login           gin.HandlerFunc
	RefreshToken    gin.HandlerFunc
	Logout          gin.HandlerFunc
	ForgotPassword  gin.HandlerFunc
	ResetPassword   gin.HandlerFunc

	// OAuth handler
	GitHubAuth      gin.HandlerFunc
	GitHubCallback  gin.HandlerFunc
	GoogleAuth      gin.HandlerFunc
	GoogleCallback  gin.HandlerFunc
	WeChatAuth      gin.HandlerFunc
	WeChatCallback  gin.HandlerFunc

	// 用户 handler
	GetUserProfile  gin.HandlerFunc
	UpdateUserProfile gin.HandlerFunc
	UpdatePassword  gin.HandlerFunc

	// API Key 管理 handler
	ListKeys        gin.HandlerFunc
	CreateKey       gin.HandlerFunc
	GetKey          gin.HandlerFunc
	UpdateKey       gin.HandlerFunc
	DeleteKey       gin.HandlerFunc

	// 使用记录 handler
	ListUsage       gin.HandlerFunc
	GetUsageStats   gin.HandlerFunc
	GetDashboard    gin.HandlerFunc

	// 网关 handler（OpenAI/Claude/Gemini 兼容）
	ChatCompletions gin.HandlerFunc
	Messages        gin.HandlerFunc
	ListModels      gin.HandlerFunc
	ImageGenerations gin.HandlerFunc
	ListModelsBeta  gin.HandlerFunc
	ModelAction     gin.HandlerFunc

	// 管理员 handler
	GetAdminDashboardStats gin.HandlerFunc
	ListAdminUsers   gin.HandlerFunc
	CreateAdminUser  gin.HandlerFunc
	GetAdminUser     gin.HandlerFunc
	UpdateAdminUser  gin.HandlerFunc
	DeleteAdminUser  gin.HandlerFunc
	ListAdminGroups  gin.HandlerFunc
	CreateAdminGroup gin.HandlerFunc
	GetAdminGroup    gin.HandlerFunc
	UpdateAdminGroup gin.HandlerFunc
	DeleteAdminGroup gin.HandlerFunc
	ListAdminAccounts gin.HandlerFunc
	CreateAdminAccount gin.HandlerFunc
	GetAdminAccount  gin.HandlerFunc
	UpdateAdminAccount gin.HandlerFunc
	DeleteAdminAccount gin.HandlerFunc
	ListRouterRules  gin.HandlerFunc
	CreateRouterRule gin.HandlerFunc
	GetRouterRule    gin.HandlerFunc
	UpdateRouterRule gin.HandlerFunc
	DeleteRouterRule gin.HandlerFunc
	GetRealtimeTraffic gin.HandlerFunc
	GetErrors        gin.HandlerFunc

	// 复杂度分析 handler
	ComplexityAnalyze    gin.HandlerFunc
	ComplexityStats      gin.HandlerFunc
	ComplexityFeedback   gin.HandlerFunc
	ComplexityTiers      gin.HandlerFunc
}

// RegisterCommonRoutes 注册通用路由（无需认证）
// - GET /health 健康检查
// - GET /setup/status 系统初始化状态
func RegisterCommonRoutes(rg *gin.RouterGroup, h HandlerGroup) {
	rg.GET("/health", wrapper(h.HealthCheck))
	rg.GET("/setup/status", wrapper(h.SetupStatus))
}

// wrapper 是一个简单的 handler 包装器，确保 handler 不为 nil 时才注册
// 如果 handler 为 nil，则返回 501 Not Implemented
func wrapper(h gin.HandlerFunc) gin.HandlerFunc {
	if h == nil {
		return func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{
				"error": gin.H{
					"code":    "NOT_IMPLEMENTED",
					"message": "该接口尚未实现",
				},
			})
		}
	}
	return h
}
