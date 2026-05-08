// Package service 业务服务层
// 提供智能路由 Agent 服务
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"maas-router/ent"
	"maas-router/internal/config"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// JudgeAgentService 智能路由 Agent 服务接口
// 调用 Judge Agent 进行复杂度评分和路由建议
type JudgeAgentService interface {
	// ScoreComplexity 复杂度评分
	// 调用 Judge Agent 分析查询复杂度
	ScoreComplexity(ctx context.Context, query string) (*ComplexityScore, error)

	// ScoreComplexityWithMessages 基于消息的复杂度评分
	ScoreComplexityWithMessages(ctx context.Context, messages interface{}) (*ComplexityScore, error)

	// GetRoutingRecommendation 获取路由建议
	// 根据复杂度评分返回推荐的路由策略
	GetRoutingRecommendation(ctx context.Context, score *ComplexityScore) (*RoutingRecommendation, error)

	// IsEnabled 检查 Agent 是否启用
	IsEnabled() bool
}

// ComplexityScore 复杂度评分结果
type ComplexityScore struct {
	// 总体评分 0-100
	Score int `json:"score"`

	// 复杂度级别: simple, medium, complex
	Level string `json:"level"`

	// 各维度评分
	Dimensions ComplexityDimensions `json:"dimensions"`

	// 推荐模型
	RecommendedModel string `json:"recommended_model,omitempty"`

	// 推荐平台
	RecommendedPlatform string `json:"recommended_platform,omitempty"`

	// 分析理由
	Reasoning string `json:"reasoning,omitempty"`

	// 处理时间
	ProcessingTimeMs int `json:"processing_time_ms,omitempty"`
}

// ComplexityDimensions 复杂度维度
type ComplexityDimensions struct {
	// 查询长度复杂度
	QueryLength int `json:"query_length"`

	// 推理复杂度
	Reasoning int `json:"reasoning"`

	// 代码相关复杂度
	CodeRelated int `json:"code_related"`

	// 数学计算复杂度
	Mathematical int `json:"mathematical"`

	// 创意写作复杂度
	Creative int `json:"creative"`

	// 多语言复杂度
	MultiLanguage int `json:"multi_language"`

	// 上下文依赖复杂度
	ContextDependency int `json:"context_dependency"`

	// 任务类型复杂度
	TaskComplexity int `json:"task_complexity"`
}

// RoutingRecommendation 路由建议
type RoutingRecommendation struct {
	// 推荐平台
	Platform string `json:"platform"`

	// 推荐模型
	Model string `json:"model"`

	// 推荐账号分组
	GroupID int64 `json:"group_id,omitempty"`

	// 是否需要流式响应
	PreferStream bool `json:"prefer_stream"`

	// 建议超时时间（秒）
	SuggestedTimeout int `json:"suggested_timeout"`

	// 建议最大 Token 数
	SuggestedMaxTokens int `json:"suggested_max_tokens"`

	// 置信度 0-1
	Confidence float64 `json:"confidence"`

	// 建议理由
	Reason string `json:"reason"`

	// 备选方案
	Alternatives []RoutingAlternative `json:"alternatives,omitempty"`
}

// RoutingAlternative 备选路由方案
type RoutingAlternative struct {
	Platform string `json:"platform"`
	Model    string `json:"model"`
	Priority int    `json:"priority"`
	Reason   string `json:"reason"`
}

// JudgeAgentRequest Judge Agent 请求
type JudgeAgentRequest struct {
	Query    string      `json:"query,omitempty"`
	Messages interface{} `json:"messages,omitempty"`
	Context  *JudgeContext `json:"context,omitempty"`
}

