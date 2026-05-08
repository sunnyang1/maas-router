package complexity

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"maas-router/internal/config"

	"go.uber.org/zap"
)

// FeatureExtractor 多维特征提取器
// 从请求中提取词法、结构、领域、对话和任务类型五个维度的特征
type FeatureExtractor struct {
	cfg    config.FeatureConfig
	logger *zap.Logger
}

// NewFeatureExtractor 创建特征提取器实例
func NewFeatureExtractor(cfg config.FeatureConfig, logger *zap.Logger) *FeatureExtractor {
	return &FeatureExtractor{
		cfg:    cfg,
		logger: logger,
	}
}

// Extract 主方法：从分析请求中提取完整特征向量
func (e *FeatureExtractor) Extract(req *AnalyzeRequest) *FeatureVector {
	fv := &FeatureVector{}

	// 合并所有文本内容用于分析
	fullText := e.collectText(req)

	// 提取各维度特征
	e.extractLexical(fullText, fv)
	e.extractStructural(fullText, fv)
	e.extractDomain(fullText, fv)
	e.extractConversational(req, fv)
	e.extractTaskType(fullText, fv)

	e.logger.Debug("特征提取完成",
		zap.Int("token_count", fv.TokenCount),
		zap.Float64("vocabulary_diversity", fv.VocabularyDiversity),
		zap.Bool("has_code_block", fv.HasCodeBlock),
		zap.String("task_category", fv.TaskCategory))

	return fv
}

// collectText 合并请求中所有文本内容
func (e *FeatureExtractor) collectText(req *AnalyzeRequest) string {
	var sb strings.Builder
	if req.System != "" {
		sb.WriteString(req.System)
		sb.WriteString(" ")
	}
	for _, msg := range req.Messages {
		sb.WriteString(msg.Content)
		sb.WriteString(" ")
	}
	return sb.String()
}

// extractLexical 提取词法特征
// 返回词法维度的综合分值 [0, 1]
func (e *FeatureExtractor) extractLexical(text string, fv *FeatureVector) float64 {
	tokens := e.tokenize(text)
	fv.TokenCount = len(tokens)

	if len(tokens) == 0 {
		fv.VocabularyDiversity = 0
		fv.AverageWordLength = 0
		fv.TechnicalTermCount = 0
		return 0
	}

	// 计算词汇多样性 (type-token ratio)
	uniqueTokens := make(map[string]struct{})
	totalWordLen := 0
	for _, token := range tokens {
		uniqueTokens[token] = struct{}{}
		totalWordLen += utf8.RuneCountInString(token)
	}
	fv.VocabularyDiversity = float64(len(uniqueTokens)) / float64(len(tokens))
	fv.AverageWordLength = float64(totalWordLen) / float64(len(tokens))

	// 检测专业术语
	fv.TechnicalTermCount = e.countTechnicalTerms(text)

	// 计算综合分值: token数归一化(0.3) + 词汇多样性(0.3) + 平均词长归一化(0.2) + 术语密度(0.2)
	tokenScore := e.normalizeFloat(float64(fv.TokenCount), 0, 500)
	diversityScore := fv.VocabularyDiversity
	wordLenScore := e.normalizeFloat(fv.AverageWordLength, 2, 10)
	termDensity := e.normalizeFloat(float64(fv.TechnicalTermCount), 0, 10)

	score := 0.3*tokenScore + 0.3*diversityScore + 0.2*wordLenScore + 0.2*termDensity
	return clampScore(score)
}

// tokenize 按空格和中文字符分割文本为 token 列表
func (e *FeatureExtractor) tokenize(text string) []string {
	var tokens []string
	var currentToken strings.Builder

	for _, r := range text {
		if unicode.IsSpace(r) {
			if currentToken.Len() > 0 {
				tokens = append(tokens, currentToken.String())
				currentToken.Reset()
			}
		} else if unicode.Is(unicode.Han, r) {
			// 中文字符单独作为一个 token
			if currentToken.Len() > 0 {
				tokens = append(tokens, currentToken.String())
				currentToken.Reset()
			}
			tokens = append(tokens, string(r))
		} else if unicode.IsPunct(r) {
			if currentToken.Len() > 0 {
				tokens = append(tokens, currentToken.String())
				currentToken.Reset()
			}
		} else {
			currentToken.WriteRune(r)
		}
	}

	if currentToken.Len() > 0 {
		tokens = append(tokens, currentToken.String())
	}

	return tokens
}

