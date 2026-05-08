package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"maas-router/internal/pkg/ctxkey"
)

// APIKeyInfo 表示从数据库中查询到的 API Key 信息
type APIKeyInfo struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	KeyPrefix    string    `json:"key_prefix"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	ExpiresAt    *time.Time `json:"expires_at"`
	DailyLimit   float64   `json:"daily_limit"`
	MonthlyLimit float64   `json:"monthly_limit"`
	UsedToday    float64   `json:"used_today"`
	UsedMonth    float64   `json:"used_month"`
	AllowedModels []string `json:"allowed_models"`
	// 用户相关信息
	UserStatus   string  `json:"user_status"`
	UserBalance  float64 `json:"user_balance"`
	// IP 限制
	IPWhitelist []string `json:"ip_whitelist"`
	IPBlacklist []string `json:"ip_blacklist"`
}

// APIKeyLookupFunc 定义 API Key 查询函数类型
// 由上层注入，用于从数据库/缓存中查询 Key 信息
type APIKeyLookupFunc func(rawKey string) (*APIKeyInfo, error)

// APIKeyAuthConfig API Key 认证中间件配置
type APIKeyAuthConfig struct {
	// API Key 查询函数
	LookupFunc APIKeyLookupFunc
	// 是否启用简单模式（跳过计费检查）
	SimpleMode bool
	// 是否跳过计费执行层（仅鉴权不扣费）
	SkipBilling bool
}

// APIKeyAuth 创建 API Key 认证中间件（核心中间件）
// 认证流程分为两层：
//
//	鉴权层: Key 有效性、用户状态、IP 白名单/黑名单
//	计费执行层: 过期/配额/余额检查（可通过配置跳过）
//
// 支持从以下头部提取 API Key：
//   - Authorization: Bearer <key>
//   - x-api-key: <key>
//   - x-goog-api-key: <key>
func APIKeyAuth(config APIKeyAuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 第一步：从多个可能的头部提取 API Key
		rawKey := extractAPIKey(c)
		if rawKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "API_KEY_REQUIRED",
					"message": "需要提供 API Key，请通过 Authorization、x-api-key 或 x-goog-api-key 头部传递",
				},
			})
			return
		}

		// 第二步：查询 Key 信息
		if config.LookupFunc == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "API Key 查询函数未配置",
				},
			})
			return
		}

		keyInfo, err := config.LookupFunc(rawKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "API_KEY_INVALID",
					"message": "无效的 API Key",
				},
			})
			return
		}

		// ========== 鉴权层 ==========

		// 检查 Key 状态
		if keyInfo.Status != "active" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "API_KEY_INACTIVE",
					"message": "API Key 已被禁用或撤销",
				},
			})
			return
		}

		// 检查用户状态
		if keyInfo.UserStatus != "active" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "USER_SUSPENDED",
					"message": "用户账户已被暂停",
				},
			})
			return
		}

		// 检查 IP 黑名单
		clientIP := c.ClientIP()
		if len(keyInfo.IPBlacklist) > 0 && isIPInList(clientIP, keyInfo.IPBlacklist) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "IP_BLOCKED",
					"message": "您的 IP 地址已被列入黑名单",
				},
			})
			return
		}

		// 检查 IP 白名单（如果配置了白名单）
		if len(keyInfo.IPWhitelist) > 0 && !isIPInList(clientIP, keyInfo.IPWhitelist) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "IP_NOT_ALLOWED",
					"message": "您的 IP 地址不在白名单中",
				},
			})
			return
		}

		// ========== 计费执行层 ==========

		// SimpleMode 或 SkipBilling 时跳过计费检查
		if !config.SimpleMode && !config.SkipBilling {
			// 检查 Key 是否过期
			if keyInfo.ExpiresAt != nil && keyInfo.ExpiresAt.Before(time.Now()) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": gin.H{
						"code":    "API_KEY_EXPIRED",
						"message": "API Key 已过期",
					},
				})
				return
			}

			// 检查日配额
			if keyInfo.DailyLimit > 0 && keyInfo.UsedToday >= keyInfo.DailyLimit {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"code":    "DAILY_QUOTA_EXCEEDED",
						"message": "已达到每日配额上限",
					},
				})
				return
			}

			// 检查月配额
			if keyInfo.MonthlyLimit > 0 && keyInfo.UsedMonth >= keyInfo.MonthlyLimit {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"code":    "MONTHLY_QUOTA_EXCEEDED",
						"message": "已达到每月配额上限",
					},
				})
				return
			}

			// 检查用户余额
			if keyInfo.UserBalance <= 0 {
				c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
					"error": gin.H{
						"code":    "INSUFFICIENT_BALANCE",
						"message": "账户余额不足，请充值后继续使用",
					},
				})
				return
			}
		}

		// ========== 认证通过 ==========

		// 将 API Key 信息写入 Context
		c.Set(string(ctxkey.ContextKeyAPIKey), keyInfo)
		c.Set(string(ctxkey.ContextKeyUserID), keyInfo.UserID)
		c.Set(string(ctxkey.ContextKeySkipBilling), config.SimpleMode || config.SkipBilling)

		c.Next()
	}
}

// extractAPIKey 从请求中提取 API Key
// 按优先级依次检查: Authorization > x-api-key > x-goog-api-key
func extractAPIKey(c *gin.Context) string {
	// 优先从 Authorization: Bearer <key> 提取
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	// 其次从 x-api-key 提取
	if key := c.GetHeader("x-api-key"); key != "" {
		return strings.TrimSpace(key)
	}

	// 最后从 x-goog-api-key 提取（Google AI 兼容）
	if key := c.GetHeader("x-goog-api-key"); key != "" {
		return strings.TrimSpace(key)
	}

	return ""
}

// isIPInList 检查 IP 是否在列表中
// 支持精确匹配和 CIDR 格式（简化实现，仅精确匹配）
func isIPInList(ip string, list []string) bool {
	for _, item := range list {
		if subtle.ConstantTimeCompare([]byte(ip), []byte(item)) == 1 {
			return true
		}
	}
	return false
}
