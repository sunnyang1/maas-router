// Package cache Wire 依赖注入 ProviderSet
package cache

import "github.com/google/wire"

// ProviderSet Cache 层 ProviderSet
// 提供 TokenBlacklist 实例
var ProviderSet = wire.NewSet(
	NewTokenBlacklist,
)
