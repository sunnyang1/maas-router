// Package cache 提供 Token 黑名单功能
package cache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// TokenBlacklist Token 黑名单服务
type TokenBlacklist struct {
	redis *redis.Client
}

// NewTokenBlacklist 创建 Token 黑名单服务
func NewTokenBlacklist(redis *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{redis: redis}
}

// AddToken 将 Token 加入黑名单
// tokenID: JWT 的 jti (JWT ID)
// expiration: Token 剩余的有效期，黑名单将在该时间后自动过期
func (b *TokenBlacklist) AddToken(ctx context.Context, tokenID string, expiration time.Duration) error {
	return b.redis.Set(ctx, "blacklist:token:"+tokenID, "1", expiration).Err()
}

// IsBlacklisted 检查 Token 是否在黑名单中
func (b *TokenBlacklist) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	exists, err := b.redis.Exists(ctx, "blacklist:token:"+tokenID).Result()
	return exists > 0, err
}

// AddRefreshToken 将刷新 Token 加入黑名单
func (b *TokenBlacklist) AddRefreshToken(ctx context.Context, tokenID string, expiration time.Duration) error {
	return b.redis.Set(ctx, "blacklist:refresh:"+tokenID, "1", expiration).Err()
}

// IsRefreshTokenBlacklisted 检查刷新 Token 是否在黑名单中
func (b *TokenBlacklist) IsRefreshTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	exists, err := b.redis.Exists(ctx, "blacklist:refresh:"+tokenID).Result()
	return exists > 0, err
}
