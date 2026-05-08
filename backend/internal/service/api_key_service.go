// Package service 业务服务层
// 提供 API Key 服务
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"maas-router/ent"
	"maas-router/internal/cache"
	"maas-router/internal/config"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// APIKeyService API Key 服务接口
// 处理 API Key 的创建、管理、验证等
type APIKeyService interface {
	// Create 创建 Key
	Create(ctx context.Context, userID int64, name string, limits *APIKeyLimits) (*APIKeyCreateResult, error)

	// List 列出 Keys
	List(ctx context.Context, userID int64) ([]*ent.APIKey, error)

	// Get 获取 Key
	Get(ctx context.Context, keyID int64) (*ent.APIKey, error)

	// Update 更新 Key
	Update(ctx context.Context, keyID int64, data *APIKeyUpdateRequest) error

	// Delete 删除 Key
	Delete(ctx context.Context, keyID int64) error

	// Validate 验证 Key
	Validate(ctx context.Context, keyHash string) (*ent.APIKey, error)

	// ValidateWithCache 带缓存的验证
	ValidateWithCache(ctx context.Context, keyHash string) (*ent.APIKey, error)

	// Revoke 撤销 Key
	Revoke(ctx context.Context, keyID int64) error

	// Regenerate 重新生成 Key
	Regenerate(ctx context.Context, keyID int64) (*APIKeyCreateResult, error)

	// UpdateLastUsed 更新最后使用时间
	UpdateLastUsed(ctx context.Context, keyID int64) error

	// CheckPermissions 检查权限
	CheckPermissions(ctx context.Context, apiKey *ent.APIKey, model string, clientIP string) error
}

