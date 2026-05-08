// Package handler 提供 MaaS-Router 的 HTTP 处理器
package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"maas-router/internal/pkg/ctxkey"
	"maas-router/internal/service"
)

// GatewayHandler Claude API 兼容网关 Handler
// 处理 Claude Messages API 格式的请求
type GatewayHandler struct {
	// AccountService 账号调度服务（智能路由）
	AccountService AccountService
	// JudgeAgentService 复杂度评估服务
	JudgeAgentService JudgeAgentService
	// ComplexityService 复杂度分析服务（增强版，支持缓存和模型推荐）
	ComplexityService ComplexityService
	// BillingService 计费服务
	BillingService BillingService
	// ProxyService 代理转发服务
	ProxyService ProxyService
	// ModelService 模型信息服务
	ModelService ModelService
	// TokenCounter Token 计数器（用于本地估算）
	TokenCounter service.TokenCounter
}

// NewGatewayHandler 创建 Claude API 兼容网关 Handler
func NewGatewayHandler(
	accountService AccountService,
	judgeAgentService JudgeAgentService,
	complexityService ComplexityService,
	billingService BillingService,
	proxyService ProxyService,
	modelService ModelService,
	tokenCounter service.TokenCounter,
) *GatewayHandler {
	return &GatewayHandler{
		AccountService:    accountService,
		JudgeAgentService: judgeAgentService,
		ComplexityService: complexityService,
		BillingService:    billingService,
		ProxyService:      proxyService,
		ModelService:      modelService,
		TokenCounter:      tokenCounter,
	}
}

// ClaudeMessagesRequest Claude Messages API 请求结构
type ClaudeMessagesRequest struct {
	Model       string                   `json:"model"`
	Messages    []ClaudeMessage          `json:"messages"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	Temperature float64                  `json:"temperature,omitempty"`
	TopP        float64                  `json:"top_p,omitempty"`
	TopK        int                      `json:"top_k,omitempty"`
	Stream      bool                     `json:"stream,omitempty"`
	System      string                   `json:"system,omitempty"`
	Tools       []ClaudeTool             `json:"tools,omitempty"`
	ToolChoice  *ClaudeToolChoice        `json:"tool_choice,omitempty"`
	Metadata    *ClaudeMetadata          `json:"metadata,omitempty"`
	StopSequences []string               `json:"stop_sequences,omitempty"`
}

// ClaudeMessage Claude 消息结构
type ClaudeMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // 支持 string 或 []ContentBlock
}

// ClaudeContentBlock Claude 内容块
type ClaudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// 工具使用相关字段
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
	// 图片相关字段
	Source *ClaudeImageSource `json:"source,omitempty"`
}

// ClaudeImageSource Claude 图片源
type ClaudeImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// ClaudeTool Claude 工具定义
type ClaudeTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ClaudeToolChoice 工具选择配置
type ClaudeToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// ClaudeMetadata 请求元数据
type ClaudeMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

// ClaudeMessagesResponse Claude Messages API 响应结构
type ClaudeMessagesResponse struct {
	ID           string                `json:"id"`
	Type         string                `json:"type"`
	Role         string                `json:"role"`
	Content      []ClaudeContentBlock  `json:"content"`
	Model        string                `json:"model"`
	StopReason   *string               `json:"stop_reason,omitempty"`
	StopSequence *string               `json:"stop_sequence,omitempty"`
	Usage        ClaudeUsage           `json:"usage"`
}

// ClaudeUsage Claude Token 使用量
type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ClaudeStreamEvent Claude SSE 流式事件
type ClaudeStreamEvent struct {
	Type         string               `json:"type"`
	Index        int                  `json:"index,omitempty"`
	Delta        *ClaudeStreamDelta   `json:"delta,omitempty"`
	ContentBlock *ClaudeContentBlock  `json:"content_block,omitempty"`
	Message      *ClaudeMessagesResponse `json:"message,omitempty"`
	Usage        *ClaudeUsage         `json:"usage,omitempty"`
}

// ClaudeStreamDelta 流式增量内容
type ClaudeStreamDelta struct {
	Type       string `json:"type,omitempty"`
	Text       string `json:"text,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
}

