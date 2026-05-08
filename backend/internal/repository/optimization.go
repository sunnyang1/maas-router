// Package repository 提供数据库查询优化功能
// 包括批量插入、分页查询、统计查询等优化实现
package repository

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"go.uber.org/zap"

	"maas-router/ent"
)

// BatchInserter 批量插入优化器
type BatchInserter struct {
	client *ent.Client
	logger *zap.Logger
	// 批处理大小
	batchSize int
}

// NewBatchInserter 创建批量插入优化器
func NewBatchInserter(client *ent.Client, logger *zap.Logger) *BatchInserter {
	return &BatchInserter{
		client:    client,
		logger:    logger,
		batchSize: 100, // 默认批处理大小
	}
}

// SetBatchSize 设置批处理大小
func (b *BatchInserter) SetBatchSize(size int) {
	if size > 0 {
		b.batchSize = size
	}
}

// BatchInsertUsageRecords 批量插入使用记录
// 使用事务和批处理优化插入性能
func (b *BatchInserter) BatchInsertUsageRecords(ctx context.Context, records []*ent.UsageRecordCreate) error {
	if len(records) == 0 {
		return nil
	}

	// 开启事务
	tx, err := b.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	// 分批处理
	for i := 0; i < len(records); i += b.batchSize {
		end := i + b.batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]
		if err := b.insertUsageRecordBatch(ctx, tx, batch); err != nil {
			return fmt.Errorf("批量插入使用记录失败: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	b.logger.Debug("批量插入使用记录完成",
		zap.Int("count", len(records)))

	return nil
}

// insertUsageRecordBatch 插入一批使用记录
func (b *BatchInserter) insertUsageRecordBatch(ctx context.Context, tx *ent.Tx, batch []*ent.UsageRecordCreate) error {
	// 使用 Builder 模式批量插入
	builders := make([]*ent.UsageRecordCreate, len(batch))
	copy(builders, batch)

	// 执行批量创建
	_, err := tx.UsageRecord.CreateBulk(builders...).Save(ctx)
	if err != nil {
		return err
	}

	return nil
}

// PaginatedQuery 分页查询优化器
type PaginatedQuery struct {
	client *ent.Client
	logger *zap.Logger
}

// NewPaginatedQuery 创建分页查询优化器
func NewPaginatedQuery(client *ent.Client, logger *zap.Logger) *PaginatedQuery {
	return &PaginatedQuery{
		client: client,
		logger: logger,
	}
}

// PaginationParams 分页参数
type PaginationParams struct {
	Page     int
	PageSize int
	SortBy   string
	SortDesc bool
}

// PaginationResult 分页结果
type PaginationResult struct {
	Total       int64
	Page        int
	PageSize    int
	TotalPages  int
	HasNext     bool
	HasPrevious bool
}

// Validate 验证分页参数
func (p *PaginationParams) Validate() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100 // 限制最大页大小
	}
}

// Offset 计算偏移量
func (p *PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// BuildPaginationResult 构建分页结果
func BuildPaginationResult(total int64, params PaginationParams) PaginationResult {
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return PaginationResult{
		Total:       total,
		Page:        params.Page,
		PageSize:    params.PageSize,
		TotalPages:  totalPages,
		HasNext:     params.Page < totalPages,
		HasPrevious: params.Page > 1,
	}
}

// QueryUsersPaginated 优化的用户分页查询
func (p *PaginatedQuery) QueryUsersPaginated(ctx context.Context, params PaginationParams) ([]*ent.User, PaginationResult, error) {
	params.Validate()

	// 构建查询
	query := p.client.User.Query()

	// 获取总数（使用 COUNT(*) 优化）
	total, err := query.Count(ctx)
	if err != nil {
		return nil, PaginationResult{}, fmt.Errorf("查询用户总数失败: %w", err)
	}

	// 应用排序
	if params.SortBy != "" {
		if params.SortDesc {
			query = query.Order(ent.Desc(params.SortBy))
		} else {
			query = query.Order(ent.Asc(params.SortBy))
		}
	} else {
		// 默认按创建时间倒序
		query = query.Order(ent.Desc("created_at"))
	}

	// 应用分页
	users, err := query.
		Offset(params.Offset()).
		Limit(params.PageSize).
		All(ctx)
	if err != nil {
		return nil, PaginationResult{}, fmt.Errorf("查询用户列表失败: %w", err)
	}

	result := BuildPaginationResult(int64(total), params)
	return users, result, nil
}

// QueryUsageRecordsPaginated 优化的使用记录分页查询
func (p *PaginatedQuery) QueryUsageRecordsPaginated(
	ctx context.Context,
	userID int64,
	startTime, endTime time.Time,
	params PaginationParams,
) ([]*ent.UsageRecord, PaginationResult, error) {
	params.Validate()

	// 构建查询
	query := p.client.UsageRecord.Query().
		Where(
			ent.UsageRecordUserID(userID),
			ent.UsageRecordCreatedAtGTE(startTime),
			ent.UsageRecordCreatedAtLTE(endTime),
		)

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, PaginationResult{}, fmt.Errorf("查询使用记录总数失败: %w", err)
	}

	// 应用排序和分页
	records, err := query.
		Order(ent.Desc("created_at")).
		Offset(params.Offset()).
		Limit(params.PageSize).
		All(ctx)
	if err != nil {
		return nil, PaginationResult{}, fmt.Errorf("查询使用记录列表失败: %w", err)
	}

	result := BuildPaginationResult(int64(total), params)
	return records, result, nil
}

