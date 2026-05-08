// Package admin 提供管理员相关的 HTTP 处理器
package admin

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"maas-router/internal/repository"
)

// OpsHandler 运维监控 Handler
type OpsHandler struct {
	accountRepo *repository.AccountRepository
	usageRepo   *repository.UsageRecordRepository
}

// NewOpsHandler 创建运维监控 Handler
func NewOpsHandler(
	accountRepo *repository.AccountRepository,
	usageRepo *repository.UsageRecordRepository,
) *OpsHandler {
	return &OpsHandler{
		accountRepo: accountRepo,
		usageRepo:   usageRepo,
	}
}

// ConcurrencyStatsResponse 并发统计响应
type ConcurrencyStatsResponse struct {
	// 当前时间
	Timestamp string `json:"timestamp"`
	// 总并发数
	TotalConcurrency int64 `json:"total_concurrency"`
	// 最大并发容量
	MaxConcurrency int64 `json:"max_concurrency"`
	// 并发使用率
	ConcurrencyRate float64 `json:"concurrency_rate"`
	// 各平台并发统计
	ByPlatform []PlatformConcurrency `json:"by_platform"`
	// 各账号并发详情
	Accounts []AccountConcurrency `json:"accounts"`
}

// PlatformConcurrency 平台并发统计
type PlatformConcurrency struct {
	// 平台名称
	Platform string `json:"platform"`
	// 当前并发数
	CurrentConcurrency int64 `json:"current_concurrency"`
	// 最大并发数
	MaxConcurrency int64 `json:"max_concurrency"`
	// 账号数量
	AccountCount int64 `json:"account_count"`
}

// AccountConcurrency 账号并发详情
type AccountConcurrency struct {
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
	// 使用率
	UsageRate float64 `json:"usage_rate"`
}

// RealtimeTrafficResponse 实时流量响应
type RealtimeTrafficResponse struct {
	// 当前时间
	Timestamp string `json:"timestamp"`
	// 最近1分钟流量
	Last1Min TrafficStats `json:"last_1_min"`
	// 最近5分钟流量
	Last5Min TrafficStats `json:"last_5_min"`
	// 最近15分钟流量
	Last15Min TrafficStats `json:"last_15_min"`
	// 最近1小时流量
	Last1Hour TrafficStats `json:"last_1_hour"`
	// 流量趋势（每分钟）
	Trend []TrafficPoint `json:"trend"`
}

// TrafficStats 流量统计
type TrafficStats struct {
	// 请求数
	Requests int64 `json:"requests"`
	// 成功数
	Success int64 `json:"success"`
	// 失败数
	Failed int64 `json:"failed"`
	// 超时数
	Timeout int64 `json:"timeout"`
	// 平均延迟（毫秒）
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	// 平均首Token延迟（毫秒）
	AvgFirstTokenMs float64 `json:"avg_first_token_ms"`
	// 总Token数
	TotalTokens int64 `json:"total_tokens"`
	// 总费用
	TotalCost float64 `json:"total_cost"`
	// QPS
	QPS float64 `json:"qps"`
	// 错误率
	ErrorRate float64 `json:"error_rate"`
}

// TrafficPoint 流量数据点
type TrafficPoint struct {
	// 时间
	Timestamp string `json:"timestamp"`
	// 请求数
	Requests int64 `json:"requests"`
	// 平均延迟
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	// 错误率
	ErrorRate float64 `json:"error_rate"`
}

// ErrorLogListRequest 错误日志列表请求
type ErrorLogListRequest struct {
	// 页码
	Page int `form:"page" binding:"min=1"`
	// 每页数量
	PageSize int `form:"page_size" binding:"min=1,max=100"`
	// 平台筛选
	Platform string `form:"platform"`
	// 账号ID筛选
	AccountID int64 `form:"account_id"`
	// 开始时间
	StartTime string `form:"start_time"`
	// 结束时间
	EndTime string `form:"end_time"`
}