// countTechnicalTerms 检测专业术语数量
func (e *FeatureExtractor) countTechnicalTerms(text string) int {
	textLower := strings.ToLower(text)
	count := 0

	// 技术术语词典
	technicalTerms := []string{
		// 编程相关
		"api", "sdk", "http", "https", "tcp", "udp", "dns", "ssl", "tls",
		"sql", "nosql", "redis", "docker", "kubernetes", "k8s", "microservice",
		"algorithm", "recursion", "polymorphism", "inheritance", "abstraction",
		"async", "concurrent", "thread", "mutex", "goroutine", "coroutine",
		"middleware", "endpoint", "payload", "schema", "protobuf", "grpc",
		"oauth", "jwt", "cors", "restful", "graphql", "websocket",
		// 数据相关
		"machine learning", "deep learning", "neural network", "transformer",
		"gradient descent", "backpropagation", "overfitting", "regularization",
		"regression", "classification", "clustering", "embedding", "attention",
		"tokenization", "fine-tuning", "prompt engineering", "rag",
		// 数学相关
		"derivative", "integral", "matrix", "tensor", "eigenvector",
		"probability", "bayesian", "convolution", "laplace", "fourier",
	}

	for _, term := range technicalTerms {
		if strings.Contains(textLower, term) {
			count++
		}
	}

	return count
}

// extractStructural 提取结构特征
// 返回结构维度的综合分值 [0, 1]
func (e *FeatureExtractor) extractStructural(text string, fv *FeatureVector) float64 {
	// 句子计数（按句号、问号、感叹号、换行分割）
	fv.SentenceCount = e.countSentences(text)

	// 问题密度
	fv.QuestionDensity = e.calculateQuestionDensity(text)

	// 嵌套条件检测
	fv.HasNestedCondition = e.detectNestedCondition(text)

	// 多部分请求检测
	fv.MultipartRequest = e.detectMultipartRequest(text)

	// 计算综合分值: 句子数归一化(0.2) + 问题密度(0.3) + 嵌套条件(0.25) + 多部分请求(0.25)
	sentenceScore := e.normalizeFloat(float64(fv.SentenceCount), 0, 20)
	questionScore := fv.QuestionDensity
	nestedScore := 0.0
	if fv.HasNestedCondition {
		nestedScore = 1.0
	}
	multipartScore := 0.0
	if fv.MultipartRequest {
		multipartScore = 1.0
	}

	score := 0.2*sentenceScore + 0.3*questionScore + 0.25*nestedScore + 0.25*multipartScore
	return clampScore(score)
}

// countSentences 计算句子数量
func (e *FeatureExtractor) countSentences(text string) int {
	count := 0
	sentenceEnders := ".!?\n。！？"
	for _, r := range text {
		if strings.ContainsRune(sentenceEnders, r) {
			count++
		}
	}
	if count == 0 && len(strings.TrimSpace(text)) > 0 {
		count = 1
	}
	return count
}

