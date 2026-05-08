// Package service 业务服务层
// 渠道测试服务 - 测试上游渠道可用性
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"maas-router/internal/repository"

	"go.uber.org/zap"
)

// ChannelTestResult 渠道测试结果
type ChannelTestResult struct {
	AccountID    string    `json:"account_id"`
	AccountName  string    `json:"account_name"`
	Platform     string    `json:"platform"`
	IsHealthy    bool      `json:"is_healthy"`
	LatencyMs    int64     `json:"latency_ms"`
	ErrorMessage string    `json:"error_message,omitempty"`
	TestedAt     time.Time `json:"tested_at"`
	Model        string    `json:"model"` // 测试使用的模型
}

// ChannelTestService 渠道测试服务接口
type ChannelTestService interface {
	// TestAccount 测试单个账户的可用性
	TestAccount(ctx context.Context, accountID string) (*ChannelTestResult, error)

	// TestAllAccounts 测试所有活跃账户
	TestAllAccounts(ctx context.Context) ([]*ChannelTestResult, error)

	// GetLatestResults 获取最新的测试结果
	GetLatestResults() []*ChannelTestResult

	// StartPeriodicTest 启动后台定期测试
	StartPeriodicTest(ctx context.Context, interval time.Duration)
}

type channelTestService struct {
	accountRepo *repository.AccountRepository
	httpClient  *http.Client
	logger      *zap.Logger

	// 最新测试结果缓存
	latestResults sync.Map // accountID -> *ChannelTestResult
	mu            sync.RWMutex
	allResults    []*ChannelTestResult
}

// NewChannelTestService 创建渠道测试服务
func NewChannelTestService(
	accountRepo *repository.AccountRepository,
	logger *zap.Logger,
) ChannelTestService {
	return &channelTestService{
		accountRepo: accountRepo,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}
}

// TestAccount 测试单个账户的可用性
func (s *channelTestService) TestAccount(ctx context.Context, accountID string) (*ChannelTestResult, error) {
	id, err := strconv.ParseInt(accountID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("无效的账户ID: %w", err)
	}

	account, err := s.accountRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取账户失败: %w", err)
	}

	result := s.testPlatformAccount(ctx, account)

	// 更新缓存
	s.latestResults.Store(accountID, result)
	s.mu.Lock()
	s.updateAllResults(result)
	s.mu.Unlock()

	return result, nil
}

// TestAllAccounts 测试所有活跃账户
func (s *channelTestService) TestAllAccounts(ctx context.Context) ([]*ChannelTestResult, error) {
	accounts, err := s.accountRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取活跃账户列表失败: %w", err)
	}

	var results []*ChannelTestResult
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 并发测试，限制并发数
	semaphore := make(chan struct{}, 5)
	for _, account := range accounts {
		wg.Add(1)
		go func(acc *repository.Account) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := s.testPlatformAccount(ctx, acc)

			accountID := strconv.FormatInt(acc.ID, 10)
			s.latestResults.Store(accountID, result)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(account)
	}

	wg.Wait()

	// 更新全部结果缓存
	s.mu.Lock()
	s.allResults = results
	s.mu.Unlock()

	return results, nil
}

// GetLatestResults 获取最新的测试结果
func (s *channelTestService) GetLatestResults() []*ChannelTestResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.allResults) > 0 {
		// 返回副本
		results := make([]*ChannelTestResult, len(s.allResults))
		copy(results, s.allResults)
		return results
	}

	// 从 sync.Map 收集
	var results []*ChannelTestResult
	s.latestResults.Range(func(key, value interface{}) bool {
		results = append(results, value.(*ChannelTestResult))
		return true
	})

	return results
}

// StartPeriodicTest 启动后台定期测试
func (s *channelTestService) StartPeriodicTest(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.logger.Info("渠道定期测试已启动", zap.Duration("interval", interval))

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("渠道定期测试已停止")
			return
		case <-ticker.C:
			s.logger.Debug("开始定期测试渠道...")
			results, err := s.TestAllAccounts(ctx)
			if err != nil {
				s.logger.Error("定期测试渠道失败", zap.Error(err))
				continue
			}

			healthy := 0
			for _, r := range results {
				if r.IsHealthy {
					healthy++
				}
			}
			s.logger.Debug("定期测试渠道完成",
				zap.Int("total", len(results)),
				zap.Int("healthy", healthy))
		}
	}
}

