// Package metrics 提供 Prometheus 指标收集功能
// 包括请求延迟统计、错误率统计、业务指标（Token 消耗、费用等）
package metrics

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Metrics 指标收集器
type Metrics struct {
	// HTTP 请求指标
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestSize     *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec

	// 业务指标
	TokenUsageTotal    *prometheus.CounterVec
	CostTotal          *prometheus.CounterVec
	RequestQueueSize   prometheus.Gauge
	ActiveConnections  prometheus.Gauge

	// 账号指标
	AccountLoadFactor   *prometheus.GaugeVec
	AccountConcurrency  *prometheus.GaugeVec
	AccountRPM          *prometheus.GaugeVec
	AccountErrorRate    *prometheus.GaugeVec

	// 缓存指标
	CacheHits      *prometheus.CounterVec
	CacheMisses    *prometheus.CounterVec
	CacheDuration  *prometheus.HistogramVec

	// 数据库指标
	DBQueryDuration *prometheus.HistogramVec
	DBConnections   prometheus.Gauge

	// 用户指标
	UserBalance    *prometheus.GaugeVec
	UserRequests   *prometheus.CounterVec

	// 注册表
	registry *prometheus.Registry
	logger   *zap.Logger
}

// NewMetrics 创建指标收集器
func NewMetrics(logger *zap.Logger) *Metrics {
	reg := prometheus.NewRegistry()

	m := &Metrics{
		registry: reg,
		logger:   logger,
	}

	// 初始化所有指标
	m.initHTTPMetrics()
	m.initBusinessMetrics()
	m.initAccountMetrics()
	m.initCacheMetrics()
	m.initDBMetrics()
	m.initUserMetrics()

	return m
}

// initHTTPMetrics 初始化 HTTP 指标
func (m *Metrics) initHTTPMetrics() {
	// HTTP 请求总数
	m.HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP 请求总数",
		},
		[]string{"method", "path", "status"},
	)

	// HTTP 请求延迟
	m.HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP 请求延迟（秒）",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// HTTP 请求大小
	m.HTTPRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP 请求大小（字节）",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	// HTTP 响应大小
	m.HTTPResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP 响应大小（字节）",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	// 注册指标
	m.registry.MustRegister(m.HTTPRequestsTotal)
	m.registry.MustRegister(m.HTTPRequestDuration)
	m.registry.MustRegister(m.HTTPRequestSize)
	m.registry.MustRegister(m.HTTPResponseSize)
}

// initBusinessMetrics 初始化业务指标
func (m *Metrics) initBusinessMetrics() {
	// Token 使用总量
	m.TokenUsageTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "token_usage_total",
			Help: "Token 使用总量",
		},
		[]string{"user_id", "model", "platform", "token_type"},
	)

	// 费用总计
	m.CostTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cost_total",
			Help: "费用总计（美元）",
		},
		[]string{"user_id", "model", "platform"},
	)

	// 请求队列大小
	m.RequestQueueSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "request_queue_size",
			Help: "当前请求队列大小",
		},
	)

	// 活跃连接数
	m.ActiveConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "当前活跃连接数",
		},
	)

	// 注册指标
	m.registry.MustRegister(m.TokenUsageTotal)
	m.registry.MustRegister(m.CostTotal)
	m.registry.MustRegister(m.RequestQueueSize)
	m.registry.MustRegister(m.ActiveConnections)
}