// APIKeyLimits API Key 限制
type APIKeyLimits struct {
	DailyLimit    *float64  `json:"daily_limit,omitempty"`
	MonthlyLimit  *float64  `json:"monthly_limit,omitempty"`
	AllowedModels []string  `json:"allowed_models,omitempty"`
	IPWhitelist   []string  `json:"ip_whitelist,omitempty"`
	IPBlacklist   []string  `json:"ip_blacklist,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

// APIKeyUpdateRequest API Key 更新请求
type APIKeyUpdateRequest struct {
	Name          *string    `json:"name,omitempty"`
	DailyLimit    *float64   `json:"daily_limit,omitempty"`
	MonthlyLimit  *float64   `json:"monthly_limit,omitempty"`
	AllowedModels []string   `json:"allowed_models,omitempty"`
	IPWhitelist   []string   `json:"ip_whitelist,omitempty"`
	IPBlacklist   []string   `json:"ip_blacklist,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

// APIKeyCreateResult API Key 创建结果
type APIKeyCreateResult struct {
	ID        int64   `json:"id"`
	Key       string  `json:"key"`        // 完整的 Key，只在创建时返回一次
	KeyPrefix string  `json:"key_prefix"` // Key 前缀，用于展示
	Name      string  `json:"name"`
	UserID    int64   `json:"user_id"`
	CreatedAt string  `json:"created_at"`
}

// apiKeyService API Key 服务实现
type apiKeyService struct {
	db       *ent.Client
	redis    *redis.Client
	cache    cache.Cache
	cacheKey *cache.CacheKey
	cfg      *config.Config
	logger   *zap.Logger
}

// NewAPIKeyService 创建 API Key 服务实例
func NewAPIKeyService(
	db *ent.Client,
	redis *redis.Client,
	cfg *config.Config,
	logger *zap.Logger,
) APIKeyService {
	// 创建统一缓存实例
	c := cache.NewCacheFromClient(redis, logger, "maas")
	return &apiKeyService{
		db:       db,
		redis:    redis,
		cache:    c,
		cacheKey: cache.NewCacheKey("maas"),
		cfg:      cfg,
		logger:   logger,
	}
}

// Create 创建 Key
func (s *apiKeyService) Create(ctx context.Context, userID int64, name string, limits *APIKeyLimits) (*APIKeyCreateResult, error) {
	// 检查用户是否存在
	user, err := s.db.User.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 检查用户状态
	if user.Status != ent.UserStatusActive {
		return nil, fmt.Errorf("用户账号已被禁用")
	}

	// 生成 API Key
	rawKey, keyHash, keyPrefix, err := s.generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("生成 API Key 失败: %w", err)
	}

	// 构建创建请求
	create := s.db.APIKey.Create().
		SetKeyHash(keyHash).
		SetKeyPrefix(keyPrefix).
		SetName(name).
		SetStatus(ent.APIKeyStatusActive).
		SetUserID(userID)

	// 设置限制
	if limits != nil {
		if limits.DailyLimit != nil {
			create = create.SetDailyLimit(*limits.DailyLimit)
		}
		if limits.MonthlyLimit != nil {
			create = create.SetMonthlyLimit(*limits.MonthlyLimit)
		}
		if len(limits.AllowedModels) > 0 {
			create = create.SetAllowedModels(limits.AllowedModels)
		}
		if len(limits.IPWhitelist) > 0 {
			create = create.SetIPWhitelist(limits.IPWhitelist)
		}
		if len(limits.IPBlacklist) > 0 {
			create = create.SetIPBlacklist(limits.IPBlacklist)
		}
		if limits.ExpiresAt != nil {
			create = create.SetExpiresAt(*limits.ExpiresAt)
		}
	}

	// 保存到数据库
	apiKey, err := create.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建 API Key 失败: %w", err)
	}

	s.logger.Info("API Key 创建成功",
		zap.Int64("key_id", apiKey.ID),
		zap.Int64("user_id", userID),
		zap.String("key_prefix", keyPrefix))

	return &APIKeyCreateResult{
		ID:        apiKey.ID,
		Key:       rawKey,
		KeyPrefix: keyPrefix,
		Name:      name,
		UserID:    userID,
		CreatedAt: apiKey.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// List 列出 Keys
func (s *apiKeyService) List(ctx context.Context, userID int64) ([]*ent.APIKey, error) {
	keys, err := s.db.APIKey.Query().
		Where(ent.APIKeyUserID(userID)).
		Order(ent.Desc(ent.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询 API Key 列表失败: %w", err)
	}
	return keys, nil
}

// Get 获取 Key
func (s *apiKeyService) Get(ctx context.Context, keyID int64) (*ent.APIKey, error) {
	key, err := s.db.APIKey.Get(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("获取 API Key 失败: %w", err)
	}
	return key, nil
}

// Update 更新 Key
func (s *apiKeyService) Update(ctx context.Context, keyID int64, data *APIKeyUpdateRequest) error {
	// 获取现有的 Key
	apiKey, err := s.db.APIKey.Get(ctx, keyID)
	if err != nil {
		return fmt.Errorf("API Key 不存在: %w", err)
	}

	// 检查状态
	if apiKey.Status == ent.APIKeyStatusRevoked {
		return fmt.Errorf("API Key 已被撤销，无法更新")
	}

	// 构建更新
	update := s.db.APIKey.UpdateOneID(keyID)

	if data.Name != nil {
		update = update.SetName(*data.Name)
	}
	if data.DailyLimit != nil {
		update = update.SetDailyLimit(*data.DailyLimit)
	}
	if data.MonthlyLimit != nil {
		update = update.SetMonthlyLimit(*data.MonthlyLimit)
	}
	if data.AllowedModels != nil {
		update = update.SetAllowedModels(data.AllowedModels)
	}
	if data.IPWhitelist != nil {
		update = update.SetIPWhitelist(data.IPWhitelist)
	}
	if data.IPBlacklist != nil {
		update = update.SetIPBlacklist(data.IPBlacklist)
	}
	if data.ExpiresAt != nil {
		update = update.SetExpiresAt(*data.ExpiresAt)
	}

	// 执行更新
	_, err = update.Save(ctx)
	if err != nil {
		return fmt.Errorf("更新 API Key 失败: %w", err)
	}

	// 清除缓存
	s.invalidateCache(ctx, keyID)

	s.logger.Info("API Key 更新成功",
		zap.Int64("key_id", keyID))

	return nil
}

// Delete 删除 Key
func (s *apiKeyService) Delete(ctx context.Context, keyID int64) error {
	// 删除数据库记录
	err := s.db.APIKey.DeleteOneID(keyID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除 API Key 失败: %w", err)
	}

	// 清除缓存
	s.invalidateCache(ctx, keyID)

	s.logger.Info("API Key 删除成功",
		zap.Int64("key_id", keyID))

	return nil
}

// Validate 验证 Key
func (s *apiKeyService) Validate(ctx context.Context, keyHash string) (*ent.APIKey, error) {
	// 查询 API Key
	apiKey, err := s.db.APIKey.Query().
		Where(ent.APIKeyKeyHash(keyHash)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("无效的 API Key")
		}
		return nil, fmt.Errorf("查询 API Key 失败: %w", err)
	}

	// 检查状态
	if apiKey.Status != ent.APIKeyStatusActive {
		return nil, fmt.Errorf("API Key 已被禁用或撤销")
	}

	// 检查过期时间
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("API Key 已过期")
	}

	return apiKey, nil
}

