package routes

import (
	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes 注册用户路由（需要 JWT 认证）
// - GET  /api/v1/user/profile  获取用户资料
// - PUT  /api/v1/user/profile  更新用户资料
// - PUT  /api/v1/user/password 修改密码
//
// 使用方式: 在注册前通过 Use() 挂载 JWT 认证中间件
func RegisterUserRoutes(rg *gin.RouterGroup, h HandlerGroup) {
	user := rg.Group("/api/v1/user")
	{
		user.GET("/profile", wrapper(h.GetUserProfile))
		user.PUT("/profile", wrapper(h.UpdateUserProfile))
		user.PUT("/password", wrapper(h.UpdatePassword))
	}
}
