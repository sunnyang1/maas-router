package complexity

import (
	"math"
	"sort"

	"maas-router/internal/config"

	"go.uber.org/zap"
)

// 默认特征权重
const (
	DefaultWeightLexical       = 0.15
	DefaultWeightStructural    = 0.20
	DefaultWeightDomain        = 0.25
	DefaultWeightConversational = 0.15
	DefaultWeightTaskType      = 0.25
)

// ComplexityClassifier 复杂度分类器
// 基于特征向量计算加权总分，映射到复杂度级别，并选择最优模型
type ComplexityClassifier struct {
	extractor *FeatureExtractor
	tiers     []config.ModelTierConfig
	weights   map[string]float64
	logger    *zap.Logger
}

// NewComplexityClassifier 创建复杂度分类器实例
func NewComplexityClassifier(
	extractor *FeatureExtractor,
	tiers []config.ModelTierConfig,
	weights map[string]float64,
	logger *zap.Logger,
) *ComplexityClassifier {
	// 设置默认权重
	if weights == nil {
		weights = map[string]float64{
			"lexical":       DefaultWeightLexical,
			"structural":    DefaultWeightStructural,
			"domain":        DefaultWeightDomain,
			"conversational": DefaultWeightConversational,
			"taskType":      DefaultWeightTaskType,
		}
	}

	// 按 threshold 升序排列 tiers
	sortedTiers := make([]config.ModelTierConfig, len(tiers))
	copy(sortedTiers, tiers)
	sort.Slice(sortedTiers, func(i, j int) bool {
		return sortedTiers[i].Threshold < sortedTiers[j].Threshold
	})

	return &ComplexityClassifier{
		extractor: extractor,
		tiers:     sortedTiers,
		weights:   weights,
		logger:    logger,
	}
}

// Classify 主方法：对特征向量进行分类，返回复杂度分析结果
func (c *ComplexityClassifier) Classify(features *FeatureVector) *ComplexityProfile {
	// 1. 计算加权总分
	score := c.calculateWeightedScore(features)

	// 2. 将 score 映射到 level
	level := c.scoreToLevel(score)

	// 3. 选择最优模型
	tier, model, costSavingRatio := c.selectOptimalModel(score)

	// 4. 评估质量风险
	qualityRisk := c.assessQualityRisk(score, tier)

	// 5. 计算置信度
	confidence := c.calculateConfidence(features)

	// 6. 判断是否需要升级
	needsUpgrade := c.determineNeedsUpgrade(score, tier)

	profile := &ComplexityProfile{
		Score:               score,
		Level:               level,
		Confidence:          confidence,
		LexicalScore:        c.weights["lexical"] * features.TaskComplexity, // 归一化后的词法贡献
		StructuralScore:     c.weights["structural"],
		DomainScore:         c.weights["domain"],
		ConversationalScore: c.weights["conversational"],
		TaskTypeScore:       c.weights["taskType"],
		RecommendedTier:     TierName(tier.Name),
		RecommendedModel:    model,
		CostSavingRatio:     costSavingRatio,
		QualityRisk:         qualityRisk,
		NeedsUpgrade:        needsUpgrade,
	}

	// 设置回退模型
	if tier.FallbackModel != "" {
		profile.FallbackModel = tier.FallbackModel
	}

	// 估算成本
	profile.EstimatedCost = tier.CostPerToken

	c.logger.Debug("复杂度分类完成",
		zap.Float64("score", score),
		zap.String("level", string(level)),
		zap.String("tier", tier.Name),
		zap.String("model", model),
		zap.Float64("confidence", confidence),
		zap.String("quality_risk", qualityRisk))

	return profile
}

// calculateWeightedScore 计算加权总分
// score = Σ(w_i * feature_i)
func (c *ComplexityClassifier) calculateWeightedScore(features *FeatureVector) float64 {
	// 计算各维度的归一化分值
	lexicalScore := c.normalizeLexicalScore(features)
	structuralScore := c.normalizeStructuralScore(features)
	domainScore := c.normalizeDomainScore(features)
	conversationalScore := c.normalizeConversationalScore(features)
	taskTypeScore := features.TaskComplexity

	// 加权求和
	score := c.weights["lexical"]*lexicalScore +
		c.weights["structural"]*structuralScore +
		c.weights["domain"]*domainScore +
		c.weights["conversational"]*conversationalScore +
		c.weights["taskType"]*taskTypeScore

	return clampScore(score)
}

// normalizeLexicalScore 归一化词法特征到 [0, 1]
func (c *ComplexityClassifier) normalizeLexicalScore(features *FeatureVector) float64 {
	tokenScore := clampScore(float64(features.TokenCount) / 500.0)
	diversityScore := features.VocabularyDiversity
	wordLenScore := clampScore(features.AverageWordLength / 10.0)
	termScore := clampScore(float64(features.TechnicalTermCount) / 10.0)

	return 0.3*tokenScore + 0.3*diversityScore + 0.2*wordLenScore + 0.2*termScore
}

// normalizeStructuralScore 归一化结构特征到 [0, 1]
func (c *ComplexityClassifier) normalizeStructuralScore(features *FeatureVector) float64 {
	sentenceScore := clampScore(float64(features.SentenceCount) / 20.0)
	questionScore := features.QuestionDensity
	nestedScore := 0.0
	if features.HasNestedCondition {
		nestedScore = 1.0
	}
	multipartScore := 0.0
	if features.MultipartRequest {
		multipartScore = 1.0
	}

	return 0.2*sentenceScore + 0.3*questionScore + 0.25*nestedScore + 0.25*multipartScore
}

