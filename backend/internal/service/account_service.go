// Package service 业务服务层
// 提供账号调度、请求转发等核心业务逻辑
package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"maas-router/ent"
	"maas-router/internal/cache"
	"maas-router/internal/config"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

// AccountService 账号调度服务接口
// 这是整个项目最核心的服务，负责账号选择和请求转发
type AccountService interface {
	// SelectAccount 智能账号选择
	// 基于权重的负载均衡、负载因子计算、Sticky Session 支持
	SelectAccount(ctx context.Context, platform, model, sessionID string) (*ent.Account, error)

	// SelectAccountWithTier 基于复杂度分层的智能账号选择
	// 如果指定了 routingTier，优先选择对应分组的账号
	SelectAccountWithTier(ctx context.Context, platform, model, sessionID, routingTier string) (*ent.Account, error)

	// ForwardRequest 请求转发
	// 支持 HTTP/2、TLS 指纹、代理、流式响应
	ForwardRequest(ctx context.Context, account *ent.Account, req *ForwardRequest) (*ForwardResponse, error)

	// UpdateAccountStatus 更新账号状态
	UpdateAccountStatus(ctx context.Context, accountID int64, status string) error

	// RecordError 记录错误
	RecordError(ctx context.Context, accountID int64, err error) error

	// GetAccountLoad 获取账号负载
	GetAccountLoad(ctx context.Context, accountID int64) (*AccountLoad, error)

	// IncrementConcurrency 增加并发数
	IncrementConcurrency(ctx context.Context, accountID int64) error

	// DecrementConcurrency 减少并发数
	DecrementConcurrency(ctx context.Context, accountID int64) error
}

// ForwardRequest 转发请求参数
type ForwardRequest struct {
	Method      string
	Path        string
	Headers     map[string]string
	Body        io.Reader
	Stream      bool // 是否流式请求
	Timeout     time.Duration
	Platform    string
	Model       string
	SessionID   string
	RequestID   string
}

// ForwardResponse 转发响应
type ForwardResponse struct {
	StatusCode int
	Headers    http.Header
	Body       io.ReadCloser
	IsStream   bool
}

// AccountLoad 账号负载信息
type AccountLoad struct {
	AccountID         int64
	CurrentConcurrency int
	MaxConcurrency     int
	CurrentRPM         int
	RPMLimit           int
	ErrorRate          float64
	LoadFactor         float64 // 综合负载因子 0-1
	LastUsedAt         time.Time
	LastErrorAt       *time.Time
}

// accountService 账号调度服务实现
type accountService struct {
	db       *ent.Client
	redis    *redis.Client
	cache    cache.Cache
	cacheKey *cache.CacheKey
	cfg      *config.Config
	logger   *zap.Logger

	// HTTP 客户端池，按平台分类
	httpClients sync.Map // map[string]*http.Client

	// 账号缓存
	accountCache sync.Map // map[int64]*ent.Account

	// 临时不可调度标记
	unschedulableAccounts sync.Map // map[int64]time.Time (过期时间)
}

// NewAccountService 创建账号调度服务实例
func NewAccountService(db *ent.Client, redis *redis.Client, cfg *config.Config, logger *zap.Logger) AccountService {
	// 创建统一缓存实例
	c := cache.NewCacheFromClient(redis, logger, "maas")
	svc := &accountService{
		db:       db,
		redis:    redis,
		cache:    c,
		cacheKey: cache.NewCacheKey("maas"),
		cfg:      cfg,
		logger:   logger,
	}

	// 初始化 HTTP 客户端
	svc.initHTTPClients()

	// 启动后台任务
	go svc.startCleanupTask()
	go svc.startMetricsSyncTask()

	return svc
}

// initHTTPClients 初始化各平台的 HTTP 客户端
func (s *accountService) initHTTPClients() {
	platforms := []string{"claude", "openai", "gemini", "self_hosted"}

	for _, platform := range platforms {
		client := s.createHTTPClient(platform)
		s.httpClients.Store(platform, client)
		s.logger.Info("HTTP 客户端已初始化",
			zap.String("platform", platform),
			zap.Int("max_idle_conns", s.cfg.Gateway.HTTPPool.MaxIdleConns),
			zap.Int("idle_timeout", s.cfg.Gateway.HTTPPool.IdleConnTimeout))
	}
}

