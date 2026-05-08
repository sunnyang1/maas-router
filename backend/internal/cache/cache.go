// Package cache 提供统一的缓存接口和实现
// 支持 Redis 作为后端存储，提供 Get/Set/Delete 和批量操作功能
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// Cache 统一缓存接口
type Cache interface {
	// Get 获取缓存值
	Get(ctx context.Context, key string) (string, error)

	// GetObject 获取缓存对象并反序列化
	GetObject(ctx context.Context, key string, dest interface{}) error

	// Set 设置缓存值
	Set(ctx context.Context, key string, value string, ttl time.Duration) error

	// SetObject 设置缓存对象（自动序列化）
	SetObject(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete 删除缓存
	Delete(ctx context.Context, keys ...string) error

	// DeletePattern 按模式删除缓存
	DeletePattern(ctx context.Context, pattern string) error

	// Exists 检查缓存是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// TTL 获取缓存剩余时间
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Expire 设置缓存过期时间
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// MGet 批量获取
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)

	// MSet 批量设置
	MSet(ctx context.Context, items map[string]string, ttl time.Duration) error

	// Incr 原子递增
	Incr(ctx context.Context, key string) (int64, error)

	// IncrBy 原子递增指定值
	IncrBy(ctx context.Context, key string, value int64) (int64, error)

	// Decr 原子递减
	Decr(ctx context.Context, key string) (int64, error)

	// DecrBy 原子递减指定值
	DecrBy(ctx context.Context, key string, value int64) (int64, error)

	// GetClient 获取底层 Redis 客户端
	GetClient() *redis.Client

	// Close 关闭缓存连接
	Close() error
}

// redisCache Redis 缓存实现
type redisCache struct {
	client *redis.Client
	logger *zap.Logger
	prefix string
}

// Config 缓存配置
type Config struct {
	// Redis 地址
	Addr string
	// Redis 密码
	Password string
	// Redis 数据库
	DB int
	// 连接池大小
	PoolSize int
	// 键前缀
	KeyPrefix string
}

// NewCache 创建缓存实例
func NewCache(cfg *Config, logger *zap.Logger) (Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接 Redis 失败: %w", err)
	}

	return &redisCache{
		client: client,
		logger: logger,
		prefix: cfg.KeyPrefix,
	}, nil
}

// NewCacheFromClient 从现有 Redis 客户端创建缓存实例
func NewCacheFromClient(client *redis.Client, logger *zap.Logger, prefix string) Cache {
	return &redisCache{
		client: client,
		logger: logger,
		prefix: prefix,
	}
}

// makeKey 生成带前缀的键
func (c *redisCache) makeKey(key string) string {
	if c.prefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", c.prefix, key)
}

// Get 获取缓存值
func (c *redisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, c.makeKey(key)).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("缓存不存在: %s", key)
	}
	if err != nil {
		return "", fmt.Errorf("获取缓存失败: %w", err)
	}
	return val, nil
}

// GetObject 获取缓存对象并反序列化
func (c *redisCache) GetObject(ctx context.Context, key string, dest interface{}) error {
	data, err := c.Get(ctx, key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("反序列化缓存对象失败: %w", err)
	}
	return nil
}

// Set 设置缓存值
func (c *redisCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if err := c.client.Set(ctx, c.makeKey(key), value, ttl).Err(); err != nil {
		return fmt.Errorf("设置缓存失败: %w", err)
	}
	return nil
}

// SetObject 设置缓存对象（自动序列化）
func (c *redisCache) SetObject(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化缓存对象失败: %w", err)
	}

	return c.Set(ctx, key, string(data), ttl)
}

// Delete 删除缓存
func (c *redisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	// 添加前缀
	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = c.makeKey(key)
	}

	if err := c.client.Del(ctx, prefixedKeys...).Err(); err != nil {
		return fmt.Errorf("删除缓存失败: %w", err)
	}
	return nil
}

// DeletePattern 按模式删除缓存
func (c *redisCache) DeletePattern(ctx context.Context, pattern string) error {
	fullPattern := c.makeKey(pattern)
	iter := c.client.Scan(ctx, 0, fullPattern, 0).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		// 每 100 个键批量删除，避免内存占用过大
		if len(keys) >= 100 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				c.logger.Warn("批量删除缓存失败", zap.Error(err))
			}
			keys = keys[:0]
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("扫描缓存键失败: %w", err)
	}

	// 删除剩余的键
	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("删除缓存失败: %w", err)
		}
	}

	return nil
}

// Exists 检查缓存是否存在
func (c *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, c.makeKey(key)).Result()
	if err != nil {
		return false, fmt.Errorf("检查缓存存在失败: %w", err)
	}
	return n > 0, nil
}

// TTL 获取缓存剩余时间
func (c *redisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := c.client.TTL(ctx, c.makeKey(key)).Result()
	if err != nil {
		return 0, fmt.Errorf("获取缓存 TTL 失败: %w", err)
	}
	return ttl, nil
}

// Expire 设置缓存过期时间
func (c *redisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if err := c.client.Expire(ctx, c.makeKey(key), ttl).Err(); err != nil {
		return fmt.Errorf("设置缓存过期时间失败: %w", err)
	}
	return nil
}

// MGet 批量获取
func (c *redisCache) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	if len(keys) == 0 {
		return []interface{}{}, nil
	}

	// 添加前缀
	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = c.makeKey(key)
	}

	vals, err := c.client.MGet(ctx, prefixedKeys...).Result()
	if err != nil {
		return nil, fmt.Errorf("批量获取缓存失败: %w", err)
	}
	return vals, nil
}

