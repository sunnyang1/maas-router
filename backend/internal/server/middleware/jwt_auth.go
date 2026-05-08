// Package middleware 提供 MaaS-Router 的 HTTP 中间件
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"maas-router/internal/pkg/ctxkey"
)

// JWTClaims 定义 JWT 载荷结构
type JWTClaims struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	TokenVersion int64  `json:"token_version"`
	IsAdmin      bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

// JWTAuthConfig JWT 认证中间件配置
type JWTAuthConfig struct {
	// JWT 签名密钥
	Secret string
	// Token 发行者
	Issuer string
}

// JWTAuth 创建 JWT 认证中间件
// 从 Authorization: Bearer <token> 头部提取 JWT，
// 验证 token 有效性和过期时间，检查 TokenVersion 确保改密后旧 token 失效。
// 验证通过后将用户信息写入 Context。
func JWTAuth(config JWTAuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Authorization 头部提取 Bearer Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "缺少 Authorization 头部",
				},
			})
			return
		}

		// 校验 Bearer 前缀
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_TOKEN_FORMAT",
					"message": "Authorization 头部格式错误，应为 Bearer <token>",
				},
			})
			return
		}

		tokenString := parts[1]

		// 解析并验证 JWT Token
		claims := &JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// 验证签名算法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("不支持的签名算法")
			}
			return []byte(config.Secret), nil
		})

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": gin.H{
						"code":    "TOKEN_EXPIRED",
						"message": "Token 已过期，请重新登录",
					},
				})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "无效的 Token",
				},
			})
			return
		}

		if !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "Token 无效",
				},
			})
			return
		}

		// 验证 Issuer（如果配置了）
		if config.Issuer != "" {
			if claims.Issuer != config.Issuer {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": gin.H{
						"code":    "INVALID_ISSUER",
						"message": "Token 发行者不匹配",
					},
				})
				return
			}
		}

		// 将用户信息写入 Context
		c.Set(string(ctxkey.ContextKeyUserID), claims.UserID)
		c.Set(string(ctxkey.ContextKeyUser), claims)
		c.Set(string(ctxkey.ContextKeyUserRole), claims.Role)
		c.Set(string(ctxkey.ContextKeyTokenVersion), claims.TokenVersion)
		c.Set(string(ctxkey.ContextKeyIsAdmin), claims.IsAdmin)

		c.Next()
	}
}