// normalizeDomainScore 归一化领域特征到 [0, 1]
func (c *ComplexityClassifier) normalizeDomainScore(features *FeatureVector) float64 {
	codeScore := 0.0
	if features.HasCodeBlock {
		codeScore = 1.0
	}
	mathScore := 0.0
	if features.HasMathSymbols {
		mathScore = 1.0
	}
	tableScore := 0.0
	if features.HasTableData {
		tableScore = 1.0
	}
	structuredScore := 0.0
	if features.HasStructuredData {
		structuredScore = 1.0
	}

	return 0.3*codeScore + 0.25*mathScore + 0.2*tableScore + 0.25*structuredScore
}

// normalizeConversationalScore 归一化对话特征到 [0, 1]
func (c *ComplexityClassifier) normalizeConversationalScore(features *FeatureVector) float64 {
	historyScore := clampScore(float64(features.HistoryLength) / 20.0)
	turnScore := clampScore(float64(features.TurnCount) / 10.0)
	toolScore := 0.0
	if features.HasToolCall {
		toolScore = 1.0
	}
	contextScore := clampScore(float64(features.ContextSize) / 50000.0)

	return 0.25*historyScore + 0.25*turnScore + 0.25*toolScore + 0.25*contextScore
}

// scoreToLevel 将分数映射到复杂度级别
// [0, 0.25) = simple, [0.25, 0.5) = medium, [0.5, 0.75) = complex, [0.75, 1.0] = expert
func (c *ComplexityClassifier) scoreToLevel(score float64) ScoreLevel {
	if score < 0.25 {
		return ScoreLevelSimple
	} else if score < 0.5 {
		return ScoreLevelMedium
	} else if score < 0.75 {
		return ScoreLevelComplex
	}
	return ScoreLevelExpert
}

// selectOptimalModel 选择最优模型
// 遍历 tiers（按 threshold 升序），找到 score <= threshold 的最便宜 tier
// 返回 tier 配置、模型名称、成本节省比例
func (c *ComplexityClassifier) selectOptimalModel(score float64) (config.ModelTierConfig, string, float64) {
	if len(c.tiers) == 0 {
		// 没有配置层级时返回默认值
		return config.ModelTierConfig{
			Name:      "standard",
			Model:     "claude-3-5-sonnet-20241022",
			Threshold: 1.0,
		}, "claude-3-5-sonnet-20241022", 0
	}

	// 找到 score <= threshold 的最便宜 tier（即 threshold 最小的满足条件的 tier）
	var selectedTier *config.ModelTierConfig
	for i := range c.tiers {
		if score <= c.tiers[i].Threshold {
			selectedTier = &c.tiers[i]
			break
		}
	}

	// 如果没有找到合适的 tier（score 太高），使用最贵的 tier
	if selectedTier == nil {
		selectedTier = &c.tiers[len(c.tiers)-1]
	}

	// 计算成本节省比例（相对于最贵的 tier）
	costSavingRatio := 0.0
	if len(c.tiers) > 0 {
		maxCost := c.tiers[len(c.tiers)-1].CostPerToken
		if maxCost > 0 {
			costSavingRatio = 1.0 - (selectedTier.CostPerToken / maxCost)
			if costSavingRatio < 0 {
				costSavingRatio = 0
			}
		}
	}

	return *selectedTier, selectedTier.Model, costSavingRatio
}

// assessQualityRisk 评估质量风险
// score 在 tier threshold 的 80% 以上 → "medium"，90% 以上 → "high"，否则 "low"
func (c *ComplexityClassifier) assessQualityRisk(score float64, tier config.ModelTierConfig) string {
	if tier.Threshold <= 0 {
		return "low"
	}

	ratio := score / tier.Threshold

	if ratio >= 0.9 {
		return "high"
	} else if ratio >= 0.8 {
		return "medium"
	}
	return "low"
}

// calculateConfidence 计算置信度
// 基于特征一致性：各维度分值的方差越小，置信度越高
func (c *ComplexityClassifier) calculateConfidence(features *FeatureVector) float64 {
	// 获取各维度归一化分值
	scores := []float64{
		c.normalizeLexicalScore(features),
		c.normalizeStructuralScore(features),
		c.normalizeDomainScore(features),
		c.normalizeConversationalScore(features),
		features.TaskComplexity,
	}

	// 计算均值
	mean := 0.0
	for _, s := range scores {
		mean += s
	}
	mean /= float64(len(scores))

	// 计算方差
	variance := 0.0
	for _, s := range scores {
		diff := s - mean
		variance += diff * diff
	}
	variance /= float64(len(scores))

	// 标准差
	stdDev := math.Sqrt(variance)

	// 标准差越小，置信度越高
	// stdDev=0 → confidence=1.0, stdDev>=0.5 → confidence=0.3
	confidence := 1.0 - stdDev*1.4
	if confidence < 0.3 {
		confidence = 0.3
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// determineNeedsUpgrade 判断是否需要升级模型
// 当质量风险为 high 且有更高层级可用时返回 true
func (c *ComplexityClassifier) determineNeedsUpgrade(score float64, currentTier config.ModelTierConfig) bool {
	// 质量风险低则不需要升级
	risk := c.assessQualityRisk(score, currentTier)
	if risk != "high" {
		return false
	}

	// 检查是否有更高层级可用
	for _, tier := range c.tiers {
		if tier.Threshold > currentTier.Threshold {
			return true
		}
	}

	return false
}
