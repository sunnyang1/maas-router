// Package handler 提供 MaaS-Router 的 HTTP 处理器
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"maas-router/internal/pkg/ctxkey"
)

// UsageHandler 使用记录 Handler
// 处理使用记录查询、统计、仪表盘数据等操作
type UsageHandler struct {
	// UsageService 使用记录服务
	UsageService UsageService
}

// NewUsageHandler 创建使用记录 Handler
func NewUsageHandler(usageService UsageService) *UsageHandler {
	return &UsageHandler{
		UsageService: usageService,
	}
}

// UsageRecordInfo 使用记录信息
type UsageRecordInfo struct {
	ID               string    `json:"id"`
	RequestID        string    `json:"request_id"`
	UserID           string    `json:"user_id"`
	APIKeyID         string    `json:"api_key_id,omitempty"`
	AccountID        string    `json:"account_id,omitempty"`
	Model            string    `json:"model"`
	Platform         string    `json:"platform"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	LatencyMs        int32     `json:"latency_ms"`
	FirstTokenMs     *int32    `json:"first_token_ms,omitempty"`
	Cost             float64   `json:"cost"`
	Status           string    `json:"status"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	ClientIP         string    `json:"client_ip,omitempty"`
	UserAgent        string    `json:"user_agent,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// ListUsageRequest 列表请求
type ListUsageRequest struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	StartDate string `form:"start_date" binding:"omitempty"`
	EndDate   string `form:"end_date" binding:"omitempty"`
	Model     string `form:"model" binding:"omitempty"`
	Platform  string `form:"platform" binding:"omitempty"`
	Status    string `form:"status" binding:"omitempty,oneof=success failed timeout"`
	APIKeyID  string `form:"api_key_id" binding:"omitempty"`
}

// ListUsageResponse 列表响应
type ListUsageResponse struct {
	Data       []UsageRecordInfo `json:"data"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

// UsageStats 使用统计
type UsageStats struct {
	TotalRequests      int64   `json:"total_requests"`
	SuccessRequests    int64   `json:"success_requests"`
	FailedRequests     int64   `json:"failed_requests"`
	TotalTokens        int64   `json:"total_tokens"`
	PromptTokens       int64   `json:"prompt_tokens"`
	CompletionTokens   int64   `json:"completion_tokens"`
	TotalCost          float64 `json:"total_cost"`
	AverageLatencyMs   int64   `json:"average_latency_ms"`
	AverageFirstTokenMs int64  `json:"average_first_token_ms"`
	RequestsByModel    map[string]int64   `json:"requests_by_model"`
	RequestsByPlatform map[string]int64   `json:"requests_by_platform"`
	TokensByModel      map[string]int64   `json:"tokens_by_model"`
	CostByModel        map[string]float64 `json:"cost_by_model"`
}

// UsageStatsRequest 统计请求
type UsageStatsRequest struct {
	StartDate string `form:"start_date" binding:"required"`
	EndDate   string `form:"end_date" binding:"required"`
	GroupBy   string `form:"group_by" binding:"omitempty,oneof=day week month model platform"`
}

// DashboardData 仪表盘数据
type DashboardData struct {
	// 今日统计
	Today struct {
		Requests int64   `json:"requests"`
		Tokens   int64   `json:"tokens"`
		Cost     float64 `json:"cost"`
	} `json:"today"`
	// 本月统计
	Month struct {
		Requests int64   `json:"requests"`
		Tokens   int64   `json:"tokens"`
		Cost     float64 `json:"cost"`
	} `json:"month"`
	// 趋势数据（最近 7 天）
	Trend []DailyUsage `json:"trend"`
	// 模型使用排行
	TopModels []ModelUsage `json:"top_models"`
	// 平台使用分布
	PlatformDistribution []PlatformUsage `json:"platform_distribution"`
	// 账户余额
	Balance float64 `json:"balance"`
}

// DailyUsage 每日使用量
type DailyUsage struct {
	Date     string  `json:"date"`
	Requests int64   `json:"requests"`
	Tokens   int64   `json:"tokens"`
	Cost     float64 `json:"cost"`
}

// ModelUsage 模型使用量
type ModelUsage struct {
	Model    string  `json:"model"`
	Requests int64   `json:"requests"`
	Tokens   int64   `json:"tokens"`
	Cost     float64 `json:"cost"`
}

// PlatformUsage 平台使用量
type PlatformUsage struct {
	Platform string  `json:"platform"`
	Requests int64   `json:"requests"`
	Tokens   int64   `json:"tokens"`
	Cost     float64 `json:"cost"`
}

// ListUsage 处理 GET /api/v1/usage
// 获取使用记录列表
func (h *UsageHandler) ListUsage(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 解析请求参数
	var req ListUsageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// 获取使用记录列表
	records, total, err := h.UsageService.ListByUser(c.Request.Context(), userID.(string), &ListUsageParams{
		Page:      req.Page,
		PageSize:  req.PageSize,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Model:     req.Model,
		Platform:  req.Platform,
		Status:    req.Status,
		APIKeyID:  req.APIKeyID,
	})
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取使用记录失败")
		return
	}

	// 转换为响应格式
	data := make([]UsageRecordInfo, 0, len(records))
	for _, record := range records {
		data = append(data, *convertToUsageRecordInfo(record))
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, ListUsageResponse{
		Data:       data,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	})
}