// CursorBasedQuery 游标分页查询器
// 适用于大数据量的分页场景
type CursorBasedQuery struct {
	client *ent.Client
	logger *zap.Logger
}

// NewCursorBasedQuery 创建游标分页查询器
func NewCursorBasedQuery(client *ent.Client, logger *zap.Logger) *CursorBasedQuery {
	return &CursorBasedQuery{
		client: client,
		logger: logger,
	}
}

// CursorParams 游标分页参数
type CursorParams struct {
	Cursor   string // 游标（上一页最后一条记录的 ID）
	Limit    int    // 每页数量
	SortDesc bool   // 是否倒序
}

// CursorResult 游标分页结果
type CursorResult struct {
	Items      interface{} `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
	HasMore    bool        `json:"has_more"`
}

// QueryUsageRecordsByCursor 游标分页查询使用记录
func (c *CursorBasedQuery) QueryUsageRecordsByCursor(
	ctx context.Context,
	userID int64,
	params CursorParams,
) (*CursorResult, error) {
	if params.Limit < 1 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	// 构建查询
	query := c.client.UsageRecord.Query().
		Where(ent.UsageRecordUserID(userID))

	// 应用游标过滤
	if params.Cursor != "" {
		// 解析游标（假设游标是记录 ID）
		var cursorID int64
		if _, err := fmt.Sscanf(params.Cursor, "%d", &cursorID); err == nil {
			if params.SortDesc {
				query = query.Where(ent.UsageRecordIDLT(cursorID))
			} else {
				query = query.Where(ent.UsageRecordIDGT(cursorID))
			}
		}
	}

	// 应用排序
	if params.SortDesc {
		query = query.Order(ent.Desc("id"))
	} else {
		query = query.Order(ent.Asc("id"))
	}

	// 查询多一条用于判断是否有更多数据
	records, err := query.Limit(params.Limit + 1).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询使用记录失败: %w", err)
	}

	// 构建结果
	result := &CursorResult{
		Items:   records,
		HasMore: len(records) > params.Limit,
	}

	// 如果有更多数据，移除多查询的一条并设置游标
	if result.HasMore {
		records = records[:params.Limit]
		result.Items = records
		if len(records) > 0 {
			lastRecord := records[len(records)-1]
			result.NextCursor = fmt.Sprintf("%d", lastRecord.ID)
		}
	}

	return result, nil
}

// StatsQuery 统计查询优化器
type StatsQuery struct {
	client *ent.Client
	logger *zap.Logger
}

// NewStatsQuery 创建统计查询优化器
func NewStatsQuery(client *ent.Client, logger *zap.Logger) *StatsQuery {
	return &StatsQuery{
		client: client,
		logger: logger,
	}
}

// DailyStats 每日统计
type DailyStats struct {
	Date          string  `json:"date"`
	TotalRequests int64   `json:"total_requests"`
	TotalTokens   int64   `json:"total_tokens"`
	TotalCost     float64 `json:"total_cost"`
}

// GetDailyUsageStats 获取每日使用统计（使用原生 SQL 优化）
func (s *StatsQuery) GetDailyUsageStats(
	ctx context.Context,
	userID int64,
	startDate, endDate string,
) ([]*DailyStats, error) {
	// 使用原生 SQL 进行分组统计，性能更好
	query := `
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as total_requests,
			SUM(total_tokens) as total_tokens,
			SUM(cost) as total_cost
		FROM usage_records
		WHERE user_id = $1 
			AND DATE(created_at) >= $2 
			AND DATE(created_at) <= $3
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`

	rows, err := s.client.QueryContext(ctx, query, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("查询每日统计失败: %w", err)
	}
	defer rows.Close()

	var stats []*DailyStats
	for rows.Next() {
		var stat DailyStats
		if err := rows.Scan(&stat.Date, &stat.TotalRequests, &stat.TotalTokens, &stat.TotalCost); err != nil {
			s.logger.Warn("扫描统计行失败", zap.Error(err))
			continue
		}
		stats = append(stats, &stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取统计结果失败: %w", err)
	}

	return stats, nil
}

// ModelStats 模型使用统计
type ModelStats struct {
	Model         string  `json:"model"`
	TotalRequests int64   `json:"total_requests"`
	TotalTokens   int64   `json:"total_tokens"`
	TotalCost     float64 `json:"total_cost"`
}

// GetModelUsageStats 获取模型使用统计
func (s *StatsQuery) GetModelUsageStats(
	ctx context.Context,
	userID int64,
	startTime, endTime time.Time,
) ([]*ModelStats, error) {
	// 使用原生 SQL 进行分组统计
	query := `
		SELECT 
			model,
			COUNT(*) as total_requests,
			SUM(total_tokens) as total_tokens,
			SUM(cost) as total_cost
		FROM usage_records
		WHERE user_id = $1 
			AND created_at >= $2 
			AND created_at <= $3
		GROUP BY model
		ORDER BY total_requests DESC
	`

	rows, err := s.client.QueryContext(ctx, query, userID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("查询模型统计失败: %w", err)
	}
	defer rows.Close()

	var stats []*ModelStats
	for rows.Next() {
		var stat ModelStats
		if err := rows.Scan(&stat.Model, &stat.TotalRequests, &stat.TotalTokens, &stat.TotalCost); err != nil {
			s.logger.Warn("扫描统计行失败", zap.Error(err))
			continue
		}
		stats = append(stats, &stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取统计结果失败: %w", err)
	}

	return stats, nil
}

// UserSummaryStats 用户汇总统计
type UserSummaryStats struct {
	TotalUsers      int64   `json:"total_users"`
	ActiveUsers     int64   `json:"active_users"`
	TotalRequests   int64   `json:"total_requests"`
	TotalTokens     int64   `json:"total_tokens"`
	TotalRevenue    float64 `json:"total_revenue"`
}

// GetSystemStats 获取系统整体统计
func (s *StatsQuery) GetSystemStats(ctx context.Context) (*UserSummaryStats, error) {
	stats := &UserSummaryStats{}

	// 用户总数
	userCount, err := s.client.User.Query().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询用户总数失败: %w", err)
	}
	stats.TotalUsers = int64(userCount)

	// 活跃用户（最近 30 天有使用记录）
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	activeUserQuery := `
		SELECT COUNT(DISTINCT user_id) 
		FROM usage_records 
		WHERE created_at >= $1
	`
	if err := s.client.QueryContext(ctx, activeUserQuery, thirtyDaysAgo).Scan(&stats.ActiveUsers); err != nil {
		s.logger.Warn("查询活跃用户数失败", zap.Error(err))
	}

	// 总请求数、Token 数、收入
	summaryQuery := `
		SELECT 
			COUNT(*) as total_requests,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(cost), 0) as total_revenue
		FROM usage_records
	`
	if err := s.client.QueryContext(ctx, summaryQuery).Scan(
		&stats.TotalRequests,
		&stats.TotalTokens,
		&stats.TotalRevenue,
	); err != nil {
		s.logger.Warn("查询汇总统计失败", zap.Error(err))
	}

	return stats, nil
}

// QueryOptimizer 查询优化器
type QueryOptimizer struct {
	client *ent.Client
	logger *zap.Logger
}

// NewQueryOptimizer 创建查询优化器
func NewQueryOptimizer(client *ent.Client, logger *zap.Logger) *QueryOptimizer {
	return &QueryOptimizer{
		client: client,
		logger: logger,
	}
}

// WithTimeout 设置查询超时
func (q *QueryOptimizer) WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// ExplainQuery 分析查询执行计划
func (q *QueryOptimizer) ExplainQuery(ctx context.Context, query string, args ...interface{}) (string, error) {
	explainQuery := "EXPLAIN " + query
	rows, err := q.client.QueryContext(ctx, explainQuery, args...)
	if err != nil {
		return "", fmt.Errorf("执行 EXPLAIN 失败: %w", err)
	}
	defer rows.Close()

	var plan string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			continue
		}
		plan += line + "\n"
	}

	return plan, nil
}

// IndexHints 索引提示配置
type IndexHints struct {
	UseIndex    []string
	IgnoreIndex []string
	ForceIndex  []string
}

// BuildIndexHint 构建索引提示 SQL
func BuildIndexHint(hints IndexHints) string {
	var parts []string
	if len(hints.UseIndex) > 0 {
		parts = append(parts, fmt.Sprintf("USE INDEX (%s)", joinStrings(hints.UseIndex, ", ")))
	}
	if len(hints.IgnoreIndex) > 0 {
		parts = append(parts, fmt.Sprintf("IGNORE INDEX (%s)", joinStrings(hints.IgnoreIndex, ", ")))
	}
	if len(hints.ForceIndex) > 0 {
		parts = append(parts, fmt.Sprintf("FORCE INDEX (%s)", joinStrings(hints.ForceIndex, ", ")))
	}
	return joinStrings(parts, " ")
}

// joinStrings 连接字符串
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// QueryCache 查询缓存
type QueryCache struct {
	cache  map[string]interface{}
	logger *zap.Logger
}

// NewQueryCache 创建查询缓存
func NewQueryCache(logger *zap.Logger) *QueryCache {
	return &QueryCache{
		cache:  make(map[string]interface{}),
		logger: logger,
	}
}

// Get 获取缓存
func (qc *QueryCache) Get(key string) (interface{}, bool) {
	val, ok := qc.cache[key]
	return val, ok
}

// Set 设置缓存
func (qc *QueryCache) Set(key string, value interface{}) {
	qc.cache[key] = value
}

// Delete 删除缓存
func (qc *QueryCache) Delete(key string) {
	delete(qc.cache, key)
}

// Clear 清空缓存
func (qc *QueryCache) Clear() {
	qc.cache = make(map[string]interface{})
}
