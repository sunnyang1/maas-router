// Package handler 提供 MaaS-Router 的 HTTP 处理器
package handler

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"
)

// RetryConfig 定义重试行为
type RetryConfig struct {
	MaxRetries          int           // 最大重试次数（默认 2）
	InitialBackoff      time.Duration // 初始退避时间（默认 100ms）
	MaxBackoff          time.Duration // 最大退避时间（默认 1s）
	RetryableStatusCodes []int        // 触发重试的 HTTP 状态码（如 429, 500, 502, 503, 504）
}

// DefaultRetryConfig 返回合理的默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 2,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		RetryableStatusCodes: []int{
			http.StatusTooManyRequests, // 429
			http.StatusInternalServerError, // 500
			http.StatusBadGateway,      // 502
			http.StatusServiceUnavailable, // 503
			http.StatusGatewayTimeout,  // 504
		},
	}
}

// RetryResult 包含重试操作的结果
type RetryResult struct {
	Response  *http.Response // 最终响应（成功时非 nil）
	Account   *Account       // 最终使用的账号
	Attempt   int            // 成功时的尝试次数（1-based）
	Retried   bool           // 是否发生过重试
	LastError error          // 最后一次错误
}

// IsRetryable 检查错误或 HTTP 状态码是否应触发重试
func (c *RetryConfig) IsRetryable(resp *http.Response, err error) bool {
	// 网络错误（连接失败、超时等）应重试
	if err != nil {
		return true
	}
	if resp == nil {
		return true
	}

	// 检查状态码是否在可重试列表中
	for _, code := range c.RetryableStatusCodes {
		if resp.StatusCode == code {
			return true
		}
	}

	return false
}

// DoRequestWithRetry 执行带重试逻辑的请求
// 每次重试时调用 selectAccountFunc 获取新账号（排除之前失败的账号）
// 参数：
//   - ctx: 请求上下文
//   - req: 代理请求
//   - selectAccountFunc: 账号选择函数，接收 excludedAccountIDs 以避免重用失败账号
//   - proxyService: 代理服务
//   - config: 重试配置
func DoRequestWithRetry(
	ctx context.Context,
	req *ProxyRequest,
	selectAccountFunc func(ctx context.Context, excludedAccountIDs []string) (*Account, error),
	proxyService ProxyService,
	config *RetryConfig,
) *RetryResult {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var excludedIDs []string
	var lastErr error
	var lastResp *http.Response
	var lastAccount *Account

	maxAttempts := config.MaxRetries + 1 // 首次尝试 + 重试次数

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return &RetryResult{
				Attempt:   attempt,
				Retried:   attempt > 1,
				LastError: ctx.Err(),
			}
		default:
		}

		// 选择账号（首次尝试不排除任何账号）
		account, err := selectAccountFunc(ctx, excludedIDs)
		if err != nil {
			lastErr = fmt.Errorf("选择账号失败: %w", err)
			// 如果无法选择账号，不再重试
			return &RetryResult{
				Attempt:   attempt,
				Retried:   attempt > 1,
				LastError: lastErr,
			}
		}

		// 更新请求中的账号 ID
		reqCopy := *req
		reqCopy.AccountID = account.ID

		// 执行请求
		resp, err := proxyService.DoRequest(ctx, &reqCopy)

		// 检查是否可重试
		if !config.IsRetryable(resp, err) {
			// 请求成功或错误不可重试，直接返回
			return &RetryResult{
				Response: resp,
				Account:  account,
				Attempt:  attempt,
				Retried:  attempt > 1,
			}
		}

		// 记录失败的账号
		lastErr = err
		lastResp = resp
		lastAccount = account

		// 关闭失败的响应体（避免资源泄漏）
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}

		// 将失败账号加入排除列表
		excludedIDs = append(excludedIDs, account.ID)

		// 如果是最后一次尝试，不再等待
		if attempt >= maxAttempts {
			break
		}

		// 指数退避等待
		backoff := calculateBackoff(config.InitialBackoff, config.MaxBackoff, attempt-1)
		select {
		case <-ctx.Done():
			return &RetryResult{
				Attempt:   attempt,
				Retried:   true,
				LastError: ctx.Err(),
			}
		case <-time.After(backoff):
		}
	}

	// 所有尝试都失败
	return &RetryResult{
		Response:  lastResp,
		Account:   lastAccount,
		Attempt:   maxAttempts,
		Retried:   true,
		LastError: lastErr,
	}
}

// calculateBackoff 计算指数退避时间
// attempt 从 0 开始（第一次重试时 attempt=0）
func calculateBackoff(initial, max time.Duration, attempt int) time.Duration {
	// 指数退避: initial * 2^attempt，加上 10% 的随机抖动
	multiplier := math.Pow(2, float64(attempt))
	backoff := time.Duration(float64(initial) * multiplier)

	// 限制最大退避时间
	if backoff > max {
		backoff = max
	}

	// 添加随机抖动（+/- 10%）防止惊群效应
	jitter := time.Duration(float64(backoff) * 0.1)
	backoff = backoff - jitter + time.Duration(float64(jitter)*2*float64(time.Now().UnixNano()%1000)/1000.0)

	if backoff < 0 {
		backoff = initial
	}
	if backoff > max {
		backoff = max
	}

	return backoff
}