// initAccountMetrics 初始化账号指标
func (m *Metrics) initAccountMetrics() {
	// 账号负载因子
	m.AccountLoadFactor = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "account_load_factor",
			Help: "账号负载因子（0-1）",
		},
		[]string{"account_id", "platform"},
	)

	// 账号并发数
	m.AccountConcurrency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "account_concurrency",
			Help: "账号当前并发数",
		},
		[]string{"account_id", "platform"},
	)

	// 账号 RPM
	m.AccountRPM = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "account_rpm",
			Help: "账号当前 RPM（每分钟请求数）",
		},
		[]string{"account_id", "platform"},
	)

	// 账号错误率
	m.AccountErrorRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "account_error_rate",
			Help: "账号错误率（0-1）",
		},
		[]string{"account_id", "platform"},
	)

	// 注册指标
	m.registry.MustRegister(m.AccountLoadFactor)
	m.registry.MustRegister(m.AccountConcurrency)
	m.registry.MustRegister(m.AccountRPM)
	m.registry.MustRegister(m.AccountErrorRate)
}

// initCacheMetrics 初始化缓存指标
func (m *Metrics) initCacheMetrics() {
	// 缓存命中次数
	m.CacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "缓存命中次数",
		},
		[]string{"cache_type", "operation"},
	)

	// 缓存未命中次数
	m.CacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "缓存未命中次数",
		},
		[]string{"cache_type", "operation"},
	)

	// 缓存操作延迟
	m.CacheDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_duration_seconds",
			Help:    "缓存操作延迟（秒）",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"cache_type", "operation"},
	)

	// 注册指标
	m.registry.MustRegister(m.CacheHits)
	m.registry.MustRegister(m.CacheMisses)
	m.registry.MustRegister(m.CacheDuration)
}

// initDBMetrics 初始化数据库指标
func (m *Metrics) initDBMetrics() {
	// 数据库查询延迟
	m.DBQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "数据库查询延迟（秒）",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	// 数据库连接数
	m.DBConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections",
			Help: "当前数据库连接数",
		},
	)

	// 注册指标
	m.registry.MustRegister(m.DBQueryDuration)
	m.registry.MustRegister(m.DBConnections)
}

// initUserMetrics 初始化用户指标
func (m *Metrics) initUserMetrics() {
	// 用户余额
	m.UserBalance = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "user_balance",
			Help: "用户余额",
		},
		[]string{"user_id"},
	)

	// 用户请求数
	m.UserRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_requests_total",
			Help: "用户请求总数",
		},
		[]string{"user_id", "status"},
	)

	// 注册指标
	m.registry.MustRegister(m.UserBalance)
	m.registry.MustRegister(m.UserRequests)
}

// Handler 返回 Prometheus HTTP 处理器
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// RecordHTTPRequest 记录 HTTP 请求指标
func (m *Metrics) RecordHTTPRequest(method, path string, status int, duration time.Duration, requestSize, responseSize int64) {
	statusStr := strconv.Itoa(status)
	m.HTTPRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
	m.HTTPRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))
	m.HTTPResponseSize.WithLabelValues(method, path).Observe(float64(responseSize))
}

// RecordTokenUsage 记录 Token 使用指标
func (m *Metrics) RecordTokenUsage(userID, model, platform string, promptTokens, completionTokens int64) {
	m.TokenUsageTotal.WithLabelValues(userID, model, platform, "prompt").Add(float64(promptTokens))
	m.TokenUsageTotal.WithLabelValues(userID, model, platform, "completion").Add(float64(completionTokens))
	m.TokenUsageTotal.WithLabelValues(userID, model, platform, "total").Add(float64(promptTokens + completionTokens))
}

// RecordCost 记录费用指标
func (m *Metrics) RecordCost(userID, model, platform string, cost float64) {
	m.CostTotal.WithLabelValues(userID, model, platform).Add(cost)
}

// RecordAccountLoad 记录账号负载指标
func (m *Metrics) RecordAccountLoad(accountID, platform string, loadFactor float64, concurrency, rpm int, errorRate float64) {
	m.AccountLoadFactor.WithLabelValues(accountID, platform).Set(loadFactor)
	m.AccountConcurrency.WithLabelValues(accountID, platform).Set(float64(concurrency))
	m.AccountRPM.WithLabelValues(accountID, platform).Set(float64(rpm))
	m.AccountErrorRate.WithLabelValues(accountID, platform).Set(errorRate)
}

