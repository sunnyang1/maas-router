// Package handler 提供 MaaS-Router 的 HTTP 处理器
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"maas-router/internal/cache"
)

// ProviderSet Handler 层 Wire ProviderSet
// 导出所有 Handler 的构造函数，供 Wire 依赖注入使用
var ProviderSet = wire.NewSet(
	// 网关 Handler
	NewGatewayHandler,
	NewOpenAIGatewayHandler,
	NewGeminiGatewayHandler,

	// 认证 Handler
	NewAuthHandler,

	// 用户 Handler
	NewUserHandler,

	// API Key Handler
	NewAPIKeyHandler,

	// 使用记录 Handler
	NewUsageHandler,

	// 复杂度分析 Handler
	NewComplexityHandler,

	// 余额查询 Handler
	NewBalanceHandler,

	// 渠道测试 Handler
	NewChannelTestHandler,

	// 品牌设置 Handler
	NewBrandingHandler,

	// 配置 Handler
	NewConfigHandler,

	// Handler 组装器
	NewHandlerAssembler,

	// 依赖：Cache 层
	cache.ProviderSet,
)

// HandlerAssembler Handler 组装器
// 负责将所有 Handler 组装成 routes.HandlerGroup
type HandlerAssembler struct {
	// 网关 Handler
	GatewayHandler       *GatewayHandler
	OpenAIGatewayHandler *OpenAIGatewayHandler
	GeminiGatewayHandler *GeminiGatewayHandler

	// 认证 Handler
	AuthHandler *AuthHandler

	// 用户 Handler
	UserHandler *UserHandler

	// API Key Handler
	APIKeyHandler *APIKeyHandler

	// 使用记录 Handler
	UsageHandler *UsageHandler

	// 复杂度分析 Handler
	ComplexityHandler *ComplexityHandler

	// 余额查询 Handler
	BalanceHandler *BalanceHandler

	// 渠道测试 Handler
	ChannelTestHandler *ChannelTestHandler

	// 品牌设置 Handler
	BrandingHandler *BrandingHandler

	// 配置 Handler
	ConfigHandler *ConfigHandler
}

// NewHandlerAssembler 创建 Handler 组装器
func NewHandlerAssembler(
	gatewayHandler *GatewayHandler,
	openaiGatewayHandler *OpenAIGatewayHandler,
	geminiGatewayHandler *GeminiGatewayHandler,
	authHandler *AuthHandler,
	userHandler *UserHandler,
	apiKeyHandler *APIKeyHandler,
	usageHandler *UsageHandler,
	complexityHandler *ComplexityHandler,
	balanceHandler *BalanceHandler,
	channelTestHandler *ChannelTestHandler,
	brandingHandler *BrandingHandler,
	configHandler *ConfigHandler,
) *HandlerAssembler {
	return &HandlerAssembler{
		GatewayHandler:       gatewayHandler,
		OpenAIGatewayHandler: openaiGatewayHandler,
		GeminiGatewayHandler: geminiGatewayHandler,
		AuthHandler:          authHandler,
		UserHandler:          userHandler,
		APIKeyHandler:        apiKeyHandler,
		UsageHandler:         usageHandler,
		ComplexityHandler:    complexityHandler,
		BalanceHandler:       balanceHandler,
		ChannelTestHandler:   channelTestHandler,
		BrandingHandler:      brandingHandler,
		ConfigHandler:        configHandler,
	}
}

