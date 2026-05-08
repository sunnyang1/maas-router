// Package main MaaS-Router 后端服务入口
// 支持 -setup（初始化模式）和 -version（查看版本号）参数
// 使用 Wire 进行依赖注入，Gin 作为 HTTP 框架
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"maas-router/internal/config"

	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// 版本号，在编译时通过 -ldflags 注入
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// 解析命令行参数
	setupMode := flag.Bool("setup", false, "初始化模式，用于数据库迁移等初始化操作")
	showVersion := flag.Bool("version", false, "显示版本信息")
	configPath := flag.String("config", "", "配置文件路径（默认搜索 ./config.yaml、./configs/config.yaml、/etc/maas-router/config.yaml）")
	flag.Parse()

	// 显示版本信息
	if *showVersion {
		printVersion()
		return
	}

	// 加载配置（不依赖 Wire，用于 setup 模式和早期初始化）
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 构建日志
	logger, err := config.BuildLogger(&cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("MaaS-Router 启动中",
		zap.String("version", version),
		zap.String("build_time", buildTime),
		zap.String("git_commit", gitCommit),
		zap.String("mode", cfg.Server.Mode),
	)

	// 初始化模式
	if *setupMode {
		runSetup(logger, cfg)
		return
	}

	// 使用 Wire 初始化应用
	app, err := InitializeApp(*configPath)
	if err != nil {
		logger.Fatal("应用初始化失败", zap.Error(err))
	}

	// 启动 HTTP 服务器
	startServer(app)
}

// printVersion 打印版本信息
func printVersion() {
	// 尝试从 VERSION 文件读取版本号
	versionFromFile := "unknown"
	if data, err := os.ReadFile("VERSION"); err == nil {
		versionFromFile = string(data)
	}

	fmt.Printf("MaaS-Router %s\n", versionFromFile)
	fmt.Printf("  Version:    %s\n", version)
	fmt.Printf("  Build Time: %s\n", buildTime)
	fmt.Printf("  Git Commit: %s\n", gitCommit)
}

// runSetup 执行初始化操作
// 包括数据库迁移、初始数据填充等
func runSetup(logger *zap.Logger, cfg *config.Config) {
	logger.Info("进入初始化模式")

	// TODO: 在此处添加数据库迁移逻辑
	// TODO: 在此处添加初始数据填充逻辑

	logger.Info("初始化完成")
}

// startServer 启动 HTTP 服务器并监听关闭信号
func startServer(app *App) {
	cfg := app.Config
	logger := app.Logger

	// 构建监听地址
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:         addr,
		Handler:      app.Engine,
		ReadTimeout:  time.Duration(cfg.Gateway.RequestTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Gateway.UpstreamTimeout) * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 根据配置决定是否启用 H2C
	if cfg.Server.EnableH2C {
		// 启用 HTTP/2 H2C（非 TLS 的 HTTP/2）
		h2cHandler := h2c.NewHandler(app.Engine, &http2.Server{
			MaxConcurrentStreams:         250,
			MaxReadFrameSize:             1 << 20, // 1MB
			InitialWindowSize:            1 << 24, // 16MB
			InitialConnectionWindowSize:  1 << 24, // 16MB
		})
		srv.Handler = h2cHandler
		logger.Info("已启用 HTTP/2 H2C")
	}

	// 在 goroutine 中启动服务器
	go func() {
		logger.Info("HTTP 服务器启动",
			zap.String("addr", addr),
			zap.String("mode", cfg.Server.Mode),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP 服务器启动失败", zap.Error(err))
		}
	}()

	// 优雅关闭：监听系统信号
	quit := make(chan os.Signal, 1)
	// 监听 SIGINT (Ctrl+C) 和 SIGTERM (kill)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("收到关闭信号，开始优雅关闭...",
		zap.String("signal", sig.String()),
	)

	// 创建带超时的关闭上下文
	shutdownTimeout := time.Duration(cfg.Server.ShutdownTimeout) * time.Second
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// 关闭 HTTP 服务器
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("HTTP 服务器关闭失败", zap.Error(err))
	} else {
		logger.Info("HTTP 服务器已优雅关闭")
	}

	logger.Info("MaaS-Router 已停止")
}