// RecordCacheHit 记录缓存命中
func (m *Metrics) RecordCacheHit(cacheType, operation string) {
	m.CacheHits.WithLabelValues(cacheType, operation).Inc()
}

// RecordCacheMiss 记录缓存未命中
func (m *Metrics) RecordCacheMiss(cacheType, operation string) {
	m.CacheMisses.WithLabelValues(cacheType, operation).Inc()
}

// RecordCacheDuration 记录缓存操作延迟
func (m *Metrics) RecordCacheDuration(cacheType, operation string, duration time.Duration) {
	m.CacheDuration.WithLabelValues(cacheType, operation).Observe(duration.Seconds())
}

// RecordDBQuery 记录数据库查询
func (m *Metrics) RecordDBQuery(operation, table string, duration time.Duration) {
	m.DBQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// SetDBConnections 设置数据库连接数
func (m *Metrics) SetDBConnections(count float64) {
	m.DBConnections.Set(count)
}

// SetUserBalance 设置用户余额
func (m *Metrics) SetUserBalance(userID string, balance float64) {
	m.UserBalance.WithLabelValues(userID).Set(balance)
}

// RecordUserRequest 记录用户请求
func (m *Metrics) RecordUserRequest(userID, status string) {
	m.UserRequests.WithLabelValues(userID, status).Inc()
}

// SetRequestQueueSize 设置请求队列大小
func (m *Metrics) SetRequestQueueSize(size float64) {
	m.RequestQueueSize.Set(size)
}

// SetActiveConnections 设置活跃连接数
func (m *Metrics) SetActiveConnections(count float64) {
	m.ActiveConnections.Set(count)
}

// GinMiddleware 返回 Gin 中间件
func (m *Metrics) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// 获取请求大小
		requestSize := c.Request.ContentLength

		// 包装响应写入器以获取响应大小
		wrapped := &responseWriter{
			ResponseWriter: c.Writer,
			size:           0,
		}
		c.Writer = wrapped

		// 执行后续处理
		c.Next()

		// 记录指标
		duration := time.Since(start)
		status := c.Writer.Status()

		m.RecordHTTPRequest(
			c.Request.Method,
			path,
			status,
			duration,
			requestSize,
			wrapped.size,
		)
	}
}

// responseWriter 包装 gin.ResponseWriter 以获取响应大小
type responseWriter struct {
	gin.ResponseWriter
	size int64
}

// Write 写入响应
func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += int64(n)
	return n, err
}

// WriteString 写入字符串
func (w *responseWriter) WriteString(s string) (int, error) {
	n, err := w.ResponseWriter.WriteString(s)
	w.size += int64(n)
	return n, err
}

// MetricsServer 指标服务器
type MetricsServer struct {
	metrics *Metrics
	server  *http.Server
	logger  *zap.Logger
}

// NewMetricsServer 创建指标服务器
func NewMetricsServer(metrics *Metrics, addr string, logger *zap.Logger) *MetricsServer {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())

	return &MetricsServer{
		metrics: metrics,
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
		logger: logger,
	}
}

// Start 启动指标服务器
func (s *MetricsServer) Start() error {
	s.logger.Info("启动指标服务器", zap.String("addr", s.server.Addr))
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("指标服务器错误", zap.Error(err))
		}
	}()
	return nil
}

// Stop 停止指标服务器
func (s *MetricsServer) Stop(ctx context.Context) error {
	s.logger.Info("停止指标服务器")
	return s.server.Shutdown(ctx)
}

// GetMetrics 获取指标收集器
func (s *MetricsServer) GetMetrics() *Metrics {
	return s.metrics
}

// DefaultPath 默认指标路径
const DefaultPath = "/metrics"

// Register 注册指标到 Gin 路由
func (m *Metrics) Register(router *gin.Engine) {
	router.GET(DefaultPath, gin.WrapH(m.Handler()))
}
