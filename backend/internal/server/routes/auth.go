package routes

import (
	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes 注册认证路由（无需认证）
// - POST /api/v1/auth/register    用户注册
// - POST /api/v1/auth/login       用户登录
// - POST /api/v1/auth/refresh     刷新 Token
// - POST /api/v1/auth/logout      用户登出
// - POST /api/v1/auth/forgot-password 忘记密码
// - POST /api/v1/auth/reset-password   重置密码
// - GET  /api/v1/auth/github      GitHub OAuth登录
// - GET  /api/v1/auth/github/callback  GitHub OAuth回调
// - GET  /api/v1/auth/google      Google OAuth登录
// - GET  /api/v1/auth/google/callback  Google OAuth回调
// - GET  /api/v1/auth/wechat      微信OAuth登录
// - GET  /api/v1/auth/wechat/callback  微信OAuth回调
func RegisterAuthRoutes(rg *gin.RouterGroup, h HandlerGroup) {
	auth := rg.Group("/api/v1/auth")
	{
		auth.POST("/register", wrapper(h.Register))
		auth.POST("/login", wrapper(h.Login))
		auth.POST("/refresh", wrapper(h.RefreshToken))
		auth.POST("/logout", wrapper(h.Logout))
		auth.POST("/forgot-password", wrapper(h.ForgotPassword))
		auth.POST("/reset-password", wrapper(h.ResetPassword))

		// OAuth路由
		auth.GET("/github", wrapper(h.GitHubAuth))
		auth.GET("/github/callback", wrapper(h.GitHubCallback))
		auth.GET("/google", wrapper(h.GoogleAuth))
		auth.GET("/google/callback", wrapper(h.GoogleCallback))
		auth.GET("/wechat", wrapper(h.WeChatAuth))
		auth.GET("/wechat/callback", wrapper(h.WeChatCallback))
	}
}