// HandlerGroup 返回 routes.HandlerGroup 格式的 Handler 组
// 用于路由注册
func (a *HandlerAssembler) HandlerGroup() HandlerGroup {
	return HandlerGroup{
		// 通用 Handler
		HealthCheck: a.HealthCheck(),
		SetupStatus: a.SetupStatus(),

		// 认证 Handler
		Register:       a.AuthHandler.Register,
		Login:          a.AuthHandler.Login,
		RefreshToken:   a.AuthHandler.RefreshToken,
		Logout:         a.AuthHandler.Logout,
		ForgotPassword: a.AuthHandler.ForgotPassword,
		ResetPassword:  a.AuthHandler.ResetPassword,

		// 用户 Handler
		GetUserProfile:   a.UserHandler.GetUserProfile,
		UpdateUserProfile: a.UserHandler.UpdateUserProfile,
		UpdatePassword:   a.UserHandler.UpdatePassword,

		// API Key Handler
		ListKeys:  a.APIKeyHandler.ListKeys,
		CreateKey: a.APIKeyHandler.CreateKey,
		GetKey:    a.APIKeyHandler.GetKey,
		UpdateKey: a.APIKeyHandler.UpdateKey,
		DeleteKey: a.APIKeyHandler.DeleteKey,

		// 使用记录 Handler
		ListUsage:       a.UsageHandler.ListUsage,
		GetUsageStats:   a.UsageHandler.GetUsageStats,
		GetDashboard:    a.UsageHandler.GetDashboard,

		// 网关 Handler
		ChatCompletions:   a.OpenAIGatewayHandler.ChatCompletions,
		Messages:          a.GatewayHandler.Messages,
		ListModels:        a.GatewayHandler.ListModels,
		ImageGenerations:  a.OpenAIGatewayHandler.ImageGenerations,
		ListModelsBeta:    a.GeminiGatewayHandler.ListModelsBeta,
		ModelAction:       a.GeminiGatewayHandler.ModelAction,

		// 复杂度分析 Handler
		ComplexityAnalyze:  a.ComplexityHandler.Analyze,
		ComplexityStats:    a.ComplexityHandler.Stats,
		ComplexityFeedback: a.ComplexityHandler.Feedback,
		ComplexityTiers:    a.ComplexityHandler.ModelTiers,

		// 余额查询 Handler
		GetAccountBalance:    a.BalanceHandler.GetBalance,
		GetAllBalances:       a.BalanceHandler.GetAllBalances,
		RefreshAccountBalance: a.BalanceHandler.RefreshBalance,

		// 渠道测试 Handler
		TestAccount:    a.ChannelTestHandler.TestAccount,
		TestAllAccounts: a.ChannelTestHandler.TestAllAccounts,
		GetTestResults: a.ChannelTestHandler.GetTestResults,

		// 品牌设置 Handler
		GetBranding:       a.BrandingHandler.GetBranding,
		UpdateBranding:    a.BrandingHandler.UpdateBranding,
		GetPublicBranding: a.BrandingHandler.GetPublicBranding,

		// 配置 Handler
		GetPublicConfig: a.ConfigHandler.GetPublicConfig,
	}
}

// HealthCheck 健康检查 Handler
func (a *HandlerAssembler) HealthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "maas-router",
		})
	}
}

// SetupStatus 系统初始化状态 Handler
func (a *HandlerAssembler) SetupStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"initialized": true,
			"version":     "1.0.0",
		})
	}
}

// HandlerGroup 聚合所有 handler，由上层注入
// 与 routes.HandlerGroup 保持一致
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
	ComplexityAnalyze  gin.HandlerFunc
	ComplexityStats    gin.HandlerFunc
	ComplexityFeedback gin.HandlerFunc
	ComplexityTiers    gin.HandlerFunc

	// 余额查询 handler
	GetAccountBalance    gin.HandlerFunc
	GetAllBalances       gin.HandlerFunc
	RefreshAccountBalance gin.HandlerFunc

	// 渠道测试 handler
	TestAccount    gin.HandlerFunc
	TestAllAccounts gin.HandlerFunc
	GetTestResults gin.HandlerFunc

	// 品牌设置 handler
	GetBranding       gin.HandlerFunc
	UpdateBranding    gin.HandlerFunc
	GetPublicBranding gin.HandlerFunc

	// 配置 handler
	GetPublicConfig gin.HandlerFunc
}
