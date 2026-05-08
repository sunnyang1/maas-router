package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery 创建异常恢复中间件
// 捕获 Gin 路由处理函数中的 panic 异常，防止服务崩溃。
// 记录完整的堆栈信息用于排查问题，并返回统一的错误响应。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录完整的堆栈信息
				log.Printf("[Recovery] panic recovered: %v\n%s", err, debug.Stack())

				// 获取 Request ID（如果存在）
				requestID, exists := c.Get("X-Request-ID")
				if !exists {
					requestID = "unknown"
				}

				// 返回统一的内部错误响应
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "INTERNAL_SERVER_ERROR",
						"message": "服务器内部错误，请稍后重试",
					},
					"request_id": requestID,
				})
			}
		}()

		c.Next()
	}
}
