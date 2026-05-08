package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"maas-router/internal/pkg/ctxkey"
)

// AdminAuthConfig 管理员认证中间件配置
type AdminAuthConfig struct {
	// Admin API Key，用于服务间管理接口调用
	AdminAPIKey string
	// JWT 配置，用于管理员用户通过 JWT 认证
	JWTConfig JWTAuthConfig
}

// AdminAuth 创建管理员认证中间件
// 支持两种认证方式：
//  1. Admin API Key: 通过 x-api-key 头部传递，使用 subtle.ConstantTimeCompare 防止时序攻击
//  2. JWT Token: 通过 Authorization: Bearer <token> 传递，额外检查 IsAdmin 字段
//  3. WebSocket: 通过 Sec-WebSocket-Protocol 头部传递 token
func AdminAuth(config AdminAuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 方式一：检查 Admin API Key
		apiKey := c.GetHeader("x-api-key")
		if apiKey != "" && config.AdminAPIKey != "" {
			// 使用 ConstantTimeCompare 防止时序攻击
			if subtle.ConstantTimeCompare([]byte(apiKey), []byte(config.AdminAPIKey)) == 1 {
				c.Set(string(ctxkey.ContextKeyIsAdmin), true)
				c.Set(string(ctxkey.ContextKeyUserRole), "admin")
				c.Next()
				return
			}
			// API Key 不匹配，继续尝试其他认证方式
		}

		// 方式二：检查 WebSocket 协议头中的 token
		// WebSocket 客户端通常通过 Sec-WebSocket-Protocol 传递认证信息
		// 格式: Sec-WebSocket-Protocol: token.<jwt_token>
		wsProtocol := c.GetHeader("Sec-WebSocket-Protocol")
		if wsProtocol != "" {
			token := extractTokenFromWebSocketProtocol(wsProtocol)
			if token != "" {
				// 将 token 设置到 Authorization 头部，复用 JWT 验证逻辑
				c.Request.Header.Set("Authorization", "Bearer "+token)
			}
		}

		// 方式三：JWT Token 认证
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				claims, err := validateJWTToken(parts[1], config.JWTConfig)
				if err == nil {
					// JWT 认证通过，但需要额外检查是否为管理员
					if !claims.IsAdmin && claims.Role != "admin" {
						c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
							"error": gin.H{
								"code":    "FORBIDDEN",
								"message": "需要管理员权限",
							},
						})
						return
					}

					c.Set(string(ctxkey.ContextKeyUserID), claims.UserID)
					c.Set(string(ctxkey.ContextKeyUser), claims)
					c.Set(string(ctxkey.ContextKeyUserRole), claims.Role)
					c.Set(string(ctxkey.ContextKeyTokenVersion), claims.TokenVersion)
					c.Set(string(ctxkey.ContextKeyIsAdmin), true)
					c.Next()
					return
				}
			}
		}

		// 所有认证方式均失败
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "ADMIN_AUTH_REQUIRED",
				"message": "需要管理员认证，请提供有效的 Admin API Key 或管理员 JWT Token",
			},
		})
	}
}

// extractTokenFromWebSocketProtocol 从 Sec-WebSocket-Protocol 头部提取 JWT Token
// 支持格式: "token.<jwt_token>" 或直接传递 token
func extractTokenFromWebSocketProtocol(protocol string) string {
	protocols := strings.Split(protocol, ",")
	for _, p := range protocols {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "token.") {
			return strings.TrimPrefix(p, "token.")
		}
	}
	// 如果没有 token. 前缀，尝试直接作为 token 使用
	if len(protocols) > 0 {
		return strings.TrimSpace(protocols[0])
	}
	return ""
}

// validateJWTToken 验证 JWT Token 并返回 Claims
func validateJWTToken(tokenString string, config JWTAuthConfig) (*JWTClaims, error) {
	claims := &JWTClaims{}
	token, err := parseToken(tokenString, claims, config.Secret)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, err
	}
	return claims, nil
}

// parseToken 解析 JWT Token
func parseToken(tokenString string, claims *JWTClaims, secret string) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
}
