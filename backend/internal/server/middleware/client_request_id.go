package middleware

import (
	"crypto/rand"
	"fmt"

	"github.com/gin-gonic/gin"

	"maas-router/internal/pkg/ctxkey"
)

// generateUUID 生成 UUID v4 格式的唯一标识符
// 使用 crypto/rand 保证随机性，不依赖外部 uuid 库
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	// 设置版本号 (v4) 和变体位
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ClientRequestID 创建请求唯一 ID 中间件
// 为每个请求生成唯一的 Request ID，用于链路追踪和日志关联。
// 如果客户端已提供 X-Request-ID 头部，则复用该值。
func ClientRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先使用客户端传入的 Request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// 生成新的 UUID v4 作为 Request ID
			requestID = generateUUID()
		}

		// 写入 Context
		c.Set(string(ctxkey.ContextKeyRequestID), requestID)

		// 写入响应头，方便客户端追踪
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}
