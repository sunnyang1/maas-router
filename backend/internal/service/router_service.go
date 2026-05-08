// Package service 业务服务层
// 提供路由服务
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"maas-router/ent"
	"maas-router/internal/config"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// RouterService 路由服务接口
// 处理路由规则的管理和评估
type RouterService interface {
	// GetRules 获取路由规则
	GetRules(ctx context.Context) ([]*ent.RouterRule, error)

	// GetActiveRules 获取启用的路由规则
	GetActiveRules(ctx context.Context) ([]*ent.RouterRule, error)

	// EvaluateRules 评估路由规则
	EvaluateRules(ctx context.Context, req *RouteRequest) (*RouteResult, error)

	// AddRule 添加规则
	AddRule(ctx context.Context, rule *RouterRuleCreate) (*ent.RouterRule, error)

	// UpdateRule 更新规则
	UpdateRule(ctx context.Context, ruleID int64, rule *RouterRuleUpdate) error

	// DeleteRule 删除规则
	DeleteRule(ctx context.Context, ruleID int64) error

	// GetRule 获取单个规则
	GetRule(ctx context.Context, ruleID int64) (*ent.RouterRule, error)

	// EnableRule 启用规则
	EnableRule(ctx context.Context, ruleID int64) error

	// DisableRule 禁用规则
	DisableRule(ctx context.Context, ruleID int64) error

	// ReorderRules 重排规则优先级
	ReorderRules(ctx context.Context, ruleIDs []int64) error
}

