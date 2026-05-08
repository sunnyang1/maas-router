package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"maas-router/ent"
	"maas-router/internal/complexity"
	"maas-router/internal/config"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// ComplexityService 复杂度分析服务接口
type ComplexityService interface {
	// Analyze 分析请求复杂度
	Analyze(ctx context.Context, req *complexity.AnalyzeRequest) (*complexity.ComplexityProfile, error)

	// AnalyzeWithCache 带缓存的复杂度分析
	AnalyzeWithCache(ctx context.Context, req *complexity.AnalyzeRequest) (*complexity.ComplexityProfile, error)

	// GetRoutingConfig 获取路由配置
	GetRoutingConfig(profile *complexity.ComplexityProfile) *RoutingConfig

	// RecordFeedback 记录质量反馈
	RecordFeedback(ctx context.Context, requestID string, profile *complexity.ComplexityProfile, satisfied bool) error

	// GetStats 获取统计数据
	GetStats(ctx context.Context) (*complexity.ComplexityStats, error)

	// IsEnabled 检查服务是否启用
	IsEnabled() bool
}

// RoutingConfig 路由配置
type RoutingConfig struct {
	// 推荐模型
	Model string `json:"model"`
	// 推荐层级
	Tier string `json:"tier"`
	// 回退模型
	FallbackModel string `json:"fallback_model,omitempty"`
	// 是否需要升级
	NeedsUpgrade bool `json:"needs_upgrade"`
	// 质量风险
	QualityRisk string `json:"quality_risk"`
	// 预估成本
	EstimatedCost float64 `json:"estimated_cost"`
	// 成本节省比例
	CostSavingRatio float64 `json:"cost_saving_ratio"`
	// 置信度
	Confidence float64 `json:"confidence"`
	// 复杂度级别
	Level string `json:"level"`
	// 复杂度分数
	Score float64 `json:"score"`
}

// complexityService 复杂度分析服务实现
type complexityService struct {
	db         *ent.Client
	redis      *redis.Client
	cfg        *config.ComplexityConfig
	classifier *complexity.ComplexityClassifier
	logger     *zap.Logger
	httpClient *http.Client
}

// NewComplexityService 创建复杂度分析服务实例
func NewComplexityService(
	db *ent.Client,
	redis *redis.Client,
	cfg *config.ComplexityConfig,
	logger *zap.Logger,
) ComplexityService {
	// 创建特征提取器
	extractor := complexity.NewFeatureExtractor(cfg.Features, logger)

	// 创建分类器
	classifier := complexity.NewComplexityClassifier(
		extractor,
		cfg.ModelTiers,
		nil, // 使用默认权重
		logger,
	)

	svc := &complexityService{
		db:         db,
		redis:      redis,
		cfg:        cfg,
		classifier: classifier,
		logger:     logger,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.TimeoutMs) * time.Millisecond,
		},
	}

	return svc
}

// Analyze 分析请求复杂度
// 本地模式直接调 classifier.Classify，远程模式调 HTTP，混合模式先本地后远程
func (s *complexityService) Analyze(ctx context.Context, req *complexity.AnalyzeRequest) (*complexity.ComplexityProfile, error) {
	if !s.cfg.Enabled {
		return s.defaultProfile(), nil
	}

	switch s.cfg.Mode {
	case "local":
		return s.analyzeLocal(ctx, req)
	case "remote":
		return s.analyzeRemote(ctx, req)
	case "hybrid":
		return s.analyzeHybrid(ctx, req)
	default:
		s.logger.Warn("未知的复杂度分析模式，使用本地模式",
			zap.String("mode", s.cfg.Mode))
		return s.analyzeLocal(ctx, req)
	}
}

// analyzeLocal 本地模式：直接使用本地分类器
func (s *complexityService) analyzeLocal(ctx context.Context, req *complexity.AnalyzeRequest) (*complexity.ComplexityProfile, error) {
	// 提取特征
	extractor := complexity.NewFeatureExtractor(s.cfg.Features, s.logger)
	features := extractor.Extract(req)

	// 分类
	profile := s.classifier.Classify(features)

	return profile, nil
}

