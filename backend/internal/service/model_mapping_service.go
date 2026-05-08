// Package service 提供 MaaS-Router 的业务逻辑服务层
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"maas-router/internal/cache"
)

// ModelMappingService 处理模型名称映射/别名
type ModelMappingService interface {
	// ResolveMapping 通过映射链解析模型名称
	// 例如 "gpt-4" -> "claude-3-5-sonnet-20241022"（如果已配置）
	ResolveMapping(model string) string

	// GetAccountModelMapping 获取特定账号的模型映射
	// 每个账号可以有自己的模型映射规则
	GetAccountModelMapping(accountID string) map[string]string

	// SetAccountModelMapping 设置特定账号的模型映射（管理员操作）
	SetAccountModelMapping(accountID string, mapping map[string]string) error

	// DeleteAccountModelMapping 删除特定账号的模型映射
	DeleteAccountModelMapping(accountID string) error

	// GetGlobalMapping 返回全局模型映射
	GetGlobalMapping() map[string]string

	// SetGlobalMapping 设置全局模型映射（管理员操作）
	SetGlobalMapping(mapping map[string]string) error
}

// modelMappingService 模型映射服务实现
type modelMappingService struct {
	cache         cache.Cache
	globalMapping map[string]string
	mu            sync.RWMutex
}

// 默认全局模型映射
var defaultGlobalMapping = map[string]string{
	// Claude 别名映射（来自 gateway_service.go）
	"claude-3-opus":    "claude-3-opus-20240229",
	"claude-3-sonnet":  "claude-3-sonnet-20240229",
	"claude-3-haiku":   "claude-3-haiku-20240307",
	"claude-3.5-sonnet": "claude-3-5-sonnet-20241022",
	"claude-3.5-haiku":  "claude-3-5-haiku-20241022",
	// GPT -> Claude 映射
	"gpt-4":        "claude-3-5-sonnet-20241022",
	"gpt-4-turbo":  "claude-3-5-sonnet-20241022",
	"gpt-3.5-turbo": "claude-3-5-haiku-20241022",
}

// 缓存键前缀
const (
	globalMappingCacheKey    = "model_mapping:global"
	accountMappingCachePrefix = "model_mapping:account:"
	accountMappingCacheTTL   = 30 * time.Minute
)

// NewModelMappingService 创建模型映射服务
func NewModelMappingService(cache cache.Cache) ModelMappingService {
	svc := &modelMappingService{
		cache:         cache,
		globalMapping: make(map[string]string),
	}

	// 初始化全局映射（深拷贝默认值）
	for k, v := range defaultGlobalMapping {
		svc.globalMapping[k] = v
	}

	// 尝试从缓存加载全局映射
	if cache != nil {
		svc.loadGlobalMappingFromCache(context.Background())
	}

	return svc
}

// ResolveMapping 通过映射链解析模型名称
func (s *modelMappingService) ResolveMapping(model string) string {
	if model == "" {
		return model
	}

	// 最多解析 10 层映射链，防止循环引用
	visited := make(map[string]bool)
	current := model

	for i := 0; i < 10; i++ {
		if visited[current] {
			// 检测到循环引用，停止解析
			break
		}
		visited[current] = true

		// 优先检查全局映射
		s.mu.RLock()
		mapped, ok := s.globalMapping[current]
		s.mu.RUnlock()

		if !ok {
			// 全局映射中没有，尝试从缓存查找
			if s.cache != nil {
				mapped = s.resolveFromAccountMappings(current)
				if mapped == "" || mapped == current {
					break
				}
			} else {
				break
			}
		}

		current = mapped
	}

	return current
}

// resolveFromAccountMappings 从所有账号映射中查找（仅用于全局映射未覆盖的情况）
func (s *modelMappingService) resolveFromAccountMappings(model string) string {
	// 账号级别映射通常需要 accountID 上下文
	// 这里返回空，表示未找到
	return ""
}

// GetAccountModelMapping 获取特定账号的模型映射
func (s *modelMappingService) GetAccountModelMapping(accountID string) map[string]string {
	if s.cache == nil || accountID == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cacheKey := accountMappingCachePrefix + accountID
	var mapping map[string]string
	err := s.cache.GetObject(ctx, cacheKey, &mapping)
	if err != nil {
		return nil
	}

	return mapping
}

