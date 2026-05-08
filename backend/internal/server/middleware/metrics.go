// Package middleware 提供 metrics 中间件
// 自动收集请求指标并记录到 Prometheus
package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"maas-router/internal/metrics"
)

// MetricsMiddleware 指标收集中间件
type MetricsMiddleware struct {
	metrics *metrics.Metrics
	logger  *zap.Logger
}

// NewMetricsMiddleware 创建指标收集中间件
func NewMetricsMiddleware(m *metrics.Metrics, logger *zap.Logger) *MetricsMiddleware {
	return &MetricsMiddleware{
		metrics: m,
		logger:  logger,
	}
}

// Handler 返回 Gin 中间件处理函数
func (mm *MetricsMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 获取请求路径（使用 FullPath 获取路由模板而不是实际路径）
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// 获取请求大小
		requestSize := c.Request.ContentLength

		// 包装响应写入器以获取响应大小
		wrapped := &metricsResponseWriter{
			ResponseWriter: c.Writer,
			size:           0,
		}
		c.Writer = wrapped

		// 执行后续处理
		c.Next()

		// 计算处理时间
		duration := time.Since(start)
		status := c.Writer.Status()

		// 记录 HTTP 请求指标
		mm.metrics.RecordHTTPRequest(
			c.Request.Method,
			path,
			status,
			duration,
			requestSize,
			wrapped.size,
		)

		// 记录用户请求指标（如果存在用户信息）
		if userID, exists := c.Get("user_id"); exists {
			userIDStr := strconv.FormatInt(userID.(int64), 10)
			statusStr := "success"
			if status >= 400 {
				statusStr = "error"
			}
			mm.metrics.RecordUserRequest(userIDStr, statusStr)
		}

		// 慢请求日志（超过 1 秒）
		if duration > time.Second {
			mm.logger.Warn("慢请求",
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.Duration("duration", duration),
				zap.Int("status", status),
			)
		}
	}
}

// metricsResponseWriter 包装 gin.ResponseWriter 以获取响应大小
type metricsResponseWriter struct {
	gin.ResponseWriter
	size int64
}

// Write 写入响应
func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += int64(n)
	return n, err
}

// WriteString 写入字符串
func (w *metricsResponseWriter) WriteString(s string) (int, error) {
	n, err := w.ResponseWriter.WriteString(s)
	w.size += int64(n)
	return n, err
}

// MetricsConfig 指标中间件配置
type MetricsConfig struct {
	// 是否启用慢请求日志
	EnableSlowRequestLog bool
	// 慢请求阈值（毫秒）
	SlowRequestThreshold int
	// 是否记录用户指标
	EnableUserMetrics bool
}

// DefaultMetricsConfig 返回默认配置
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		EnableSlowRequestLog: true,
		SlowRequestThreshold: 1000,
		EnableUserMetrics:    true,
	}
}

// MetricsWithConfig 使用配置的指标中间件
func MetricsWithConfig(m *metrics.Metrics, logger *zap.Logger, config MetricsConfig) gin.HandlerFunc {
	mm := NewMetricsMiddleware(m, logger)
	return func(c *gin.Context) {
		start := time.Now()

		// 获取请求路径
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// 获取请求大小
		requestSize := c.Request.ContentLength

		// 包装响应写入器
		wrapped := &metricsResponseWriter{
			ResponseWriter: c.Writer,
			size:           0,
		}
		c.Writer = wrapped

		// 执行后续处理
		c.Next()

		// 计算处理时间
		duration := time.Since(start)
		status := c.Writer.Status()

		// 记录 HTTP 请求指标
		m.RecordHTTPRequest(
			c.Request.Method,
			path,
			status,
			duration,
			requestSize,
			wrapped.size,
		)

		// 记录用户请求指标
		if config.EnableUserMetrics {
			if userID, exists := c.Get("user_id"); exists {
				userIDStr := strconv.FormatInt(userID.(int64), 10)
				statusStr := "success"
				if status >= 400 {
					statusStr = "error"
				}
				m.RecordUserRequest(userIDStr, statusStr)
			}
		}

		// 慢请求日志
		if config.EnableSlowRequestLog && duration > time.Duration(config.SlowRequestThreshold)*time.Millisecond {
			logger.Warn("慢请求",
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.Duration("duration", duration),
				zap.Int("status", status),
			)
		}
	}
}

