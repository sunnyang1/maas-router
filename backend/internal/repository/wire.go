// Package repository 提供数据访问层实现
// Wire ProviderSet 定义
package repository

import (
	"github.com/google/wire"
)

// ProviderSet Repository 层的 Wire ProviderSet
// 包含所有 Repository 的构造函数，用于依赖注入
var ProviderSet = wire.NewSet(
	NewUserRepository,
	NewAPIKeyRepository,
	NewAccountRepository,
	NewGroupRepository,
	NewUsageRecordRepository,
	NewRouterRuleRepository,
)
