// Package cache 提供缓存中间件实现
// 支持自动缓存响应和 Cache-Control 头处理
package cache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ResponseCache 响应缓存中间件配置
type ResponseCache struct {
	// 缓存实例
	cache Cache
	// 日志
	logger *zap.Logger
	// 默认 TTL
	defaultTTL time.Duration
	// 缓存键前缀
	keyPrefix string
	// 跳过缓存的条件
	skipConditions []SkipCondition
	// 缓存状态码
	cacheableStatus []int
}

// SkipCondition 跳过缓存的条件函数
type SkipCondition func(*gin.Context) bool

// ResponseCacheConfig 响应缓存配置
type ResponseCacheConfig struct {
	// 缓存实例（必需）
	Cache Cache
	// 日志实例（必需）
	Logger *zap.Logger
	// 默认缓存时间（默认 5 分钟）
	DefaultTTL time.Duration
	// 缓存键前缀（默认 "response"）
	KeyPrefix string
	// 跳过缓存的条件
	SkipConditions []SkipCondition
	// 可缓存的状态码（默认 200）
	CacheableStatus []int
}

// NewResponseCache 创建响应缓存中间件
func NewResponseCache(cfg ResponseCacheConfig) *ResponseCache {
	if cfg.DefaultTTL == 0 {
		cfg.DefaultTTL = 5 * time.Minute
	}
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "response"
	}
	if len(cfg.CacheableStatus) == 0 {
		cfg.CacheableStatus = []int{http.StatusOK}
	}

	return &ResponseCache{
		cache:           cfg.Cache,
		logger:          cfg.Logger,
		defaultTTL:      cfg.DefaultTTL,
		keyPrefix:       cfg.KeyPrefix,
		skipConditions:  cfg.SkipConditions,
		cacheableStatus: cfg.CacheableStatus,
	}
}

// Middleware 返回 Gin 中间件
func (rc *ResponseCache) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否应该跳过缓存
		if rc.shouldSkip(c) {
			c.Next()
			return
		}

		// 生成缓存键
		cacheKey := rc.generateCacheKey(c)

		// 尝试从缓存获取响应
		if cached := rc.getFromCache(c, cacheKey); cached != nil {
			// 设置缓存命中头
			c.Header("X-Cache", "HIT")
			c.Data(cached.StatusCode, cached.ContentType, cached.Body)
			c.Abort()
			return
		}

		// 缓存未命中，设置缓存未命中头
		c.Header("X-Cache", "MISS")

		// 包装响应写入器
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		// 执行后续处理
		c.Next()

		// 检查是否应该缓存响应
		if rc.shouldCacheResponse(c, writer.Status()) {
			rc.saveToCache(cacheKey, writer)
		}
	}
}

// shouldSkip 检查是否应该跳过缓存
func (rc *ResponseCache) shouldSkip(c *gin.Context) bool {
	// 非 GET 请求不缓存
	if c.Request.Method != http.MethodGet {
		return true
	}

	// 检查 Cache-Control 头
	cacheControl := c.GetHeader("Cache-Control")
	if strings.Contains(cacheControl, "no-cache") ||
		strings.Contains(cacheControl, "no-store") ||
		strings.Contains(cacheControl, "max-age=0") {
		return true
	}

	// 执行自定义跳过条件
	for _, condition := range rc.skipConditions {
		if condition(c) {
			return true
		}
	}

	return false
}