// CountTokensRequest Token 计数请求
type CountTokensRequest struct {
	Model    string          `json:"model"`
	Messages []ClaudeMessage `json:"messages"`
	System   string          `json:"system,omitempty"`
	Tools    []ClaudeTool    `json:"tools,omitempty"`
}

// CountTokensResponse Token 计数响应
type CountTokensResponse struct {
	InputTokens int `json:"input_tokens"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Created    int64    `json:"created"`
	OwnedBy    string   `json:"owned_by"`
	Permission []string `json:"permission,omitempty"`
}

// ListModelsResponse 模型列表响应
type ListModelsResponse struct {
	Data   []ModelInfo `json:"data"`
	Object string      `json:"object"`
}

// UsageResponse 使用量响应
type UsageResponse struct {
	TotalTokens     int64   `json:"total_tokens"`
	TotalCost       float64 `json:"total_cost"`
	RequestsCount   int64   `json:"requests_count"`
	InputTokens     int64   `json:"input_tokens"`
	OutputTokens    int64   `json:"output_tokens"`
}

// Messages 处理 POST /v1/messages (Claude Messages API)
// 这是核心网关接口，支持流式和非流式响应
func (h *GatewayHandler) Messages(c *gin.Context) {
	startTime := time.Now()
	requestID := c.GetString(string(ctxkey.ContextKeyRequestID))

	// 获取 API Key 信息
	apiKeyInfo, exists := c.Get(string(ctxkey.ContextKeyAPIKey))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}
	apiKey := apiKeyInfo.(*APIKeyContext)

	// 解析请求体
	var req ClaudeMessagesRequest
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "无法读取请求体")
		return
	}

	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", fmt.Sprintf("请求格式错误: %v", err))
		return
	}

	// 验证必填字段
	if req.Model == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_MODEL", "缺少 model 参数")
		return
	}
	if len(req.Messages) == 0 {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_MESSAGES", "缺少 messages 参数")
		return
	}

	// 检查模型权限
	if !h.checkModelPermission(c, apiKey, req.Model) {
		ErrorResponse(c, http.StatusForbidden, "MODEL_NOT_ALLOWED", fmt.Sprintf("无权使用模型: %s", req.Model))
		return
	}

	// 获取复杂度评分（可选，用于智能路由）
	complexity := 0.0
	var complexityProfile *ComplexityProfile
	originalModel := req.Model

	if h.ComplexityService != nil {
		// 使用增强版 ComplexityService 进行分析（支持缓存和模型推荐）
		profile, err := h.ComplexityService.AnalyzeWithCache(c.Request.Context(), &ComplexityAnalyzeRequest{
			Model:     req.Model,
			Messages:  claudeMessagesToGeneric(req.Messages),
			System:    req.System,
			MaxTokens: req.MaxTokens,
			Stream:    req.Stream,
		})
		if err == nil && profile != nil {
			complexityProfile = profile
			complexity = profile.Score

			// 如果推荐模型不为空且用户未禁止覆盖，使用推荐模型
			if profile.RecommendedModel != "" {
				allowOverride := c.GetHeader("X-Allow-Model-Override")
				if allowOverride == "" || allowOverride != "false" {
					req.Model = profile.RecommendedModel
					if req.Model != originalModel {
						// 记录模型升级日志
					}
				}
			}
		}
	}

	// 回退到原有 JudgeAgent 逻辑
	if complexityProfile == nil && h.JudgeAgentService != nil {
		complexity, _ = h.JudgeAgentService.GetComplexityScore(c.Request.Context(), &req)
	}

	// 选择最优上游账号（带重试）
	retryConfig := DefaultRetryConfig()

	// 构建上游请求模板
	upstreamReq := &ProxyRequest{
		Method:    http.MethodPost,
		Headers:   nil, // 将在 selectAccountFunc 中根据账号动态设置
		Body:      bodyBytes,
		Stream:    req.Stream,
		RequestID: requestID,
		Model:     req.Model,
		UserID:    apiKey.UserID,
		APIKeyID:  apiKey.ID,
	}

	result := DoRequestWithRetry(
		c.Request.Context(),
		upstreamReq,
		func(ctx context.Context, excluded []string) (*Account, error) {
			account, err := h.AccountService.SelectAccount(ctx, &AccountSelectRequest{
				Model:             req.Model,
				Platform:          "claude",
				Complexity:        complexity,
				UserID:            apiKey.UserID,
				APIKeyID:          apiKey.ID,
				RoutingTier:       getComplexityTier(complexityProfile),
				ExcludedAccountIDs: excluded,
			})
			if err != nil {
				return nil, err
			}
			// 动态设置上游 URL 和请求头
			upstreamReq.URL = fmt.Sprintf("%s/v1/messages", account.BaseURL)
			upstreamReq.Headers = h.buildClaudeHeaders(account, apiKey)
			upstreamReq.AccountID = account.ID
			return account, nil
		},
		h.ProxyService,
		retryConfig,
	)

	if result.Response == nil {
		ErrorResponse(c, http.StatusBadGateway, "UPSTREAM_ERROR",
			fmt.Sprintf("上游服务错误（已重试 %d 次）: %v", result.Attempt-1, result.LastError))
		return
	}
	resp := result.Response
	account = result.Account
	defer resp.Body.Close()

	// 处理流式响应
	if req.Stream {
		h.handleStreamResponse(c, resp, account, apiKey, startTime, req.Model, complexityProfile, originalModel)
		return
	}

	// 处理非流式响应
	h.handleNonStreamResponse(c, resp, account, apiKey, startTime, req.Model, complexityProfile, originalModel)
}

// CountTokens 处理 POST /v1/messages/count_tokens
// 计算 Claude API 请求的 Token 数量
func (h *GatewayHandler) CountTokens(c *gin.Context) {
	// 获取 API Key 信息
	apiKeyInfo, exists := c.Get(string(ctxkey.ContextKeyAPIKey))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}
	apiKey := apiKeyInfo.(*APIKeyContext)

	// 解析请求
	var req CountTokensRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", fmt.Sprintf("请求格式错误: %v", err))
		return
	}

	// 验证必填字段
	if req.Model == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_MODEL", "缺少 model 参数")
		return
	}

	// 检查模型权限
	if !h.checkModelPermission(c, apiKey, req.Model) {
		ErrorResponse(c, http.StatusForbidden, "MODEL_NOT_ALLOWED", fmt.Sprintf("无权使用模型: %s", req.Model))
		return
	}

	// 计算 Token 数量（简化实现，实际应调用上游或本地计算）
	inputTokens := h.estimateTokens(req.Messages, req.System, req.Tools)

	c.JSON(http.StatusOK, CountTokensResponse{
		InputTokens: inputTokens,
	})
}

// ListModels 处理 GET /v1/models
// 返回可用模型列表
func (h *GatewayHandler) ListModels(c *gin.Context) {
	// 获取 API Key 信息
	apiKeyInfo, exists := c.Get(string(ctxkey.ContextKeyAPIKey))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}
	apiKey := apiKeyInfo.(*APIKeyContext)

	// 获取可用模型列表
	models, err := h.ModelService.ListModels(c.Request.Context(), apiKey.AllowedModels)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取模型列表失败")
		return
	}

	c.JSON(http.StatusOK, ListModelsResponse{
		Object: "list",
		Data:   models,
	})
}

// GetUsage 处理 GET /v1/usage
// 返回当前用户的使用量统计
func (h *GatewayHandler) GetUsage(c *gin.Context) {
	// 获取 API Key 信息
	apiKeyInfo, exists := c.Get(string(ctxkey.ContextKeyAPIKey))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}
	apiKey := apiKeyInfo.(*APIKeyContext)

	// 获取使用量统计
	usage, err := h.BillingService.GetUserUsage(c.Request.Context(), apiKey.UserID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取使用量失败")
		return
	}

	c.JSON(http.StatusOK, UsageResponse{
		TotalTokens:     usage.TotalTokens,
		TotalCost:       usage.TotalCost,
		RequestsCount:   usage.RequestsCount,
		InputTokens:     usage.InputTokens,
		OutputTokens:    usage.OutputTokens,
	})
}

// handleStreamResponse 处理 SSE 流式响应
func (h *GatewayHandler) handleStreamResponse(c *gin.Context, resp *http.Response, account *Account, apiKey *APIKeyContext, startTime time.Time, model string, complexityProfile *ComplexityProfile, originalModel string) {
	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// 获取响应写入器
	writer := c.Writer

	// 创建流式读取器
	reader := bufio.NewReader(resp.Body)

	var totalInputTokens, totalOutputTokens int
	var lastEvent ClaudeStreamEvent
	var streamedContent strings.Builder // accumulate streamed text for local token counting

	// 流式转发
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}

		line = strings.TrimSpace(line)

		// 跳过空行
		if line == "" {
			continue
		}

		// 处理 SSE 事件
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// 解析事件
			var event ClaudeStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err == nil {
				lastEvent = event

				// 提取 token 使用量
				if event.Usage != nil {
					totalInputTokens = event.Usage.InputTokens
					totalOutputTokens = event.Usage.OutputTokens
				}

				// 累积流式文本内容用于本地 token 估算
				if event.Delta != nil && event.Delta.Text != "" {
					streamedContent.WriteString(event.Delta.Text)
				}
			}

			// 转发给客户端
			fmt.Fprintf(writer, "data: %s\n\n", data)
			writer.Flush()
		} else if strings.HasPrefix(line, "event: ") {
			// 转发事件类型
			eventType := strings.TrimPrefix(line, "event: ")
			fmt.Fprintf(writer, "event: %s\n", eventType)
		}
	}

	// 如果上游未提供 token 使用量，使用本地 token 计数器估算
	if totalInputTokens == 0 && totalOutputTokens == 0 && h.TokenCounter != nil {
		// 估算输出 token（基于累积的流式内容）
		totalOutputTokens = h.TokenCounter.CountTokens(streamedContent.String())
		// 输入 token 保持为 0，因为我们在流式场景下无法轻易获取原始输入
		// 但如果需要，可以从请求中估算
	}

	// 记录使用量和计费
	latency := time.Since(startTime).Milliseconds()
	h.recordUsage(c, account, apiKey, model, totalInputTokens, totalOutputTokens, latency, "success", "", complexityProfile, originalModel)
}

// handleNonStreamResponse 处理非流式响应
func (h *GatewayHandler) handleNonStreamResponse(c *gin.Context, resp *http.Response, account *Account, apiKey *APIKeyContext, startTime time.Time, model string, complexityProfile *ComplexityProfile, originalModel string) {
	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		ErrorResponse(c, http.StatusBadGateway, "UPSTREAM_ERROR", "读取上游响应失败")
		return
	}

	// 解析响应
	var claudeResp ClaudeMessagesResponse
	if err := json.Unmarshal(bodyBytes, &claudeResp); err != nil {
		// 如果解析失败，直接转发原始响应
		c.Data(resp.StatusCode, "application/json", bodyBytes)
		return
	}

	// 记录使用量和计费
	latency := time.Since(startTime).Milliseconds()
	h.recordUsage(c, account, apiKey, model, claudeResp.Usage.InputTokens, claudeResp.Usage.OutputTokens, latency, "success", "", complexityProfile, originalModel)

	// 返回响应
	c.Data(resp.StatusCode, "application/json", bodyBytes)
}

// recordUsage 记录使用量和计费
func (h *GatewayHandler) recordUsage(c *gin.Context, account *Account, apiKey *APIKeyContext, model string, inputTokens, outputTokens int, latencyMs int64, status, errorMsg string, complexityProfile *ComplexityProfile, originalModel string) {
	if h.BillingService == nil {
		return
	}

	// 跳过计费标记
	if skipBilling, _ := c.Get(string(ctxkey.ContextKeySkipBilling)); skipBilling == true {
		return
	}

	// 计算费用
	cost := h.BillingService.CalculateCost(model, inputTokens, outputTokens)

	// 构建使用记录
	record := &UsageRecord{
		RequestID:        c.GetString(string(ctxkey.ContextKeyRequestID)),
		UserID:           apiKey.UserID,
		APIKeyID:         apiKey.ID,
		AccountID:        account.ID,
		Model:            model,
		Platform:         "claude",
		PromptTokens:     inputTokens,
		CompletionTokens: outputTokens,
		TotalTokens:      inputTokens + outputTokens,
		LatencyMs:        int32(latencyMs),
		Cost:             cost,
		Status:           status,
		ErrorMessage:     errorMsg,
		ClientIP:         c.ClientIP(),
		UserAgent:        c.GetHeader("User-Agent"),
	}

	// 填充复杂度分析字段
	if complexityProfile != nil {
		record.ComplexityScore = complexityProfile.Score
		record.ComplexityLevel = complexityProfile.Level
		record.RoutingTier = complexityProfile.RecommendedTier
		record.ComplexityModel = complexityProfile.RecommendedModel
		record.CostSavingRatio = complexityProfile.CostSavingRatio
		record.QualityRisk = complexityProfile.QualityRisk
		record.WasUpgraded = (originalModel != "" && originalModel != model)
	}

	// 异步记录（不阻塞响应）
	go h.BillingService.RecordUsage(c.Request.Context(), record)
}

// buildClaudeHeaders 构建 Claude API 请求头
func (h *GatewayHandler) buildClaudeHeaders(account *Account, apiKey *APIKeyContext) map[string]string {
	headers := map[string]string{
		"Content-Type":      "application/json",
		"anthropic-version": "2023-06-01",
		"x-api-key":         account.APIKey,
	}

	// 添加用户标识
	if apiKey.UserID != "" {
		headers["anthropic-user-id"] = apiKey.UserID
	}

	return headers
}

// checkModelPermission 检查模型使用权限
func (h *GatewayHandler) checkModelPermission(c *gin.Context, apiKey *APIKeyContext, model string) bool {
	// 如果没有限制，允许所有模型
	if len(apiKey.AllowedModels) == 0 {
		return true
	}

	// 检查模型是否在允许列表中
	for _, allowed := range apiKey.AllowedModels {
		if allowed == model || allowed == "*" {
			return true
		}
		// 支持前缀匹配，如 "claude-*" 匹配所有 claude 模型
		if strings.HasSuffix(allowed, "*") {
			prefix := strings.TrimSuffix(allowed, "*")
			if strings.HasPrefix(model, prefix) {
				return true
			}
		}
	}

	return false
}

// estimateTokens 估算 Token 数量（简化实现）
func (h *GatewayHandler) estimateTokens(messages []ClaudeMessage, system string, tools []ClaudeTool) int {
	totalChars := len(system)

	for _, msg := range messages {
		totalChars += len(msg.Role)
		totalChars += len(msg.Content)
	}

	for _, tool := range tools {
		totalChars += len(tool.Name)
		totalChars += len(tool.Description)
		totalChars += len(tool.InputSchema)
	}

	// 粗略估算：平均 4 个字符 = 1 token
	return totalChars / 4
}

// RegisterClaudeHandlers 注册 Claude API 兼容路由到 HandlerGroup
func RegisterClaudeHandlers(h *GatewayHandler) map[string]gin.HandlerFunc {
	return map[string]gin.HandlerFunc{
		"Messages":    h.Messages,
		"ListModels":  h.ListModels,
		"GetUsage":    h.GetUsage,
	}
}

// ===== 复杂度分析服务接口 =====

// ComplexityService 复杂度分析服务接口（增强版，支持缓存和模型推荐）
type ComplexityService interface {
	// AnalyzeWithCache 带缓存的复杂度分析
	AnalyzeWithCache(ctx interface{}, req *ComplexityAnalyzeRequest) (*ComplexityProfile, error)
	// RecordFeedback 记录质量反馈
	RecordFeedback(ctx interface{}, requestID string, qualityScore float64) error
	// HealthCheck 健康检查
	HealthCheck(ctx interface{}) (map[string]interface{}, error)
}

// ComplexityAnalyzeRequest 复杂度分析请求
type ComplexityAnalyzeRequest struct {
	Model     string            `json:"model"`
	Messages  []GenericMessage  `json:"messages"`
	System    string            `json:"system,omitempty"`
	MaxTokens int               `json:"max_tokens,omitempty"`
	Stream    bool              `json:"stream,omitempty"`
}

// GenericMessage 通用消息结构（用于复杂度分析）
type GenericMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ComplexityProfile 复杂度分析结果
type ComplexityProfile struct {
	Score            float64 `json:"score"`
	Level            string  `json:"level"`
	Confidence       float64 `json:"confidence"`
	LexicalScore     float64 `json:"lexical_score"`
	StructuralScore  float64 `json:"structural_score"`
	DomainScore      float64 `json:"domain_score"`
	ConversationalScore float64 `json:"conversational_score"`
	TaskTypeScore    float64 `json:"task_type_score"`
	RecommendedTier  string  `json:"recommended_tier"`
	RecommendedModel string  `json:"recommended_model"`
	FallbackModel    string  `json:"fallback_model"`
	EstimatedCost    float64 `json:"estimated_cost"`
	CostSavingRatio  float64 `json:"cost_saving_ratio"`
	QualityRisk      string  `json:"quality_risk"`
	NeedsUpgrade     bool    `json:"needs_upgrade"`
}

// ===== 服务接口定义 =====

// AccountService 账号调度服务接口
type AccountService interface {
	// SelectAccount 选择最优上游账号
	SelectAccount(ctx interface{}, req *AccountSelectRequest) (*Account, error)
}

// AccountSelectRequest 账号选择请求
type AccountSelectRequest struct {
	Model             string
	Platform          string
	Complexity        float64
	UserID            string
	APIKeyID          string
	RoutingTier       string // 复杂度分析推荐的路由层级
	ExcludedAccountIDs []string // 排除的账号 ID 列表（用于重试时避免重用失败账号）
}

// Account 上游账号信息
type Account struct {
	ID             string
	Name           string
	Platform       string
	BaseURL        string
	APIKey         string
	MaxConcurrency int
	Status         string
}

// JudgeAgentService 复杂度评估服务接口
type JudgeAgentService interface {
	// GetComplexityScore 获取请求复杂度评分
	GetComplexityScore(ctx interface{}, req *ClaudeMessagesRequest) (float64, error)
}

// BillingService 计费服务接口
type BillingService interface {
	// CalculateCost 计算请求费用
	CalculateCost(model string, inputTokens, outputTokens int) float64
	// RecordUsage 记录使用量
	RecordUsage(ctx interface{}, record *UsageRecord) error
	// GetUserUsage 获取用户使用量统计
	GetUserUsage(ctx interface{}, userID string) (*UserUsage, error)
}

// UserUsage 用户使用量统计
type UserUsage struct {
	TotalTokens   int64
	TotalCost     float64
	RequestsCount int64
	InputTokens   int64
	OutputTokens  int64
}

// UsageRecord 使用记录
type UsageRecord struct {
	RequestID        string
	UserID           string
	APIKeyID         string
	AccountID        string
	Model            string
	Platform         string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	LatencyMs        int32
	Cost             float64
	Status           string
	ErrorMessage     string
	ClientIP         string
	UserAgent        string
	// 复杂度分析字段
	ComplexityScore   float64
	ComplexityLevel   string
	RoutingTier       string
	ComplexityModel   string
	CostSavingRatio   float64
	QualityRisk       string
	WasUpgraded       bool
}

// ProxyService 代理转发服务接口
type ProxyService interface {
	// DoRequest 执行代理请求
	DoRequest(ctx interface{}, req *ProxyRequest) (*http.Response, error)
}

// ProxyRequest 代理请求
type ProxyRequest struct {
	Method    string
	URL       string
	Headers   map[string]string
	Body      []byte
	Stream    bool
	AccountID string
	RequestID string
	Model     string
	UserID    string
	APIKeyID  string
}

// ModelService 模型信息服务接口
type ModelService interface {
	// ListModels 获取模型列表
	ListModels(ctx interface{}, allowedModels []string) ([]ModelInfo, error)
}

// APIKeyContext API Key 上下文信息
type APIKeyContext struct {
	ID            string
	UserID        string
	KeyPrefix     string
	Name          string
	Status        string
	AllowedModels []string
	DailyLimit    float64
	MonthlyLimit  float64
	UserBalance   float64
}

// ErrorResponse 统一错误响应
func ErrorResponse(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"type":    "error",
			"code":    code,
			"message": message,
		},
	})
}

// SSEWriter SSE 写入器
type SSEWriter struct {
	writer  http.ResponseWriter
	flusher http.Flusher
}

// NewSSEWriter 创建 SSE 写入器
func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}
	return &SSEWriter{
		writer:  w,
		flusher: flusher,
	}
}

// WriteEvent 写入 SSE 事件
func (w *SSEWriter) WriteEvent(event string, data []byte) error {
	if _, err := fmt.Fprintf(w.writer, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w.writer, "data: %s\n\n", string(data)); err != nil {
		return err
	}
	w.flusher.Flush()
	return nil
}

// WriteData 写入 SSE 数据
func (w *SSEWriter) WriteData(data []byte) error {
	if _, err := fmt.Fprintf(w.writer, "data: %s\n\n", string(data)); err != nil {
		return err
	}
	w.flusher.Flush()
	return nil
}

// ParseStreamEvent 解析流式事件
func ParseStreamEvent(line string) (*ClaudeStreamEvent, error) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "data: ") {
		return nil, fmt.Errorf("invalid SSE line format")
	}

	data := strings.TrimPrefix(line, "data: ")
	var event ClaudeStreamEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return nil, err
	}

	return &event, nil
}

// CloneRequest 克隆 HTTP 请求
func CloneRequest(req *http.Request) (*http.Request, error) {
	// 读取请求体
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		// 恢复原始请求体
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// 创建新请求
	newReq, err := http.NewRequest(req.Method, req.URL.String(), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	// 复制请求头
	newReq.Header = req.Header.Clone()

	return newReq, nil
}

// claudeMessagesToGeneric 将 Claude 消息转换为通用消息格式（用于复杂度分析）
func claudeMessagesToGeneric(messages []ClaudeMessage) []GenericMessage {
	result := make([]GenericMessage, 0, len(messages))
	for _, msg := range messages {
		// 尝试将 json.RawMessage 解析为字符串
		var content string
		if err := json.Unmarshal(msg.Content, &content); err == nil {
			result = append(result, GenericMessage{
				Role:    msg.Role,
				Content: content,
			})
		} else {
			// 如果解析失败，使用原始 JSON 字符串
			result = append(result, GenericMessage{
				Role:    msg.Role,
				Content: string(msg.Content),
			})
		}
	}
	return result
}

// getComplexityTier 从复杂度配置文件中获取推荐的路由层级
func getComplexityTier(profile *ComplexityProfile) string {
	if profile == nil {
		return ""
	}
	return profile.RecommendedTier
}
