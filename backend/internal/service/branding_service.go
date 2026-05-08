// Package service 业务服务层
// 品牌设置服务 - 管理站点品牌化配置
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"maas-router/internal/cache"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// BrandingSettings 品牌化设置
type BrandingSettings struct {
	SiteName      string `json:"site_name" yaml:"site_name"`
	LogoURL       string `json:"logo_url" yaml:"logo_url"`
	FaviconURL    string `json:"favicon_url" yaml:"favicon_url"`
	PrimaryColor  string `json:"primary_color" yaml:"primary_color"`
	FooterText    string `json:"footer_text" yaml:"footer_text"`
	CustomCSS     string `json:"custom_css" yaml:"custom_css"`
	AboutPage     string `json:"about_page" yaml:"about_page"`     // HTML 或 Markdown
	Announcement  string `json:"announcement" yaml:"announcement"` // 公告
	ContactEmail  string `json:"contact_email" yaml:"contact_email"`
	Theme         string `json:"theme" yaml:"theme"` // light, dark, system
}

// DefaultBrandingSettings 默认品牌设置
func DefaultBrandingSettings() *BrandingSettings {
	return &BrandingSettings{
		SiteName:     "MaaS-Router",
		LogoURL:      "",
		FaviconURL:   "",
		PrimaryColor: "#1890ff",
		FooterText:   "Powered by MaaS-Router",
		CustomCSS:    "",
		AboutPage:    "",
		Announcement: "",
		ContactEmail: "",
		Theme:        "system",
	}
}

// PublicBrandingSettings 公开可见的品牌设置（隐藏管理配置）
type PublicBrandingSettings struct {
	SiteName     string `json:"site_name"`
	LogoURL      string `json:"logo_url"`
	FaviconURL   string `json:"favicon_url"`
	PrimaryColor string `json:"primary_color"`
	FooterText   string `json:"footer_text"`
	Theme        string `json:"theme"`
	Announcement string `json:"announcement"`
}

// BrandingService 品牌设置服务接口
type BrandingService interface {
	// GetSettings 获取品牌设置
	GetSettings(ctx context.Context) (*BrandingSettings, error)

	// UpdateSettings 更新品牌设置
	UpdateSettings(ctx context.Context, settings *BrandingSettings) error

	// GetPublicSettings 获取公开的品牌设置（用于前端）
	GetPublicSettings(ctx context.Context) (*PublicBrandingSettings, error)
}

type brandingService struct {
	redis  *redis.Client
	cache  cache.Cache
	logger *zap.Logger

	// 本地缓存
	localCache *BrandingSettings
	mu         sync.RWMutex
}

const brandingCacheKey = "maas:branding:settings"

// NewBrandingService 创建品牌设置服务
func NewBrandingService(
	redis *redis.Client,
	logger *zap.Logger,
) BrandingService {
	c := cache.NewCacheFromClient(redis, logger, "maas")
	svc := &brandingService{
		redis:  redis,
		cache:  c,
		logger: logger,
	}

	// 初始化时加载缓存
	svc.loadFromCache(context.Background())

	return svc
}

// GetSettings 获取品牌设置
func (s *brandingService) GetSettings(ctx context.Context) (*BrandingSettings, error) {
	// 先从本地缓存获取
	s.mu.RLock()
	if s.localCache != nil {
		settings := s.localCache
		s.mu.RUnlock()
		return settings, nil
	}
	s.mu.RUnlock()

	// 从 Redis 获取
	settings, err := s.loadFromCache(ctx)
	if err != nil {
		// 如果 Redis 中没有，返回默认设置
		s.logger.Warn("从缓存加载品牌设置失败，使用默认值", zap.Error(err))
		return DefaultBrandingSettings(), nil
	}

	return settings, nil
}

// UpdateSettings 更新品牌设置
func (s *brandingService) UpdateSettings(ctx context.Context, settings *BrandingSettings) error {
	if settings == nil {
		return fmt.Errorf("设置不能为空")
	}

	// 验证主题值
	validThemes := map[string]bool{
		"light":  true,
		"dark":   true,
		"system": true,
	}
	if settings.Theme != "" && !validThemes[settings.Theme] {
		return fmt.Errorf("无效的主题值: %s，可选值: light, dark, system", settings.Theme)
	}

	// 序列化设置
	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("序列化设置失败: %w", err)
	}

	// 保存到 Redis
	if err := s.cache.Set(ctx, brandingCacheKey, string(data), cache.CommonCacheTTL.VeryLong); err != nil {
		return fmt.Errorf("保存设置到缓存失败: %w", err)
	}

	// 更新本地缓存
	s.mu.Lock()
	s.localCache = settings
	s.mu.Unlock()

	s.logger.Info("品牌设置已更新",
		zap.String("site_name", settings.SiteName),
		zap.String("theme", settings.Theme))

	return nil
}

// GetPublicSettings 获取公开的品牌设置
func (s *brandingService) GetPublicSettings(ctx context.Context) (*PublicBrandingSettings, error) {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	return &PublicBrandingSettings{
		SiteName:     settings.SiteName,
		LogoURL:      settings.LogoURL,
		FaviconURL:   settings.FaviconURL,
		PrimaryColor: settings.PrimaryColor,
		FooterText:   settings.FooterText,
		Theme:        settings.Theme,
		Announcement: settings.Announcement,
	}, nil
}

// loadFromCache 从 Redis 缓存加载设置
func (s *brandingService) loadFromCache(ctx context.Context) (*BrandingSettings, error) {
	data, err := s.cache.Get(ctx, brandingCacheKey)
	if err != nil {
		return nil, err
	}

	var settings BrandingSettings
	if err := json.Unmarshal([]byte(data), &settings); err != nil {
		return nil, fmt.Errorf("反序列化设置失败: %w", err)
	}

	// 更新本地缓存
	s.mu.Lock()
	s.localCache = &settings
	s.mu.Unlock()

	return &settings, nil
}