// GetUsageDetail 处理 GET /api/v1/usage/:id
// 获取指定使用记录详情
func (h *UsageHandler) GetUsageDetail(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 获取记录 ID
	recordID := c.Param("id")
	if recordID == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_ID", "缺少记录 ID")
		return
	}

	// 获取使用记录
	record, err := h.UsageService.GetByID(c.Request.Context(), recordID)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", "使用记录不存在")
		return
	}

	// 验证所有权
	if record.UserID != userID.(string) {
		ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "无权访问此记录")
		return
	}

	c.JSON(http.StatusOK, convertToUsageRecordInfo(record))
}

// GetUsageStats 处理 GET /api/v1/usage/stats
// 获取使用统计
func (h *UsageHandler) GetUsageStats(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 解析请求参数
	var req UsageStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	// 获取统计数据
	stats, err := h.UsageService.GetStats(c.Request.Context(), userID.(string), &GetStatsParams{
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		GroupBy:   req.GroupBy,
	})
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取统计数据失败")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetDashboard 处理 GET /api/v1/usage/dashboard
// 获取仪表盘数据
func (h *UsageHandler) GetDashboard(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 获取仪表盘数据
	dashboard, err := h.UsageService.GetDashboard(c.Request.Context(), userID.(string))
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取仪表盘数据失败")
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// ExportUsage 导出使用记录
func (h *UsageHandler) ExportUsage(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 解析请求参数
	var req ListUsageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	// 获取导出数据
	data, err := h.UsageService.Export(c.Request.Context(), userID.(string), &ListUsageParams{
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Model:     req.Model,
		Platform:  req.Platform,
		Status:    req.Status,
		APIKeyID:  req.APIKeyID,
	})
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "导出数据失败")
		return
	}

	// 设置响应头
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=usage_export.csv")

	c.Data(http.StatusOK, "text/csv", data)
}

// RegisterUsageHandlers 注册使用记录路由到 HandlerGroup
func RegisterUsageHandlers(h *UsageHandler) map[string]gin.HandlerFunc {
	return map[string]gin.HandlerFunc{
		"ListUsage":     h.ListUsage,
		"GetUsageStats": h.GetUsageStats,
		"GetDashboard":  h.GetDashboard,
	}
}

// ===== 辅助函数 =====

func convertToUsageRecordInfo(record *UsageRecordData) *UsageRecordInfo {
	return &UsageRecordInfo{
		ID:               record.ID,
		RequestID:        record.RequestID,
		UserID:           record.UserID,
		APIKeyID:         record.APIKeyID,
		AccountID:        record.AccountID,
		Model:            record.Model,
		Platform:         record.Platform,
		PromptTokens:     record.PromptTokens,
		CompletionTokens: record.CompletionTokens,
		TotalTokens:      record.TotalTokens,
		LatencyMs:        record.LatencyMs,
		FirstTokenMs:     record.FirstTokenMs,
		Cost:             record.Cost,
		Status:           record.Status,
		ErrorMessage:     record.ErrorMessage,
		ClientIP:         record.ClientIP,
		UserAgent:        record.UserAgent,
		CreatedAt:        record.CreatedAt,
	}
}

// ===== 服务接口定义 =====

// UsageRecordData 使用记录数据
type UsageRecordData struct {
	ID               string
	RequestID        string
	UserID           string
	APIKeyID         string
	AccountID        string
	Model            string
	Platform         string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	LatencyMs        int32
	FirstTokenMs     *int32
	Cost             float64
	Status           string
	ErrorMessage     string
	ClientIP         string
	UserAgent        string
	CreatedAt        time.Time
}

// UsageService 使用记录服务接口
type UsageService interface {
	// ListByUser 获取用户的使用记录列表
	ListByUser(ctx interface{}, userID string, params *ListUsageParams) ([]*UsageRecordData, int64, error)
	// GetByID 根据 ID 获取使用记录
	GetByID(ctx interface{}, id string) (*UsageRecordData, error)
	// GetStats 获取使用统计
	GetStats(ctx interface{}, userID string, params *GetStatsParams) (*UsageStats, error)
	// GetDashboard 获取仪表盘数据
	GetDashboard(ctx interface{}, userID string) (*DashboardData, error)
	// Export 导出使用记录
	Export(ctx interface{}, userID string, params *ListUsageParams) ([]byte, error)
}

// ListUsageParams 列表参数
type ListUsageParams struct {
	Page      int
	PageSize  int
	StartDate string
	EndDate   string
	Model     string
	Platform  string
	Status    string
	APIKeyID  string
}

// GetStatsParams 统计参数
type GetStatsParams struct {
	StartDate string
	EndDate   string
	GroupBy   string
}
