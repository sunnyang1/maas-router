// Package service 业务服务层
// 余额查询服务 - 查询上游供应商账户余额
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"maas-router/internal/cache"
	"maas-router/internal/repository"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// BalanceInfo 账户余额信息
type BalanceInfo struct {
	AccountID   string    `json:"account_id"`
	AccountName string    `json:"account_name"`
	Platform    string    `json:"platform"`
	Balance     float64   `json:"balance"`
	Currency    string    `json:"currency"` // USD, CNY, etc.
	UsedToday   float64   `json:"used_today"`
	Limit       float64   `json:"limit"`
	UpdatedAt   time.Time `json:"updated_at"`
	Error       string    `json:"error,omitempty"`
}

// BalanceService 余额查询服务接口
type BalanceService interface {
	// QueryBalance 查询指定账户的余额
	QueryBalance(ctx context.Context, accountID string) (*BalanceInfo, error)

	// QueryAllBalances 查询所有活跃账户的余额
	QueryAllBalances(ctx context.Context) ([]*BalanceInfo, error)

	// GetCachedBalance 获取账户的缓存余额
	GetCachedBalance(accountID string) (*BalanceInfo, error)

	// StartPeriodicQuery 启动后台定期查询 goroutine
	StartPeriodicQuery(ctx context.Context, interval time.Duration)
}

type balanceService struct {
	cache       cache.Cache
	accountRepo *repository.AccountRepository
	httpClient  *http.Client
	logger      *zap.Logger

	// 本地缓存 accountID -> *BalanceInfo
	cacheMap sync.Map
}

// NewBalanceService 创建余额查询服务
func NewBalanceService(
	redis *redis.Client,
	accountRepo *repository.AccountRepository,
	logger *zap.Logger,
) BalanceService {
	c := cache.NewCacheFromClient(redis, logger, "maas")
	svc := &balanceService{
		cache:       c,
		accountRepo: accountRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
	return svc
}

// QueryBalance 查询指定账户的余额
func (s *balanceService) QueryBalance(ctx context.Context, accountID string) (*BalanceInfo, error) {
	// 解析 accountID (string -> int64)
	id, err := strconv.ParseInt(accountID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("无效的账户ID: %w", err)
	}

	// 从数据库获取账户信息
	account, err := s.accountRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取账户失败: %w", err)
	}

	// 根据平台查询余额
	info, err := s.queryPlatformBalance(ctx, account)
	if err != nil {
		s.logger.Warn("查询余额失败",
			zap.String("account_id", accountID),
			zap.String("platform", account.Platform),
			zap.Error(err))
		// 返回带错误信息的 BalanceInfo，而不是返回 error
		info = &BalanceInfo{
			AccountID:   accountID,
			AccountName: account.Name,
			Platform:    account.Platform,
			Error:       err.Error(),
			UpdatedAt:   time.Now(),
		}
	}

	// 更新缓存
	s.cacheMap.Store(accountID, info)
	s.updateRedisCache(ctx, accountID, info)

	return info, nil
}

// QueryAllBalances 查询所有活跃账户的余额
func (s *balanceService) QueryAllBalances(ctx context.Context) ([]*BalanceInfo, error) {
	// 获取所有活跃账户
	accounts, err := s.accountRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取活跃账户列表失败: %w", err)
	}

	var results []*BalanceInfo
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 并发查询每个账户的余额
	semaphore := make(chan struct{}, 5) // 限制并发数
	for _, account := range accounts {
		wg.Add(1)
		go func(acc *repository.Account) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			info, err := s.queryPlatformBalance(ctx, acc)
			if err != nil {
				s.logger.Warn("查询余额失败",
					zap.Int64("account_id", acc.ID),
					zap.String("platform", acc.Platform),
					zap.Error(err))
				info = &BalanceInfo{
					AccountID:   strconv.FormatInt(acc.ID, 10),
					AccountName: acc.Name,
					Platform:    acc.Platform,
					Error:       err.Error(),
					UpdatedAt:   time.Now(),
				}
			}

			accountID := strconv.FormatInt(acc.ID, 10)
			s.cacheMap.Store(accountID, info)
			s.updateRedisCache(ctx, accountID, info)

			mu.Lock()
			results = append(results, info)
			mu.Unlock()
		}(account)
	}

	wg.Wait()
	return results, nil
}

