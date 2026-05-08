// Package routes 定义支付相关的路由
package routes

import (
	"github.com/gin-gonic/gin"

	"maas-router/backend/internal/handler"
)

// PaymentRoutes 支付路由组
type PaymentRoutes struct {
	paymentHandler *handler.PaymentHandler
	authMiddleware gin.HandlerFunc
}

// NewPaymentRoutes 创建支付路由组
func NewPaymentRoutes(paymentHandler *handler.PaymentHandler, authMiddleware gin.HandlerFunc) *PaymentRoutes {
	return &PaymentRoutes{
		paymentHandler: paymentHandler,
		authMiddleware: authMiddleware,
	}
}

// Register 注册支付路由
func (r *PaymentRoutes) Register(router *gin.RouterGroup) {
	// 支付订单相关路由（需要认证）
	payments := router.Group("/payments")
	payments.Use(r.authMiddleware)
	{
		// 创建支付订单
		// POST /api/v1/payments
		payments.POST("", r.paymentHandler.CreatePayment)

		// 查询支付列表
		// GET /api/v1/payments?page=1&page_size=10&status=pending
		payments.GET("", r.paymentHandler.ListPayments)

		// 查询支付状态
		// GET /api/v1/payments/:payment_id
		payments.GET("/:payment_id", r.paymentHandler.QueryPayment)

		// 取消支付订单
		// POST /api/v1/payments/:payment_id/cancel
		payments.POST("/:payment_id/cancel", r.paymentHandler.CancelPayment)

		// 退款
		// POST /api/v1/payments/:payment_id/refund
		payments.POST("/:payment_id/refund", r.paymentHandler.Refund)
	}

	// 支付回调路由（不需要认证，由第三方支付平台调用）
	// POST /api/v1/payments/webhook/:provider
	router.POST("/payments/webhook/:provider", r.paymentHandler.HandleWebhook)
}

// RegisterPublicRoutes 注册公开支付路由（不需要认证）
func (r *PaymentRoutes) RegisterPublicRoutes(router *gin.RouterGroup) {
	// 支付回调路由
	// POST /api/v1/payments/webhook/:provider
	router.POST("/payments/webhook/:provider", r.paymentHandler.HandleWebhook)
}

// SetupPaymentRoutes 快速设置支付路由（便捷函数）
func SetupPaymentRoutes(router *gin.Engine, paymentHandler *handler.PaymentHandler, authMiddleware gin.HandlerFunc) {
	routes := NewPaymentRoutes(paymentHandler, authMiddleware)

	// API v1 版本组
	v1 := router.Group("/api/v1")
	{
		routes.Register(v1)
	}
}

// PaymentRouteConfig 支付路由配置
type PaymentRouteConfig struct {
	// 是否启用支付功能
	Enabled bool
	// 是否需要认证
	RequireAuth bool
	// 允许的来源IP（回调路由）
	AllowedWebhookIPs []string
	// 速率限制
	RateLimit int
}

// DefaultPaymentRouteConfig 默认支付路由配置
func DefaultPaymentRouteConfig() *PaymentRouteConfig {
	return &PaymentRouteConfig{
		Enabled:           true,
		RequireAuth:       true,
		AllowedWebhookIPs: []string{},
		RateLimit:         100,
	}
}

// RegisterWithConfig 使用配置注册支付路由
func (r *PaymentRoutes) RegisterWithConfig(router *gin.RouterGroup, config *PaymentRouteConfig) {
	if !config.Enabled {
		return
	}

	// 公开路由（回调）
	router.POST("/payments/webhook/:provider", r.paymentHandler.HandleWebhook)

	// 需要认证的路由
	payments := router.Group("/payments")
	if config.RequireAuth {
		payments.Use(r.authMiddleware)
	}

	payments.POST("", r.paymentHandler.CreatePayment)
	payments.GET("", r.paymentHandler.ListPayments)
	payments.GET("/:payment_id", r.paymentHandler.QueryPayment)
	payments.POST("/:payment_id/cancel", r.paymentHandler.CancelPayment)
	payments.POST("/:payment_id/refund", r.paymentHandler.Refund)
}
