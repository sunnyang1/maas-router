// Package repository 提供数据访问层实现
package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"entgo.io/ent/dialect/sql"

	"maas-router/ent"
	"maas-router/ent/apikey"
)

// APIKeyRepository API Key 数据访问仓库
type APIKeyRepository struct {
	client *ent.Client
}

// NewAPIKeyRepository 创建 API Key 仓库实例
func NewAPIKeyRepository(client *ent.Client) *APIKeyRepository {
	return &APIKeyRepository{client: client}
}

// APIKey API Key 实体
type APIKey struct {
	ID            int64
	UserID        int64
	KeyHash       string
	KeyPrefix     string
	Name          string
	Status        string
	DailyLimit    *float64
	MonthlyLimit  *float64
	AllowedModels []string
	IPWhitelist   []string
	IPBlacklist   []string
	ExpiresAt     *time.Time
	LastUsedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// APIKeyListFilter API Key 列表筛选条件
type APIKeyListFilter struct {
	UserID    int64
	Status    string
	Keyword   string
	SortBy    string
	SortOrder string
}

// CreateAPIKeyInput 创建 API Key 输入
type CreateAPIKeyInput struct {
	UserID        int64
	Key           string
	Name          string
	DailyLimit    *float64
	MonthlyLimit  *float64
	AllowedModels []string
	IPWhitelist   []string
	IPBlacklist   []string
	ExpiresAt     *time.Time
}

// UpdateAPIKeyInput 更新 API Key 输入
type UpdateAPIKeyInput struct {
	Name          *string
	Status        *string
	DailyLimit    *float64
	MonthlyLimit  *float64
	AllowedModels []string
	IPWhitelist   []string
	IPBlacklist   []string
	ExpiresAt     *time.Time
}

// Create 创建 API Key
func (r *APIKeyRepository) Create(ctx context.Context, input *CreateAPIKeyInput) (*APIKey, error) {
	// 计算Key哈希
	keyHash := r.hashKey(input.Key)
	keyPrefix := input.Key
	if len(input.Key) > 8 {
		keyPrefix = input.Key[:8]
	}

	create := r.client.APIKey.Create().
		SetUserID(input.UserID).
		SetKeyHash(keyHash).
		SetKeyPrefix(keyPrefix).
		SetNillableName(&input.Name)

	if input.DailyLimit != nil {
		create.SetDailyLimit(*input.DailyLimit)
	}
	if input.MonthlyLimit != nil {
		create.SetMonthlyLimit(*input.MonthlyLimit)
	}
	if input.AllowedModels != nil {
		create.SetAllowedModels(input.AllowedModels)
	}
	if input.IPWhitelist != nil {
		create.SetIPWhitelist(input.IPWhitelist)
	}
	if input.IPBlacklist != nil {
		create.SetIPBlacklist(input.IPBlacklist)
	}
	if input.ExpiresAt != nil {
		create.SetExpiresAt(*input.ExpiresAt)
	}

	k, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToAPIKey(k), nil
}

// Get 根据ID获取 API Key
func (r *APIKeyRepository) Get(ctx context.Context, id int64) (*APIKey, error) {
	k, err := r.client.APIKey.Query().
		Where(apikey.ID(id)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("API Key不存在")
		}
		return nil, err
	}

	return r.convertToAPIKey(k), nil
}

// GetByKeyHash 根据Key哈希获取 API Key
func (r *APIKeyRepository) GetByKeyHash(ctx context.Context, keyHash string) (*APIKey, error) {
	k, err := r.client.APIKey.Query().
		Where(apikey.KeyHash(keyHash)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("API Key不存在")
		}
		return nil, err
	}

	return r.convertToAPIKey(k), nil
}

// List 获取 API Key 列表
func (r *APIKeyRepository) List(ctx context.Context, filter APIKeyListFilter, page, pageSize int) ([]*APIKey, int64, error) {
	query := r.client.APIKey.Query()

	// 应用筛选条件
	if filter.UserID > 0 {
		query.Where(apikey.UserID(filter.UserID))
	}
	if filter.Status != "" {
		query.Where(apikey.Status(apikey.Status(filter.Status)))
	}
	if filter.Keyword != "" {
		query.Where(apikey.NameContains(filter.Keyword))
	}

	// 应用排序
	orderBy := apikey.FieldCreatedAt
	if filter.SortBy != "" {
		orderBy = filter.SortBy
	}
	orderDirection := sql.OrderDesc()
	if filter.SortOrder == "asc" {
		orderDirection = sql.OrderAsc()
	}
	query.Order(ent.OrderByFunc(func(s *sql.Selector) {
		s.OrderBy(sql.OrderExpr(orderDirection, orderBy))
	}))

	// 获取总数
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 分页
	offset := (page - 1) * pageSize
	keys, err := query.Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 转换
	result := make([]*APIKey, 0, len(keys))
	for _, k := range keys {
		result = append(result, r.convertToAPIKey(k))
	}

	return result, int64(total), nil
}

