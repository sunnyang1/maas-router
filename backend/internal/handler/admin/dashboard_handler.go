// Package admin 提供管理员相关的 HTTP 处理器
package admin

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"maas-router/internal/repository"
)

// DashboardHandler 管理仪表盘 Handler
type DashboardHandler struct {
	userRepo       *repository.UserRepository
	usageRepo      *repository.UsageRecordRepository
	accountRepo    *repository.AccountRepository
	groupRepo      *repository.GroupRepository
	routerRuleRepo *repository.RouterRuleRepository
}

// NewDashboardHandler 创建管理仪表盘 Handler
func NewDashboardHandler(
	userRepo *repository.UserRepository,
	usageRepo *repository.UsageRecordRepository,
	accountRepo *repository.AccountRepository,
	groupRepo *repository.GroupRepository,
	routerRuleRepo *repository.RouterRuleRepository,
) *DashboardHandler {
	return &DashboardHandler{
		userRepo:       userRepo,
		usageRepo:      usageRepo,
		accountRepo:    accountRepo,
		groupRepo:      groupRepo,
		routerRuleRepo: routerRuleRepo,
	}
}

// DashboardStatsResponse 仪表盘统计响应
type DashboardStatsResponse struct {
	// 用户统计
	UserStats UserStats `json:"user_stats"`
	// 账号统计
	AccountStats AccountStats `json:"account_stats"`
	// 分组统计
	GroupStats GroupStats `json:"group_stats"`
	// 请求统计
	RequestStats RequestStats `json:"request_stats"`
	// 费用统计
	CostStats CostStats `json:"cost_stats"`
	// 路由规则统计
	RouterRuleStats RouterRuleStats `json:"router_rule_stats"`
}

// UserStats 用户统计
type UserStats struct {
	// 总用户数
	Total int64 `json:"total"`
	// 活跃用户数
	Active int64 `json:"active"`
	// 今日新增用户数
	TodayNew int64 `json:"today_new"`
	// 本月新增用户数
	MonthNew int64 `json:"month_new"`
}

// AccountStats 账号统计
type AccountStats struct {
	// 总账号数
	Total int64 `json:"total"`
	// 活跃账号数
	Active int64 `json:"active"`
	// 禁用账号数
	Disabled int64 `json:"disabled"`
	// 各平台账号数
	ByPlatform map[string]int64 `json:"by_platform"`
}

// GroupStats 分组统计
type GroupStats struct {
	// 总分组数
	Total int64 `json:"total"`
	// 活跃分组数
	Active int64 `json:"active"`
	// 各平台分组数
	ByPlatform map[string]int64 `json:"by_platform"`
}

// RequestStats 请求统计
type RequestStats struct {
	// 今日请求数
	TodayTotal int64 `json:"today_total"`
	// 今日成功请求数
	TodaySuccess int64 `json:"today_success"`
	// 今日失败请求数
	TodayFailed int64 `json:"today_failed"`
	// 今日超时请求数
	TodayTimeout int64 `json:"today_timeout"`
	// 本月请求数
	MonthTotal int64 `json:"month_total"`
	// 总请求数
	Total int64 `json:"total"`
	// 平均延迟（毫秒）
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	// 平均首Token延迟（毫秒）
	AvgFirstTokenMs float64 `json:"avg_first_token_ms"`
}

// CostStats 费用统计
type CostStats struct {
	// 今日费用
	TodayCost float64 `json:"today_cost"`
	// 本月费用
	MonthCost float64 `json:"month_cost"`
	// 总费用
	TotalCost float64 `json:"total_cost"`
	// 今日Token数
	TodayTokens int64 `json:"today_tokens"`
	// 本月Token数
	MonthTokens int64 `json:"month_tokens"`
}

// RouterRuleStats 路由规则统计
type RouterRuleStats struct {
	// 总规则数
	Total int64 `json:"total"`
	// 启用规则数
	Active int64 `json:"active"`
}

