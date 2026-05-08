// Package handler 提供 MaaS-Router 的 HTTP 处理器
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"maas-router/internal/complexity"
	"maas-router/internal/service"
)

// ComplexityHandler 复杂度分析 Handler
// 处理复杂度分析、统计、反馈和模型分级配置等操作
type ComplexityHandler struct {
	complexityService service.ComplexityService
	logger            *zap.Logger
}

// NewComplexityHandler 创建复杂度分析 Handler
func NewComplexityHandler(complexityService service.ComplexityService, logger *zap.Logger) *ComplexityHandler {
	return &ComplexityHandler{
		complexityService: complexityService,
		logger:            logger,
	}
}

// AnalyzeRequest 复杂度分析请求体
type AnalyzeRequest struct {
	Model    string                   `json:"model" binding:"required"`
	Messages []complexity.Message     `json:"messages" binding:"required,min=1"`
	System   string                   `json:"system,omitempty"`
	MaxTokens int                     `json:"max_tokens,omitempty"`
	Stream   bool                     `json:"stream,omitempty"`
}

// AnalyzeResponse 复杂度分析响应体
type AnalyzeResponse struct {
	Score               float64  `json:"score"`
	Level               string   `json:"level"`
	Confidence          float64  `json:"confidence"`
	LexicalScore        float64  `json:"lexical_score"`
	StructuralScore     float64  `json:"structural_score"`
	DomainScore         float64  `json:"domain_score"`
	ConversationalScore float64  `json:"conversational_score"`
	TaskTypeScore       float64  `json:"task_type_score"`
	RecommendedTier     string   `json:"recommended_tier"`
	RecommendedModel    string   `json:"recommended_model"`
	FallbackModel       string   `json:"fallback_model,omitempty"`
	EstimatedCost       float64  `json:"estimated_cost"`
	CostSavingRatio     float64  `json:"cost_saving_ratio"`
	QualityRisk         string   `json:"quality_risk"`
	NeedsUpgrade        bool     `json:"needs_upgrade"`
}

// FeedbackRequest 质量反馈请求体
type FeedbackRequest struct {
	RequestID string `json:"request_id" binding:"required"`
	Satisfied bool   `json:"satisfied"`
}

// Analyze 处理 POST /v1/complexity/analyze
// 分析请求复杂度，返回推荐的路由配置
func (h *ComplexityHandler) Analyze(c *gin.Context) {
	// 解析请求体
	var req AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	// 构建分析请求
	analyzeReq := &complexity.AnalyzeRequest{
		Model:     req.Model,
		Messages:  req.Messages,
		System:    req.System,
		MaxTokens: req.MaxTokens,
		Stream:    req.Stream,
	}

	// 调用带缓存的分析服务
	profile, err := h.complexityService.AnalyzeWithCache(c.Request.Context(), analyzeReq)
	if err != nil {
		h.logger.Error("复杂度分析失败",
			zap.Error(err),
			zap.String("model", req.Model))
		ErrorResponse(c, http.StatusInternalServerError, "ANALYSIS_FAILED", "复杂度分析失败")
		return
	}

	// 构建响应
	resp := AnalyzeResponse{
		Score:               profile.Score,
		Level:               string(profile.Level),
		Confidence:          profile.Confidence,
		LexicalScore:        profile.LexicalScore,
		StructuralScore:     profile.StructuralScore,
		DomainScore:         profile.DomainScore,
		ConversationalScore: profile.ConversationalScore,
		TaskTypeScore:       profile.TaskTypeScore,
		RecommendedTier:     string(profile.RecommendedTier),
		RecommendedModel:    profile.RecommendedModel,
		FallbackModel:       profile.FallbackModel,
		EstimatedCost:       profile.EstimatedCost,
		CostSavingRatio:     profile.CostSavingRatio,
		QualityRisk:         profile.QualityRisk,
		NeedsUpgrade:        profile.NeedsUpgrade,
	}

	c.JSON(http.StatusOK, gin.H{
		"data": resp,
	})
}

// Stats 处理 GET /v1/complexity/stats
// 获取复杂度分析统计数据，支持 period 和 user_id 查询参数
func (h *ComplexityHandler) Stats(c *gin.Context) {
	// 检查服务是否启用
	if !h.complexityService.IsEnabled() {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"enabled": false,
				"message": "复杂度分析服务未启用",
			},
		})
		return
	}

	// 获取统计数据
	stats, err := h.complexityService.GetStats(c.Request.Context())
	if err != nil {
		h.logger.Error("获取复杂度统计失败", zap.Error(err))
		ErrorResponse(c, http.StatusInternalServerError, "STATS_FAILED", "获取统计数据失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}

// Feedback 处理 POST /v1/complexity/feedback
// 记录复杂度分析的质量反馈
func (h *ComplexityHandler) Feedback(c *gin.Context) {
	// 解析请求体
	var req FeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	// 构建默认 profile 用于记录反馈
	profile := &complexity.ComplexityProfile{
		Score:           0,
		Level:           complexity.ScoreLevelMedium,
		RecommendedTier: complexity.TierNameStandard,
		RecommendedModel: "",
		QualityRisk:     "low",
		Confidence:      0,
	}

	// 记录反馈
	err := h.complexityService.RecordFeedback(c.Request.Context(), req.RequestID, profile, req.Satisfied)
	if err != nil {
		h.logger.Error("记录质量反馈失败",
			zap.Error(err),
			zap.String("request_id", req.RequestID))
		ErrorResponse(c, http.StatusInternalServerError, "FEEDBACK_FAILED", "记录质量反馈失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "反馈记录成功",
	})
}

// ModelTiers 处理 GET /v1/complexity/tiers
// 返回模型分级配置
func (h *ComplexityHandler) ModelTiers(c *gin.Context) {
	// 检查服务是否启用
	if !h.complexityService.IsEnabled() {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"enabled": false,
				"message": "复杂度分析服务未启用",
			},
		})
		return
	}

	// 返回模型分级配置
	tiers := gin.H{
		"economy": gin.H{
			"tier":       "economy",
			"description": "经济层级，适用于简单任务",
			"models":     []string{"claude-3-5-haiku-20241022"},
			"max_score":  0.3,
		},
		"standard": gin.H{
			"tier":       "standard",
			"description": "标准层级，适用于中等复杂度任务",
			"models":     []string{"claude-3-5-sonnet-20241022"},
			"max_score":  0.6,
		},
		"advanced": gin.H{
			"tier":       "advanced",
			"description": "高级层级，适用于复杂任务",
			"models":     []string{"claude-3-5-sonnet-20241022"},
			"max_score":  0.8,
		},
		"premium": gin.H{
			"tier":       "premium",
			"description": "旗舰层级，适用于专家级任务",
			"models":     []string{"claude-3-opus-20240229"},
			"max_score":  1.0,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tiers,
	})
}
