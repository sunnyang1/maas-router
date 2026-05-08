// Package config Wire 依赖注入 ProviderSet
// 将配置加载和日志构建注册到 Wire 容器中
package config

import "github.com/google/wire"

// ProviderSet 配置层 ProviderSet
// 提供 Config 和 Logger 实例
var ProviderSet = wire.NewSet(
	LoadConfig,
	BuildLogger,
)