// CacheMetricsMiddleware 缓存指标中间件
// 用于记录缓存命中/未命中情况
type CacheMetricsMiddleware struct {
	metrics *metrics.Metrics
}

// NewCacheMetricsMiddleware 创建缓存指标中间件
func NewCacheMetricsMiddleware(m *metrics.Metrics) *CacheMetricsMiddleware {
	return &CacheMetricsMiddleware{
		metrics: m,
	}
}

// RecordCacheHit 记录缓存命中
func (cmm *CacheMetricsMiddleware) RecordCacheHit(cacheType, operation string) {
	cmm.metrics.RecordCacheHit(cacheType, operation)
}

// RecordCacheMiss 记录缓存未命中
func (cmm *CacheMetricsMiddleware) RecordCacheMiss(cacheType, operation string) {
	cmm.metrics.RecordCacheMiss(cacheType, operation)
}

// RecordCacheDuration 记录缓存操作延迟
func (cmm *CacheMetricsMiddleware) RecordCacheDuration(cacheType, operation string, duration time.Duration) {
	cmm.metrics.RecordCacheDuration(cacheType, operation, duration)
}

// DBMetricsMiddleware 数据库指标中间件
type DBMetricsMiddleware struct {
	metrics *metrics.Metrics
}

// NewDBMetricsMiddleware 创建数据库指标中间件
func NewDBMetricsMiddleware(m *metrics.Metrics) *DBMetricsMiddleware {
	return &DBMetricsMiddleware{
		metrics: m,
	}
}

// RecordQuery 记录数据库查询
func (dbm *DBMetricsMiddleware) RecordQuery(operation, table string, duration time.Duration) {
	dbm.metrics.RecordDBQuery(operation, table, duration)
}

// SetConnections 设置数据库连接数
func (dbm *DBMetricsMiddleware) SetConnections(count float64) {
	dbm.metrics.SetDBConnections(count)
}

// BusinessMetricsMiddleware 业务指标中间件
type BusinessMetricsMiddleware struct {
	metrics *metrics.Metrics
}

// NewBusinessMetricsMiddleware 创建业务指标中间件
func NewBusinessMetricsMiddleware(m *metrics.Metrics) *BusinessMetricsMiddleware {
	return &BusinessMetricsMiddleware{
		metrics: m,
	}
}

// RecordTokenUsage 记录 Token 使用
func (bmm *BusinessMetricsMiddleware) RecordTokenUsage(userID, model, platform string, promptTokens, completionTokens int64) {
	bmm.metrics.RecordTokenUsage(userID, model, platform, promptTokens, completionTokens)
}

// RecordCost 记录费用
func (bmm *BusinessMetricsMiddleware) RecordCost(userID, model, platform string, cost float64) {
	bmm.metrics.RecordCost(userID, model, platform, cost)
}

// SetUserBalance 设置用户余额
func (bmm *BusinessMetricsMiddleware) SetUserBalance(userID string, balance float64) {
	bmm.metrics.SetUserBalance(userID, balance)
}

// SetRequestQueueSize 设置请求队列大小
func (bmm *BusinessMetricsMiddleware) SetRequestQueueSize(size float64) {
	bmm.metrics.SetRequestQueueSize(size)
}

// SetActiveConnections 设置活跃连接数
func (bmm *BusinessMetricsMiddleware) SetActiveConnections(count float64) {
	bmm.metrics.SetActiveConnections(count)
}

// AccountMetricsMiddleware 账号指标中间件
type AccountMetricsMiddleware struct {
	metrics *metrics.Metrics
}

// NewAccountMetricsMiddleware 创建账号指标中间件
func NewAccountMetricsMiddleware(m *metrics.Metrics) *AccountMetricsMiddleware {
	return &AccountMetricsMiddleware{
		metrics: m,
	}
}

// RecordAccountLoad 记录账号负载
func (amm *AccountMetricsMiddleware) RecordAccountLoad(accountID, platform string, loadFactor float64, concurrency, rpm int, errorRate float64) {
	amm.metrics.RecordAccountLoad(accountID, platform, loadFactor, concurrency, rpm, errorRate)
}