// RouteRequest 路由请求
type RouteRequest struct {
	Model       string                 `json:"model"`
	Platform    string                 `json:"platform"`
	UserID      int64                  `json:"user_id"`
	APIKeyID    int64                  `json:"api_key_id"`
	SessionID   string                 `json:"session_id,omitempty"`
	Query       string                 `json:"query,omitempty"`
	Messages    interface{}            `json:"messages,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ClientIP    string                 `json:"client_ip,omitempty"`
}

// RouteResult 路由结果
type RouteResult struct {
	MatchedRule   *ent.RouterRule `json:"matched_rule,omitempty"`
	Platform      string          `json:"platform"`
	Model         string          `json:"model"`
	GroupID       *int64          `json:"group_id,omitempty"`
	AccountID     *int64          `json:"account_id,omitempty"`
	ModelMapping  string          `json:"model_mapping,omitempty"`
	RateMultiplier float64        `json:"rate_multiplier,omitempty"`
	Timeout       int             `json:"timeout,omitempty"`
	MaxTokens     int             `json:"max_tokens,omitempty"`
	Priority      int             `json:"priority"`
	Reason        string          `json:"reason"`
}

// RouterRuleCreate 路由规则创建请求
type RouterRuleCreate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Priority    int                    `json:"priority"`
	Condition   map[string]interface{} `json:"condition"`
	Action      map[string]interface{} `json:"action"`
	IsActive    bool                   `json:"is_active"`
}

// RouterRuleUpdate 路由规则更新请求
type RouterRuleUpdate struct {
	Name        *string                 `json:"name,omitempty"`
	Description *string                 `json:"description,omitempty"`
	Priority    *int                    `json:"priority,omitempty"`
	Condition   map[string]interface{}  `json:"condition,omitempty"`
	Action      map[string]interface{}  `json:"action,omitempty"`
	IsActive    *bool                   `json:"is_active,omitempty"`
}

// RuleCondition 规则条件
type RuleCondition struct {
	// 模型匹配
	Models    []string `json:"models,omitempty"`
	ModelRegex string  `json:"model_regex,omitempty"`

	// 平台匹配
	Platforms []string `json:"platforms,omitempty"`

	// 用户匹配
	UserIDs   []int64  `json:"user_ids,omitempty"`
	UserRoles []string `json:"user_roles,omitempty"`

	// 时间匹配
	TimeRange *TimeRangeCondition `json:"time_range,omitempty"`

	// 负载匹配
	MaxLoadFactor *float64 `json:"max_load_factor,omitempty"`

	// 自定义表达式
	Expression string `json:"expression,omitempty"`
}

// TimeRangeCondition 时间范围条件
type TimeRangeCondition struct {
	StartHour int `json:"start_hour"` // 0-23
	EndHour   int `json:"end_hour"`   // 0-23
	Days      []int `json:"days,omitempty"` // 0=Sunday, 1=Monday, ...
}

// RuleAction 规则动作
type RuleAction struct {
	// 目标平台
	Platform string `json:"platform,omitempty"`

	// 目标模型
	Model string `json:"model,omitempty"`

	// 模型映射
	ModelMapping map[string]string `json:"model_mapping,omitempty"`

	// 目标分组
	GroupID int64 `json:"group_id,omitempty"`

	// 目标账号
	AccountID int64 `json:"account_id,omitempty"`

	// 费率倍率
	RateMultiplier float64 `json:"rate_multiplier,omitempty"`

	// 超时设置
	Timeout int `json:"timeout,omitempty"`

	// 最大 Token
	MaxTokens int `json:"max_tokens,omitempty"`

	// 重试策略
	RetryPolicy *RetryPolicy `json:"retry_policy,omitempty"`
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries int      `json:"max_retries"`
	RetryDelay int      `json:"retry_delay"` // 毫秒
	RetryOn    []string `json:"retry_on"`    // 错误类型
}

// routerService 路由服务实现
type routerService struct {
	db     *ent.Client
	redis  *redis.Client
	cfg    *config.Config
	logger *zap.Logger

	// 规则缓存
	rulesCache []*ent.RouterRule
	cacheTime  time.Time
}

// NewRouterService 创建路由服务实例
func NewRouterService(
	db *ent.Client,
	redis *redis.Client,
	cfg *config.Config,
	logger *zap.Logger,
) RouterService {
	return &routerService{
		db:     db,
		redis:  redis,
		cfg:    cfg,
		logger: logger,
	}
}

// GetRules 获取路由规则
func (s *routerService) GetRules(ctx context.Context) ([]*ent.RouterRule, error) {
	rules, err := s.db.RouterRule.Query().
		Order(ent.Desc(ent.FieldPriority)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询路由规则失败: %w", err)
	}
	return rules, nil
}

// GetActiveRules 获取启用的路由规则
func (s *routerService) GetActiveRules(ctx context.Context) ([]*ent.RouterRule, error) {
	// 检查缓存
	if s.rulesCache != nil && time.Since(s.cacheTime) < 5*time.Minute {
		return s.rulesCache, nil
	}

	// 从数据库查询
	rules, err := s.db.RouterRule.Query().
		Where(ent.RouterRuleIsActive(true)).
		Order(ent.Desc(ent.FieldPriority)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询启用的路由规则失败: %w", err)
	}

	// 更新缓存
	s.rulesCache = rules
	s.cacheTime = time.Now()

	return rules, nil
}

// EvaluateRules 评估路由规则
func (s *routerService) EvaluateRules(ctx context.Context, req *RouteRequest) (*RouteResult, error) {
	// 获取启用的规则
	rules, err := s.GetActiveRules(ctx)
	if err != nil {
		return nil, err
	}

	// 按优先级排序（已在查询时排序）
	// 遍历规则进行匹配
	for _, rule := range rules {
		matched, err := s.matchRule(rule, req)
		if err != nil {
			s.logger.Warn("规则匹配失败",
				zap.Int64("rule_id", rule.ID),
				zap.Error(err))
			continue
		}

		if matched {
			// 解析动作
			result, err := s.parseAction(rule, req)
			if err != nil {
				s.logger.Warn("解析规则动作失败",
					zap.Int64("rule_id", rule.ID),
					zap.Error(err))
				continue
			}

			result.MatchedRule = rule
			result.Reason = fmt.Sprintf("匹配规则: %s (优先级: %d)", rule.Name, rule.Priority)

			s.logger.Debug("路由规则匹配成功",
				zap.Int64("rule_id", rule.ID),
				zap.String("rule_name", rule.Name),
				zap.String("platform", result.Platform),
				zap.String("model", result.Model))

			return result, nil
		}
	}

	// 没有匹配的规则，返回默认路由
	return s.getDefaultRoute(req), nil
}

// AddRule 添加规则
func (s *routerService) AddRule(ctx context.Context, rule *RouterRuleCreate) (*ent.RouterRule, error) {
	// 验证规则
	if err := s.validateRule(rule.Condition, rule.Action); err != nil {
		return nil, fmt.Errorf("规则验证失败: %w", err)
	}

	// 创建规则
	created, err := s.db.RouterRule.Create().
		SetName(rule.Name).
		SetNillableDescription(&rule.Description).
		SetPriority(rule.Priority).
		SetCondition(rule.Condition).
		SetAction(rule.Action).
		SetIsActive(rule.IsActive).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建路由规则失败: %w", err)
	}

	// 清除缓存
	s.invalidateCache()

	s.logger.Info("路由规则创建成功",
		zap.Int64("rule_id", created.ID),
		zap.String("name", rule.Name),
		zap.Int("priority", rule.Priority))

	return created, nil
}

// UpdateRule 更新规则
func (s *routerService) UpdateRule(ctx context.Context, ruleID int64, rule *RouterRuleUpdate) error {
	// 获取现有规则
	existing, err := s.db.RouterRule.Get(ctx, ruleID)
	if err != nil {
		return fmt.Errorf("路由规则不存在: %w", err)
	}

	// 构建更新
	update := s.db.RouterRule.UpdateOneID(ruleID)

	if rule.Name != nil {
		update = update.SetName(*rule.Name)
	}
	if rule.Description != nil {
		update = update.SetDescription(*rule.Description)
	}
	if rule.Priority != nil {
		update = update.SetPriority(*rule.Priority)
	}
	if rule.Condition != nil {
		update = update.SetCondition(rule.Condition)
	}
	if rule.Action != nil {
		update = update.SetAction(rule.Action)
	}
	if rule.IsActive != nil {
		update = update.SetIsActive(*rule.IsActive)
	}

	// 执行更新
	_, err = update.Save(ctx)
	if err != nil {
		return fmt.Errorf("更新路由规则失败: %w", err)
	}

	// 清除缓存
	s.invalidateCache()

	s.logger.Info("路由规则更新成功",
		zap.Int64("rule_id", ruleID))

	return nil
}

// DeleteRule 删除规则
func (s *routerService) DeleteRule(ctx context.Context, ruleID int64) error {
	err := s.db.RouterRule.DeleteOneID(ruleID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除路由规则失败: %w", err)
	}

	// 清除缓存
	s.invalidateCache()

	s.logger.Info("路由规则删除成功",
		zap.Int64("rule_id", ruleID))

	return nil
}

// GetRule 获取单个规则
func (s *routerService) GetRule(ctx context.Context, ruleID int64) (*ent.RouterRule, error) {
	rule, err := s.db.RouterRule.Get(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("获取路由规则失败: %w", err)
	}
	return rule, nil
}

// EnableRule 启用规则
func (s *routerService) EnableRule(ctx context.Context, ruleID int64) error {
	_, err := s.db.RouterRule.UpdateOneID(ruleID).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("启用路由规则失败: %w", err)
	}

	// 清除缓存
	s.invalidateCache()

	s.logger.Info("路由规则已启用",
		zap.Int64("rule_id", ruleID))

	return nil
}

// DisableRule 禁用规则
func (s *routerService) DisableRule(ctx context.Context, ruleID int64) error {
	_, err := s.db.RouterRule.UpdateOneID(ruleID).
		SetIsActive(false).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("禁用路由规则失败: %w", err)
	}

	// 清除缓存
	s.invalidateCache()

	s.logger.Info("路由规则已禁用",
		zap.Int64("rule_id", ruleID))

	return nil
}

// ReorderRules 重排规则优先级
func (s *routerService) ReorderRules(ctx context.Context, ruleIDs []int64) error {
	// 开启事务
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	// 更新每个规则的优先级
	for i, ruleID := range ruleIDs {
		priority := len(ruleIDs) - i // 越靠前优先级越高
		_, err := tx.RouterRule.UpdateOneID(ruleID).
			SetPriority(priority).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("更新规则优先级失败: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 清除缓存
	s.invalidateCache()

	s.logger.Info("路由规则优先级已更新",
		zap.Int("count", len(ruleIDs)))

	return nil
}

// matchRule 匹配规则
func (s *routerService) matchRule(rule *ent.RouterRule, req *RouteRequest) (bool, error) {
	condition := rule.Condition
	if condition == nil {
		return false, nil
	}

	// 检查模型匹配
	if models, ok := condition["models"].([]interface{}); ok && len(models) > 0 {
		modelMatched := false
		for _, m := range models {
			if model, ok := m.(string); ok && model == req.Model {
				modelMatched = true
				break
			}
		}
		if !modelMatched {
			return false, nil
		}
	}

	// 检查平台匹配
	if platforms, ok := condition["platforms"].([]interface{}); ok && len(platforms) > 0 {
		platformMatched := false
		for _, p := range platforms {
			if platform, ok := p.(string); ok && platform == req.Platform {
				platformMatched = true
				break
			}
		}
		if !platformMatched {
			return false, nil
		}
	}

	// 检查用户匹配
	if userIDs, ok := condition["user_ids"].([]interface{}); ok && len(userIDs) > 0 {
		userMatched := false
		for _, u := range userIDs {
			if userID, ok := u.(float64); ok && int64(userID) == req.UserID {
				userMatched = true
				break
			}
		}
		if !userMatched {
			return false, nil
		}
	}

	// 检查时间范围
	if timeRange, ok := condition["time_range"].(map[string]interface{}); ok {
		if !s.matchTimeRange(timeRange) {
			return false, nil
		}
	}

	return true, nil
}

// matchTimeRange 匹配时间范围
func (s *routerService) matchTimeRange(timeRange map[string]interface{}) bool {
	now := time.Now()

	// 检查小时范围
	if startHour, ok := timeRange["start_hour"].(float64); ok {
		if endHour, ok := timeRange["end_hour"].(float64); ok {
			hour := now.Hour()
			if hour < int(startHour) || hour >= int(endHour) {
				return false
			}
		}
	}

	// 检查星期几
	if days, ok := timeRange["days"].([]interface{}); ok && len(days) > 0 {
		dayMatched := false
		weekday := int(now.Weekday())
		for _, d := range days {
			if day, ok := d.(float64); ok && int(day) == weekday {
				dayMatched = true
				break
			}
		}
		if !dayMatched {
			return false
		}
	}

	return true
}

// parseAction 解析规则动作
func (s *routerService) parseAction(rule *ent.RouterRule, req *RouteRequest) (*RouteResult, error) {
	action := rule.Action
	if action == nil {
		return &RouteResult{
			Platform: req.Platform,
			Model:    req.Model,
			Priority: rule.Priority,
		}, nil
	}

	result := &RouteResult{
		Priority: rule.Priority,
	}

	// 解析平台
	if platform, ok := action["platform"].(string); ok {
		result.Platform = platform
	} else {
		result.Platform = req.Platform
	}

	// 解析模型
	if model, ok := action["model"].(string); ok {
		result.Model = model
	} else {
		result.Model = req.Model
	}

	// 解析模型映射
	if modelMapping, ok := action["model_mapping"].(map[string]interface{}); ok {
		if mapped, exists := modelMapping[req.Model]; exists {
			if mappedModel, ok := mapped.(string); ok {
				result.ModelMapping = mappedModel
				result.Model = mappedModel
			}
		}
	}

	// 解析分组 ID
	if groupID, ok := action["group_id"].(float64); ok {
		gid := int64(groupID)
		result.GroupID = &gid
	}

	// 解析账号 ID
	if accountID, ok := action["account_id"].(float64); ok {
		aid := int64(accountID)
		result.AccountID = &aid
	}

	// 解析费率倍率
	if rateMultiplier, ok := action["rate_multiplier"].(float64); ok {
		result.RateMultiplier = rateMultiplier
	}

	// 解析超时
	if timeout, ok := action["timeout"].(float64); ok {
		result.Timeout = int(timeout)
	}

	// 解析最大 Token
	if maxTokens, ok := action["max_tokens"].(float64); ok {
		result.MaxTokens = int(maxTokens)
	}

	return result, nil
}

// getDefaultRoute 获取默认路由
func (s *routerService) getDefaultRoute(req *RouteRequest) *RouteResult {
	// 默认路由逻辑
	platform := req.Platform
	if platform == "" {
		// 根据模型推断平台
		platform = s.inferPlatform(req.Model)
	}

	return &RouteResult{
		Platform: platform,
		Model:    req.Model,
		Priority: 0,
		Reason:   "默认路由",
	}
}

// inferPlatform 根据模型推断平台
func (s *routerService) inferPlatform(model string) string {
	// Claude 模型
	if containsAny(model, []string{"claude", "claude-3", "claude-3.5"}) {
		return "claude"
	}

	// OpenAI 模型
	if containsAny(model, []string{"gpt", "o1", "dall-e", "whisper", "tts", "embedding"}) {
		return "openai"
	}

	// Gemini 模型
	if containsAny(model, []string{"gemini", "palm"}) {
		return "gemini"
	}

	// 默认使用 Claude
	return "claude"
}

// validateRule 验证规则
func (s *routerService) validateRule(condition, action map[string]interface{}) error {
	// 验证条件
	if condition != nil {
		// 检查 models 格式
		if models, ok := condition["models"]; ok {
			if _, ok := models.([]interface{}); !ok {
				return fmt.Errorf("models 必须是数组")
			}
		}

		// 检查 platforms 格式
		if platforms, ok := condition["platforms"]; ok {
			if _, ok := platforms.([]interface{}); !ok {
				return fmt.Errorf("platforms 必须是数组")
			}
		}
	}

	// 验证动作
	if action != nil {
		// 检查 platform 格式
		if platform, ok := action["platform"]; ok {
			if _, ok := platform.(string); !ok {
				return fmt.Errorf("platform 必须是字符串")
			}
		}

		// 检查 model 格式
		if model, ok := action["model"]; ok {
			if _, ok := model.(string); !ok {
				return fmt.Errorf("model 必须是字符串")
			}
		}
	}

	return nil
}

// invalidateCache 清除缓存
func (s *routerService) invalidateCache() {
	s.rulesCache = nil
	s.cacheTime = time.Time{}
}

// containsAny 检查字符串是否包含任意子串
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

// GetRulesByModel 获取指定模型的规则
func (s *routerService) GetRulesByModel(ctx context.Context, model string) ([]*ent.RouterRule, error) {
	allRules, err := s.GetActiveRules(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*ent.RouterRule
	for _, rule := range allRules {
		if models, ok := rule.Condition["models"].([]interface{}); ok {
			for _, m := range models {
				if mStr, ok := m.(string); ok && (mStr == model || mStr == "*") {
					filtered = append(filtered, rule)
					break
				}
			}
		} else {
			// 没有模型限制的规则也包含
			filtered = append(filtered, rule)
		}
	}

	return filtered, nil
}

// ExportRules 导出规则
func (s *routerService) ExportRules(ctx context.Context) ([]byte, error) {
	rules, err := s.GetRules(ctx)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(rules, "", "  ")
}

// ImportRules 导入规则
func (s *routerService) ImportRules(ctx context.Context, data []byte, overwrite bool) error {
	var rules []*ent.RouterRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("解析规则失败: %w", err)
	}

	// 如果覆盖，先删除现有规则
	if overwrite {
		_, err := s.db.RouterRule.Delete().Exec(ctx)
		if err != nil {
			return fmt.Errorf("删除现有规则失败: %w", err)
		}
	}

	// 导入规则
	for _, rule := range rules {
		_, err := s.db.RouterRule.Create().
			SetName(rule.Name).
			SetDescription(rule.Description).
			SetPriority(rule.Priority).
			SetCondition(rule.Condition).
			SetAction(rule.Action).
			SetIsActive(rule.IsActive).
			Save(ctx)
		if err != nil {
			s.logger.Warn("导入规则失败",
				zap.String("name", rule.Name),
				zap.Error(err))
		}
	}

	// 清除缓存
	s.invalidateCache()

	s.logger.Info("规则导入完成",
		zap.Int("total", len(rules)))

	return nil
}

// GetRuleStats 获取规则统计
func (s *routerService) GetRuleStats(ctx context.Context) (map[string]interface{}, error) {
	rules, err := s.GetRules(ctx)
	if err != nil {
		return nil, err
	}

	activeCount := 0
	inactiveCount := 0
	for _, rule := range rules {
		if rule.IsActive {
			activeCount++
		} else {
			inactiveCount++
		}
	}

	return map[string]interface{}{
		"total":    len(rules),
		"active":   activeCount,
		"inactive": inactiveCount,
	}, nil
}

// SortRulesByPriority 按优先级排序规则
func SortRulesByPriority(rules []*ent.RouterRule) {
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})
}
