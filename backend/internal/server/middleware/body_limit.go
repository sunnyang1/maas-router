package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const MaxRequestBodySize = 10 << 20 // 10MB

// BodyLimit 请求体大小限制中间件
// 防止恶意客户端发送超大请求体导致服务器内存耗尽
func BodyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxRequestBodySize)
		c.Next()
	}
}