// generateCacheKey 生成缓存键
func (rc *ResponseCache) generateCacheKey(c *gin.Context) string {
	// 构建键的基础部分
	key := fmt.Sprintf("%s:%s:%s", rc.keyPrefix, c.Request.Method, c.Request.URL.Path)

	// 添加查询参数
	if query := c.Request.URL.RawQuery; query != "" {
		key = fmt.Sprintf("%s?%s", key, query)
	}

	// 添加用户特定标识（如果有）
	if userID, exists := c.Get("user_id"); exists {
		key = fmt.Sprintf("%s:user:%v", key, userID)
	}

	// 使用 SHA256 哈希键，避免键过长
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// CachedResponse 缓存的响应数据
type CachedResponse struct {
	StatusCode  int           `json:"status_code"`
	ContentType string        `json:"content_type"`
	Body        []byte        `json:"body"`
	Headers     http.Header   `json:"headers"`
	CachedAt    time.Time     `json:"cached_at"`
	TTL         time.Duration `json:"ttl"`
}

// getFromCache 从缓存获取响应
func (rc *ResponseCache) getFromCache(c *gin.Context, key string) *CachedResponse {
	var response CachedResponse
	if err := rc.cache.GetObject(c.Request.Context(), key, &response); err != nil {
		return nil
	}

	// 检查缓存是否过期
	if time.Since(response.CachedAt) > response.TTL {
		return nil
	}

	rc.logger.Debug("缓存命中",
		zap.String("key", key),
		zap.String("path", c.Request.URL.Path))

	return &response
}

// shouldCacheResponse 检查是否应该缓存响应
func (rc *ResponseCache) shouldCacheResponse(c *gin.Context, statusCode int) bool {
	// 检查状态码是否在可缓存列表中
	for _, code := range rc.cacheableStatus {
		if code == statusCode {
			return true
		}
	}
	return false
}

// saveToCache 保存响应到缓存
func (rc *ResponseCache) saveToCache(key string, writer *responseWriter) {
	response := &CachedResponse{
		StatusCode:  writer.Status(),
		ContentType: writer.Header().Get("Content-Type"),
		Body:        writer.body.Bytes(),
		Headers:     writer.Header().Clone(),
		CachedAt:    time.Now(),
		TTL:         rc.defaultTTL,
	}

	// 移除不应缓存的头
	response.Headers.Del("Set-Cookie")
	response.Headers.Del("X-Cache")

	ctx := context.Background()
	if err := rc.cache.SetObject(ctx, key, response, rc.defaultTTL); err != nil {
		rc.logger.Warn("保存响应到缓存失败",
			zap.String("key", key),
			zap.Error(err))
	} else {
		rc.logger.Debug("响应已缓存",
			zap.String("key", key),
			zap.Duration("ttl", rc.defaultTTL))
	}
}

// responseWriter 包装 Gin 的 ResponseWriter
type responseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

// Write 写入响应体
func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// WriteHeader 写入状态码
func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Status 获取状态码
func (w *responseWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

// CacheControl 生成 Cache-Control 头值
func CacheControl(maxAge time.Duration, directives ...string) string {
	parts := []string{fmt.Sprintf("max-age=%d", int(maxAge.Seconds()))}
	parts = append(parts, directives...)
	return strings.Join(parts, ", ")
}

// NoCache 返回禁止缓存的 Cache-Control 头值
func NoCache() string {
	return "no-cache, no-store, must-revalidate"
}

// PrivateCache 返回私有缓存的 Cache-Control 头值
func PrivateCache(maxAge time.Duration) string {
	return fmt.Sprintf("private, max-age=%d", int(maxAge.Seconds()))
}

// PublicCache 返回公共缓存的 Cache-Control 头值
func PublicCache(maxAge time.Duration) string {
	return fmt.Sprintf("public, max-age=%d", int(maxAge.Seconds()))
}

// SetCacheHeaders 设置缓存相关的响应头
func SetCacheHeaders(c *gin.Context, maxAge time.Duration, directives ...string) {
	c.Header("Cache-Control", CacheControl(maxAge, directives...))
	c.Header("Expires", time.Now().Add(maxAge).Format(http.TimeFormat))
}

// SetNoCacheHeaders 设置禁止缓存的响应头
func SetNoCacheHeaders(c *gin.Context) {
	c.Header("Cache-Control", NoCache())
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
}

// InvalidateCache 使指定模式的缓存失效
func InvalidateCache(cache Cache, pattern string) error {
	ctx := context.Background()
	return cache.DeletePattern(ctx, pattern)
}

// InvalidateUserCache 使用户相关的缓存失效
func InvalidateUserCache(cache Cache, userID int64) error {
	ctx := context.Background()
	key := fmt.Sprintf("*user:%d*", userID)
	return cache.DeletePattern(ctx, key)
}

// InvalidateAPIKeyCache 使 API Key 相关的缓存失效
func InvalidateAPIKeyCache(cache Cache, keyHash string) error {
	ctx := context.Background()
	key := fmt.Sprintf("*apikey:%s*", keyHash)
	return cache.DeletePattern(ctx, key)
}
