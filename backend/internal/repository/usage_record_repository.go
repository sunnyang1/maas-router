// Package repository 提供数据访问层实现
package repository

import (
	"context"
	"errors"
	"time"

	"entgo.io/ent/dialect/sql"

	"maas-router/ent"
	"maas-router/ent/usagerecord"
)

// UsageRecordRepository 使用记录数据访问仓库
type UsageRecordRepository struct {
	client *ent.Client
}

// NewUsageRecordRepository 创建使用记录仓库实例
func NewUsageRecordRepository(client *ent.Client) *UsageRecordRepository {
	return &UsageRecordRepository{client: client}
}

// UsageRecord 使用记录实体
type UsageRecord struct {
	ID              int64
	RequestID       string
	UserID          int64
	APIKeyID        *int64
	AccountID       *int64
	GroupID         *int64
	Model           string
	Platform        string
	PromptTokens    int32
	CompletionTokens int32
	TotalTokens     int32
	LatencyMs       *int32
	FirstTokenMs    *int32
	Cost            float64
	Status          string
	ErrorMessage     string
	ClientIP        string
	UserAgent       string
	CreatedAt       time.Time
}

// UsageRecordListFilter 使用记录列表筛选条件
type UsageRecordListFilter struct {
	UserID    int64
	APIKeyID  int64
	AccountID int64
	GroupID   int64
	Platform  string
	Model     string
	Status    string
	StartTime *time.Time
	EndTime   *time.Time
	SortBy    string
	SortOrder string
}

// UsageStats 使用统计
type UsageStats struct {
	TotalRequests     int64
	SuccessRequests   int64
	FailedRequests    int64
	TimeoutRequests   int64
	TotalTokens       int64
	TotalCost         float64
	AvgLatencyMs      float64
	AvgFirstTokenMs   float64
}

// UserUsageStats 用户使用统计
type UserUsageStats struct {
	TodayRequests int64
	TodayCost     float64
	MonthRequests int64
	MonthCost     float64
	TotalRequests int64
	TotalCost     float64
}

// ErrorLogFilter 错误日志筛选条件
type ErrorLogFilter struct {
	Platform  string
	AccountID int64
	StartTime *time.Time
	EndTime   *time.Time
}

// CreateUsageRecordInput 创建使用记录输入
type CreateUsageRecordInput struct {
	RequestID        string
	UserID           int64
	APIKeyID         *int64
	AccountID        *int64
	GroupID          *int64
	Model            string
	Platform         string
	PromptTokens     int32
	CompletionTokens int32
	TotalTokens      int32
	LatencyMs        *int32
	FirstTokenMs     *int32
	Cost             float64
	Status           string
	ErrorMessage      string
	ClientIP         string
	UserAgent        string
}

// Create 创建使用记录
func (r *UsageRecordRepository) Create(ctx context.Context, input *CreateUsageRecordInput) (*UsageRecord, error) {
	create := r.client.UsageRecord.Create().
		SetRequestID(input.RequestID).
		SetUserID(input.UserID).
		SetModel(input.Model).
		SetPlatform(input.Platform).
		SetPromptTokens(input.PromptTokens).
		SetCompletionTokens(input.CompletionTokens).
		SetTotalTokens(input.TotalTokens).
		SetCost(input.Cost).
		SetStatus(usagerecord.Status(input.Status))

	if input.APIKeyID != nil {
		create.SetAPIKeyID(*input.APIKeyID)
	}
	if input.AccountID != nil {
		create.SetAccountID(*input.AccountID)
	}
	if input.GroupID != nil {
		create.SetGroupID(*input.GroupID)
	}
	if input.LatencyMs != nil {
		create.SetLatencyMs(*input.LatencyMs)
	}
	if input.FirstTokenMs != nil {
		create.SetFirstTokenMs(*input.FirstTokenMs)
	}
	if input.ErrorMessage != "" {
		create.SetErrorMessage(input.ErrorMessage)
	}
	if input.ClientIP != "" {
		create.SetClientIP(input.ClientIP)
	}
	if input.UserAgent != "" {
		create.SetUserAgent(input.UserAgent)
	}

	record, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToUsageRecord(record), nil
}

// Get 根据ID获取使用记录
func (r *UsageRecordRepository) Get(ctx context.Context, id int64) (*UsageRecord, error) {
	record, err := r.client.UsageRecord.Query().
		Where(usagerecord.ID(id)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("使用记录不存在")
		}
		return nil, err
	}

	return r.convertToUsageRecord(record), nil
}

// GetByRequestID 根据请求ID获取使用记录
func (r *UsageRecordRepository) GetByRequestID(ctx context.Context, requestID string) (*UsageRecord, error) {
	record, err := r.client.UsageRecord.Query().
		Where(usagerecord.RequestID(requestID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("使用记录不存在")
		}
		return nil, err
	}

	return r.convertToUsageRecord(record), nil
}

