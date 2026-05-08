// Package service 业务服务层 Wire ProviderSet
// 导出所有 Service 的 ProviderSet，用于依赖注入
package service

import (
	"context"

	"maas-router/ent"
	"maas-router/internal/config"

	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
	"go.uber.org/zap"
)

// ProviderSet Service 层 ProviderSet
// 包含所有服务的构造函数，由 Wire 自动处理依赖关系
var ProviderSet = wire.NewSet(
	// 账号调度服务（核心）
	NewAccountService,

	// Claude 网关服务
	NewClaudeGatewayService,

	// OpenAI 网关服务
	NewOpenAIGatewayService,

	// 智能路由 Agent 服务
	NewJudgeAgentService,

	// 计费服务
	NewBillingService,

	// 用户服务
	NewUserService,

	// API Key 服务
	NewAPIKeyService,

	// 路由服务
	NewRouterService,

	// 复杂度分析服务
	NewComplexityService,
)

// ServiceRegistry 服务注册表
// 用于管理所有服务的生命周期
type ServiceRegistry struct {
	AccountService     AccountService
	ClaudeGateway      ClaudeGatewayService
	OpenAIGateway      OpenAIGatewayService
	JudgeAgent         JudgeAgentService
	BillingService     BillingService
	UserService        UserService
	APIKeyService      APIKeyService
	RouterService      RouterService
	ComplexityService  ComplexityService
}

// NewServiceRegistry 创建服务注册表
// 这个函数由 Wire 自动生成依赖注入代码
func NewServiceRegistry(
	accountService AccountService,
	claudeGateway ClaudeGatewayService,
	openaiGateway OpenAIGatewayService,
	judgeAgent JudgeAgentService,
	billingService BillingService,
	userService UserService,
	apiKeyService APIKeyService,
	routerService RouterService,
	complexityService ComplexityService,
) *ServiceRegistry {
	return &ServiceRegistry{
		AccountService:     accountService,
		ClaudeGateway:      claudeGateway,
		OpenAIGateway:      openaiGateway,
		JudgeAgent:         judgeAgent,
		BillingService:     billingService,
		UserService:        userService,
		APIKeyService:      apiKeyService,
		RouterService:      routerService,
		ComplexityService:  complexityService,
	}
}

// InitServices 初始化所有服务
// 这是一个便捷函数，用于手动初始化服务（不使用 Wire）
func InitServices(
	db *ent.Client,
	redis *redis.Client,
	cfg *config.Config,
	logger *zap.Logger,
) *ServiceRegistry {
	// 创建计费服务（无依赖）
	billingService := NewBillingService(db, redis, cfg, logger)

	// 创建用户服务（无依赖）
	userService := NewUserService(db, redis, cfg, logger)

	// 创建 API Key 服务（无依赖）
	apiKeyService := NewAPIKeyService(db, redis, cfg, logger)

	// 创建路由服务（无依赖）
	routerService := NewRouterService(db, redis, cfg, logger)

	// 创建 Judge Agent 服务（无依赖）
	judgeAgent := NewJudgeAgentService(db, redis, cfg, logger)

	// 创建账号调度服务（无依赖）
	accountService := NewAccountService(db, redis, cfg, logger)

	// 创建 Claude 网关服务（依赖账号服务和计费服务）
	claudeGateway := NewClaudeGatewayService(db, redis, cfg, logger, accountService, billingService)

	// 创建 OpenAI 网关服务（依赖账号服务和计费服务）
	openaiGateway := NewOpenAIGatewayService(db, redis, cfg, logger, accountService, billingService)

	// 创建复杂度分析服务（无依赖）
	complexityService := NewComplexityService(db, redis, &cfg.Complexity, logger)

	return &ServiceRegistry{
		AccountService:     accountService,
		ClaudeGateway:      claudeGateway,
		OpenAIGateway:      openaiGateway,
		JudgeAgent:         judgeAgent,
		BillingService:     billingService,
		UserService:        userService,
		APIKeyService:      apiKeyService,
		RouterService:      routerService,
		ComplexityService:  complexityService,
	}
}

// Close 关闭所有服务
// 用于优雅关闭时释放资源
func (r *ServiceRegistry) Close() error {
	// 目前服务没有需要关闭的资源
	// 如果将来添加了连接池等资源，可以在这里关闭
	return nil
}

// HealthCheck 健康检查
// 检查所有服务的健康状态
func (r *ServiceRegistry) HealthCheck(ctx context.Context) map[string]bool {
	results := make(map[string]bool)

	// 检查账号服务
	results["account"] = r.AccountService != nil

	// 检查 Claude 网关
	results["claude_gateway"] = r.ClaudeGateway != nil

	// 检查 OpenAI 网关
	results["openai_gateway"] = r.OpenAIGateway != nil

	// 检查 Judge Agent
	results["judge_agent"] = r.JudgeAgent != nil

	// 检查计费服务
	results["billing"] = r.BillingService != nil

	// 检查用户服务
	results["user"] = r.UserService != nil

	// 检查 API Key 服务
	results["api_key"] = r.APIKeyService != nil

	// 检查路由服务
	results["router"] = r.RouterService != nil

	// 检查复杂度分析服务
	results["complexity"] = r.ComplexityService != nil

	return results
}

// GetServiceStats 获取服务统计信息
func (r *ServiceRegistry) GetServiceStats() map[string]interface{} {
	return map[string]interface{}{
		"services": []string{
			"account_service",
			"claude_gateway_service",
			"openai_gateway_service",
			"judge_agent_service",
			"billing_service",
			"user_service",
			"api_key_service",
			"router_service",
			"complexity_service",
		},
		"total": 9,
	}
}
