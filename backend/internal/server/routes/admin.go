package routes

import (
	"github.com/gin-gonic/gin"
)

// RegisterAdminRoutes 注册管理员路由（需要管理员认证）
//
// 仪表盘:
//   - GET /api/v1/admin/dashboard/stats  仪表盘统计数据
//
// 用户管理:
//   - GET    /api/v1/admin/users         用户列表
//   - POST   /api/v1/admin/users         创建用户
//   - PUT    /api/v1/admin/users/:id     更新用户
//   - DELETE /api/v1/admin/users/:id     删除用户
//
// 用户组管理:
//   - GET    /api/v1/admin/groups        用户组列表
//   - POST   /api/v1/admin/groups        创建用户组
//   - PUT    /api/v1/admin/groups/:id    更新用户组
//   - DELETE /api/v1/admin/groups/:id    删除用户组
//
// 账户管理:
//   - GET    /api/v1/admin/accounts      账户列表
//   - POST   /api/v1/admin/accounts      创建账户
//   - PUT    /api/v1/admin/accounts/:id  更新账户
//   - DELETE /api/v1/admin/accounts/:id  删除账户
//
// 路由规则管理:
//   - GET    /api/v1/admin/router-rules       路由规则列表
//   - POST   /api/v1/admin/router-rules       创建路由规则
//   - PUT    /api/v1/admin/router-rules/:id   更新路由规则
//   - DELETE /api/v1/admin/router-rules/:id   删除路由规则
//
// 运维:
//   - GET /api/v1/admin/ops/realtime-traffic  实时流量
//   - GET /api/v1/admin/ops/errors            错误日志
//
// 使用方式: 在注册前通过 Use() 挂载管理员认证中间件
func RegisterAdminRoutes(rg *gin.RouterGroup, h HandlerGroup) {
	admin := rg.Group("/api/v1/admin")

	// 仪表盘
	admin.GET("/dashboard/stats", wrapper(h.GetAdminDashboardStats))

	// 用户管理
	users := admin.Group("/users")
	{
		users.GET("", wrapper(h.ListAdminUsers))
		users.POST("", wrapper(h.CreateAdminUser))
		users.GET("/:id", wrapper(h.GetAdminUser))
		users.PUT("/:id", wrapper(h.UpdateAdminUser))
		users.DELETE("/:id", wrapper(h.DeleteAdminUser))
	}

	// 用户组管理
	groups := admin.Group("/groups")
	{
		groups.GET("", wrapper(h.ListAdminGroups))
		groups.POST("", wrapper(h.CreateAdminGroup))
		groups.GET("/:id", wrapper(h.GetAdminGroup))
		groups.PUT("/:id", wrapper(h.UpdateAdminGroup))
		groups.DELETE("/:id", wrapper(h.DeleteAdminGroup))
	}

	// 账户管理
	accounts := admin.Group("/accounts")
	{
		accounts.GET("", wrapper(h.ListAdminAccounts))
		accounts.POST("", wrapper(h.CreateAdminAccount))
		accounts.GET("/:id", wrapper(h.GetAdminAccount))
		accounts.PUT("/:id", wrapper(h.UpdateAdminAccount))
		accounts.DELETE("/:id", wrapper(h.DeleteAdminAccount))
	}

	// 路由规则管理
	routerRules := admin.Group("/router-rules")
	{
		routerRules.GET("", wrapper(h.ListRouterRules))
		routerRules.POST("", wrapper(h.CreateRouterRule))
		routerRules.GET("/:id", wrapper(h.GetRouterRule))
		routerRules.PUT("/:id", wrapper(h.UpdateRouterRule))
		routerRules.DELETE("/:id", wrapper(h.DeleteRouterRule))
	}

	// 运维接口
	ops := admin.Group("/ops")
	{
		ops.GET("/realtime-traffic", wrapper(h.GetRealtimeTraffic))
		ops.GET("/errors", wrapper(h.GetErrors))
	}
}