// createHTTPClient 创建优化的 HTTP 客户端
func (s *accountService) createHTTPClient(platform string) *http.Client {
	poolCfg := s.cfg.Gateway.HTTPPool

	// 创建自定义 Transport，使用配置连接池参数
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// 连接池配置
		MaxIdleConns:        poolCfg.MaxIdleConns,
		MaxIdleConnsPerHost: poolCfg.MaxIdleConnsPerHost,
		IdleConnTimeout:     time.Duration(poolCfg.IdleConnTimeout) * time.Second,
		// TLS 配置
		TLSHandshakeTimeout:   time.Duration(poolCfg.TLSHandshakeTimeout) * time.Second,
		ExpectContinueTimeout: time.Duration(poolCfg.ExpectContinueTimeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(poolCfg.ResponseHeaderTimeout) * time.Second,
		// 自定义 TLS 配置，支持 TLS 指纹
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		},
		// 启用 HTTP/2
		ForceAttemptHTTP2: true,
		// Keep-Alive 配置
		DisableKeepAlives: poolCfg.DisableKeepAlives,
	}

	// 配置 HTTP/2 支持
	if err := http2.ConfigureTransport(transport); err != nil {
		s.logger.Warn("配置 HTTP/2 失败", zap.String("platform", platform), zap.Error(err))
	}

	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(s.cfg.Gateway.UpstreamTimeout) * time.Second,
	}
}

// getHTTPClient 获取指定平台的 HTTP 客户端（支持连接复用）
func (s *accountService) getHTTPClient(platform string) *http.Client {
	// 尝试获取已存在的客户端
	if client, ok := s.httpClients.Load(platform); ok {
		return client.(*http.Client)
	}

	// 如果不存在，创建新的客户端
	s.logger.Info("创建新的 HTTP 客户端", zap.String("platform", platform))
	client := s.createHTTPClient(platform)
	s.httpClients.Store(platform, client)
	return client
}

// closeIdleConnections 关闭所有空闲连接（用于优雅关闭或健康检查）
func (s *accountService) closeIdleConnections() {
	s.httpClients.Range(func(key, value interface{}) bool {
		if client, ok := value.(*http.Client); ok {
			if transport, ok := client.Transport.(*http.Transport); ok {
				transport.CloseIdleConnections()
				s.logger.Debug("关闭空闲连接", zap.String("platform", key.(string)))
			}
		}
		return true
	})
}

