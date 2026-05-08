package routes

import (
	"github.com/gin-gonic/gin"
)

// RegisterUsageRoutes 注册使用记录路由（需要 JWT 认证）
// - GET /api/v1/usage          获取使用记录列表
// - GET /api/v1/usage/stats    获取使用统计
// - GET /api/v1/usage/dashboard 获取仪表盘数据
//
// 使用方式: 在注册前通过 Use() 挂载 JWT 认证中间件
func RegisterUsageRoutes(rg *gin.RouterGroup, h HandlerGroup) {
	usage := rg.Group("/api/v1/usage")
	{
		usage.GET("", wrapper(h.ListUsage))
		usage.GET("/stats", wrapper(h.GetUsageStats))
		usage.GET("/dashboard", wrapper(h.GetDashboard))
	}
}