// List 获取使用记录列表
func (r *UsageRecordRepository) List(ctx context.Context, filter UsageRecordListFilter, page, pageSize int) ([]*UsageRecord, int64, error) {
	query := r.client.UsageRecord.Query()

	// 应用筛选条件
	if filter.UserID > 0 {
		query.Where(usagerecord.UserID(filter.UserID))
	}
	if filter.APIKeyID > 0 {
		query.Where(usagerecord.APIKeyID(filter.APIKeyID))
	}
	if filter.AccountID > 0 {
		query.Where(usagerecord.AccountID(filter.AccountID))
	}
	if filter.GroupID > 0 {
		query.Where(usagerecord.GroupID(filter.GroupID))
	}
	if filter.Platform != "" {
		query.Where(usagerecord.Platform(filter.Platform))
	}
	if filter.Model != "" {
		query.Where(usagerecord.Model(filter.Model))
	}
	if filter.Status != "" {
		query.Where(usagerecord.Status(usagerecord.Status(filter.Status)))
	}
	if filter.StartTime != nil {
		query.Where(usagerecord.CreatedAtGTE(*filter.StartTime))
	}
	if filter.EndTime != nil {
		query.Where(usagerecord.CreatedAtLTE(*filter.EndTime))
	}

	// 应用排序
	orderBy := usagerecord.FieldCreatedAt
	if filter.SortBy != "" {
		orderBy = filter.SortBy
	}
	orderDirection := sql.OrderDesc()
	if filter.SortOrder == "asc" {
		orderDirection = sql.OrderAsc()
	}
	query.Order(ent.OrderByFunc(func(s *sql.Selector) {
		s.OrderBy(sql.OrderExpr(orderDirection, orderBy))
	}))

	// 获取总数
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 分页
	offset := (page - 1) * pageSize
	records, err := query.Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 转换
	result := make([]*UsageRecord, 0, len(records))
	for _, record := range records {
		result = append(result, r.convertToUsageRecord(record))
	}

	return result, int64(total), nil
}

// GetStats 获取使用统计
func (r *UsageRecordRepository) GetStats(ctx context.Context, filter UsageRecordListFilter) (*UsageStats, error) {
	query := r.client.UsageRecord.Query()

	// 应用筛选条件
	if filter.UserID > 0 {
		query.Where(usagerecord.UserID(filter.UserID))
	}
	if filter.Platform != "" {
		query.Where(usagerecord.Platform(filter.Platform))
	}
	if filter.StartTime != nil {
		query.Where(usagerecord.CreatedAtGTE(*filter.StartTime))
	}
	if filter.EndTime != nil {
		query.Where(usagerecord.CreatedAtLTE(*filter.EndTime))
	}

	var stats UsageStats

	// 总请求数
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalRequests = int64(total)

	// 成功请求数
	success, err := query.Clone().Where(usagerecord.Status(usagerecord.StatusSuccess)).Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.SuccessRequests = int64(success)

	// 失败请求数
	failed, err := query.Clone().Where(usagerecord.Status(usagerecord.StatusFailed)).Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.FailedRequests = int64(failed)

	// 超时请求数
	timeout, err := query.Clone().Where(usagerecord.Status(usagerecord.StatusTimeout)).Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.TimeoutRequests = int64(timeout)

	// 聚合统计
	type aggResult struct {
		TotalTokens    int64   `json:"total_tokens"`
		TotalCost      float64 `json:"total_cost"`
		AvgLatencyMs   float64 `json:"avg_latency_ms"`
		AvgFirstTokenMs float64 `json:"avg_first_token_ms"`
	}

	var agg aggResult
	err = query.Clone().
		Modify(func(s *sql.Selector) {
			s.Select(
				sql.As(sql.Sum(sql.Expr("total_tokens")), "total_tokens"),
				sql.As(sql.Sum(sql.Expr("cost")), "total_cost"),
				sql.As(sql.Avg(sql.Expr("latency_ms")), "avg_latency_ms"),
				sql.As(sql.Avg(sql.Expr("first_token_ms")), "avg_first_token_ms"),
			)
		}).
		Scan(ctx, &agg)
	if err != nil {
		return nil, err
	}

	stats.TotalTokens = agg.TotalTokens
	stats.TotalCost = agg.TotalCost
	stats.AvgLatencyMs = agg.AvgLatencyMs
	stats.AvgFirstTokenMs = agg.AvgFirstTokenMs

	return &stats, nil
}

// GetStatsByTimeRange 获取指定时间范围的统计
func (r *UsageRecordRepository) GetStatsByTimeRange(ctx context.Context, startTime, endTime time.Time) (*UsageStats, error) {
	filter := UsageRecordListFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
	}
	return r.GetStats(ctx, filter)
}