// Update 更新 API Key
func (r *APIKeyRepository) Update(ctx context.Context, id int64, input *UpdateAPIKeyInput) (*APIKey, error) {
	update := r.client.APIKey.UpdateOneID(id)

	if input.Name != nil {
		update.SetName(*input.Name)
	}
	if input.Status != nil {
		update.SetStatus(apikey.Status(*input.Status))
	}
	if input.DailyLimit != nil {
		update.SetDailyLimit(*input.DailyLimit)
	} else {
		update.ClearDailyLimit()
	}
	if input.MonthlyLimit != nil {
		update.SetMonthlyLimit(*input.MonthlyLimit)
	} else {
		update.ClearMonthlyLimit()
	}
	if input.AllowedModels != nil {
		update.SetAllowedModels(input.AllowedModels)
	}
	if input.IPWhitelist != nil {
		update.SetIPWhitelist(input.IPWhitelist)
	}
	if input.IPBlacklist != nil {
		update.SetIPBlacklist(input.IPBlacklist)
	}
	if input.ExpiresAt != nil {
		update.SetExpiresAt(*input.ExpiresAt)
	} else {
		update.ClearExpiresAt()
	}

	k, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToAPIKey(k), nil
}

// Delete 删除 API Key
func (r *APIKeyRepository) Delete(ctx context.Context, id int64) error {
	return r.client.APIKey.DeleteOneID(id).Exec(ctx)
}

// Revoke 撤销 API Key
func (r *APIKeyRepository) Revoke(ctx context.Context, id int64) error {
	_, err := r.client.APIKey.UpdateOneID(id).
		SetStatus(apikey.StatusRevoked).
		Save(ctx)
	return err
}

// UpdateLastUsedAt 更新最后使用时间
func (r *APIKeyRepository) UpdateLastUsedAt(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := r.client.APIKey.UpdateOneID(id).
		SetLastUsedAt(now).
		Save(ctx)
	return err
}

// GetActiveByUserID 获取用户的活跃 API Key 列表
func (r *APIKeyRepository) GetActiveByUserID(ctx context.Context, userID int64) ([]*APIKey, error) {
	now := time.Now()
	keys, err := r.client.APIKey.Query().
		Where(
			apikey.UserID(userID),
			apikey.Status(apikey.StatusActive),
			apikey.Or(
				apikey.ExpiresAtIsNil(),
				apikey.ExpiresAtGT(now),
			),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*APIKey, 0, len(keys))
	for _, k := range keys {
		result = append(result, r.convertToAPIKey(k))
	}

	return result, nil
}

// CountByUserID 统计用户的 API Key 数量
func (r *APIKeyRepository) CountByUserID(ctx context.Context, userID int64) (int64, error) {
	count, err := r.client.APIKey.Query().
		Where(apikey.UserID(userID)).
		Count(ctx)
	return int64(count), err
}

// hashKey 计算 API Key 的哈希值
func (r *APIKeyRepository) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// convertToAPIKey 转换Ent实体到领域实体
func (r *APIKeyRepository) convertToAPIKey(k *ent.APIKey) *APIKey {
	return &APIKey{
		ID:            k.ID,
		UserID:        k.UserID,
		KeyHash:       k.KeyHash,
		KeyPrefix:     k.KeyPrefix,
		Name:          k.Name,
		Status:        string(k.Status),
		DailyLimit:    k.DailyLimit,
		MonthlyLimit:  k.MonthlyLimit,
		AllowedModels: k.AllowedModels,
		IPWhitelist:   k.IPWhitelist,
		IPBlacklist:   k.IPBlacklist,
		ExpiresAt:     k.ExpiresAt,
		LastUsedAt:    k.LastUsedAt,
		CreatedAt:     k.CreatedAt,
		UpdatedAt:     k.UpdatedAt,
	}
}