// GetStats 获取系统统计数据
// GET /api/v1/admin/dashboard/stats
func (h *DashboardHandler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()

	// 获取用户统计
	userStats, err := h.getUserStats(ctx)
	if err != nil {
		c.JSON(500, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取用户统计失败",
			},
		})
		return
	}

	// 获取账号统计
	accountStats, err := h.getAccountStats(ctx)
	if err != nil {
		c.JSON(500, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取账号统计失败",
			},
		})
		return
	}

	// 获取分组统计
	groupStats, err := h.getGroupStats(ctx)
	if err != nil {
		c.JSON(500, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取分组统计失败",
			},
		})
		return
	}

	// 获取请求统计
	requestStats, err := h.getRequestStats(ctx)
	if err != nil {
		c.JSON(500, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取请求统计失败",
			},
		})
		return
	}

	// 获取费用统计
	costStats, err := h.getCostStats(ctx)
	if err != nil {
		c.JSON(500, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取费用统计失败",
			},
		})
		return
	}

	// 获取路由规则统计
	routerRuleStats, err := h.getRouterRuleStats(ctx)
	if err != nil {
		c.JSON(500, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取路由规则统计失败",
			},
		})
		return
	}

	c.JSON(200, DashboardStatsResponse{
		UserStats:       userStats,
		AccountStats:    accountStats,
		GroupStats:      groupStats,
		RequestStats:    requestStats,
		CostStats:       costStats,
		RouterRuleStats: routerRuleStats,
	})
}

// RealtimeDataResponse 实时数据响应
type RealtimeDataResponse struct {
	// 当前时间
	Timestamp time.Time `json:"timestamp"`
	// 实时请求统计
	RealtimeRequests RealtimeRequests `json:"realtime_requests"`
	// 实时账号状态
	RealtimeAccounts []RealtimeAccount `json:"realtime_accounts"`
	// 系统健康状态
	SystemHealth SystemHealth `json:"system_health"`
}

// RealtimeRequests 实时请求统计
type RealtimeRequests struct {
	// 最近1分钟请求数
	Last1Min int64 `json:"last_1_min"`
	// 最近5分钟请求数
	Last5Min int64 `json:"last_5_min"`
	// 最近15分钟请求数
	Last15Min int64 `json:"last_15_min"`
	// 当前并发请求数
	CurrentConcurrency int64 `json:"current_concurrency"`
	// 最近1分钟平均延迟
	AvgLatency1Min float64 `json:"avg_latency_1_min"`
	// 最近1分钟错误率
	ErrorRate1Min float64 `json:"error_rate_1_min"`
}

// RealtimeAccount 实时账号状态
type RealtimeAccount struct {
	// 账号ID
	ID int64 `json:"id"`
	// 账号名称
	Name string `json:"name"`
	// 平台
	Platform string `json:"platform"`
	// 状态
	Status string `json:"status"`
	// 当前并发数
	CurrentConcurrency int `json:"current_concurrency"`
	// 最大并发数
	MaxConcurrency int `json:"max_concurrency"`
	// 最近1分钟请求数
	Requests1Min int64 `json:"requests_1_min"`
	// 最近错误时间
	LastErrorAt *time.Time `json:"last_error_at,omitempty"`
}

// SystemHealth 系统健康状态
type SystemHealth struct {
	// 数据库连接状态
	Database string `json:"database"`
	// Redis连接状态
	Redis string `json:"redis"`
	// 系统负载
	LoadAvg string `json:"load_avg"`
	// 内存使用率
	MemoryUsage string `json:"memory_usage"`
}

// GetRealtime 获取实时数据
// GET /api/v1/admin/dashboard/realtime
func (h *DashboardHandler) GetRealtime(c *gin.Context) {
	ctx := c.Request.Context()

	// 获取实时请求统计
	realtimeRequests, err := h.getRealtimeRequests(ctx)
	if err != nil {
		c.JSON(500, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取实时请求统计失败",
			},
		})
		return
	}

	// 获取实时账号状态
	realtimeAccounts, err := h.getRealtimeAccounts(ctx)
	if err != nil {
		c.JSON(500, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取实时账号状态失败",
			},
		})
		return
	}

	// 获取系统健康状态
	systemHealth := h.getSystemHealth()

	c.JSON(200, RealtimeDataResponse{
		Timestamp:         time.Now(),
		RealtimeRequests:  realtimeRequests,
		RealtimeAccounts:  realtimeAccounts,
		SystemHealth:      systemHealth,
	})
}

