package routes

import (
	"github.com/gin-gonic/gin"
)

// RegisterGatewayRoutes 注册 API 网关路由（需要 API Key 认证，核心路由）
//
// 兼容多个 AI 服务商的 API 格式：
//
//	OpenAI 兼容:
//	  - POST /v1/chat/completions   聊天补全
//	  - GET  /v1/models             模型列表
//	  - POST /v1/images/generations 图像生成
//
//	Claude 兼容:
//	  - POST /v1/messages           消息接口
//
//	Gemini 兼容:
//	  - GET  /v1beta/models         模型列表
//	  - POST /v1beta/models/*modelAction 模型操作（通配符）
//
// 使用方式: 在注册前通过 Use() 挂载 API Key 认证中间件
func RegisterGatewayRoutes(rg *gin.RouterGroup, h HandlerGroup) {
	// OpenAI 兼容路由
	rg.POST("/v1/chat/completions", wrapper(h.ChatCompletions))
	rg.GET("/v1/models", wrapper(h.ListModels))
	rg.POST("/v1/images/generations", wrapper(h.ImageGenerations))

	// Claude 兼容路由
	rg.POST("/v1/messages", wrapper(h.Messages))

	// Gemini 兼容路由
	rg.GET("/v1beta/models", wrapper(h.ListModelsBeta))
	rg.POST("/v1beta/models/*modelAction", wrapper(h.ModelAction))

	// 复杂度分析引擎路由
	rg.POST("/v1/complexity/analyze", wrapper(h.ComplexityAnalyze))
	rg.GET("/v1/complexity/stats", wrapper(h.ComplexityStats))
	rg.POST("/v1/complexity/feedback", wrapper(h.ComplexityFeedback))
	rg.GET("/v1/complexity/tiers", wrapper(h.ComplexityTiers))
}
