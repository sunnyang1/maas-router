// Package service 业务服务层
// 提供计费服务
package service

import (
	"context"
	"fmt"
	"time"

	"maas-router/ent"
	"maas-router/internal/cache"
	"maas-router/internal/config"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// BillingService 计费服务接口
// 处理费用计算、余额管理、使用记录等
type BillingService interface {
	// CalculateCost 计算费用
	CalculateCost(model string, promptTokens, completionTokens int) float64

	// DeductBalance 扣除余额
	DeductBalance(ctx context.Context, userID int64, amount float64) error

	// CheckBalance 检查余额
	CheckBalance(ctx context.Context, userID int64) (float64, error)

	// RecordUsage 记录使用量
	RecordUsage(ctx context.Context, record *UsageRecord) error

	// GetUsageStats 获取使用统计
	GetUsageStats(ctx context.Context, userID int64, period string) (*UsageStats, error)

	// GetDailyUsage 获取每日使用量
	GetDailyUsage(ctx context.Context, userID int64, date string) (*DailyUsage, error)

	// CheckQuota 检查配额
	CheckQuota(ctx context.Context, apiKeyID int64) (bool, error)

	// UpdateQuotaUsage 更新配额使用
	UpdateQuotaUsage(ctx context.Context, apiKeyID int64, amount float64) error
}

// UsageRecord 使用记录
type UsageRecord struct {
	RequestID        string    `json:"request_id"`
	UserID           int64     `json:"user_id"`
	APIKeyID         int64     `json:"api_key_id,omitempty"`
	AccountID        int64     `json:"account_id,omitempty"`
	GroupID          int64     `json:"group_id,omitempty"`
	Model            string    `json:"model"`
	Platform         string    `json:"platform"`
	PromptTokens     int32     `json:"prompt_tokens"`
	CompletionTokens int32     `json:"completion_tokens"`
	TotalTokens      int32     `json:"total_tokens"`
	LatencyMs        int32     `json:"latency_ms,omitempty"`
	FirstTokenMs     int32     `json:"first_token_ms,omitempty"`
	Cost             float64   `json:"cost"`
	Status           string    `json:"status"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	ClientIP         string    `json:"client_ip,omitempty"`
	UserAgent        string    `json:"user_agent,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// UsageStats 使用统计
type UsageStats struct {
	UserID           int64     `json:"user_id"`
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
	TotalRequests    int64     `json:"total_requests"`
	SuccessRequests  int64     `json:"success_requests"`
	FailedRequests   int64     `json:"failed_requests"`
	TotalTokens      int64     `json:"total_tokens"`
	PromptTokens     int64     `json:"prompt_tokens"`
	CompletionTokens int64     `json:"completion_tokens"`
	TotalCost        float64   `json:"total_cost"`
	AvgLatencyMs     int64     `json:"avg_latency_ms"`
	ModelBreakdown   map[string]*ModelUsage `json:"model_breakdown"`
	PlatformBreakdown map[string]*PlatformUsage `json:"platform_breakdown"`
}

// ModelUsage 模型使用统计
type ModelUsage struct {
	Model            string  `json:"model"`
	TotalRequests    int64   `json:"total_requests"`
	TotalTokens      int64   `json:"total_tokens"`
	TotalCost        float64 `json:"total_cost"`
}

// PlatformUsage 平台使用统计
type PlatformUsage struct {
	Platform      string  `json:"platform"`
	TotalRequests int64   `json:"total_requests"`
	TotalTokens   int64   `json:"total_tokens"`
	TotalCost     float64 `json:"total_cost"`
}

// DailyUsage 每日使用量
type DailyUsage struct {
	Date             string  `json:"date"`
	TotalRequests    int64   `json:"total_requests"`
	TotalTokens      int64   `json:"total_tokens"`
	TotalCost        float64 `json:"total_cost"`
	RemainingQuota   float64 `json:"remaining_quota"`
	QuotaLimit       float64 `json:"quota_limit"`
}

// billingService 计费服务实现
type billingService struct {
	db       *ent.Client
	redis    *redis.Client
	cache    cache.Cache
	cacheKey *cache.CacheKey
	cfg      *config.Config
	logger   *zap.Logger

	// 模型定价表
	pricing map[string]ModelPricing
}

// ModelPricing 模型定价
type ModelPricing struct {
	InputPrice  float64 // 输入价格（美元/百万 Token）
	OutputPrice float64 // 输出价格（美元/百万 Token）
}

// NewBillingService 创建计费服务实例
func NewBillingService(
	db *ent.Client,
	redis *redis.Client,
	cfg *config.Config,
	logger *zap.Logger,
) BillingService {
	// 创建统一缓存实例
	c := cache.NewCacheFromClient(redis, logger, "maas")
	svc := &billingService{
		db:       db,
		redis:    redis,
		cache:    c,
		cacheKey: cache.NewCacheKey("maas"),
		cfg:      cfg,
		logger:   logger,
	}

	// 初始化定价表
	svc.initPricing()

	return svc
}

// initPricing 初始化模型定价表
func (s *billingService) initPricing() {
	s.pricing = map[string]ModelPricing{
		// Claude 模型定价
		"claude-3-opus-20240229":     {InputPrice: 15, OutputPrice: 75},
		"claude-3-opus-latest":       {InputPrice: 15, OutputPrice: 75},
		"claude-3-sonnet-20240229":   {InputPrice: 3, OutputPrice: 15},
		"claude-3-haiku-20240307":    {InputPrice: 0.25, OutputPrice: 1.25},
		"claude-3-5-sonnet-20240620": {InputPrice: 3, OutputPrice: 15},
		"claude-3-5-sonnet-20241022": {InputPrice: 3, OutputPrice: 15},
		"claude-3-5-sonnet-latest":   {InputPrice: 3, OutputPrice: 15},
		"claude-3-5-haiku-20241022":  {InputPrice: 0.8, OutputPrice: 4},
		"claude-3-5-haiku-latest":    {InputPrice: 0.8, OutputPrice: 4},

		// OpenAI 模型定价
		"gpt-4o":                    {InputPrice: 2.5, OutputPrice: 10},
		"gpt-4o-2024-11-20":         {InputPrice: 2.5, OutputPrice: 10},
		"gpt-4o-2024-08-06":         {InputPrice: 2.5, OutputPrice: 10},
		"gpt-4o-2024-05-13":         {InputPrice: 5, OutputPrice: 15},
		"gpt-4o-mini":               {InputPrice: 0.15, OutputPrice: 0.6},
		"gpt-4o-mini-2024-07-18":    {InputPrice: 0.15, OutputPrice: 0.6},
		"gpt-4-turbo":               {InputPrice: 10, OutputPrice: 30},
		"gpt-4-turbo-2024-04-09":    {InputPrice: 10, OutputPrice: 30},
		"gpt-4":                     {InputPrice: 30, OutputPrice: 60},
		"gpt-4-32k":                 {InputPrice: 60, OutputPrice: 120},
		"gpt-3.5-turbo":             {InputPrice: 0.5, OutputPrice: 1.5},
		"gpt-3.5-turbo-0125":        {InputPrice: 0.5, OutputPrice: 1.5},
		"o1-preview":                {InputPrice: 15, OutputPrice: 60},
		"o1-mini":                   {InputPrice: 3, OutputPrice: 12},

		// Embedding 模型定价
		"text-embedding-3-small":    {InputPrice: 0.02, OutputPrice: 0},
		"text-embedding-3-large":    {InputPrice: 0.13, OutputPrice: 0},
		"text-embedding-ada-002":    {InputPrice: 0.1, OutputPrice: 0},

		// 图片生成模型定价（按次计费）
		"dall-e-3":                  {InputPrice: 0.04, OutputPrice: 0}, // 标准 1024x1024
		"dall-e-2":                  {InputPrice: 0.02, OutputPrice: 0},
	}
}

// CalculateCost 计算费用
func (s *billingService) CalculateCost(model string, promptTokens, completionTokens int) float64 {
	pricing, ok := s.pricing[model]
	if !ok {
		// 默认使用中等定价
		pricing = ModelPricing{InputPrice: 3, OutputPrice: 15}
	}

	inputCost := float64(promptTokens) * pricing.InputPrice / 1_000_000
	outputCost := float64(completionTokens) * pricing.OutputPrice / 1_000_000

	cost := inputCost + outputCost

	// 应用最低计费单位
	if cost > 0 && cost < s.cfg.Billing.MinChargeUnit {
		cost = s.cfg.Billing.MinChargeUnit
	}

	return cost
}

// DeductBalance 扣除余额
func (s *billingService) DeductBalance(ctx context.Context, userID int64, amount float64) error {
	// 开启事务
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	// 获取用户当前余额
	user, err := tx.User.Get(ctx, userID)
	if err != nil {
		return fmt.Errorf("获取用户失败: %w", err)
	}

	// 检查余额是否足够
	if user.Balance < amount {
		return fmt.Errorf("余额不足: 当前 %.6f, 需要 %.6f", user.Balance, amount)
	}

	// 扣除余额
	newBalance := user.Balance - amount
	_, err = tx.User.UpdateOneID(userID).
		SetBalance(newBalance).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("更新余额失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 更新 Redis 缓存
	cacheKey := fmt.Sprintf("user:balance:%d", userID)
	s.redis.Set(ctx, cacheKey, newBalance, 5*time.Minute)

	s.logger.Info("余额扣除成功",
		zap.Int64("user_id", userID),
		zap.Float64("amount", amount),
		zap.Float64("new_balance", newBalance))

	return nil
}

// CheckBalance 检查余额（带缓存）
func (s *billingService) CheckBalance(ctx context.Context, userID int64) (float64, error) {
	// 先从缓存获取
	cacheKey := s.cacheKey.Balance(userID)
	if balance, err := s.cache.Get(ctx, cacheKey); err == nil {
		var balanceFloat float64
		if _, err := fmt.Sscanf(balance, "%f", &balanceFloat); err == nil {
			return balanceFloat, nil
		}
	}

	// 从数据库获取
	user, err := s.db.User.Get(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("获取用户失败: %w", err)
	}

	// 更新缓存
	if err := s.cache.Set(ctx, cacheKey, fmt.Sprintf("%f", user.Balance), cache.CommonCacheTTL.Medium); err != nil {
		s.logger.Warn("缓存余额失败", zap.Error(err))
	}

	return user.Balance, nil
}

// RecordUsage 记录使用量
func (s *billingService) RecordUsage(ctx context.Context, record *UsageRecord) error {
	// 检查是否跳过计费
	if record.UserID == 0 {
		return nil
	}

	// 创建使用记录
	_, err := s.db.UsageRecord.Create().
		SetRequestID(record.RequestID).
		SetUserID(record.UserID).
		SetNillableAPIKeyID(&record.APIKeyID).
		SetNillableAccountID(&record.AccountID).
		SetNillableGroupID(&record.GroupID).
		SetModel(record.Model).
		SetPlatform(record.Platform).
		SetPromptTokens(record.PromptTokens).
		SetCompletionTokens(record.CompletionTokens).
		SetTotalTokens(record.TotalTokens).
		SetNillableLatencyMs(&record.LatencyMs).
		SetNillableFirstTokenMs(&record.FirstTokenMs).
		SetCost(record.Cost).
		SetStatus(ent.UsageRecordStatus(record.Status)).
		SetNillableErrorMessage(&record.ErrorMessage).
		SetNillableClientIP(&record.ClientIP).
		SetNillableUserAgent(&record.UserAgent).
		Save(ctx)
	if err != nil {
		s.logger.Error("创建使用记录失败",
			zap.String("request_id", record.RequestID),
			zap.Error(err))
		return fmt.Errorf("创建使用记录失败: %w", err)
	}

	// 扣除余额
	if record.Cost > 0 && s.cfg.Billing.Enabled {
		if err := s.DeductBalance(ctx, record.UserID, record.Cost); err != nil {
			s.logger.Warn("扣除余额失败",
				zap.Int64("user_id", record.UserID),
				zap.Float64("cost", record.Cost),
				zap.Error(err))
		}
	}

	// 更新 Redis 统计
	s.updateRedisStats(ctx, record)

	s.logger.Debug("使用记录已保存",
		zap.String("request_id", record.RequestID),
		zap.Int64("user_id", record.UserID),
		zap.String("model", record.Model),
		zap.Float64("cost", record.Cost))

	return nil
}

// GetUsageStats 获取使用统计
func (s *billingService) GetUsageStats(ctx context.Context, userID int64, period string) (*UsageStats, error) {
	// 解析时间范围
	var start, end time.Time
	now := time.Now()

	switch period {
	case "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = now
	case "yesterday":
		start = time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
		end = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		start = now.AddDate(0, 0, -7)
		end = now
	case "month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = now
	case "last_month":
		start = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		end = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = now
	}

	// 从数据库查询统计
	records, err := s.db.UsageRecord.Query().
		Where(
			ent.UsageRecordUserID(userID),
			ent.UsageRecordCreatedAtGTE(start),
			ent.UsageRecordCreatedAtLTE(end),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询使用记录失败: %w", err)
	}

	// 计算统计数据
	stats := &UsageStats{
		UserID:           userID,
		PeriodStart:      start,
		PeriodEnd:        end,
		ModelBreakdown:   make(map[string]*ModelUsage),
		PlatformBreakdown: make(map[string]*PlatformUsage),
	}

	for _, record := range records {
		stats.TotalRequests++
		stats.TotalTokens += int64(record.TotalTokens)
		stats.PromptTokens += int64(record.PromptTokens)
		stats.CompletionTokens += int64(record.CompletionTokens)
		stats.TotalCost += record.Cost
		stats.AvgLatencyMs += int64(record.LatencyMs)

		if record.Status == ent.UsageRecordStatusSuccess {
			stats.SuccessRequests++
		} else {
			stats.FailedRequests++
		}

		// 模型统计
		if _, ok := stats.ModelBreakdown[record.Model]; !ok {
			stats.ModelBreakdown[record.Model] = &ModelUsage{Model: record.Model}
		}
		stats.ModelBreakdown[record.Model].TotalRequests++
		stats.ModelBreakdown[record.Model].TotalTokens += int64(record.TotalTokens)
		stats.ModelBreakdown[record.Model].TotalCost += record.Cost

		// 平台统计
		if _, ok := stats.PlatformBreakdown[record.Platform]; !ok {
			stats.PlatformBreakdown[record.Platform] = &PlatformUsage{Platform: record.Platform}
		}
		stats.PlatformBreakdown[record.Platform].TotalRequests++
		stats.PlatformBreakdown[record.Platform].TotalTokens += int64(record.TotalTokens)
		stats.PlatformBreakdown[record.Platform].TotalCost += record.Cost
	}

	if stats.TotalRequests > 0 {
		stats.AvgLatencyMs /= stats.TotalRequests
	}

	return stats, nil
}

// GetDailyUsage 获取每日使用量（带缓存）
func (s *billingService) GetDailyUsage(ctx context.Context, userID int64, date string) (*DailyUsage, error) {
	// 解析日期
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("日期格式错误: %w", err)
	}

	start := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, parsedDate.Location())
	end := start.Add(24 * time.Hour)

	// 从缓存获取
	cacheKey := s.cacheKey.DailyUsage(userID, date)
	var cachedUsage DailyUsage
	if err := s.cache.GetObject(ctx, cacheKey, &cachedUsage); err == nil {
		s.logger.Debug("从缓存获取每日使用量", zap.Int64("user_id", userID), zap.String("date", date))
		return &cachedUsage, nil
	}

	// 从数据库查询
	records, err := s.db.UsageRecord.Query().
		Where(
			ent.UsageRecordUserID(userID),
			ent.UsageRecordCreatedAtGTE(start),
			ent.UsageRecordCreatedAtLT(end),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询使用记录失败: %w", err)
	}

	// 计算统计
	usage := &DailyUsage{
		Date: date,
	}

	for _, record := range records {
		usage.TotalRequests++
		usage.TotalTokens += int64(record.TotalTokens)
		usage.TotalCost += record.Cost
	}

	// 缓存结果（1小时）
	if err := s.cache.SetObject(ctx, cacheKey, usage, cache.CommonCacheTTL.Long); err != nil {
		s.logger.Warn("缓存每日使用量失败", zap.Error(err))
	}

	return usage, nil
}

// CheckQuota 检查配额
func (s *billingService) CheckQuota(ctx context.Context, apiKeyID int64) (bool, error) {
	// 获取 API Key 信息
	apiKey, err := s.db.APIKey.Get(ctx, apiKeyID)
	if err != nil {
		return false, fmt.Errorf("获取 API Key 失败: %w", err)
	}

	// 检查状态
	if apiKey.Status != ent.APIKeyStatusActive {
		return false, nil
	}

	// 检查过期时间
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return false, nil
	}

	// 检查每日限额
	if apiKey.DailyLimit != nil {
		today := time.Now().Format("2006-01-02")
		usage, err := s.GetDailyUsage(ctx, apiKey.UserID, today)
		if err != nil {
			return false, err
		}

		if usage.TotalCost >= *apiKey.DailyLimit {
			return false, nil
		}
	}

	// 检查每月限额
	if apiKey.MonthlyLimit != nil {
		now := time.Now()
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		stats, err := s.GetUsageStats(ctx, apiKey.UserID, "month")
		if err != nil {
			return false, err
		}

		if stats.TotalCost >= *apiKey.MonthlyLimit {
			return false, nil
		}
	}

	return true, nil
}