// getUserStats 获取用户统计
func (h *DashboardHandler) getUserStats(ctx context.Context) (UserStats, error) {
	// 获取总用户数
	total, err := h.userRepo.Count(ctx)
	if err != nil {
		return UserStats{}, err
	}

	// 获取活跃用户数
	active, err := h.userRepo.CountByStatus(ctx, "active")
	if err != nil {
		return UserStats{}, err
	}

	// 获取今日新增用户数
	todayStart := time.Now().Truncate(24 * time.Hour)
	todayNew, err := h.userRepo.CountByCreatedAt(ctx, todayStart, time.Now())
	if err != nil {
		return UserStats{}, err
	}

	// 获取本月新增用户数
	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)
	monthNew, err := h.userRepo.CountByCreatedAt(ctx, monthStart, time.Now())
	if err != nil {
		return UserStats{}, err
	}

	return UserStats{
		Total:    total,
		Active:   active,
		TodayNew: todayNew,
		MonthNew: monthNew,
	}, nil
}

// getAccountStats 获取账号统计
func (h *DashboardHandler) getAccountStats(ctx context.Context) (AccountStats, error) {
	// 获取总账号数
	total, err := h.accountRepo.Count(ctx)
	if err != nil {
		return AccountStats{}, err
	}

	// 获取活跃账号数
	active, err := h.accountRepo.CountByStatus(ctx, "active")
	if err != nil {
		return AccountStats{}, err
	}

	// 获取禁用账号数
	disabled, err := h.accountRepo.CountByStatus(ctx, "disabled")
	if err != nil {
		return AccountStats{}, err
	}

	// 获取各平台账号数
	byPlatform, err := h.accountRepo.CountGroupByPlatform(ctx)
	if err != nil {
		return AccountStats{}, err
	}

	return AccountStats{
		Total:      total,
		Active:     active,
		Disabled:   disabled,
		ByPlatform: byPlatform,
	}, nil
}

// getGroupStats 获取分组统计
func (h *DashboardHandler) getGroupStats(ctx context.Context) (GroupStats, error) {
	// 获取总分组数
	total, err := h.groupRepo.Count(ctx)
	if err != nil {
		return GroupStats{}, err
	}

	// 获取活跃分组数
	active, err := h.groupRepo.CountByStatus(ctx, "active")
	if err != nil {
		return GroupStats{}, err
	}

	// 获取各平台分组数
	byPlatform, err := h.groupRepo.CountGroupByPlatform(ctx)
	if err != nil {
		return GroupStats{}, err
	}

	return GroupStats{
		Total:      total,
		Active:     active,
		ByPlatform: byPlatform,
	}, nil
}

// getRequestStats 获取请求统计
func (h *DashboardHandler) getRequestStats(ctx context.Context) (RequestStats, error) {
	// 获取今日请求统计
	todayStart := time.Now().Truncate(24 * time.Hour)
	todayStats, err := h.usageRepo.GetStatsByTimeRange(ctx, todayStart, time.Now())
	if err != nil {
		return RequestStats{}, err
	}

	// 获取本月请求统计
	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)
	monthStats, err := h.usageRepo.GetStatsByTimeRange(ctx, monthStart, time.Now())
	if err != nil {
		return RequestStats{}, err
	}

	// 获取总请求统计
	totalStats, err := h.usageRepo.GetStatsByTimeRange(ctx, time.Time{}, time.Now())
	if err != nil {
		return RequestStats{}, err
	}

	return RequestStats{
		TodayTotal:      todayStats.TotalRequests,
		TodaySuccess:    todayStats.SuccessRequests,
		TodayFailed:     todayStats.FailedRequests,
		TodayTimeout:    todayStats.TimeoutRequests,
		MonthTotal:      monthStats.TotalRequests,
		Total:           totalStats.TotalRequests,
		AvgLatencyMs:    todayStats.AvgLatencyMs,
		AvgFirstTokenMs: todayStats.AvgFirstTokenMs,
	}, nil
}

