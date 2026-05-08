package routes

import (
	"github.com/gin-gonic/gin"
)

// RegisterKeyRoutes 注册 API Key 管理路由（需要 JWT 认证）
// - GET    /api/v1/keys      获取 API Key 列表
// - POST   /api/v1/keys      创建 API Key
// - GET    /api/v1/keys/:id  获取指定 API Key 详情
// - PUT    /api/v1/keys/:id  更新 API Key
// - DELETE /api/v1/keys/:id  删除 API Key
//
// 使用方式: 在注册前通过 Use() 挂载 JWT 认证中间件
func RegisterKeyRoutes(rg *gin.RouterGroup, h HandlerGroup) {
	keys := rg.Group("/api/v1/keys")
	{
		keys.GET("", wrapper(h.ListKeys))
		keys.POST("", wrapper(h.CreateKey))
		keys.GET("/:id", wrapper(h.GetKey))
		keys.PUT("/:id", wrapper(h.UpdateKey))
		keys.DELETE("/:id", wrapper(h.DeleteKey))
	}
}