// ErrorLogListResponse 错误日志列表响应
type ErrorLogListResponse struct {
	// 错误日志列表
	List []*ErrorLogInfo `json:"list"`
	// 总数
	Total int64 `json:"total"`
	// 当前页码
	Page int `json:"page"`
	// 每页数量
	PageSize int `json:"page_size"`
}

// ErrorLogInfo 错误日志信息
type ErrorLogInfo struct {
	// 请求ID
	RequestID string `json:"request_id"`
	// 用户ID
	UserID int64 `json:"user_id"`
	// 账号ID
	AccountID int64 `json:"account_id,omitempty"`
	// 分组ID
	GroupID int64 `json:"group_id,omitempty"`
	// 模型
	Model string `json:"model"`
	// 平台
	Platform string `json:"platform"`
	// 状态
	Status string `json:"status"`
	// 错误信息
	ErrorMessage string `json:"error_message"`
	// 客户端IP
	ClientIP string `json:"client_ip,omitempty"`
	// 创建时间
	CreatedAt string `json:"created_at"`
}

// AlertRuleResponse 告警规则响应
type AlertRuleResponse struct {
	// 告警规则列表
	Rules []AlertRuleInfo `json:"rules"`
}

// AlertRuleInfo 告警规则信息
type AlertRuleInfo struct {
	// 规则ID
	ID string `json:"id"`
	// 规则名称
	Name string `json:"name"`
	// 规则描述
	Description string `json:"description"`
	// 规则类型
	Type string `json:"type"`
	// 阈值
	Threshold float64 `json:"threshold"`
	// 持续时间（秒）
	Duration int `json:"duration"`
	// 是否启用
	Enabled bool `json:"enabled"`
	// 通知方式
	NotifyMethods []string `json:"notify_methods"`
	// 通知目标
	NotifyTargets []string `json:"notify_targets"`
	// 创建时间
	CreatedAt string `json:"created_at"`
	// 更新时间
	UpdatedAt string `json:"updated_at"`
}

// GetConcurrency 获取并发统计
// GET /api/v1/admin/ops/concurrency
func (h *OpsHandler) GetConcurrency(c *gin.Context) {
	ctx := c.Request.Context()

	// 获取所有活跃账号
	accounts, err := h.accountRepo.ListActive(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取账号列表失败",
			},
		})
		return
	}

	// 统计各平台并发
	platformStats := make(map[string]*PlatformConcurrency)
	var totalConcurrency, maxConcurrency int64

	accountDetails := make([]AccountConcurrency, 0, len(accounts))
	for _, acc := range accounts {
		// 计算总并发
		totalConcurrency += int64(acc.CurrentConcurrency)
		maxConcurrency += int64(acc.MaxConcurrency)

		// 计算使用率
		var usageRate float64
		if acc.MaxConcurrency > 0 {
			usageRate = float64(acc.CurrentConcurrency) / float64(acc.MaxConcurrency) * 100
		}

		// 账号详情
		accountDetails = append(accountDetails, AccountConcurrency{
			ID:                 acc.ID,
			Name:               acc.Name,
			Platform:           string(acc.Platform),
			Status:             string(acc.Status),
			CurrentConcurrency: acc.CurrentConcurrency,
			MaxConcurrency:     acc.MaxConcurrency,
			UsageRate:          usageRate,
		})

		// 平台统计
		platform := string(acc.Platform)
		if ps, ok := platformStats[platform]; ok {
			ps.CurrentConcurrency += int64(acc.CurrentConcurrency)
			ps.MaxConcurrency += int64(acc.MaxConcurrency)
			ps.AccountCount++
		} else {
			platformStats[platform] = &PlatformConcurrency{
				Platform:           platform,
				CurrentConcurrency: int64(acc.CurrentConcurrency),
				MaxConcurrency:     int64(acc.MaxConcurrency),
				AccountCount:       1,
			}
		}
	}

	// 计算总使用率
	var concurrencyRate float64
	if maxConcurrency > 0 {
		concurrencyRate = float64(totalConcurrency) / float64(maxConcurrency) * 100
	}

	// 转换平台统计为切片
	byPlatform := make([]PlatformConcurrency, 0, len(platformStats))
	for _, ps := range platformStats {
		byPlatform = append(byPlatform, *ps)
	}

	c.JSON(http.StatusOK, ConcurrencyStatsResponse{
		Timestamp:         time.Now().Format("2006-01-02 15:04:05"),
		TotalConcurrency:  totalConcurrency,
		MaxConcurrency:    maxConcurrency,
		ConcurrencyRate:   concurrencyRate,
		ByPlatform:        byPlatform,
		Accounts:          accountDetails,
	})
}