// GetUserStats 获取用户使用统计
func (r *UsageRecordRepository) GetUserStats(ctx context.Context, userID int64) (*UserUsageStats, error) {
	now := time.Now()

	// 今日统计
	todayStart := now.Truncate(24 * time.Hour)
	todayStats, err := r.GetStatsByTimeRange(ctx, todayStart, now)
	if err != nil {
		return nil, err
	}

	// 本月统计
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthStats, err := r.GetStatsByTimeRange(ctx, monthStart, now)
	if err != nil {
		return nil, err
	}

	// 总统计
	filter := UsageRecordListFilter{UserID: userID}
	totalStats, err := r.GetStats(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &UserUsageStats{
		TodayRequests: todayStats.TotalRequests,
		TodayCost:     todayStats.TotalCost,
		MonthRequests: monthStats.TotalRequests,
		MonthCost:     monthStats.TotalCost,
		TotalRequests: totalStats.TotalRequests,
		TotalCost:     totalStats.TotalCost,
	}, nil
}

// GetDashboard 获取仪表盘数据
func (r *UsageRecordRepository) GetDashboard(ctx context.Context, userID int64) (map[string]interface{}, error) {
	stats, err := r.GetUserStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"today_requests": stats.TodayRequests,
		"today_cost":     stats.TodayCost,
		"month_requests": stats.MonthRequests,
		"month_cost":     stats.MonthCost,
		"total_requests": stats.TotalRequests,
		"total_cost":     stats.TotalCost,
	}, nil
}

// CountByTimeRange 统计指定时间范围的请求数
func (r *UsageRecordRepository) CountByTimeRange(ctx context.Context, startTime, endTime time.Time) (int64, error) {
	count, err := r.client.UsageRecord.Query().
		Where(
			usagerecord.CreatedAtGTE(startTime),
			usagerecord.CreatedAtLTE(endTime),
		).
		Count(ctx)
	return int64(count), err
}

// CountByAccountAndTimeRange 统计指定账号和时间范围的请求数
func (r *UsageRecordRepository) CountByAccountAndTimeRange(ctx context.Context, accountID int64, startTime, endTime time.Time) (int64, error) {
	count, err := r.client.UsageRecord.Query().
		Where(
			usagerecord.AccountID(accountID),
			usagerecord.CreatedAtGTE(startTime),
			usagerecord.CreatedAtLTE(endTime),
		).
		Count(ctx)
	return int64(count), err
}

// ListErrors 获取错误日志列表
func (r *UsageRecordRepository) ListErrors(ctx context.Context, filter ErrorLogFilter, page, pageSize int) ([]*UsageRecord, int64, error) {
	query := r.client.UsageRecord.Query().
		Where(usagerecord.StatusNEQ(usagerecord.StatusSuccess))

	// 应用筛选条件
	if filter.Platform != "" {
		query.Where(usagerecord.Platform(filter.Platform))
	}
	if filter.AccountID > 0 {
		query.Where(usagerecord.AccountID(filter.AccountID))
	}
	if filter.StartTime != nil {
		query.Where(usagerecord.CreatedAtGTE(*filter.StartTime))
	}
	if filter.EndTime != nil {
		query.Where(usagerecord.CreatedAtLTE(*filter.EndTime))
	}

	// 按时间倒序
	query.Order(ent.Desc(usagerecord.FieldCreatedAt))

	// 获取总数
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 分页
	offset := (page - 1) * pageSize
	records, err := query.Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 转换
	result := make([]*UsageRecord, 0, len(records))
	for _, record := range records {
		result = append(result, r.convertToUsageRecord(record))
	}

	return result, int64(total), nil
}

// convertToUsageRecord 转换Ent实体到领域实体
func (r *UsageRecordRepository) convertToUsageRecord(record *ent.UsageRecord) *UsageRecord {
	return &UsageRecord{
		ID:               record.ID,
		RequestID:        record.RequestID,
		UserID:           record.UserID,
		APIKeyID:         record.APIKeyID,
		AccountID:        record.AccountID,
		GroupID:          record.GroupID,
		Model:            record.Model,
		Platform:         record.Platform,
		PromptTokens:     record.PromptTokens,
		CompletionTokens: record.CompletionTokens,
		TotalTokens:      record.TotalTokens,
		LatencyMs:        record.LatencyMs,
		FirstTokenMs:     record.FirstTokenMs,
		Cost:             record.Cost,
		Status:           string(record.Status),
		ErrorMessage:      record.ErrorMessage,
		ClientIP:         record.ClientIP,
		UserAgent:        record.UserAgent,
		CreatedAt:        record.CreatedAt,
	}
}
