//go:build wireinject
// +build wireinject

// Package main Wire 依赖注入定义
// 定义应用各层的依赖关系，由 wire 工具生成实际的注入代码
package main

import (
	"maas-router/internal/config"

	"github.com/google/wire"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// App 应用结构体，持有所有核心依赖
type App struct {
	Config *config.Config
	Logger *zap.Logger
	Engine *gin.Engine
}

// InitializeApp 使用 Wire 初始化应用
// wire 会根据此函数签名自动生成依赖注入代码
func InitializeApp(configPath string) (*App, error) {
	wire.Build(
		// 配置层：加载配置和构建日志
		config.ProviderSet,

		// 以下为预留的各层 ProviderSet，待后续实现后取消注释
		// repository.ProviderSet,  // 数据访问层
		// service.ProviderSet,     // 业务逻辑层
		// handler.ProviderSet,     // HTTP 处理层

		// 服务器层：构建 Gin 引擎
		NewGinEngine,

		// 组装应用
		NewApp,
	)
	return nil, nil
}

// NewApp 创建应用实例
func NewApp(cfg *config.Config, logger *zap.Logger, engine *gin.Engine) *App {
	return &App{
		Config: cfg,
		Logger: logger,
		Engine: engine,
	}
}

// NewGinEngine 创建并配置 Gin 引擎
func NewGinEngine(cfg *config.Config, logger *zap.Logger) *gin.Engine {
	// 根据 Gin 模式设置运行模式
	switch cfg.Server.Mode {
	case "simple":
		gin.SetMode(gin.ReleaseMode)
	case "normal":
		gin.SetMode(gin.ReleaseMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// 使用 zap 作为日志中间件
	engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health", "/ready"},
	}))
	engine.Use(gin.Recovery())

	return engine
}