// ValidateWithCache 带缓存的验证
func (s *apiKeyService) ValidateWithCache(ctx context.Context, keyHash string) (*ent.APIKey, error) {
	// 先从缓存获取
	cacheKey := s.cacheKey.APIKey(keyHash)
	var apiKey ent.APIKey
	if err := s.cache.GetObject(ctx, cacheKey, &apiKey); err == nil {
		// 缓存命中，检查是否仍然有效
		if apiKey.Status == ent.APIKeyStatusActive {
			if apiKey.ExpiresAt == nil || apiKey.ExpiresAt.After(time.Now()) {
				s.logger.Debug("API Key 缓存命中", zap.Int64("key_id", apiKey.ID))
				return &apiKey, nil
			}
		}
		// 缓存过期或无效，删除缓存
		s.cache.Delete(ctx, cacheKey)
	}

	// 缓存未命中，从数据库验证
	apiKeyDB, err := s.Validate(ctx, keyHash)
	if err != nil {
		return nil, err
	}

	// 缓存结果（5分钟）
	if err := s.cache.SetObject(ctx, cacheKey, apiKeyDB, cache.CommonCacheTTL.Medium); err != nil {
		s.logger.Warn("缓存 API Key 失败", zap.Error(err))
	}

	return apiKeyDB, nil
}

// Revoke 撤销 Key
func (s *apiKeyService) Revoke(ctx context.Context, keyID int64) error {
	// 更新状态为已撤销
	_, err := s.db.APIKey.UpdateOneID(keyID).
		SetStatus(ent.APIKeyStatusRevoked).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("撤销 API Key 失败: %w", err)
	}

	// 清除缓存
	s.invalidateCache(ctx, keyID)

	s.logger.Info("API Key 已撤销",
		zap.Int64("key_id", keyID))

	return nil
}

// Regenerate 重新生成 Key
func (s *apiKeyService) Regenerate(ctx context.Context, keyID int64) (*APIKeyCreateResult, error) {
	// 获取现有的 Key
	oldKey, err := s.db.APIKey.Get(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("API Key 不存在: %w", err)
	}

	// 生成新的 Key
	rawKey, keyHash, keyPrefix, err := s.generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("生成 API Key 失败: %w", err)
	}

	// 更新数据库
	_, err = s.db.APIKey.UpdateOneID(keyID).
		SetKeyHash(keyHash).
		SetKeyPrefix(keyPrefix).
		SetStatus(ent.APIKeyStatusActive).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新 API Key 失败: %w", err)
	}

	// 清除缓存
	s.invalidateCache(ctx, keyID)

	s.logger.Info("API Key 重新生成成功",
		zap.Int64("key_id", keyID),
		zap.String("key_prefix", keyPrefix))

	return &APIKeyCreateResult{
		ID:        keyID,
		Key:       rawKey,
		KeyPrefix: keyPrefix,
		Name:      oldKey.Name,
		UserID:    oldKey.UserID,
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// UpdateLastUsed 更新最后使用时间
func (s *apiKeyService) UpdateLastUsed(ctx context.Context, keyID int64) error {
	_, err := s.db.APIKey.UpdateOneID(keyID).
		SetLastUsedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Warn("更新最后使用时间失败",
			zap.Int64("key_id", keyID),
			zap.Error(err))
	}
	return err
}

// CheckPermissions 检查权限
func (s *apiKeyService) CheckPermissions(ctx context.Context, apiKey *ent.APIKey, model string, clientIP string) error {
	// 检查模型权限
	if len(apiKey.AllowedModels) > 0 {
		allowed := false
		for _, m := range apiKey.AllowedModels {
			if m == model || m == "*" {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("模型 %s 不在允许列表中", model)
		}
	}

	// 检查 IP 白名单
	if len(apiKey.IPWhitelist) > 0 {
		inWhitelist := false
		for _, ip := range apiKey.IPWhitelist {
			if ip == clientIP || ip == "*" {
				inWhitelist = true
				break
			}
		}
		if !inWhitelist {
			return fmt.Errorf("IP %s 不在白名单中", clientIP)
		}
	}

	// 检查 IP 黑名单
	if len(apiKey.IPBlacklist) > 0 {
		for _, ip := range apiKey.IPBlacklist {
			if ip == clientIP {
				return fmt.Errorf("IP %s 已被禁止", clientIP)
			}
		}
	}

	return nil
}

// generateAPIKey 生成 API Key
func (s *apiKeyService) generateAPIKey() (rawKey, keyHash, keyPrefix string, err error) {
	// 生成 32 字节随机数
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", err
	}

	// 生成原始 Key (maas_xxxx 格式)
	rawKey = "maas_" + hex.EncodeToString(bytes)

	// 计算 Key Hash
	hash := sha256.Sum256([]byte(rawKey))
	keyHash = hex.EncodeToString(hash[:])

	// 提取 Key 前缀 (前 8 位)
	keyPrefix = rawKey[:8]

	return rawKey, keyHash, keyPrefix, nil
}