// getCostStats 获取费用统计
func (h *DashboardHandler) getCostStats(ctx context.Context) (CostStats, error) {
	// 获取今日费用统计
	todayStart := time.Now().Truncate(24 * time.Hour)
	todayStats, err := h.usageRepo.GetStatsByTimeRange(ctx, todayStart, time.Now())
	if err != nil {
		return CostStats{}, err
	}

	// 获取本月费用统计
	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)
	monthStats, err := h.usageRepo.GetStatsByTimeRange(ctx, monthStart, time.Now())
	if err != nil {
		return CostStats{}, err
	}

	// 获取总费用统计
	totalStats, err := h.usageRepo.GetStatsByTimeRange(ctx, time.Time{}, time.Now())
	if err != nil {
		return CostStats{}, err
	}

	return CostStats{
		TodayCost:   todayStats.TotalCost,
		MonthCost:   monthStats.TotalCost,
		TotalCost:   totalStats.TotalCost,
		TodayTokens: int64(todayStats.TotalTokens),
		MonthTokens: int64(monthStats.TotalTokens),
	}, nil
}

// getRouterRuleStats 获取路由规则统计
func (h *DashboardHandler) getRouterRuleStats(ctx context.Context) (RouterRuleStats, error) {
	// 获取总规则数
	total, err := h.routerRuleRepo.Count(ctx)
	if err != nil {
		return RouterRuleStats{}, err
	}

	// 获取启用规则数
	active, err := h.routerRuleRepo.CountActive(ctx)
	if err != nil {
		return RouterRuleStats{}, err
	}

	return RouterRuleStats{
		Total:  total,
		Active: active,
	}, nil
}

// getRealtimeRequests 获取实时请求统计
func (h *DashboardHandler) getRealtimeRequests(ctx context.Context) (RealtimeRequests, error) {
	now := time.Now()

	// 获取最近1分钟请求数
	last1Min, err := h.usageRepo.CountByTimeRange(ctx, now.Add(-1*time.Minute), now)
	if err != nil {
		return RealtimeRequests{}, err
	}

	// 获取最近5分钟请求数
	last5Min, err := h.usageRepo.CountByTimeRange(ctx, now.Add(-5*time.Minute), now)
	if err != nil {
		return RealtimeRequests{}, err
	}

	// 获取最近15分钟请求数
	last15Min, err := h.usageRepo.CountByTimeRange(ctx, now.Add(-15*time.Minute), now)
	if err != nil {
		return RealtimeRequests{}, err
	}

	// 获取最近1分钟统计
	stats1Min, err := h.usageRepo.GetStatsByTimeRange(ctx, now.Add(-1*time.Minute), now)
	if err != nil {
		return RealtimeRequests{}, err
	}

	// 计算错误率
	var errorRate float64
	if stats1Min.TotalRequests > 0 {
		errorRate = float64(stats1Min.FailedRequests+stats1Min.TimeoutRequests) / float64(stats1Min.TotalRequests) * 100
	}

	// 获取当前并发数（从账号汇总）
	currentConcurrency, err := h.accountRepo.SumCurrentConcurrency(ctx)
	if err != nil {
		currentConcurrency = 0
	}

	return RealtimeRequests{
		Last1Min:           last1Min,
		Last5Min:           last5Min,
		Last15Min:          last15Min,
		CurrentConcurrency: currentConcurrency,
		AvgLatency1Min:     stats1Min.AvgLatencyMs,
		ErrorRate1Min:      errorRate,
	}, nil
}

// getRealtimeAccounts 获取实时账号状态
func (h *DashboardHandler) getRealtimeAccounts(ctx context.Context) ([]RealtimeAccount, error) {
	// 获取所有活跃账号
	accounts, err := h.accountRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	result := make([]RealtimeAccount, 0, len(accounts))

	for _, acc := range accounts {
		// 获取账号最近1分钟请求数
		requests1Min, _ := h.usageRepo.CountByAccountAndTimeRange(ctx, acc.ID, now.Add(-1*time.Minute), now)

		realtimeAcc := RealtimeAccount{
			ID:                 acc.ID,
			Name:               acc.Name,
			Platform:           string(acc.Platform),
			Status:             string(acc.Status),
			CurrentConcurrency: acc.CurrentConcurrency,
			MaxConcurrency:     acc.MaxConcurrency,
			Requests1Min:       requests1Min,
			LastErrorAt:        acc.LastErrorAt,
		}
		result = append(result, realtimeAcc)
	}

	return result, nil
}

// getSystemHealth 获取系统健康状态
func (h *DashboardHandler) getSystemHealth() SystemHealth {
	// TODO: 实现真实的健康检查
	return SystemHealth{
		Database:     "healthy",
		Redis:        "healthy",
		LoadAvg:      "0.5, 0.3, 0.2",
		MemoryUsage:  "45%",
	}
}