// analyzeRemote 远程模式：调用远程分析服务
func (s *complexityService) analyzeRemote(ctx context.Context, req *complexity.AnalyzeRequest) (*complexity.ComplexityProfile, error) {
	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		s.logger.Warn("序列化复杂度分析请求失败，返回默认配置",
			zap.Error(err))
		return s.defaultProfile(), nil
	}

	// 构建 HTTP 请求
	url := fmt.Sprintf("%s/v1/complexity/analyze", s.cfg.RemoteAddr)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		s.logger.Warn("创建复杂度分析请求失败，返回默认配置",
			zap.Error(err))
		return s.defaultProfile(), nil
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求（带重试）
	var resp *http.Response
	var lastErr error
	for attempt := 0; attempt <= s.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避
			backoff := time.Duration(attempt*attempt*100) * time.Millisecond
			select {
			case <-ctx.Done():
				return s.defaultProfile(), ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, lastErr = s.httpClient.Do(httpReq)
		if lastErr == nil && resp.StatusCode < 500 {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	if lastErr != nil {
		s.logger.Warn("调用远程复杂度分析服务失败，返回默认配置",
			zap.Error(lastErr),
			zap.Int("max_retries", s.cfg.MaxRetries))
		return s.defaultProfile(), nil
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Warn("远程复杂度分析服务返回错误，返回默认配置",
			zap.Int("status_code", resp.StatusCode),
			zap.String("body", string(body)))
		return s.defaultProfile(), nil
	}

	// 解析响应
	var profile complexity.ComplexityProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		s.logger.Warn("解析复杂度分析响应失败，返回默认配置",
			zap.Error(err))
		return s.defaultProfile(), nil
	}

	return &profile, nil
}

// analyzeHybrid 混合模式：先本地分析，再远程验证
func (s *complexityService) analyzeHybrid(ctx context.Context, req *complexity.AnalyzeRequest) (*complexity.ComplexityProfile, error) {
	// 先进行本地分析
	localProfile, err := s.analyzeLocal(ctx, req)
	if err != nil {
		s.logger.Warn("本地复杂度分析失败，尝试远程分析",
			zap.Error(err))
		return s.analyzeRemote(ctx, req)
	}

	// 如果本地置信度足够高，直接返回
	if localProfile.Confidence >= 0.8 {
		return localProfile, nil
	}

	// 置信度不够高时，尝试远程分析
	remoteProfile, err := s.analyzeRemote(ctx, req)
	if err != nil {
		s.logger.Debug("远程复杂度分析失败，使用本地结果",
			zap.Error(err))
		return localProfile, nil
	}

	// 取置信度更高的结果
	if remoteProfile.Confidence > localProfile.Confidence {
		return remoteProfile, nil
	}

	return localProfile, nil
}

// AnalyzeWithCache 带缓存的复杂度分析
// 先查 Redis 缓存（key=sha256(messages)），未命中则调 Analyze 并缓存
func (s *complexityService) AnalyzeWithCache(ctx context.Context, req *complexity.AnalyzeRequest) (*complexity.ComplexityProfile, error) {
	// 生成缓存 key
	cacheKey := s.generateCacheKey(req)

	// 尝试从缓存获取
	cached, err := s.getFromCache(ctx, cacheKey)
	if err == nil && cached != nil {
		s.logger.Debug("复杂度分析缓存命中",
			zap.String("cache_key", cacheKey))
		return cached, nil
	}

	// 缓存未命中，执行分析
	profile, err := s.Analyze(ctx, req)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	if cacheErr := s.setToCache(ctx, cacheKey, profile); cacheErr != nil {
		s.logger.Warn("写入复杂度分析缓存失败",
			zap.Error(cacheErr),
			zap.String("cache_key", cacheKey))
	}

	return profile, nil
}

// generateCacheKey 生成缓存 key（基于消息内容的 sha256 哈希）
func (s *complexityService) generateCacheKey(req *complexity.AnalyzeRequest) string {
	hash := sha256.New()

	// 写入模型
	hash.Write([]byte(req.Model))

	// 写入系统提示
	hash.Write([]byte(req.System))

	// 写入所有消息
	for _, msg := range req.Messages {
		hash.Write([]byte(msg.Role))
		hash.Write([]byte(msg.Content))
	}

	hashBytes := hash.Sum(nil)
	hashStr := hex.EncodeToString(hashBytes)

	return fmt.Sprintf("complexity:analyze:%s", hashStr)
}

// getFromCache 从 Redis 缓存获取复杂度分析结果
func (s *complexityService) getFromCache(ctx context.Context, key string) (*complexity.ComplexityProfile, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("redis 未初始化")
	}

	data, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var profile complexity.ComplexityProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

// setToCache 将复杂度分析结果写入 Redis 缓存
func (s *complexityService) setToCache(ctx context.Context, key string, profile *complexity.ComplexityProfile) error {
	if s.redis == nil {
		return fmt.Errorf("redis 未初始化")
	}

	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}

	ttl := time.Duration(s.cfg.CacheTTLSec) * time.Second
	return s.redis.Set(ctx, key, data, ttl).Err()
}