// checkHTTPClientHealth 检查 HTTP 客户端健康状态
func (s *accountService) checkHTTPClientHealth(ctx context.Context, platform string) error {
	client := s.getHTTPClient(platform)

	// 发送健康检查请求（HEAD 请求）
	var healthURL string
	switch platform {
	case "claude":
		healthURL = "https://api.anthropic.com/v1/health"
	case "openai":
		healthURL = "https://api.openai.com/v1/models"
	case "gemini":
		healthURL = "https://generativelanguage.googleapis.com/v1beta/models"
	default:
		return nil // 其他平台跳过健康检查
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, healthURL, nil)
	if err != nil {
		return fmt.Errorf("创建健康检查请求失败: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("健康检查请求失败: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// SelectAccount 智能账号选择
// 实现基于权重的负载均衡，考虑并发数、RPM 限制、错误率等因素
func (s *accountService) SelectAccount(ctx context.Context, platform, model, sessionID string) (*ent.Account, error) {
	// 1. 如果有 Sticky Session，尝试复用之前的账号
	if sessionID != "" {
		if account, err := s.getStickyAccount(ctx, sessionID); err == nil && account != nil {
			s.logger.Debug("使用 Sticky Session 账号",
				zap.String("session_id", sessionID),
				zap.Int64("account_id", account.ID))
			return account, nil
		}
	}

	// 2. 获取该平台下所有可用账号
	accounts, err := s.getAvailableAccounts(ctx, platform)
	if err != nil {
		return nil, fmt.Errorf("获取可用账号失败: %w", err)
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("平台 %s 没有可用账号", platform)
	}

	// 3. 计算每个账号的负载因子
	type accountScore struct {
		account   *ent.Account
		loadScore float64
	}
	scores := make([]accountScore, 0, len(accounts))

	for _, acc := range accounts {
		// 检查是否在临时不可调度列表中
		if s.isUnschedulable(acc.ID) {
			continue
		}

		// 获取账号负载
		load, err := s.GetAccountLoad(ctx, acc.ID)
		if err != nil {
			s.logger.Warn("获取账号负载失败",
				zap.Int64("account_id", acc.ID),
				zap.Error(err))
			continue
		}

		// 检查是否超过限制
		if load.CurrentConcurrency >= load.MaxConcurrency {
			continue
		}
		if load.CurrentRPM >= load.RPMLimit {
			continue
		}

		// 计算综合评分（负载因子越低，评分越高）
		// 评分 = (1 - 负载因子) * 权重
		weight := 100.0 // 默认权重，可从分组获取
		score := (1 - load.LoadFactor) * weight

		scores = append(scores, accountScore{
			account:   acc,
			loadScore: score,
		})
	}

	if len(scores) == 0 {
		return nil, fmt.Errorf("平台 %s 所有账号均已达到负载上限", platform)
	}

	// 4. 基于评分进行加权随机选择
	selected := s.weightedRandomSelect(scores)

	// 5. 更新 Sticky Session
	if sessionID != "" {
		if err := s.setStickyAccount(ctx, sessionID, selected.ID); err != nil {
			s.logger.Warn("设置 Sticky Session 失败",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}

	s.logger.Info("选择账号成功",
		zap.Int64("account_id", selected.ID),
		zap.String("platform", platform),
		zap.String("model", model))

	return selected, nil
}

// SelectAccountWithTier 基于复杂度分层的智能账号选择
// 如果指定了 routingTier，优先选择对应分组的账号
func (s *accountService) SelectAccountWithTier(ctx context.Context, platform, model, sessionID, routingTier string) (*ent.Account, error) {
	// 如果没有指定路由层级，回退到标准选择逻辑
	if routingTier == "" {
		return s.SelectAccount(ctx, platform, model, sessionID)
	}

	// 1. 如果有 Sticky Session，尝试复用之前的账号
	if sessionID != "" {
		if account, err := s.getStickyAccount(ctx, sessionID); err == nil && account != nil {
			s.logger.Debug("使用 Sticky Session 账号",
				zap.String("session_id", sessionID),
				zap.Int64("account_id", account.ID))
			return account, nil
		}
	}

	// 2. 获取该平台下所有可用账号
	accounts, err := s.getAvailableAccounts(ctx, platform)
	if err != nil {
		return nil, fmt.Errorf("获取可用账号失败: %w", err)
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("平台 %s 没有可用账号", platform)
	}

	// 3. 计算每个账号的负载因子，并优先选择匹配分组的账号
	type accountScore struct {
		account    *ent.Account
		loadScore  float64
		tierMatch  bool // 是否匹配复杂度分层
	}
	scores := make([]accountScore, 0, len(accounts))

	for _, acc := range accounts {
		// 检查是否在临时不可调度列表中
		if s.isUnschedulable(acc.ID) {
			continue
		}

		// 获取账号负载
		load, err := s.GetAccountLoad(ctx, acc.ID)
		if err != nil {
			s.logger.Warn("获取账号负载失败",
				zap.Int64("account_id", acc.ID),
				zap.Error(err))
			continue
		}

		// 检查是否超过限制
		if load.CurrentConcurrency >= load.MaxConcurrency {
			continue
		}
		if load.CurrentRPM >= load.RPMLimit {
			continue
		}

		// 计算综合评分（负载因子越低，评分越高）
		weight := 100.0
		score := (1 - load.LoadFactor) * weight

		// 检查账号是否匹配复杂度分层
		// 通过账号的 Group 或 Tags 字段判断是否属于对应层级
		tierMatch := s.checkAccountTierMatch(acc, routingTier)
		if tierMatch {
			// 匹配分组的账号获得额外权重加成
			score *= 1.5
		}

		scores = append(scores, accountScore{
			account:   acc,
			loadScore: score,
			tierMatch: tierMatch,
		})
	}

	if len(scores) == 0 {
		return nil, fmt.Errorf("平台 %s 所有账号均已达到负载上限", platform)
	}

	// 4. 优先从匹配分组的账号中选择
	var tierScores []struct {
		account   *ent.Account
		loadScore float64
	}
	for _, s := range scores {
		if s.tierMatch {
			tierScores = append(tierScores, struct {
				account   *ent.Account
				loadScore float64
			}{s.account, s.loadScore})
		}
	}

	var selected *ent.Account
	if len(tierScores) > 0 {
		// 从匹配分组的账号中选择
		selected = s.weightedRandomSelect(tierScores)
		s.logger.Debug("选择匹配复杂度分组的账号",
			zap.String("routing_tier", routingTier),
			zap.Int64("account_id", selected.ID))
	} else {
		// 没有匹配的分组账号，从所有可用账号中选择
		var allScores []struct {
			account   *ent.Account
			loadScore float64
		}
		for _, s := range scores {
			allScores = append(allScores, struct {
				account   *ent.Account
				loadScore float64
			}{s.account, s.loadScore})
		}
		selected = s.weightedRandomSelect(allScores)
		s.logger.Debug("未找到匹配分组的账号，从全部可用账号中选择",
			zap.String("routing_tier", routingTier),
			zap.Int64("account_id", selected.ID))
	}

	// 5. 更新 Sticky Session
	if sessionID != "" {
		if err := s.setStickyAccount(ctx, sessionID, selected.ID); err != nil {
			s.logger.Warn("设置 Sticky Session 失败",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}

	s.logger.Info("选择账号成功（带复杂度分层）",
		zap.Int64("account_id", selected.ID),
		zap.String("platform", platform),
		zap.String("model", model),
		zap.String("routing_tier", routingTier))

	return selected, nil
}

// checkAccountTierMatch 检查账号是否匹配指定的复杂度分层
func (s *accountService) checkAccountTierMatch(account *ent.Account, routingTier string) bool {
	// 通过账号的 Group 字段判断是否匹配
	// 如果账号有 Group 标签且包含对应的层级名称，则认为匹配
	if account.Group != "" && strings.Contains(strings.ToLower(account.Group), strings.ToLower(routingTier)) {
		return true
	}

	// 通过账号名称中的关键词匹配
	accountName := strings.ToLower(account.Name)
	tierKeywords := map[string][]string{
		"economy":  {"economy", "flash", "mini", "lite", "haiku"},
		"standard": {"standard", "pro", "sonnet", "normal"},
		"premium":  {"premium", "opus", "advanced", "pro-max"},
	}

	if keywords, ok := tierKeywords[strings.ToLower(routingTier)]; ok {
		for _, keyword := range keywords {
			if strings.Contains(accountName, keyword) {
				return true
			}
		}
	}

	return false
}

// ForwardRequest 请求转发
// 支持流式响应、代理、TLS 指纹等
func (s *accountService) ForwardRequest(ctx context.Context, account *ent.Account, req *ForwardRequest) (*ForwardResponse, error) {
	// 1. 构建 HTTP 请求
	var body io.Reader = req.Body
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.Path, body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 2. 设置请求头
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// 3. 获取账号凭证并设置认证
	if err := s.setAuthentication(httpReq, account); err != nil {
		return nil, fmt.Errorf("设置认证失败: %w", err)
	}

	// 4. 获取 HTTP 客户端（使用连接池复用）
	httpClient := s.getHTTPClient(account.Platform)

	// 5. 如果配置了代理，设置代理
	if account.ProxyURL != nil && *account.ProxyURL != "" {
		if err := s.setProxy(httpClient, *account.ProxyURL); err != nil {
			s.logger.Warn("设置代理失败",
				zap.Int64("account_id", account.ID),
				zap.String("proxy_url", *account.ProxyURL),
				zap.Error(err))
		}
	}

	// 6. 发送请求
	startTime := time.Now()
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		// 记录错误
		s.RecordError(ctx, account.ID, err)
		return nil, fmt.Errorf("请求转发失败: %w", err)
	}

	// 7. 更新账号最后使用时间
	s.updateLastUsedAt(ctx, account.ID)

	s.logger.Info("请求转发成功",
		zap.Int64("account_id", account.ID),
		zap.String("request_id", req.RequestID),
		zap.Int("status_code", resp.StatusCode),
		zap.Duration("latency", time.Since(startTime)))

	// 8. 构建响应
	return &ForwardResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       resp.Body,
		IsStream:   req.Stream,
	}, nil
}

// UpdateAccountStatus 更新账号状态
func (s *accountService) UpdateAccountStatus(ctx context.Context, accountID int64, status string) error {
	// 验证状态值
	validStatuses := map[string]bool{
		"active":       true,
		"disabled":     true,
		"unschedulable": true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("无效的账号状态: %s", status)
	}

	// 更新数据库
	_, err := s.db.Account.UpdateOneID(accountID).
		SetStatus(ent.AccountStatus(status)).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("更新账号状态失败: %w", err)
	}

	// 更新缓存
	s.accountCache.Delete(accountID)

	// 如果设置为不可调度，添加到临时不可调度列表
	if status == "unschedulable" {
		s.markUnschedulable(accountID, 5*time.Minute)
	}

	s.logger.Info("账号状态已更新",
		zap.Int64("account_id", accountID),
		zap.String("status", status))

	return nil
}

// RecordError 记录错误
func (s *accountService) RecordError(ctx context.Context, accountID int64, err error) error {
	now := time.Now()

	// 更新数据库中的错误计数
	_, dbErr := s.db.Account.UpdateOneID(accountID).
		SetLastErrorAt(now).
		AddErrorCount(1).
		Save(ctx)
	if dbErr != nil {
		s.logger.Error("更新错误计数失败",
			zap.Int64("account_id", accountID),
			zap.Error(dbErr))
	}

	// 更新 Redis 中的错误计数（用于计算错误率）
	key := fmt.Sprintf("account:errors:%d:%s", accountID, now.Format("2006-01-02"))
	if _, redisErr := s.redis.Incr(ctx, key).Result(); redisErr != nil {
		s.logger.Error("Redis 记录错误失败", zap.Error(redisErr))
	}
	// 设置过期时间
	s.redis.Expire(ctx, key, 24*time.Hour)

	// 检查是否需要临时标记为不可调度
	errorCount, _ := s.redis.Get(ctx, key).Int()
	if errorCount >= 5 {
		s.markUnschedulable(accountID, 5*time.Minute)
		s.logger.Warn("账号错误次数过多，临时标记为不可调度",
			zap.Int64("account_id", accountID),
			zap.Int("error_count", errorCount))
	}

	s.logger.Warn("账号请求错误",
		zap.Int64("account_id", accountID),
		zap.Error(err))

	return nil
}

// GetAccountLoad 获取账号负载（带缓存）
func (s *accountService) GetAccountLoad(ctx context.Context, accountID int64) (*AccountLoad, error) {
	// 尝试从缓存获取
	cacheKey := s.cacheKey.AccountLoad(accountID)
	var cachedLoad AccountLoad
	if err := s.cache.GetObject(ctx, cacheKey, &cachedLoad); err == nil {
		s.logger.Debug("从缓存获取账号负载", zap.Int64("account_id", accountID))
		return &cachedLoad, nil
	}

	// 从数据库获取账号信息
	account, err := s.db.Account.Get(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("获取账号失败: %w", err)
	}

	// 从 Redis 获取实时数据
	concurrencyKey := fmt.Sprintf("account:concurrency:%d", accountID)
	rpmKey := fmt.Sprintf("account:rpm:%d:%s", accountID, time.Now().Format("2006-01-02-15:04"))

	currentConcurrency, _ := s.redis.Get(ctx, concurrencyKey).Int()
	currentRPM, _ := s.redis.Get(ctx, rpmKey).Int()

	// 计算错误率
	errorKey := fmt.Sprintf("account:errors:%d:%s", accountID, time.Now().Format("2006-01-02"))
	errorCount, _ := s.redis.Get(ctx, errorKey).Int()
	totalKey := fmt.Sprintf("account:total:%d:%s", accountID, time.Now().Format("2006-01-02"))
	totalCount, _ := s.redis.Get(ctx, totalKey).Int()

	errorRate := 0.0
	if totalCount > 0 {
		errorRate = float64(errorCount) / float64(totalCount)
	}

	// 计算综合负载因子
	// 负载因子 = 并发占比 * 0.4 + RPM占比 * 0.3 + 错误率 * 0.3
	concurrencyRatio := float64(currentConcurrency) / float64(account.MaxConcurrency)
	rpmRatio := float64(currentRPM) / float64(account.RpmLimit)
	loadFactor := concurrencyRatio*0.4 + rpmRatio*0.3 + errorRate*0.3

	// 确保负载因子在 0-1 范围内
	loadFactor = math.Max(0, math.Min(1, loadFactor))

	load := &AccountLoad{
		AccountID:          accountID,
		CurrentConcurrency: currentConcurrency,
		MaxConcurrency:     account.MaxConcurrency,
		CurrentRPM:         currentRPM,
		RPMLimit:           account.RpmLimit,
		ErrorRate:          errorRate,
		LoadFactor:         loadFactor,
		LastUsedAt:         time.Now(),
		LastErrorAt:        account.LastErrorAt,
	}

	// 缓存负载信息（短缓存，因为负载变化频繁）
	if err := s.cache.SetObject(ctx, cacheKey, load, cache.CommonCacheTTL.Short); err != nil {
		s.logger.Warn("缓存账号负载失败", zap.Error(err))
	}

	return load, nil
}

// IncrementConcurrency 增加并发数
func (s *accountService) IncrementConcurrency(ctx context.Context, accountID int64) error {
	key := fmt.Sprintf("account:concurrency:%d", accountID)
	if _, err := s.redis.Incr(ctx, key).Result(); err != nil {
		return fmt.Errorf("增加并发数失败: %w", err)
	}

	// 同时更新 RPM 计数
	rpmKey := fmt.Sprintf("account:rpm:%d:%s", accountID, time.Now().Format("2006-01-02-15:04"))
	s.redis.Incr(ctx, rpmKey)
	s.redis.Expire(ctx, rpmKey, 2*time.Minute)

	// 更新总请求数
	totalKey := fmt.Sprintf("account:total:%d:%s", accountID, time.Now().Format("2006-01-02"))
	s.redis.Incr(ctx, totalKey)
	s.redis.Expire(ctx, totalKey, 24*time.Hour)

	return nil
}

// DecrementConcurrency 减少并发数
func (s *accountService) DecrementConcurrency(ctx context.Context, accountID int64) error {
	key := fmt.Sprintf("account:concurrency:%d", accountID)
	if _, err := s.redis.Decr(ctx, key).Result(); err != nil {
		return fmt.Errorf("减少并发数失败: %w", err)
	}
	return nil
}

// getAvailableAccounts 获取平台下所有可用账号
func (s *accountService) getAvailableAccounts(ctx context.Context, platform string) ([]*ent.Account, error) {
	// 从数据库查询
	accounts, err := s.db.Account.Query().
		Where(
			ent.AccountPlatformEQ(ent.AccountPlatform(platform)),
			ent.AccountStatusEQ(ent.AccountStatusActive),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

// getStickyAccount 获取 Sticky Session 绑定的账号
func (s *accountService) getStickyAccount(ctx context.Context, sessionID string) (*ent.Account, error) {
	key := fmt.Sprintf("sticky:session:%s", sessionID)
	accountID, err := s.redis.Get(ctx, key).Int64()
	if err != nil {
		return nil, err
	}

	// 从缓存或数据库获取账号
	account, err := s.db.Account.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// 检查账号是否仍然可用
	if account.Status != ent.AccountStatusActive {
		return nil, fmt.Errorf("账号不可用")
	}

	return account, nil
}

// setStickyAccount 设置 Sticky Session
func (s *accountService) setStickyAccount(ctx context.Context, sessionID string, accountID int64) error {
	key := fmt.Sprintf("sticky:session:%s", sessionID)
	return s.redis.Set(ctx, key, accountID, 30*time.Minute).Err()
}

// isUnschedulable 检查账号是否在临时不可调度列表中
func (s *accountService) isUnschedulable(accountID int64) bool {
	if expireAt, ok := s.unschedulableAccounts.Load(accountID); ok {
		if time.Now().Before(expireAt.(time.Time)) {
			return true
		}
		// 已过期，删除标记
		s.unschedulableAccounts.Delete(accountID)
	}
	return false
}

// markUnschedulable 标记账号临时不可调度
func (s *accountService) markUnschedulable(accountID int64, duration time.Duration) {
	s.unschedulableAccounts.Store(accountID, time.Now().Add(duration))
}

// weightedRandomSelect 基于评分的加权随机选择
func (s *accountService) weightedRandomSelect(scores []struct {
	account   *ent.Account
	loadScore float64
}) *ent.Account {
	// 计算总权重
	totalScore := 0.0
	for _, s := range scores {
		totalScore += s.loadScore
	}

	// 随机选择
	r := rand.Float64() * totalScore
	cumulative := 0.0
	for _, s := range scores {
		cumulative += s.loadScore
		if r <= cumulative {
			return s.account
		}
	}

	// 默认返回第一个
	return scores[0].account
}

// setAuthentication 设置请求认证
func (s *accountService) setAuthentication(req *http.Request, account *ent.Account) error {
	credentials := account.Credentials
	if credentials == nil {
		return fmt.Errorf("账号凭证为空")
	}

	switch account.AccountType {
	case ent.AccountAccountTypeApiKey:
		// API Key 认证
		if apiKey, ok := credentials["api_key"].(string); ok {
			switch account.Platform {
			case ent.AccountPlatformClaude:
				req.Header.Set("x-api-key", apiKey)
			case ent.AccountPlatformOpenai:
				req.Header.Set("Authorization", "Bearer "+apiKey)
			case ent.AccountPlatformGemini:
				req.Header.Set("Authorization", "Bearer "+apiKey)
			}
		}
	case ent.AccountAccountTypeOauth:
		// OAuth 认证
		if accessToken, ok := credentials["access_token"].(string); ok {
			req.Header.Set("Authorization", "Bearer "+accessToken)
		}
	case ent.AccountAccountTypeCookie:
		// Cookie 认证
		if cookie, ok := credentials["cookie"].(string); ok {
			req.Header.Set("Cookie", cookie)
		}
	}

	return nil
}

// setProxy 设置代理
func (s *accountService) setProxy(client *http.Client, proxyURL string) error {
	// 这里需要重新创建 Transport 来设置代理
	// 实际实现中应该考虑连接池复用
	return nil
}

// updateLastUsedAt 更新账号最后使用时间
func (s *accountService) updateLastUsedAt(ctx context.Context, accountID int64) {
	now := time.Now()
	_, err := s.db.Account.UpdateOneID(accountID).
		SetLastUsedAt(now).
		Save(ctx)
	if err != nil {
		s.logger.Warn("更新最后使用时间失败",
			zap.Int64("account_id", accountID),
			zap.Error(err))
	}
}

// startCleanupTask 启动清理任务
func (s *accountService) startCleanupTask() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// 清理过期的不可调度标记
		now := time.Now()
		s.unschedulableAccounts.Range(func(key, value interface{}) bool {
			if value.(time.Time).Before(now) {
				s.unschedulableAccounts.Delete(key)
			}
			return true
		})
	}
}

// startMetricsSyncTask 启动指标同步任务
func (s *accountService) startMetricsSyncTask() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// 同步 Redis 中的指标到数据库
		// 这里可以实现定期将 Redis 中的统计数据持久化到数据库
	}
}

// GetAccountByID 根据 ID 获取账号（带统一缓存）
func (s *accountService) GetAccountByID(ctx context.Context, accountID int64) (*ent.Account, error) {
	// 先从本地缓存获取
	if cached, ok := s.accountCache.Load(accountID); ok {
		return cached.(*ent.Account), nil
	}

	// 尝试从 Redis 缓存获取
	cacheKey := s.cacheKey.Account(accountID)
	var cachedAccount ent.Account
	if err := s.cache.GetObject(ctx, cacheKey, &cachedAccount); err == nil {
		// 更新本地缓存
		s.accountCache.Store(accountID, &cachedAccount)
		return &cachedAccount, nil
	}

	// 从数据库获取
	account, err := s.db.Account.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// 更新本地缓存
	s.accountCache.Store(accountID, account)

	// 更新 Redis 缓存
	if err := s.cache.SetObject(ctx, cacheKey, account, cache.CommonCacheTTL.Medium); err != nil {
		s.logger.Warn("缓存账号信息失败", zap.Error(err))
	}

	return account, nil
}

// GetAccountCredentials 获取账号凭证（解密后）
func (s *accountService) GetAccountCredentials(account *ent.Account) (map[string]interface{}, error) {
	// 实际实现中应该解密凭证
	// 这里返回原始凭证
	return account.Credentials, nil
}

// ParseUpstreamURL 解析上游 URL
func (s *accountService) ParseUpstreamURL(platform string, path string) string {
	var baseURL string
	switch platform {
	case "claude":
		baseURL = "https://api.anthropic.com"
	case "openai":
		baseURL = "https://api.openai.com"
	case "gemini":
		baseURL = "https://generativelanguage.googleapis.com"
	default:
		baseURL = "https://api.anthropic.com"
	}

	// 确保 path 以 / 开头
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return baseURL + path
}

// StreamResponse 流式响应处理
func (s *accountService) StreamResponse(ctx context.Context, resp *http.Response, callback func(data []byte) error) error {
	reader := resp.Body
	defer reader.Close()

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := reader.Read(buf)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			if n > 0 {
				if err := callback(buf[:n]); err != nil {
					return err
				}
			}
		}
	}
}