// calculateQuestionDensity 计算问题密度
// 基于问号和疑问词的比例
func (e *FeatureExtractor) calculateQuestionDensity(text string) float64 {
	if len(text) == 0 {
		return 0
	}

	// 统计问号数量
	questionMarkCount := strings.Count(text, "?") + strings.Count(text, "？")

	// 统计疑问词数量
	questionWords := []string{
		"what", "why", "how", "when", "where", "who", "which",
		"什么", "为什么", "如何", "怎么", "什么时候", "哪里", "谁", "哪个",
		"是否", "能不能", "可不可以", "是否可以",
	}
	textLower := strings.ToLower(text)
	questionWordCount := 0
	for _, qw := range questionWords {
		questionWordCount += strings.Count(textLower, qw)
	}

	// 基于句子数归一化
	sentenceCount := e.countSentences(text)
	if sentenceCount == 0 {
		sentenceCount = 1
	}

	density := float64(questionMarkCount+questionWordCount) / float64(sentenceCount)
	// 归一化到 [0, 1]，超过3个问题/句子视为最大值
	if density > 3.0 {
		density = 3.0
	}
	return density / 3.0
}

// detectNestedCondition 检测嵌套条件结构
// 检测 if...then...else 等条件逻辑
func (e *FeatureExtractor) detectNestedCondition(text string) bool {
	textLower := strings.ToLower(text)

	// 英文条件模式
	conditionPatterns := []string{
		"if", "then", "else", "elif", "unless", "otherwise",
		"switch", "case", "default", "break",
	}
	hasCondition := false
	for _, p := range conditionPatterns {
		if strings.Contains(textLower, p) {
			hasCondition = true
			break
		}
	}
	if !hasCondition {
		return false
	}

	// 检测嵌套：同时出现多个条件关键词
	conditionCount := 0
	for _, p := range conditionPatterns {
		if strings.Contains(textLower, p) {
			conditionCount++
		}
	}

	return conditionCount >= 2
}

// detectMultipartRequest 检测多部分请求
// 检测编号列表、分点请求等
func (e *FeatureExtractor) detectMultipartRequest(text string) bool {
	// 编号列表模式: 1. 2. 3. 或 1) 2) 3) 或 (1) (2) (3)
	numberedList := false
	for i := 1; i <= 10; i++ {
		patterns := []string{
			fmt.Sprintf("%d.", i),
			fmt.Sprintf("%d)", i),
			fmt.Sprintf("(%d)", i),
			fmt.Sprintf("%d、", i),
		}
		for _, p := range patterns {
			if strings.Contains(text, p) {
				numberedList = true
				break
			}
		}
		if numberedList {
			break
		}
	}

	if numberedList {
		// 确认至少有两个编号
		matchCount := 0
		for i := 1; i <= 10; i++ {
			if strings.Contains(text, fmt.Sprintf("%d.", i)) ||
				strings.Contains(text, fmt.Sprintf("%d)", i)) ||
				strings.Contains(text, fmt.Sprintf("(%d)", i)) {
				matchCount++
			}
		}
		if matchCount >= 2 {
			return true
		}
	}

	// 分点模式: "first...second..." 或 "首先...其次..."
	multiPartPatterns := []string{
		"first", "second", "third",
		"首先", "其次", "再次", "最后",
		"firstly", "secondly", "thirdly",
		"第一步", "第二步", "第三步",
		"step 1", "step 2", "step 3",
	}
	matchCount := 0
	textLower := strings.ToLower(text)
	for _, p := range multiPartPatterns {
		if strings.Contains(textLower, p) {
			matchCount++
		}
	}
	return matchCount >= 2
}

// extractDomain 提取领域特征
// 返回领域维度的综合分值 [0, 1]
func (e *FeatureExtractor) extractDomain(text string, fv *FeatureVector) float64 {
	// 代码块检测
	fv.HasCodeBlock = e.detectCodeBlock(text)

	// 数学符号检测
	fv.HasMathSymbols = e.detectMathSymbols(text)

	// 表格数据检测
	fv.HasTableData = e.detectTableData(text)

	// 结构化数据检测
	fv.HasStructuredData = e.detectStructuredData(text)

	// 计算综合分值: 代码块(0.3) + 数学符号(0.25) + 表格(0.2) + 结构化数据(0.25)
	codeScore := 0.0
	if fv.HasCodeBlock {
		codeScore = 1.0
	}
	mathScore := 0.0
	if fv.HasMathSymbols {
		mathScore = 1.0
	}
	tableScore := 0.0
	if fv.HasTableData {
		tableScore = 1.0
	}
	structuredScore := 0.0
	if fv.HasStructuredData {
		structuredScore = 1.0
	}

	score := 0.3*codeScore + 0.25*mathScore + 0.2*tableScore + 0.25*structuredScore
	return clampScore(score)
}

