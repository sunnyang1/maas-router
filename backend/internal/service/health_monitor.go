// Package service 提供 MaaS-Router 的业务逻辑服务层
package service

import (
	"context"
	"sync"
	"time"
)

// AccountHealthMonitor 使用滑动窗口成功率监控账号健康状态
type AccountHealthMonitor interface {
	// RecordResult 记录账号的请求结果
	RecordResult(accountID string, success bool)

	// GetSuccessRate 获取账号当前成功率
	GetSuccessRate(accountID string) float64

	// IsHealthy 检查账号是否健康（成功率高于阈值）
	IsHealthy(accountID string) bool

	// GetUnhealthyAccounts 返回所有当前不健康的账号 ID
	GetUnhealthyAccounts() []string

	// GetAccountStats 返回账号的详细统计信息
	GetAccountStats(accountID string) *AccountHealthStats

	// Start 启动后台清理 goroutine
	Start(ctx context.Context)

	// SetThreshold 设置健康判定的成功率阈值
	SetThreshold(threshold float64)

	// SetRecoveryThreshold 设置恢复判定的成功率阈值
	SetRecoveryThreshold(threshold float64)
}

// AccountHealthStats 账号健康统计信息
type AccountHealthStats struct {
	AccountID           string    `json:"account_id"`
	TotalRequests       int64     `json:"total_requests"`
	SuccessCount        int64     `json:"success_count"`
	FailureCount        int64     `json:"failure_count"`
	SuccessRate         float64   `json:"success_rate"`
	IsHealthy           bool      `json:"is_healthy"`
	LastFailure         time.Time `json:"last_failure"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
}

// accountHealthMonitor 健康监控实现
type accountHealthMonitor struct {
	mu                sync.RWMutex
	windows           map[string]*slidingWindow
	healthyState      map[string]bool // 账号当前健康状态（带迟滞）
	disableThreshold  float64         // 低于此值 -> 标记为不健康（默认 0.8）
	recoveryThreshold float64         // 高于此值 -> 恢复为健康（默认 0.9）
	windowSize        int             // 保留的结果数量（默认 100）
	cleanupInterval   time.Duration
	lastAccess        map[string]time.Time // 最后访问时间，用于清理
}

// slidingWindow 滑动窗口实现（环形缓冲区）
type slidingWindow struct {
	results   []bool // 环形缓冲区，true=成功，false=失败
	index     int    // 当前写入位置
	count     int    // 已记录的总数
	successes int    // 成功数
}

// newSlidingWindow 创建指定大小的滑动窗口
func newSlidingWindow(size int) *slidingWindow {
	return &slidingWindow{
		results: make([]bool, size),
		index:   0,
		count:   0,
		successes: 0,
	}
}

// record 记录一个结果
func (w *slidingWindow) record(success bool) {
	if w.count < len(w.results) {
		// 窗口未满，直接写入
		if w.results[w.index] {
			// 覆盖前如果是成功，减少成功计数（仅在窗口已满时需要）
		}
		w.results[w.index] = success
		w.index = (w.index + 1) % len(w.results)
		w.count++
		if success {
			w.successes++
		}
	} else {
		// 窗口已满，覆盖最旧的记录
		old := w.results[w.index]
		w.results[w.index] = success
		w.index = (w.index + 1) % len(w.results)
		if success {
			w.successes++
		}
		if old {
			w.successes--
		}
	}
}

// successRate 计算成功率
func (w *slidingWindow) successRate() float64 {
	if w.count == 0 {
		return 1.0 // 没有记录时视为健康
	}
	return float64(w.successes) / float64(w.count)
}

// consecutiveFailures 计算从最新记录往回的连续失败次数
func (w *slidingWindow) consecutiveFailures() int {
	failures := 0
	// 从当前位置往前遍历
	start := (w.index - 1 + len(w.results)) % len(w.results)
	for i := 0; i < w.count; i++ {
		pos := (start - i + len(w.results)) % len(w.results)
		if w.results[pos] {
			break // 遇到成功就停止
		}
		failures++
	}
	return failures
}

// NewAccountHealthMonitor 创建账号健康监控器
func NewAccountHealthMonitor() AccountHealthMonitor {
	return &accountHealthMonitor{
		windows:           make(map[string]*slidingWindow),
		healthyState:      make(map[string]bool),
		disableThreshold:  0.8,
		recoveryThreshold: 0.9,
		windowSize:        100,
		cleanupInterval:   5 * time.Minute,
		lastAccess:        make(map[string]time.Time),
	}
}

// RecordResult 记录账号的请求结果
func (m *accountHealthMonitor) RecordResult(accountID string, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	window, ok := m.windows[accountID]
	if !ok {
		window = newSlidingWindow(m.windowSize)
		m.windows[accountID] = window
		m.healthyState[accountID] = true // 新账号默认健康
	}

	window.record(success)
	m.lastAccess[accountID] = time.Now()

	// 检查是否需要更新健康状态（带迟滞逻辑）
	rate := window.successRate()
	currentHealthy := m.healthyState[accountID]

	if currentHealthy {
		// 当前健康：检查是否需要标记为不健康
		if rate < m.disableThreshold {
			m.healthyState[accountID] = false
		}
	} else {
		// 当前不健康：检查是否需要恢复
		if rate >= m.recoveryThreshold {
			m.healthyState[accountID] = true
		}
	}
}

// GetSuccessRate 获取账号当前成功率
func (m *accountHealthMonitor) GetSuccessRate(accountID string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	window, ok := m.windows[accountID]
	if !ok {
		return 1.0 // 没有记录时视为健康
	}
	return window.successRate()
}

// IsHealthy 检查账号是否健康
func (m *accountHealthMonitor) IsHealthy(accountID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	healthy, ok := m.healthyState[accountID]
	if !ok {
		return true // 未知账号默认健康
	}
	return healthy
}

// GetUnhealthyAccounts 返回所有不健康的账号 ID
func (m *accountHealthMonitor) GetUnhealthyAccounts() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var unhealthy []string
	for id, healthy := range m.healthyState {
		if !healthy {
			unhealthy = append(unhealthy, id)
		}
	}
	return unhealthy
}

// GetAccountStats 返回账号的详细统计信息
func (m *accountHealthMonitor) GetAccountStats(accountID string) *AccountHealthStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	window, ok := m.windows[accountID]
	if !ok {
		return &AccountHealthStats{
			AccountID:   accountID,
			IsHealthy:   true,
			SuccessRate: 1.0,
		}
	}

	stats := &AccountHealthStats{
		AccountID:           accountID,
		TotalRequests:       int64(window.count),
		SuccessCount:        int64(window.successes),
		FailureCount:        int64(window.count - window.successes),
		SuccessRate:         window.successRate(),
		IsHealthy:           m.healthyState[accountID],
		ConsecutiveFailures: window.consecutiveFailures(),
	}

	// 计算最后失败时间（从窗口中查找最后一次失败的位置）
	// 由于环形缓冲区不记录时间戳，这里用零值表示
	// 实际使用中可以通过外部事件记录获取更精确的时间
	if stats.FailureCount > 0 {
		stats.LastFailure = time.Time{} // 占位，实际由调用方补充
	}

	return stats
}

// Start 启动后台清理 goroutine
func (m *accountHealthMonitor) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(m.cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.cleanup()
			}
		}
	}()
}

// cleanup 清理长时间未访问的账号数据
func (m *accountHealthMonitor) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	staleThreshold := 30 * time.Minute

	for id, lastAccess := range m.lastAccess {
		if now.Sub(lastAccess) > staleThreshold {
			delete(m.windows, id)
			delete(m.healthyState, id)
			delete(m.lastAccess, id)
		}
	}
}

// SetThreshold 设置健康判定的成功率阈值
func (m *accountHealthMonitor) SetThreshold(threshold float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disableThreshold = threshold
}

// SetRecoveryThreshold 设置恢复判定的成功率阈值
func (m *accountHealthMonitor) SetRecoveryThreshold(threshold float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recoveryThreshold = threshold
}
