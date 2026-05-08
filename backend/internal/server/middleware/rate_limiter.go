package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// RateLimitConfig 滑动窗口限流器配置
type RateLimitConfig struct {
	// Redis 客户端
	RedisClient *redis.Client
	// 时间窗口大小（秒）
	WindowSeconds int
	// 窗口内最大请求数
	MaxRequests int
	// 故障模式: "fail_open" 表示 Redis 不可用时放行, "fail_close" 表示 Redis 不可用时拒绝
	FailMode string
	// Key 前缀
	KeyPrefix string
}

// Redis Lua 脚本：滑动窗口限流（原子操作）
// KEYS[1] = 限流 key
// ARGV[1] = 窗口大小（秒）
// ARGV[2] = 最大请求数
// ARGV[3] = 当前时间戳（毫秒）
// ARGV[4] = 唯一请求标识
//
// 脚本逻辑:
//  1. 移除窗口外的旧记录
//  2. 统计当前窗口内的请求数
//  3. 如果未超限，添加当前请求记录并设置过期时间
//  4. 返回当前窗口内的请求数和 TTL
const slidingWindowLuaScript = `
local key = KEYS[1]
local window = tonumber(ARGV[1])
local max_requests = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local request_id = ARGV[4]

-- 计算窗口起始时间
local window_start = now - (window * 1000)

-- 移除窗口外的旧记录
redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

-- 统计当前窗口内的请求数
local current = redis.call('ZCARD', key)

if current < max_requests then
	-- 未超限，添加当前请求
	redis.call('ZADD', key, now, request_id)
	-- 设置过期时间，防止内存泄漏
	redis.call('PEXPIRE', key, window * 1000)
	return {current + 1, window * 1000}
else
	-- 超限，返回当前计数和 TTL
	local ttl = redis.call('PTTL', key)
	if ttl < 0 then
		ttl = window * 1000
	end
	return {current, ttl}
end
`

// RateLimiter 创建基于 Redis 滑动窗口的限流中间件
// 使用 Redis Lua 脚本保证原子性，支持 FailOpen/FailClose 两种故障模式
func RateLimiter(config RateLimitConfig) gin.HandlerFunc {
	// 设置默认值
	if config.WindowSeconds <= 0 {
		config.WindowSeconds = 60
	}
	if config.MaxRequests <= 0 {
		config.MaxRequests = 100
	}
	if config.FailMode == "" {
		config.FailMode = "fail_open"
	}
	if config.KeyPrefix == "" {
		config.KeyPrefix = "ratelimit"
	}

	return func(c *gin.Context) {
		// 生成限流 key，基于客户端 IP
		clientIP := c.ClientIP()
		rateLimitKey := fmt.Sprintf("%s:%s:%s", config.KeyPrefix, "sliding", clientIP)

		// 生成唯一请求标识
		requestID, _ := c.Get("X-Request-ID")
		if requestID == nil {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}

		// 执行 Lua 脚本
		now := time.Now().UnixMilli()
		result, err := config.RedisClient.Eval(
			context.Background(),
			slidingWindowLuaScript,
			[]string{rateLimitKey},
			config.WindowSeconds,
			config.MaxRequests,
			now,
			fmt.Sprintf("%v", requestID),
		).Slice()

		if err != nil {
			// Redis 执行失败，根据故障模式处理
			if config.FailMode == "fail_close" {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"error": gin.H{
						"code":    "RATE_LIMIT_SERVICE_ERROR",
						"message": "限流服务暂时不可用",
					},
				})
				return
			}
			// fail_open: Redis 不可用时放行
			c.Next()
			return
		}

		// 解析结果
		currentCount := 0
		ttlMs := int64(0)
		if len(result) >= 2 {
			if count, ok := result[0].(int64); ok {
				currentCount = int(count)
			}
			if ttl, ok := result[1].(int64); ok {
				ttlMs = ttl
			}
		}

		// 设置限流相关响应头
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.MaxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", intMax(0, config.MaxRequests-currentCount)))
		if ttlMs > 0 {
			resetTime := time.Now().Add(time.Duration(ttlMs) * time.Millisecond).Unix()
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))
		}

		// 检查是否超限
		if currentCount > config.MaxRequests {
			retryAfter := ttlMs / 1000
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": fmt.Sprintf("请求过于频繁，请在 %d 秒后重试", retryAfter),
				},
			})
			return
		}

		c.Next()
	}
}

// intMax 返回两个整数中的较大值
func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