// MSet 批量设置
func (c *redisCache) MSet(ctx context.Context, items map[string]string, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	// 添加前缀
	prefixedItems := make(map[string]string, len(items))
	for key, value := range items {
		prefixedItems[c.makeKey(key)] = value
	}

	// 使用 Pipeline 批量设置
	pipe := c.client.Pipeline()
	for key, value := range prefixedItems {
		pipe.Set(ctx, key, value, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("批量设置缓存失败: %w", err)
	}
	return nil
}

// Incr 原子递增
func (c *redisCache) Incr(ctx context.Context, key string) (int64, error) {
	val, err := c.client.Incr(ctx, c.makeKey(key)).Result()
	if err != nil {
		return 0, fmt.Errorf("缓存递增失败: %w", err)
	}
	return val, nil
}

// IncrBy 原子递增指定值
func (c *redisCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	val, err := c.client.IncrBy(ctx, c.makeKey(key), value).Result()
	if err != nil {
		return 0, fmt.Errorf("缓存递增失败: %w", err)
	}
	return val, nil
}

// Decr 原子递减
func (c *redisCache) Decr(ctx context.Context, key string) (int64, error) {
	val, err := c.client.Decr(ctx, c.makeKey(key)).Result()
	if err != nil {
		return 0, fmt.Errorf("缓存递减失败: %w", err)
	}
	return val, nil
}

// DecrBy 原子递减指定值
func (c *redisCache) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	val, err := c.client.DecrBy(ctx, c.makeKey(key), value).Result()
	if err != nil {
		return 0, fmt.Errorf("缓存递减失败: %w", err)
	}
	return val, nil
}

// GetClient 获取底层 Redis 客户端
func (c *redisCache) GetClient() *redis.Client {
	return c.client
}

// Close 关闭缓存连接
func (c *redisCache) Close() error {
	return c.client.Close()
}

// CacheKey 缓存键生成器
type CacheKey struct {
	prefix string
}

// NewCacheKey 创建缓存键生成器
func NewCacheKey(prefix string) *CacheKey {
	return &CacheKey{prefix: prefix}
}

// User 用户相关缓存键
func (k *CacheKey) User(userID int64) string {
	return fmt.Sprintf("%s:user:%d", k.prefix, userID)
}

// UserByEmail 根据邮箱缓存键
func (k *CacheKey) UserByEmail(email string) string {
	return fmt.Sprintf("%s:user:email:%s", k.prefix, email)
}

// APIKey API Key 缓存键
func (k *CacheKey) APIKey(keyHash string) string {
	return fmt.Sprintf("%s:apikey:%s", k.prefix, keyHash)
}

// APIKeyByUser 用户 API Key 列表缓存键
func (k *CacheKey) APIKeyByUser(userID int64) string {
	return fmt.Sprintf("%s:apikey:user:%d", k.prefix, userID)
}

// Account 账号缓存键
func (k *CacheKey) Account(accountID int64) string {
	return fmt.Sprintf("%s:account:%d", k.prefix, accountID)
}

// AccountLoad 账号负载缓存键
func (k *CacheKey) AccountLoad(accountID int64) string {
	return fmt.Sprintf("%s:account:%d:load", k.prefix, accountID)
}

// AccountList 账号列表缓存键
func (k *CacheKey) AccountList(platform string) string {
	return fmt.Sprintf("%s:account:list:%s", k.prefix, platform)
}

// Balance 用户余额缓存键
func (k *CacheKey) Balance(userID int64) string {
	return fmt.Sprintf("%s:balance:%d", k.prefix, userID)
}

// UsageStats 使用统计缓存键
func (k *CacheKey) UsageStats(userID int64, period string) string {
	return fmt.Sprintf("%s:usage:%d:%s", k.prefix, userID, period)
}

// DailyUsage 每日使用缓存键
func (k *CacheKey) DailyUsage(userID int64, date string) string {
	return fmt.Sprintf("%s:usage:%d:daily:%s", k.prefix, userID, date)
}

// BillingRule 计费规则缓存键
func (k *CacheKey) BillingRule(model string) string {
	return fmt.Sprintf("%s:billing:rule:%s", k.prefix, model)
}

// RateLimit 限流缓存键
func (k *CacheKey) RateLimit(key string) string {
	return fmt.Sprintf("%s:ratelimit:%s", k.prefix, key)
}

// StickySession Sticky Session 缓存键
func (k *CacheKey) StickySession(sessionID string) string {
	return fmt.Sprintf("%s:sticky:%s", k.prefix, sessionID)
}

// RefreshToken 刷新令牌缓存键
func (k *CacheKey) RefreshToken(userID int64) string {
	return fmt.Sprintf("%s:refresh_token:%d", k.prefix, userID)
}

// CommonCacheTTL 常用缓存过期时间
var CommonCacheTTL = struct {
	// Short 短缓存（1分钟）
	Short time.Duration
	// Medium 中等缓存（5分钟）
	Medium time.Duration
	// Long 长缓存（1小时）
	Long time.Duration
	// VeryLong 超长缓存（24小时）
	VeryLong time.Duration
	// Session Session 缓存（30分钟）
	Session time.Duration
}{
	Short:    1 * time.Minute,
	Medium:   5 * time.Minute,
	Long:     1 * time.Hour,
	VeryLong: 24 * time.Hour,
	Session:  30 * time.Minute,
}