// AccountStats 账号统计信息
type AccountStats struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	AvgLatency      time.Duration
	TotalTokens     int64
	TotalCost       float64
}

// GetAccountStats 获取账号统计信息
func (s *accountService) GetAccountStats(ctx context.Context, accountID int64, period string) (*AccountStats, error) {
	// 从 Redis 或数据库获取统计数据
	// 这里返回模拟数据
	return &AccountStats{
		TotalRequests:   0,
		SuccessRequests: 0,
		FailedRequests:  0,
		AvgLatency:      0,
		TotalTokens:     0,
		TotalCost:       0,
	}, nil
}

// ToJSON 将账号信息转换为 JSON（隐藏敏感信息）
func (s *accountService) ToJSON(account *ent.Account) map[string]interface{} {
	return map[string]interface{}{
		"id":                account.ID,
		"name":              account.Name,
		"platform":          account.Platform,
		"account_type":      account.AccountType,
		"status":            account.Status,
		"max_concurrency":   account.MaxConcurrency,
		"current_concurrency": account.CurrentConcurrency,
		"rpm_limit":         account.RpmLimit,
		"total_requests":    account.TotalRequests,
		"error_count":       account.ErrorCount,
		"last_used_at":      account.LastUsedAt,
		"created_at":        account.CreatedAt,
		"updated_at":        account.UpdatedAt,
	}
}

// MarshalJSON 实现 JSON 序列化
func (a *AccountLoad) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"account_id":           a.AccountID,
		"current_concurrency":  a.CurrentConcurrency,
		"max_concurrency":      a.MaxConcurrency,
		"current_rpm":          a.CurrentRPM,
		"rpm_limit":            a.RPMLimit,
		"error_rate":           a.ErrorRate,
		"load_factor":          a.LoadFactor,
		"last_used_at":         a.LastUsedAt,
		"last_error_at":        a.LastErrorAt,
	})
}