// detectCodeBlock 检测代码块（``` 包裹）
func (e *FeatureExtractor) detectCodeBlock(text string) bool {
	return strings.Contains(text, "```")
}

// detectMathSymbols 检测数学符号
func (e *FeatureExtractor) detectMathSymbols(text string) bool {
	mathSymbols := []string{
		"∑", "∫", "√", "∂", "∇", "∞", "≈", "≠", "≤", "≥",
		"±", "×", "÷", "∈", "∉", "⊂", "⊃", "∪", "∩", "∧",
		"∨", "¬", "⇒", "⇔", "∀", "∃", "α", "β", "γ", "δ",
		"theta", "lambda", "pi", "sigma", "omega",
		"\\frac", "\\sum", "\\int", "\\sqrt", "\\partial",
		"\\alpha", "\\beta", "\\gamma", "\\delta", "\\theta",
		"\\lambda", "\\pi", "\\sigma", "\\omega", "\\infty",
	}
	for _, sym := range mathSymbols {
		if strings.Contains(text, sym) {
			return true
		}
	}
	return false
}

// detectTableData 检测表格数据（| 分隔）
func (e *FeatureExtractor) detectTableData(text string) bool {
	// 检测 Markdown 表格格式: 至少两行包含 | 分隔符
	lines := strings.Split(text, "\n")
	tableLineCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			tableLineCount++
		}
	}
	return tableLineCount >= 2
}

// detectStructuredData 检测结构化数据（JSON/XML 标签）
func (e *FeatureExtractor) detectStructuredData(text string) bool {
	// JSON 检测
	if strings.Contains(text, "{") && strings.Contains(text, "}") {
		if strings.Contains(text, ":") || strings.Contains(text, "\"") {
			return true
		}
	}

	// XML/HTML 标签检测
	if strings.Contains(text, "<") && strings.Contains(text, ">") {
		if strings.Contains(text, "</") {
			return true
		}
	}

	return false
}

// extractConversational 提取对话特征
// 返回对话维度的综合分值 [0, 1]
func (e *FeatureExtractor) extractConversational(req *AnalyzeRequest, fv *FeatureVector) float64 {
	// 历史长度（排除最后一条用户消息）
	if len(req.Messages) > 1 {
		fv.HistoryLength = len(req.Messages) - 1
	} else {
		fv.HistoryLength = 0
	}

	// 轮次计数（user-assistant 为一轮）
	fv.TurnCount = e.countTurns(req.Messages)

	// 工具调用检测
	fv.HasToolCall = e.detectToolCall(req.Messages)

	// 上下文大小
	fv.ContextSize = 0
	for _, msg := range req.Messages {
		fv.ContextSize += len(msg.Content)
	}
	if req.System != "" {
		fv.ContextSize += len(req.System)
	}

	// 计算综合分值: 历史长度归一化(0.25) + 轮次归一化(0.25) + 工具调用(0.25) + 上下文大小归一化(0.25)
	historyScore := e.normalizeFloat(float64(fv.HistoryLength), 0, 20)
	turnScore := e.normalizeFloat(float64(fv.TurnCount), 0, 10)
	toolScore := 0.0
	if fv.HasToolCall {
		toolScore = 1.0
	}
	contextScore := e.normalizeFloat(float64(fv.ContextSize), 0, 50000)

	score := 0.25*historyScore + 0.25*turnScore + 0.25*toolScore + 0.25*contextScore
	return clampScore(score)
}

// countTurns 计算对话轮次
func (e *FeatureExtractor) countTurns(messages []Message) int {
	turns := 0
	inTurn := false
	for _, msg := range messages {
		if msg.Role == "user" && !inTurn {
			inTurn = true
		} else if msg.Role == "assistant" && inTurn {
			turns++
			inTurn = false
		}
	}
	// 如果最后一条是用户消息且未完成轮次，也算一轮
	if inTurn {
		turns++
	}
	return turns
}