// invalidateCache 清除缓存
func (s *apiKeyService) invalidateCache(ctx context.Context, keyID int64) {
	// 获取 Key Hash 并删除缓存
	apiKey, err := s.db.APIKey.Get(ctx, keyID)
	if err == nil {
		cacheKey := s.cacheKey.APIKey(apiKey.KeyHash)
		userKey := s.cacheKey.APIKeyByUser(apiKey.UserID)
		if err := s.cache.Delete(ctx, cacheKey, userKey); err != nil {
			s.logger.Warn("清除 API Key 缓存失败", zap.Error(err))
		}
	}
}

// HashKey 计算 Key Hash
func HashKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(hash[:])
}

// GetKeyPrefix 提取 Key 前缀
func GetKeyPrefix(rawKey string) string {
	if len(rawKey) >= 8 {
		return rawKey[:8]
	}
	return rawKey
}

// CountByUser 统计用户的 API Key 数量
func (s *apiKeyService) CountByUser(ctx context.Context, userID int64) (int, error) {
	return s.db.APIKey.Query().
		Where(
			ent.APIKeyUserID(userID),
			ent.APIKeyStatusEQ(ent.APIKeyStatusActive),
		).
		Count(ctx)
}

// GetExpiredKeys 获取已过期的 Key
func (s *apiKeyService) GetExpiredKeys(ctx context.Context) ([]*ent.APIKey, error) {
	return s.db.APIKey.Query().
		Where(
			ent.APIKeyStatusEQ(ent.APIKeyStatusActive),
			ent.APIKeyExpiresAtLT(time.Now()),
		).
		All(ctx)
}

// MarkExpiredKeys 标记过期的 Key
func (s *apiKeyService) MarkExpiredKeys(ctx context.Context) (int, error) {
	keys, err := s.GetExpiredKeys(ctx)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, key := range keys {
		_, err := s.db.APIKey.UpdateOneID(key.ID).
			SetStatus(ent.APIKeyStatusExpired).
			Save(ctx)
		if err != nil {
			s.logger.Warn("标记过期 Key 失败",
				zap.Int64("key_id", key.ID),
				zap.Error(err))
			continue
		}
		count++

		// 清除缓存
		s.invalidateCache(ctx, key.ID)
	}

	if count > 0 {
		s.logger.Info("已标记过期的 API Key",
			zap.Int("count", count))
	}

	return count, nil
}

// GetUsageByKey 获取 Key 的使用统计
func (s *apiKeyService) GetUsageByKey(ctx context.Context, keyID int64, period string) (*UsageStats, error) {
	// 获取 API Key
	apiKey, err := s.db.APIKey.Get(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("获取 API Key 失败: %w", err)
	}

	// 计算时间范围
	var start, end time.Time
	now := time.Now()

	switch period {
	case "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = now
	case "month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = now
	default:
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = now
	}

	// 查询使用记录
	records, err := s.db.UsageRecord.Query().
		Where(
			ent.UsageRecordAPIKeyID(keyID),
			ent.UsageRecordCreatedAtGTE(start),
			ent.UsageRecordCreatedAtLTE(end),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询使用记录失败: %w", err)
	}

	// 计算统计
	stats := &UsageStats{
		UserID:      apiKey.UserID,
		PeriodStart: start,
		PeriodEnd:   end,
	}

	for _, record := range records {
		stats.TotalRequests++
		stats.TotalTokens += int64(record.TotalTokens)
		stats.TotalCost += record.Cost
	}

	return stats, nil
}

// CleanupRevokedKeys 清理已撤销的旧 Key
func (s *apiKeyService) CleanupRevokedKeys(ctx context.Context, olderThanDays int) (int, error) {
	cutoff := time.Now().AddDate(0, 0, -olderThanDays)

	// 查找需要清理的 Key
	keys, err := s.db.APIKey.Query().
		Where(
			ent.APIKeyStatusEQ(ent.APIKeyStatusRevoked),
			ent.APIKeyUpdatedAtLT(cutoff),
		).
		All(ctx)
	if err != nil {
		return 0, fmt.Errorf("查询需要清理的 Key 失败: %w", err)
	}

	count := 0
	for _, key := range keys {
		if err := s.db.APIKey.DeleteOneID(key.ID).Exec(ctx); err != nil {
			s.logger.Warn("删除已撤销的 Key 失败",
				zap.Int64("key_id", key.ID),
				zap.Error(err))
			continue
		}
		count++
	}

	if count > 0 {
		s.logger.Info("已清理已撤销的旧 API Key",
			zap.Int("count", count))
	}

	return count, nil
}
