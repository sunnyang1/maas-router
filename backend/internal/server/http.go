// Package server 提供 MaaS-Router 的 HTTP 服务器
package server

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"maas-router/internal/server/middleware"
)

// HTTPServerConfig HTTP 服务器配置
type HTTPServerConfig struct {
	// 监听地址，如 ":8080"
	Addr string
	// 运行模式: "debug", "release", "test"
	Mode string
	// 是否启用 H2C（非 TLS 的 HTTP/2）
	EnableH2C bool
	// TLS 配置（可选）
	TLSConfig *tls.Config
}

// HTTPServer 封装 HTTP 服务器及其依赖
type HTTPServer struct {
	engine *gin.Engine
	server *http.Server
	config HTTPServerConfig
}

// NewHTTPServer 创建并配置 HTTP 服务器
// 创建 Gin 引擎，注册全局中间件，支持 HTTP/2 H2C
func NewHTTPServer(config HTTPServerConfig) *HTTPServer {
	// 设置 Gin 运行模式
	if config.Mode != "" {
		gin.SetMode(config.Mode)
	}

	// 创建 Gin 引擎
	engine := gin.New()

	// ========== 注册全局中间件 ==========
	// 注意：中间件的注册顺序很重要，先注册的先执行

	// 1. 异常恢复中间件（最外层，确保所有 panic 都能被捕获）
	engine.Use(middleware.Recovery())

	// 2. 请求唯一 ID 中间件（尽早生成，方便后续中间件和 handler 使用）
	engine.Use(middleware.ClientRequestID())

	// 3. 请求体大小限制中间件
	engine.Use(middleware.BodyLimit())

	// 4. CORS 跨域中间件
	corsConfig := middleware.CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:8000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
	engine.Use(middleware.CORS(corsConfig))

	// 4. 可选：限流中间件（需要 Redis，在 RegisterRoutes 中按需挂载）

	srv := &HTTPServer{
		engine: engine,
		config: config,
	}

	// 配置底层 HTTP Server
	srv.server = &http.Server{
		Addr:         config.Addr,
		Handler:      srv.handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return srv
}

// Engine 返回 Gin 引擎实例，用于注册路由
func (s *HTTPServer) Engine() *gin.Engine {
	return s.engine
}

// handler 返回 HTTP Handler
// 如果启用了 H2C，则包装为支持 HTTP/2 Cleartext 的 Handler
func (s *HTTPServer) handler() http.Handler {
	if s.config.EnableH2C {
		// 使用 h2c 包装，支持非 TLS 的 HTTP/2
		h2s := &http2.Server{}
		return h2c.NewHandler(s.engine, h2s)
	}
	return s.engine
}

// ListenAndServe 启动 HTTP 服务器
func (s *HTTPServer) ListenAndServe() error {
	if s.config.TLSConfig != nil {
		// TLS 模式启动
		s.server.TLSConfig = s.config.TLSConfig
		return s.server.ListenAndServeTLS("", "")
	}
	// 普通模式启动（如果启用了 H2C，则自动支持 HTTP/2）
	return s.server.ListenAndServe()
}

// Shutdown 优雅关闭 HTTP 服务器
func (s *HTTPServer) Shutdown() error {
	return s.server.Close()
}