// detectToolCall 检测工具调用（tool_result/function）
func (e *FeatureExtractor) detectToolCall(messages []Message) bool {
	for _, msg := range messages {
		contentLower := strings.ToLower(msg.Content)
		if strings.Contains(contentLower, "tool_result") ||
			strings.Contains(contentLower, "function") ||
			strings.Contains(contentLower, "tool_use") ||
			msg.Role == "tool" {
			return true
		}
	}
	return false
}

// extractTaskType 提取任务类型特征
// 返回任务类型维度的综合分值 [0, 1]
func (e *FeatureExtractor) extractTaskType(text string, fv *FeatureVector) float64 {
	textLower := strings.ToLower(text)

	// 祈使动词词典映射
	lowComplexityVerbs := []string{
		"format", "list", "translate", "convert", "define",
		"格式化", "列出", "翻译", "转换", "定义", "解释", "说明",
		"查找", "搜索", "查询", "告诉我",
	}
	mediumComplexityVerbs := []string{
		"summarize", "rewrite", "paraphrase", "expand", "simplify",
		"总结", "重写", "改写", "扩展", "简化", "概括", "归纳",
		"描述", "介绍", "对比",
	}
	highComplexityVerbs := []string{
		"analyze", "compare", "design", "evaluate", "optimize",
		"refactor", "architect", "implement", "integrate", "debug",
		"code", "program", "develop", "build", "create",
		"math", "calculate", "compute", "solve", "prove",
		"推导", "证明", "计算", "求解",
		"分析", "比较", "设计", "评估", "优化", "重构",
		"架构", "实现", "集成", "调试", "编程", "开发",
		"构建", "创建", "生成",
	}

	lowCount := 0
	mediumCount := 0
	highCount := 0

	for _, verb := range lowComplexityVerbs {
		if strings.Contains(textLower, verb) {
			lowCount++
		}
	}
	for _, verb := range mediumComplexityVerbs {
		if strings.Contains(textLower, verb) {
			mediumCount++
		}
	}
	for _, verb := range highComplexityVerbs {
		if strings.Contains(textLower, verb) {
			highCount++
		}
	}

	// 确定任务类别和分值
	totalMatches := lowCount + mediumCount + highCount
	if totalMatches == 0 {
		// 没有匹配到任何动词，根据文本长度给出默认分值
		if len(text) < 100 {
			fv.TaskCategory = "low"
			fv.TaskComplexity = 0.2
		} else if len(text) < 500 {
			fv.TaskCategory = "medium"
			fv.TaskComplexity = 0.5
		} else {
			fv.TaskCategory = "high"
			fv.TaskComplexity = 0.7
		}
		return fv.TaskComplexity
	}

	// 加权计算: high * 1.0 + medium * 0.6 + low * 0.2
	weightedScore := float64(highCount)*1.0 + float64(mediumCount)*0.6 + float64(lowCount)*0.2
	maxPossible := float64(totalMatches) * 1.0
	normalizedScore := weightedScore / maxPossible

	fv.TaskComplexity = clampScore(normalizedScore)

	// 确定类别
	if highCount > 0 {
		fv.TaskCategory = "high"
	} else if mediumCount > 0 {
		fv.TaskCategory = "medium"
	} else {
		fv.TaskCategory = "low"
	}

	return fv.TaskComplexity
}

// normalizeFloat 将值从 [min, max] 范围归一化到 [0, 1]
func (e *FeatureExtractor) normalizeFloat(value, min, max float64) float64 {
	if max <= min {
		return 0
	}
	normalized := (value - min) / (max - min)
	if normalized < 0 {
		return 0
	}
	if normalized > 1 {
		return 1
	}
	return normalized
}

// clampScore 将分值限制在 [0, 1] 范围内
func clampScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}