// SetAccountModelMapping 设置特定账号的模型映射
func (s *modelMappingService) SetAccountModelMapping(accountID string, mapping map[string]string) error {
	if accountID == "" {
		return fmt.Errorf("accountID 不能为空")
	}

	if s.cache == nil {
		return fmt.Errorf("缓存未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cacheKey := accountMappingCachePrefix + accountID
	err := s.cache.SetObject(ctx, cacheKey, mapping, accountMappingCacheTTL)
	if err != nil {
		return fmt.Errorf("设置账号模型映射缓存失败: %w", err)
	}

	return nil
}

// DeleteAccountModelMapping 删除特定账号的模型映射
func (s *modelMappingService) DeleteAccountModelMapping(accountID string) error {
	if accountID == "" {
		return fmt.Errorf("accountID 不能为空")
	}

	if s.cache == nil {
		return fmt.Errorf("缓存未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cacheKey := accountMappingCachePrefix + accountID
	err := s.cache.Delete(ctx, cacheKey)
	if err != nil {
		return fmt.Errorf("删除账号模型映射缓存失败: %w", err)
	}

	return nil
}

// GetGlobalMapping 返回全局模型映射
func (s *modelMappingService) GetGlobalMapping() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 返回深拷贝
	result := make(map[string]string, len(s.globalMapping))
	for k, v := range s.globalMapping {
		result[k] = v
	}
	return result
}

// SetGlobalMapping 设置全局模型映射
func (s *modelMappingService) SetGlobalMapping(mapping map[string]string) error {
	if mapping == nil {
		return fmt.Errorf("映射不能为 nil")
	}

	s.mu.Lock()
	s.globalMapping = make(map[string]string, len(mapping))
	for k, v := range mapping {
		s.globalMapping[k] = v
	}
	s.mu.Unlock()

	// 持久化到缓存
	if s.cache != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := s.cache.SetObject(ctx, globalMappingCacheKey, mapping, accountMappingCacheTTL)
		if err != nil {
			return fmt.Errorf("持久化全局映射到缓存失败: %w", err)
		}
	}

	return nil
}

// loadGlobalMappingFromCache 从缓存加载全局映射
func (s *modelMappingService) loadGlobalMappingFromCache(ctx context.Context) {
	var mapping map[string]string
	err := s.cache.GetObject(ctx, globalMappingCacheKey, &mapping)
	if err != nil {
		// 缓存中没有，使用默认值
		return
	}

	if len(mapping) > 0 {
		s.mu.Lock()
		s.globalMapping = mapping
		s.mu.Unlock()
	}
}

// ResolveMappingWithAccount 使用账号级别映射解析模型名称
// 账号级别映射优先于全局映射
func ResolveMappingWithAccount(svc ModelMappingService, model string, accountID string) string {
	if model == "" {
		return model
	}

	// 先检查账号级别映射
	if accountID != "" {
		accountMapping := svc.GetAccountModelMapping(accountID)
		if accountMapping != nil {
			if mapped, ok := accountMapping[model]; ok && mapped != "" {
				// 对映射结果继续解析（可能需要多级映射）
				return svc.ResolveMapping(mapped)
			}
		}
	}

	// 回退到全局映射
	return svc.ResolveMapping(model)
}

// SerializeMapping 序列化映射为 JSON 字符串（用于存储）
func SerializeMapping(mapping map[string]string) (string, error) {
	if mapping == nil {
		return "{}", nil
	}
	data, err := json.Marshal(mapping)
	if err != nil {
		return "", fmt.Errorf("序列化映射失败: %w", err)
	}
	return string(data), nil
}

// DeserializeMapping 从 JSON 字符串反序列化映射
func DeserializeMapping(data string) (map[string]string, error) {
	if data == "" {
		return nil, nil
	}
	var mapping map[string]string
	if err := json.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, fmt.Errorf("反序列化映射失败: %w", err)
	}
	return mapping, nil
}