// GetRealtimeTraffic 获取实时流量
// GET /api/v1/admin/ops/realtime-traffic
func (h *OpsHandler) GetRealtimeTraffic(c *gin.Context) {
	ctx := c.Request.Context()
	now := time.Now()

	// 获取各时间段的流量统计
	last1Min := h.getTrafficStats(ctx, now.Add(-1*time.Minute), now)
	last5Min := h.getTrafficStats(ctx, now.Add(-5*time.Minute), now)
	last15Min := h.getTrafficStats(ctx, now.Add(-15*time.Minute), now)
	last1Hour := h.getTrafficStats(ctx, now.Add(-1*time.Hour), now)

	// 获取趋势数据（最近15分钟，每分钟一个点）
	trend := make([]TrafficPoint, 0, 15)
	for i := 14; i >= 0; i-- {
		endTime := now.Add(-time.Duration(i) * time.Minute)
		startTime := endTime.Add(-1 * time.Minute)
		stats := h.getTrafficStats(ctx, startTime, endTime)
		trend = append(trend, TrafficPoint{
			Timestamp:    startTime.Format("2006-01-02 15:04:05"),
			Requests:     stats.Requests,
			AvgLatencyMs: stats.AvgLatencyMs,
			ErrorRate:    stats.ErrorRate,
		})
	}

	c.JSON(http.StatusOK, RealtimeTrafficResponse{
		Timestamp:  now.Format("2006-01-02 15:04:05"),
		Last1Min:   last1Min,
		Last5Min:   last5Min,
		Last15Min:  last15Min,
		Last1Hour:  last1Hour,
		Trend:      trend,
	})
}

// getTrafficStats 获取指定时间段的流量统计
func (h *OpsHandler) getTrafficStats(ctx context.Context, startTime, endTime time.Time) TrafficStats {
	stats, err := h.usageRepo.GetStatsByTimeRange(ctx, startTime, endTime)
	if err != nil {
		return TrafficStats{}
	}

	// 计算QPS
	duration := endTime.Sub(startTime).Seconds()
	var qps float64
	if duration > 0 {
		qps = float64(stats.TotalRequests) / duration
	}

	// 计算错误率
	var errorRate float64
	if stats.TotalRequests > 0 {
		errorRate = float64(stats.FailedRequests+stats.TimeoutRequests) / float64(stats.TotalRequests) * 100
	}

	return TrafficStats{
		Requests:       stats.TotalRequests,
		Success:        stats.SuccessRequests,
		Failed:         stats.FailedRequests,
		Timeout:        stats.TimeoutRequests,
		AvgLatencyMs:   stats.AvgLatencyMs,
		AvgFirstTokenMs: stats.AvgFirstTokenMs,
		TotalTokens:    stats.TotalTokens,
		TotalCost:      stats.TotalCost,
		QPS:            qps,
		ErrorRate:      errorRate,
	}
}