// UpdateQuotaUsage 更新配额使用
func (s *billingService) UpdateQuotaUsage(ctx context.Context, apiKeyID int64, amount float64) error {
	// 更新 Redis 中的配额使用
	today := time.Now().Format("2006-01-02")
	dailyKey := fmt.Sprintf("quota:daily:%d:%s", apiKeyID, today)
	monthlyKey := fmt.Sprintf("quota:monthly:%d:%s", apiKeyID, time.Now().Format("2006-01"))

	// 增加每日使用量
	s.redis.IncrByFloat(ctx, dailyKey, amount)
	s.redis.Expire(ctx, dailyKey, 48*time.Hour)

	// 增加每月使用量
	s.redis.IncrByFloat(ctx, monthlyKey, amount)
	s.redis.Expire(ctx, monthlyKey, 35*24*time.Hour)

	return nil
}

// updateRedisStats 更新 Redis 统计
func (s *billingService) updateRedisStats(ctx context.Context, record *UsageRecord) {
	now := time.Now()
	date := now.Format("2006-01-02")
	hour := now.Format("2006-01-02-15")

	// 用户每日统计
	userDailyKey := fmt.Sprintf("stats:user:%d:daily:%s", record.UserID, date)
	s.redis.IncrBy(ctx, userDailyKey+":requests", 1)
	s.redis.IncrBy(ctx, userDailyKey+":tokens", int64(record.TotalTokens))
	s.redis.IncrByFloat(ctx, userDailyKey+":cost", record.Cost)
	s.redis.Expire(ctx, userDailyKey+":requests", 7*24*time.Hour)
	s.redis.Expire(ctx, userDailyKey+":tokens", 7*24*time.Hour)
	s.redis.Expire(ctx, userDailyKey+":cost", 7*24*time.Hour)

	// 用户每小时统计
	userHourlyKey := fmt.Sprintf("stats:user:%d:hourly:%s", record.UserID, hour)
	s.redis.IncrBy(ctx, userHourlyKey+":requests", 1)
	s.redis.Expire(ctx, userHourlyKey+":requests", 48*time.Hour)

	// 模型统计
	modelKey := fmt.Sprintf("stats:model:%s:daily:%s", record.Model, date)
	s.redis.IncrBy(ctx, modelKey+":requests", 1)
	s.redis.Expire(ctx, modelKey+":requests", 30*24*time.Hour)

	// 平台统计
	platformKey := fmt.Sprintf("stats:platform:%s:daily:%s", record.Platform, date)
	s.redis.IncrBy(ctx, platformKey+":requests", 1)
	s.redis.Expire(ctx, platformKey+":requests", 30*24*time.Hour)
}

// GetPricing 获取模型定价
func (s *billingService) GetPricing(model string) (*ModelPricing, error) {
	pricing, ok := s.pricing[model]
	if !ok {
		return nil, fmt.Errorf("模型 %s 定价信息不存在", model)
	}
	return &pricing, nil
}

// GetAllPricing 获取所有模型定价
func (s *billingService) GetAllPricing() map[string]ModelPricing {
	return s.pricing
}

// EstimateCost 预估费用
func (s *billingService) EstimateCost(model string, estimatedTokens int) float64 {
	pricing, ok := s.pricing[model]
	if !ok {
		pricing = ModelPricing{InputPrice: 3, OutputPrice: 15}
	}

	// 假设输入输出比例为 3:1
	inputTokens := int(float64(estimatedTokens) * 0.75)
	outputTokens := estimatedTokens - inputTokens

	return s.CalculateCost(model, inputTokens, outputTokens)
}

// 辅助函数
func toJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func parseJSON(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}
