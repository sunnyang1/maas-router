package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig CORS 中间件配置
type CORSConfig struct {
	// AllowedOrigins 允许的来源列表
	// 支持 "*" 表示允许所有来源
	// 支持具体域名 "https://example.com"
	AllowedOrigins []string
	// AllowedMethods 允许的 HTTP 方法
	AllowedMethods []string
	// AllowedHeaders 允许的请求头
	AllowedHeaders []string
	// ExposeHeaders 暴露给客户端的响应头
	ExposeHeaders []string
	// AllowCredentials 是否允许携带凭证（Cookie 等）
	AllowCredentials bool
	// MaxAge 预检请求缓存时间（秒）
	MaxAge int
}

// DefaultCORSConfig 返回默认的 CORS 配置
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-API-Key",
			"X-Goog-Api-Key",
			"X-Request-ID",
			"X-Client-Version",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"X-Request-ID",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},
		AllowCredentials: false,
		MaxAge:           86400,
	}
}

// CORS 创建 CORS 跨域中间件
// 支持配置化 AllowedOrigins/Methods/Headers
func CORS(config CORSConfig) gin.HandlerFunc {
	// 设置默认值
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{
			http.MethodGet, http.MethodPost, http.MethodPut,
			http.MethodPatch, http.MethodDelete, http.MethodOptions,
		}
	}
	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{
			"Origin", "Content-Type", "Accept", "Authorization",
		}
	}
	if config.MaxAge <= 0 {
		config.MaxAge = 86400
	}

	allowAllOrigins := len(config.AllowedOrigins) == 1 && config.AllowedOrigins[0] == "*"

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// 处理来源
		if allowAllOrigins {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if isOriginAllowed(origin, config.AllowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			if config.AllowCredentials {
				c.Header("Vary", "Origin")
			}
		}

		// 凭证支持
		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// 暴露的响应头
		if len(config.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
		}

		// 处理预检请求
		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isOriginAllowed 检查请求来源是否在允许列表中
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if strings.EqualFold(origin, allowed) {
			return true
		}
		// 支持通配符后缀匹配，如 *.example.com
		if strings.HasPrefix(allowed, "*.") {
			suffix := strings.TrimPrefix(allowed, "*")
			if strings.HasSuffix(strings.ToLower(origin), strings.ToLower(suffix)) {
				return true
			}
		}
	}
	return false
}