// GetCachedBalance 获取账户的缓存余额
func (s *balanceService) GetCachedBalance(accountID string) (*BalanceInfo, error) {
	// 先从本地缓存获取
	if cached, ok := s.cacheMap.Load(accountID); ok {
		return cached.(*BalanceInfo), nil
	}

	// 从 Redis 缓存获取
	cacheKey := fmt.Sprintf("maas:balance:cache:%s", accountID)
	var info BalanceInfo
	if err := s.cache.GetObject(context.Background(), cacheKey, &info); err == nil {
		s.cacheMap.Store(accountID, &info)
		return &info, nil
	}

	return nil, fmt.Errorf("未找到账户 %s 的缓存余额信息", accountID)
}

// StartPeriodicQuery 启动后台定期查询
func (s *balanceService) StartPeriodicQuery(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.logger.Info("余额定期查询已启动", zap.Duration("interval", interval))

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("余额定期查询已停止")
			return
		case <-ticker.C:
			s.logger.Debug("开始定期查询余额...")
			results, err := s.QueryAllBalances(ctx)
			if err != nil {
				s.logger.Error("定期查询余额失败", zap.Error(err))
				continue
			}
			s.logger.Debug("定期查询余额完成",
				zap.Int("account_count", len(results)))
		}
	}
}

// queryPlatformBalance 根据平台查询余额
func (s *balanceService) queryPlatformBalance(ctx context.Context, account *repository.Account) (*BalanceInfo, error) {
	switch account.Platform {
	case "claude", "anthropic":
		return s.queryAnthropicBalance(ctx, account)
	case "openai":
		return s.queryOpenAIBalance(ctx, account)
	case "self_hosted":
		return &BalanceInfo{
			AccountID:   strconv.FormatInt(account.ID, 10),
			AccountName: account.Name,
			Platform:    account.Platform,
			Balance:     -1,
			Currency:    "N/A",
			UpdatedAt:   time.Now(),
		}, nil
	default:
		return &BalanceInfo{
			AccountID:   strconv.FormatInt(account.ID, 10),
			AccountName: account.Name,
			Platform:    account.Platform,
			Balance:     0,
			Currency:    "N/A",
			UpdatedAt:   time.Now(),
			Error:       "不支持的平台: " + account.Platform,
		}, nil
	}
}