// JudgeContext Judge 上下文
type JudgeContext struct {
	UserID      string `json:"user_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	PreviousQueries []string `json:"previous_queries,omitempty"`
	UserPreferences map[string]interface{} `json:"user_preferences,omitempty"`
}

// JudgeAgentResponse Judge Agent 响应
type JudgeAgentResponse struct {
	Score           ComplexityScore    `json:"score"`
	Recommendation  RoutingRecommendation `json:"recommendation"`
	ProcessingTimeMs int `json:"processing_time_ms"`
}

// judgeAgentService 智能路由 Agent 服务实现
type judgeAgentService struct {
	db         *ent.Client
	redis      *redis.Client
	cfg        *config.Config
	logger     *zap.Logger
	httpClient *http.Client

	// 规则回退配置
	fallbackRules []FallbackRule
}

// FallbackRule 回退规则
type FallbackRule struct {
	Name        string
	Condition   func(query string) bool
	Platform    string
	Model       string
	Confidence  float64
}

// NewJudgeAgentService 创建智能路由 Agent 服务实例
func NewJudgeAgentService(
	db *ent.Client,
	redis *redis.Client,
	cfg *config.Config,
	logger *zap.Logger,
) JudgeAgentService {
	svc := &judgeAgentService{
		db:     db,
		redis:  redis,
		cfg:    cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.JudgeAgent.TimeoutMs) * time.Millisecond,
		},
	}

	// 初始化回退规则
	svc.initFallbackRules()

	return svc
}

// initFallbackRules 初始化回退规则
func (s *judgeAgentService) initFallbackRules() {
	s.fallbackRules = []FallbackRule{
		{
			Name: "代码生成",
			Condition: func(query string) bool {
				keywords := []string{"代码", "code", "编程", "函数", "实现", "写一个"}
				for _, kw := range keywords {
					if containsIgnoreCase(query, kw) {
						return true
					}
				}
				return false
			},
			Platform:   "claude",
			Model:      "claude-3-5-sonnet-20241022",
			Confidence: 0.8,
		},
		{
			Name: "数学计算",
			Condition: func(query string) bool {
				keywords := []string{"计算", "数学", "方程", "求解", "证明"}
				for _, kw := range keywords {
					if containsIgnoreCase(query, kw) {
						return true
					}
				}
				return false
			},
			Platform:   "openai",
			Model:      "gpt-4o",
			Confidence: 0.75,
		},
		{
			Name: "创意写作",
			Condition: func(query string) bool {
				keywords := []string{"写一篇", "创作", "故事", "小说", "文章"}
				for _, kw := range keywords {
					if containsIgnoreCase(query, kw) {
						return true
					}
				}
				return false
			},
			Platform:   "claude",
			Model:      "claude-3-opus-20240229",
			Confidence: 0.7,
		},
		{
			Name: "简单问答",
			Condition: func(query string) bool {
				// 短查询或简单问题
				return len(query) < 50
			},
			Platform:   "claude",
			Model:      "claude-3-5-haiku-20241022",
			Confidence: 0.9,
		},
	}
}

// ScoreComplexity 复杂度评分
func (s *judgeAgentService) ScoreComplexity(ctx context.Context, query string) (*ComplexityScore, error) {
	// 检查是否启用 Agent
	if !s.cfg.JudgeAgent.Enabled {
		return s.fallbackScore(query), nil
	}

	// 构建请求
	req := &JudgeAgentRequest{
		Query: query,
	}

	// 调用 Agent
	resp, err := s.callAgent(ctx, req)
	if err != nil {
		s.logger.Warn("调用 Judge Agent 失败，使用回退规则",
			zap.Error(err),
			zap.String("query", query))
		return s.fallbackScore(query), nil
	}

	return &resp.Score, nil
}

// ScoreComplexityWithMessages 基于消息的复杂度评分
func (s *judgeAgentService) ScoreComplexityWithMessages(ctx context.Context, messages interface{}) (*ComplexityScore, error) {
	// 检查是否启用 Agent
	if !s.cfg.JudgeAgent.Enabled {
		return &ComplexityScore{
			Score:             50,
			Level:             "medium",
			RecommendedModel:  "claude-3-5-sonnet-20241022",
			RecommendedPlatform: "claude",
		}, nil
	}

	// 构建请求
	req := &JudgeAgentRequest{
		Messages: messages,
	}

	// 调用 Agent
	resp, err := s.callAgent(ctx, req)
	if err != nil {
		s.logger.Warn("调用 Judge Agent 失败，使用默认配置",
			zap.Error(err))
		return &ComplexityScore{
			Score:             50,
			Level:             "medium",
			RecommendedModel:  "claude-3-5-sonnet-20241022",
			RecommendedPlatform: "claude",
		}, nil
	}

	return &resp.Score, nil
}

// GetRoutingRecommendation 获取路由建议
func (s *judgeAgentService) GetRoutingRecommendation(ctx context.Context, score *ComplexityScore) (*RoutingRecommendation, error) {
	// 如果评分中已有推荐，直接返回
	if score.RecommendedModel != "" && score.RecommendedPlatform != "" {
		return &RoutingRecommendation{
			Platform:           score.RecommendedPlatform,
			Model:              score.RecommendedModel,
			PreferStream:       true,
			SuggestedTimeout:   s.getSuggestedTimeout(score),
			SuggestedMaxTokens: s.getSuggestedMaxTokens(score),
			Confidence:         0.85,
			Reason:             score.Reasoning,
		}, nil
	}

	// 根据评分级别返回推荐
	switch score.Level {
	case "simple":
		return &RoutingRecommendation{
			Platform:           "claude",
			Model:              "claude-3-5-haiku-20241022",
			PreferStream:       true,
			SuggestedTimeout:   30,
			SuggestedMaxTokens: 2048,
			Confidence:         0.9,
			Reason:             "简单查询，使用快速模型",
		}, nil
	case "complex":
		return &RoutingRecommendation{
			Platform:           "claude",
			Model:              "claude-3-opus-20240229",
			PreferStream:       true,
			SuggestedTimeout:   300,
			SuggestedMaxTokens: 8192,
			Confidence:         0.85,
			Reason:             "复杂查询，使用高级模型",
			Alternatives: []RoutingAlternative{
				{Platform: "openai", Model: "gpt-4o", Priority: 1, Reason: "备选方案"},
			},
		}, nil
	default: // medium
		return &RoutingRecommendation{
			Platform:           "claude",
			Model:              "claude-3-5-sonnet-20241022",
			PreferStream:       true,
			SuggestedTimeout:   120,
			SuggestedMaxTokens: 4096,
			Confidence:         0.85,
			Reason:             "中等复杂度查询，使用平衡模型",
		}, nil
	}
}

// IsEnabled 检查 Agent 是否启用
func (s *judgeAgentService) IsEnabled() bool {
	return s.cfg.JudgeAgent.Enabled
}

// callAgent 调用 Judge Agent
func (s *judgeAgentService) callAgent(ctx context.Context, req *JudgeAgentRequest) (*JudgeAgentResponse, error) {
	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 构建 HTTP 请求
	url := fmt.Sprintf("%s/v1/judge", s.cfg.JudgeAgent.Addr)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	startTime := time.Now()
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 Agent 失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Agent 返回错误: %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var agentResp JudgeAgentResponse
	if err := json.NewDecoder(resp.Body).Decode(&agentResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 设置处理时间
	agentResp.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())

	s.logger.Debug("Judge Agent 调用成功",
		zap.Int("score", agentResp.Score.Score),
		zap.String("level", agentResp.Score.Level),
		zap.String("recommended_model", agentResp.Score.RecommendedModel),
		zap.Int("processing_time_ms", agentResp.ProcessingTimeMs))

	return &agentResp, nil
}

// fallbackScore 回退评分（当 Agent 不可用时）
func (s *judgeAgentService) fallbackScore(query string) *ComplexityScore {
	// 尝试匹配回退规则
	for _, rule := range s.fallbackRules {
		if rule.Condition(query) {
			return &ComplexityScore{
				Score:              50,
				Level:              "medium",
				RecommendedModel:   rule.Model,
				RecommendedPlatform: rule.Platform,
				Reasoning:          fmt.Sprintf("匹配规则: %s", rule.Name),
			}
		}
	}

	// 默认评分
	score := s.calculateSimpleScore(query)
	return &ComplexityScore{
		Score:              score,
		Level:              s.scoreToLevel(score),
		RecommendedModel:   "claude-3-5-sonnet-20241022",
		RecommendedPlatform: "claude",
		Reasoning:          "基于查询特征的简单评分",
	}
}

// calculateSimpleScore 计算简单评分
func (s *judgeAgentService) calculateSimpleScore(query string) int {
	score := 0

	// 基于长度
	length := len(query)
	if length > 500 {
		score += 20
	} else if length > 200 {
		score += 10
	}

	// 基于关键词
	complexKeywords := []string{"分析", "比较", "评估", "设计", "架构", "优化", "重构"}
	for _, kw := range complexKeywords {
		if containsIgnoreCase(query, kw) {
			score += 15
		}
	}

	// 基于问题类型
	if containsIgnoreCase(query, "为什么") || containsIgnoreCase(query, "如何") {
		score += 10
	}
	if containsIgnoreCase(query, "代码") || containsIgnoreCase(query, "编程") {
		score += 15
	}

	// 限制范围
	if score > 100 {
		score = 100
	}

	return score
}

// scoreToLevel 评分转级别
func (s *judgeAgentService) scoreToLevel(score int) string {
	if score < 30 {
		return "simple"
	} else if score < 70 {
		return "medium"
	}
	return "complex"
}

// getSuggestedTimeout 获取建议超时时间
func (s *judgeAgentService) getSuggestedTimeout(score *ComplexityScore) int {
	switch score.Level {
	case "simple":
		return 30
	case "complex":
		return 300
	default:
		return 120
	}
}

// getSuggestedMaxTokens 获取建议最大 Token 数
func (s *judgeAgentService) getSuggestedMaxTokens(score *ComplexityScore) int {
	switch score.Level {
	case "simple":
		return 2048
	case "complex":
		return 8192
	default:
		return 4096
	}
}

// containsIgnoreCase 忽略大小写包含检查
func containsIgnoreCase(s, substr string) bool {
	sLower := make([]byte, len(s))
	substrLower := make([]byte, len(substr))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		sLower[i] = c
	}
	for i := 0; i < len(substr); i++ {
		c := substr[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		substrLower[i] = c
	}
	return contains(string(sLower), string(substrLower))
}

// contains 字符串包含检查
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BatchScoreComplexity 批量复杂度评分
func (s *judgeAgentService) BatchScoreComplexity(ctx context.Context, queries []string) ([]*ComplexityScore, error) {
	results := make([]*ComplexityScore, len(queries))

	for i, query := range queries {
		score, err := s.ScoreComplexity(ctx, query)
		if err != nil {
			results[i] = s.fallbackScore(query)
		} else {
			results[i] = score
		}
	}

	return results, nil
}

// GetModelRecommendation 获取模型推荐
func (s *judgeAgentService) GetModelRecommendation(ctx context.Context, query string, availableModels []string) (string, error) {
	// 获取复杂度评分
	score, err := s.ScoreComplexity(ctx, query)
	if err != nil {
		return availableModels[0], nil
	}

	// 根据评分选择模型
	for _, model := range availableModels {
		if s.isModelSuitable(model, score) {
			return model, nil
		}
	}

	// 默认返回第一个可用模型
	return availableModels[0], nil
}

// isModelSuitable 检查模型是否适合
func (s *judgeAgentService) isModelSuitable(model string, score *ComplexityScore) bool {
	// 简单的模型匹配逻辑
	switch score.Level {
	case "simple":
		// 简单任务优先使用快速模型
		return containsIgnoreCase(model, "haiku") || containsIgnoreCase(model, "mini")
	case "complex":
		// 复杂任务使用高级模型
		return containsIgnoreCase(model, "opus") || containsIgnoreCase(model, "gpt-4")
	default:
		// 中等任务使用平衡模型
		return containsIgnoreCase(model, "sonnet") || containsIgnoreCase(model, "gpt-4o")
	}
}

// CacheScore 缓存评分结果
func (s *judgeAgentService) CacheScore(ctx context.Context, queryHash string, score *ComplexityScore) error {
	key := fmt.Sprintf("judge:score:%s", queryHash)
	data, err := json.Marshal(score)
	if err != nil {
		return err
	}
	return s.redis.Set(ctx, key, data, 1*time.Hour).Err()
}

// GetCachedScore 获取缓存的评分
func (s *judgeAgentService) GetCachedScore(ctx context.Context, queryHash string) (*ComplexityScore, error) {
	key := fmt.Sprintf("judge:score:%s", queryHash)
	data, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var score ComplexityScore
	if err := json.Unmarshal(data, &score); err != nil {
		return nil, err
	}

	return &score, nil
}
