// Package complexity 提供智能推理资源优化引擎的核心功能
// 包含复杂度分析、特征提取、分类器和模型路由推荐
package complexity

// ScoreLevel 复杂度评分级别常量
type ScoreLevel string

const (
	// ScoreLevelSimple 简单级别
	ScoreLevelSimple ScoreLevel = "simple"
	// ScoreLevelMedium 中等级别
	ScoreLevelMedium ScoreLevel = "medium"
	// ScoreLevelComplex 复杂级别
	ScoreLevelComplex ScoreLevel = "complex"
	// ScoreLevelExpert 专家级别
	ScoreLevelExpert ScoreLevel = "expert"
)

// TierName 模型层级名称常量
type TierName string

const (
	// TierNameEconomy 经济层级
	TierNameEconomy TierName = "economy"
	// TierNameStandard 标准层级
	TierNameStandard TierName = "standard"
	// TierNameAdvanced 高级层级
	TierNameAdvanced TierName = "advanced"
	// TierNamePremium 旗舰层级
	TierNamePremium TierName = "premium"
)

// ComplexityProfile 复杂度分析结果
type ComplexityProfile struct {
	// Score 加权总分 [0, 1]
	Score float64 `json:"score"`
	// Level 复杂度级别: simple, medium, complex, expert
	Level ScoreLevel `json:"level"`
	// Confidence 置信度 [0, 1]，基于特征一致性
	Confidence float64 `json:"confidence"`

	// 五个特征分解分值
	LexicalScore       float64 `json:"lexical_score"`
	StructuralScore    float64 `json:"structural_score"`
	DomainScore        float64 `json:"domain_score"`
	ConversationalScore float64 `json:"conversational_score"`
	TaskTypeScore      float64 `json:"task_type_score"`

	// 推荐路由信息
	RecommendedTier      TierName `json:"recommended_tier"`
	RecommendedModel     string   `json:"recommended_model"`
	FallbackModel        string   `json:"fallback_model,omitempty"`

	// 成本与质量评估
	EstimatedCost   float64 `json:"estimated_cost"`
	CostSavingRatio float64 `json:"cost_saving_ratio"`
	QualityRisk     string  `json:"quality_risk"` // low, medium, high
	NeedsUpgrade    bool    `json:"needs_upgrade"`
}

// FeatureVector 多维特征向量
type FeatureVector struct {
	// 词法特征 (4个字段)
	TokenCount      int     `json:"token_count"`       // token 计数
	VocabularyDiversity float64 `json:"vocabulary_diversity"` // 词汇多样性 (type-token ratio)
	AverageWordLength   float64 `json:"average_word_length"`  // 平均词长
	TechnicalTermCount  int     `json:"technical_term_count"` // 专业术语数量

	// 结构特征 (4个字段)
	SentenceCount     int     `json:"sentence_count"`      // 句子计数
	QuestionDensity   float64 `json:"question_density"`    // 问题密度
	HasNestedCondition bool   `json:"has_nested_condition"` // 是否包含嵌套条件
	MultipartRequest  bool    `json:"multipart_request"`   // 是否多部分请求

	// 领域特征 (4个字段)
	HasCodeBlock    bool `json:"has_code_block"`     // 是否包含代码块
	HasMathSymbols  bool `json:"has_math_symbols"`   // 是否包含数学符号
	HasTableData    bool `json:"has_table_data"`     // 是否包含表格数据
	HasStructuredData bool `json:"has_structured_data"` // 是否包含结构化数据

	// 对话特征 (4个字段)
	HistoryLength   int `json:"history_length"`    // 历史消息长度
	TurnCount       int `json:"turn_count"`        // 轮次计数
	HasToolCall     bool `json:"has_tool_call"`     // 是否包含工具调用
	ContextSize     int `json:"context_size"`      // 上下文大小（字符数）

	// 任务类型特征 (2个字段)
	TaskComplexity float64 `json:"task_complexity"` // 任务复杂度分值 [0, 1]
	TaskCategory   string  `json:"task_category"`   // 任务类别: low, medium, high
}

// ComplexityStats 复杂度分析统计数据
type ComplexityStats struct {
	// 总请求数
	TotalRequests int64 `json:"total_requests"`
	// 平均分值
	AvgScore float64 `json:"avg_score"`
	// 级别分布
	LevelDistribution map[ScoreLevel]int64 `json:"level_distribution"`
	// 层级分布
	TierDistribution map[TierName]int64 `json:"tier_distribution"`
	// 模型分布
	ModelDistribution map[string]int64 `json:"model_distribution"`
	// 平均成本节省比例
	AvgCostSaving float64 `json:"avg_cost_saving"`
	// 升级率
	UpgradeRate float64 `json:"upgrade_rate"`
	// 质量通过率
	QualityPassRate float64 `json:"quality_pass_rate"`
}

// AnalyzeRequest 复杂度分析请求
type AnalyzeRequest struct {
	// 当前请求使用的模型
	Model string `json:"model"`
	// 对话消息列表
	Messages []Message `json:"messages"`
	// 系统提示词
	System string `json:"system,omitempty"`
	// 最大 token 数
	MaxTokens int `json:"max_tokens,omitempty"`
	// 是否流式请求
	Stream bool `json:"stream,omitempty"`
}

// Message 对话消息
type Message struct {
	// 角色: user, assistant, system
	Role    string `json:"role"`
	// 消息内容
	Content string `json:"content"`
}