// queryAnthropicBalance 查询 Anthropic/Claude 账户余额
func (s *balanceService) queryAnthropicBalance(ctx context.Context, account *repository.Account) (*BalanceInfo, error) {
	apiKey, err := s.getAPIKey(account)
	if err != nil {
		return nil, fmt.Errorf("获取 API Key 失败: %w", err)
	}

	// Anthropic 没有直接的余额查询 API，通过 usage 端点获取用量信息
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.anthropic.com/v1/organizations/current/usage", nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Anthropic API 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic API 返回错误: status=%d, body=%s", resp.StatusCode, string(body))
	}

	// 解析响应
	var usageResp struct {
		Usage []struct {
			InputTokens  int64   `json:"input_tokens"`
			OutputTokens int64   `json:"output_tokens"`
			CostUSD      float64 `json:"cost_usd"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &usageResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 计算今日用量
	var usedToday float64
	for _, u := range usageResp.Usage {
		usedToday += u.CostUSD
	}

	return &BalanceInfo{
		AccountID:   strconv.FormatInt(account.ID, 10),
		AccountName: account.Name,
		Platform:    account.Platform,
		Balance:     0, // Anthropic 不提供余额 API
		Currency:    "USD",
		UsedToday:   usedToday,
		Limit:       0,
		UpdatedAt:   time.Now(),
	}, nil
}

// queryOpenAIBalance 查询 OpenAI 账户余额
func (s *balanceService) queryOpenAIBalance(ctx context.Context, account *repository.Account) (*BalanceInfo, error) {
	apiKey, err := s.getAPIKey(account)
	if err != nil {
		return nil, fmt.Errorf("获取 API Key 失败: %w", err)
	}

	// 查询账单用量
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.openai.com/v1/dashboard/billing/usage?start_date="+
			time.Now().AddDate(0, 0, -1).Format("2006-01-02")+
			"&end_date="+time.Now().Format("2006-01-02"), nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 OpenAI API 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API 返回错误: status=%d, body=%s", resp.StatusCode, string(body))
	}

	// 解析用量响应
	var usageResp struct {
		TotalUsage float64 `json:"total_usage"` // 单位: 美分
	}

	if err := json.Unmarshal(body, &usageResp); err != nil {
		return nil, fmt.Errorf("解析用量响应失败: %w", err)
	}

	// 查询余额/额度
	balanceReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.openai.com/v1/dashboard/billing/subscription", nil)
	if err != nil {
		return nil, fmt.Errorf("创建余额请求失败: %w", err)
	}

	balanceReq.Header.Set("Authorization", "Bearer "+apiKey)

	balanceResp, err := s.httpClient.Do(balanceReq)
	if err != nil {
		return nil, fmt.Errorf("请求 OpenAI 余额 API 失败: %w", err)
	}
	defer balanceResp.Body.Close()

	balanceBody, err := io.ReadAll(balanceResp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取余额响应失败: %w", err)
	}

	var subResp struct {
		HardLimitUSD       float64 `json:"hard_limit_usd"`
		SystemHardLimitUSD float64 `json:"system_hard_limit_usd"`
	}

	if err := json.Unmarshal(balanceBody, &subResp); err != nil {
		// 如果解析失败，使用默认值
		subResp.HardLimitUSD = 0
	}

	limit := subResp.HardLimitUSD
	if limit == 0 {
		limit = subResp.SystemHardLimitUSD
	}

	// total_usage 单位是美分，转换为美元
	usedToday := usageResp.TotalUsage / 100.0
	balance := limit - usedToday

	return &BalanceInfo{
		AccountID:   strconv.FormatInt(account.ID, 10),
		AccountName: account.Name,
		Platform:    account.Platform,
		Balance:     balance,
		Currency:    "USD",
		UsedToday:   usedToday,
		Limit:       limit,
		UpdatedAt:   time.Now(),
	}, nil
}

// getAPIKey 从账户凭证中获取 API Key
func (s *balanceService) getAPIKey(account *repository.Account) (string, error) {
	if account.Credentials == nil {
		return "", fmt.Errorf("账户凭证为空")
	}

	switch account.AccountType {
	case "api_key":
		if apiKey, ok := account.Credentials["api_key"].(string); ok && apiKey != "" {
			return apiKey, nil
		}
	case "oauth":
		if accessToken, ok := account.Credentials["access_token"].(string); ok && accessToken != "" {
			return accessToken, nil
		}
	}

	return "", fmt.Errorf("无法从账户凭证中获取有效的认证信息")
}

// updateRedisCache 更新 Redis 缓存
func (s *balanceService) updateRedisCache(ctx context.Context, accountID string, info *BalanceInfo) {
	cacheKey := fmt.Sprintf("maas:balance:cache:%s", accountID)
	if err := s.cache.SetObject(ctx, cacheKey, info, cache.CommonCacheTTL.Medium); err != nil {
		s.logger.Warn("缓存余额信息失败",
			zap.String("account_id", accountID),
			zap.Error(err))
	}
}