// GetErrors 获取错误日志
// GET /api/v1/admin/ops/errors
func (h *OpsHandler) GetErrors(c *gin.Context) {
	var req ErrorLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "请求参数无效: " + err.Error(),
			},
		})
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// 解析时间
	var startTime, endTime *time.Time
	if req.StartTime != "" {
		t, err := time.Parse("2006-01-02 15:04:05", req.StartTime)
		if err == nil {
			startTime = &t
		}
	}
	if req.EndTime != "" {
		t, err := time.Parse("2006-01-02 15:04:05", req.EndTime)
		if err == nil {
			endTime = &t
		}
	}

	// 构建查询条件
	filter := repository.ErrorLogFilter{
		Platform:  req.Platform,
		AccountID: req.AccountID,
		StartTime: startTime,
		EndTime:   endTime,
	}

	// 查询错误日志
	logs, total, err := h.usageRepo.ListErrors(c.Request.Context(), filter, req.Page, req.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "查询错误日志失败",
			},
		})
		return
	}

	// 转换为响应格式
	list := make([]*ErrorLogInfo, 0, len(logs))
	for _, log := range logs {
		list = append(list, &ErrorLogInfo{
			RequestID:     log.RequestID,
			UserID:        log.UserID,
			AccountID:     log.AccountID,
			GroupID:       log.GroupID,
			Model:         log.Model,
			Platform:      log.Platform,
			Status:        string(log.Status),
			ErrorMessage:  log.ErrorMessage,
			ClientIP:      log.ClientIP,
			CreatedAt:     log.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, ErrorLogListResponse{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

// GetAlertRules 获取告警规则
// GET /api/v1/admin/ops/alert-rules
func (h *OpsHandler) GetAlertRules(c *gin.Context) {
	// TODO: 从数据库或配置中读取告警规则
	// 目前返回默认规则
	rules := []AlertRuleInfo{
		{
			ID:            "high_error_rate",
			Name:          "高错误率告警",
			Description:   "当错误率超过阈值时触发告警",
			Type:          "error_rate",
			Threshold:     5.0,
			Duration:      60,
			Enabled:       true,
			NotifyMethods: []string{"email", "webhook"},
			NotifyTargets: []string{"admin@example.com"},
			CreatedAt:     time.Now().Format("2006-01-02 15:04:05"),
			UpdatedAt:     time.Now().Format("2006-01-02 15:04:05"),
		},
		{
			ID:            "high_latency",
			Name:          "高延迟告警",
			Description:   "当平均延迟超过阈值时触发告警",
			Type:          "latency",
			Threshold:     3000.0,
			Duration:      60,
			Enabled:       true,
			NotifyMethods: []string{"email"},
			NotifyTargets: []string{"admin@example.com"},
			CreatedAt:     time.Now().Format("2006-01-02 15:04:05"),
			UpdatedAt:     time.Now().Format("2006-01-02 15:04:05"),
		},
		{
			ID:            "account_error",
			Name:          "账号错误告警",
			Description:   "当账号连续出错时触发告警",
			Type:          "account_error",
			Threshold:     10.0,
			Duration:      300,
			Enabled:       true,
			NotifyMethods: []string{"email", "webhook"},
			NotifyTargets: []string{"admin@example.com"},
			CreatedAt:     time.Now().Format("2006-01-02 15:04:05"),
			UpdatedAt:     time.Now().Format("2006-01-02 15:04:05"),
		},
		{
			ID:            "low_balance",
			Name:          "余额不足告警",
			Description:   "当用户余额低于阈值时触发告警",
			Type:          "balance",
			Threshold:     10.0,
			Duration:      0,
			Enabled:       true,
			NotifyMethods: []string{"email"},
			NotifyTargets: []string{"user@example.com"},
			CreatedAt:     time.Now().Format("2006-01-02 15:04:05"),
			UpdatedAt:     time.Now().Format("2006-01-02 15:04:05"),
		},
	}

	c.JSON(http.StatusOK, AlertRuleResponse{
		Rules: rules,
	})
}