// testPlatformAccount 根据平台测试账户
func (s *channelTestService) testPlatformAccount(ctx context.Context, account *repository.Account) *ChannelTestResult {
	accountID := strconv.FormatInt(account.ID, 10)
	startTime := time.Now()

	result := &ChannelTestResult{
		AccountID:   accountID,
		AccountName: account.Name,
		Platform:    account.Platform,
		TestedAt:    startTime,
	}

	switch account.Platform {
	case "claude", "anthropic":
		result.Model = "claude-3-haiku-20240307"
		s.testAnthropicAccount(ctx, account, result)
	case "openai":
		result.Model = "gpt-3.5-turbo"
		s.testOpenAIAccount(ctx, account, result)
	case "self_hosted":
		result.Model = "default"
		result.IsHealthy = true
		result.LatencyMs = 0
		result.ErrorMessage = "自托管平台无需测试"
	default:
		result.IsHealthy = false
		result.ErrorMessage = "不支持的平台: " + account.Platform
	}

	result.LatencyMs = time.Since(startTime).Milliseconds()
	return result
}

// testAnthropicAccount 测试 Anthropic 账户
func (s *channelTestService) testAnthropicAccount(ctx context.Context, account *repository.Account, result *ChannelTestResult) {
	apiKey, err := s.getAPIKey(account)
	if err != nil {
		result.IsHealthy = false
		result.ErrorMessage = "获取 API Key 失败: " + err.Error()
		return
	}

	// 构建最小测试请求
	reqBody := map[string]interface{}{
		"model":      "claude-3-haiku-20240307",
		"max_tokens": 1,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": "hi",
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		result.IsHealthy = false
		result.ErrorMessage = "序列化请求失败: " + err.Error()
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		result.IsHealthy = false
		result.ErrorMessage = "创建请求失败: " + err.Error()
		return
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		result.IsHealthy = false
		result.ErrorMessage = "请求失败: " + err.Error()
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		result.IsHealthy = true
	} else {
		result.IsHealthy = false
		result.ErrorMessage = fmt.Sprintf("API 返回错误: status=%d, body=%s", resp.StatusCode, string(respBody))
	}
}

// testOpenAIAccount 测试 OpenAI 账户
func (s *channelTestService) testOpenAIAccount(ctx context.Context, account *repository.Account, result *ChannelTestResult) {
	apiKey, err := s.getAPIKey(account)
	if err != nil {
		result.IsHealthy = false
		result.ErrorMessage = "获取 API Key 失败: " + err.Error()
		return
	}

	// 构建最小测试请求
	reqBody := map[string]interface{}{
		"model":      "gpt-3.5-turbo",
		"max_tokens": 1,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": "hi",
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		result.IsHealthy = false
		result.ErrorMessage = "序列化请求失败: " + err.Error()
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		result.IsHealthy = false
		result.ErrorMessage = "创建请求失败: " + err.Error()
		return
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		result.IsHealthy = false
		result.ErrorMessage = "请求失败: " + err.Error()
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		result.IsHealthy = true
	} else {
		result.IsHealthy = false
		result.ErrorMessage = fmt.Sprintf("API 返回错误: status=%d, body=%s", resp.StatusCode, string(respBody))
	}
}

// getAPIKey 从账户凭证中获取 API Key
func (s *channelTestService) getAPIKey(account *repository.Account) (string, error) {
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

// updateAllResults 更新全部结果列表（替换已有账户的结果）
func (s *channelTestService) updateAllResults(newResult *ChannelTestResult) {
	found := false
	for i, r := range s.allResults {
		if r.AccountID == newResult.AccountID {
			s.allResults[i] = newResult
			found = true
			break
		}
	}
	if !found {
		s.allResults = append(s.allResults, newResult)
	}
}