// GetRoutingConfig 从 profile 生成路由配置
func (s *complexityService) GetRoutingConfig(profile *complexity.ComplexityProfile) *RoutingConfig {
	if profile == nil {
		profile = s.defaultProfile()
	}

	return &RoutingConfig{
		Model:           profile.RecommendedModel,
		Tier:            string(profile.RecommendedTier),
		FallbackModel:   profile.FallbackModel,
		NeedsUpgrade:    profile.NeedsUpgrade,
		QualityRisk:     profile.QualityRisk,
		EstimatedCost:   profile.EstimatedCost,
		CostSavingRatio: profile.CostSavingRatio,
		Confidence:      profile.Confidence,
		Level:           string(profile.Level),
		Score:           profile.Score,
	}
}

// RecordFeedback 记录质量反馈到 Redis
func (s *complexityService) RecordFeedback(ctx context.Context, requestID string, profile *complexity.ComplexityProfile, satisfied bool) error {
	if s.redis == nil {
		return fmt.Errorf("redis 未初始化")
	}

	feedback := map[string]interface{}{
		"request_id":    requestID,
		"score":         profile.Score,
		"level":         string(profile.Level),
		"tier":          string(profile.RecommendedTier),
		"model":         profile.RecommendedModel,
		"quality_risk":  profile.QualityRisk,
		"confidence":    profile.Confidence,
		"satisfied":     satisfied,
		"timestamp":     time.Now().Unix(),
	}

	data, err := json.Marshal(feedback)
	if err != nil {
		return fmt.Errorf("序列化反馈数据失败: %w", err)
	}

	// 写入反馈列表
	key := fmt.Sprintf("complexity:feedback:%s", requestID)
	if err := s.redis.Set(ctx, key, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("写入反馈缓存失败: %w", err)
	}

	// 更新统计计数
	if satisfied {
		s.redis.Incr(ctx, "complexity:stats:satisfied_count")
	} else {
		s.redis.Incr(ctx, "complexity:stats:unsatisfied_count")
	}
	s.redis.Incr(ctx, "complexity:stats:total_count")

	s.logger.Debug("记录复杂度分析质量反馈",
		zap.String("request_id", requestID),
		zap.String("level", string(profile.Level)),
		zap.String("model", profile.RecommendedModel),
		zap.Bool("satisfied", satisfied))

	return nil
}

// GetStats 从数据库/Redis 聚合统计数据
func (s *complexityService) GetStats(ctx context.Context) (*complexity.ComplexityStats, error) {
	stats := &complexity.ComplexityStats{
		LevelDistribution: make(map[complexity.ScoreLevel]int64),
		TierDistribution:  make(map[complexity.TierName]int64),
		ModelDistribution: make(map[string]int64),
	}

	if s.redis == nil {
		return stats, nil
	}

	// 从 Redis 获取统计计数
	totalCount, _ := s.redis.Get(ctx, "complexity:stats:total_count").Int64()
	satisfiedCount, _ := s.redis.Get(ctx, "complexity:stats:satisfied_count").Int64()

	stats.TotalRequests = totalCount
	if totalCount > 0 {
		stats.QualityPassRate = float64(satisfiedCount) / float64(totalCount)
	}

	// 获取级别分布
	for _, level := range []complexity.ScoreLevel{
		complexity.ScoreLevelSimple,
		complexity.ScoreLevelMedium,
		complexity.ScoreLevelComplex,
		complexity.ScoreLevelExpert,
	} {
		count, _ := s.redis.Get(ctx, fmt.Sprintf("complexity:stats:level:%s", level)).Int64()
		stats.LevelDistribution[level] = count
	}

	// 获取层级分布
	for _, tier := range []complexity.TierName{
		complexity.TierNameEconomy,
		complexity.TierNameStandard,
		complexity.TierNameAdvanced,
		complexity.TierNamePremium,
	} {
		count, _ := s.redis.Get(ctx, fmt.Sprintf("complexity:stats:tier:%s", tier)).Int64()
		stats.TierDistribution[tier] = count
	}

	return stats, nil
}

// IsEnabled 检查服务是否启用
func (s *complexityService) IsEnabled() bool {
	return s.cfg != nil && s.cfg.Enabled
}

// defaultProfile 返回默认的复杂度分析结果（降级逻辑）
func (s *complexityService) defaultProfile() *complexity.ComplexityProfile {
	return &complexity.ComplexityProfile{
		Score:               0.5,
		Level:               complexity.ScoreLevelMedium,
		Confidence:          0.5,
		LexicalScore:        0.1,
		StructuralScore:     0.1,
		DomainScore:         0.1,
		ConversationalScore: 0.1,
		TaskTypeScore:       0.1,
		RecommendedTier:     complexity.TierNameStandard,
		RecommendedModel:    "claude-3-5-sonnet-20241022",
		FallbackModel:       "claude-3-5-haiku-20241022",
		EstimatedCost:       0.000003,
		CostSavingRatio:     0,
		QualityRisk:         "low",
		NeedsUpgrade:        false,
	}
}
