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
)

// OpenAIGatewayHandler OpenAI API 兼容网关 Handler
// 处理 OpenAI Chat Completions、Responses、Images、Embeddings API 格式的请求
type OpenAIGatewayHandler struct {
	// AccountService 账号调度服务
	AccountService AccountService
	// ComplexityService 复杂度分析服务（增强版，支持缓存和模型推荐）
	ComplexityService ComplexityService
	// BillingService 计费服务
	BillingService BillingService
	// ProxyService 代理转发服务
	ProxyService ProxyService
	// ModelService 模型信息服务
	ModelService ModelService
	// RouterService 路由服务（自动路由到 Claude/OpenAI）
	RouterService RouterService
	// ModelMappingService 模型映射服务
	ModelMappingService ModelMappingService
}

// NewOpenAIGatewayHandler 创建 OpenAI API 兼容网关 Handler
func NewOpenAIGatewayHandler(
	accountService AccountService,
	complexityService ComplexityService,
	billingService BillingService,
	proxyService ProxyService,
	modelService ModelService,
	routerService RouterService,
	modelMappingService ModelMappingService,
) *OpenAIGatewayHandler {
	return &OpenAIGatewayHandler{
		AccountService:      accountService,
		ComplexityService:   complexityService,
		BillingService:      billingService,
		ProxyService:        proxyService,
		ModelService:        modelService,
		RouterService:       routerService,
		ModelMappingService: modelMappingService,
	}
}

// ChatCompletionRequest OpenAI Chat Completions API 请求结构
type ChatCompletionRequest struct {
	Model            string                 `json:"model"`
	Messages         []ChatMessage          `json:"messages"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	Temperature      *float64               `json:"temperature,omitempty"`
	TopP             *float64               `json:"top_p,omitempty"`
	N                *int                   `json:"n,omitempty"`
	Stream           bool                   `json:"stream,omitempty"`
	Stop             interface{}            `json:"stop,omitempty"`
	PresencePenalty  *float64               `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64               `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]float64     `json:"logit_bias,omitempty"`
	User             string                 `json:"user,omitempty"`
	Tools            []ChatTool             `json:"tools,omitempty"`
	ToolChoice       interface{}            `json:"tool_choice,omitempty"`
	ResponseFormat   *ResponseFormat        `json:"response_format,omitempty"`
	Seed             *int                   `json:"seed,omitempty"`
}

// ChatMessage OpenAI 聊天消息结构
type ChatMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // 支持 string 或 []ContentPart
	Name    string          `json:"name,omitempty"`
	// 工具调用相关
	ToolCallID   string       `json:"tool_call_id,omitempty"`
	ToolCalls    []ToolCall   `json:"tool_calls,omitempty"`
}

// ContentPart 多模态内容部分
type ContentPart struct {
	Type     string          `json:"type"` // "text", "image_url"
	Text     string          `json:"text,omitempty"`
	ImageURL *ImageURL       `json:"image_url,omitempty"`
}

// ImageURL 图片 URL 结构
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "auto", "low", "high"
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function FunctionCall    `json:"function"`
}

// FunctionCall 函数调用
type FunctionCall struct {
	Name      string          `json:"name"`
	Arguments string          `json:"arguments"`
}

// ChatTool OpenAI 工具定义
type ChatTool struct {
	Type     string          `json:"type"`
	Function FunctionDef     `json:"function"`
}

// FunctionDef 函数定义
type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ResponseFormat 响应格式配置
type ResponseFormat struct {
	Type       string `json:"type"` // "text", "json_object"
	JSONSchema *JSONSchema `json:"json_schema,omitempty"`
}

// JSONSchema JSON Schema 定义
type JSONSchema struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict bool            `json:"strict,omitempty"`
}

// ChatCompletionResponse OpenAI Chat Completions API 响应结构
type ChatCompletionResponse struct {
	ID                string                 `json:"id"`
	Object            string                 `json:"object"`
	Created           int64                  `json:"created"`
	Model             string                 `json:"model"`
	Choices           []ChatCompletionChoice `json:"choices"`
	Usage             ChatUsage              `json:"usage"`
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"`
}

// ChatCompletionChoice 选择项
type ChatCompletionChoice struct {
	Index        int          `json:"index"`
	Message      *ChatMessage `json:"message"`
	Delta        *ChatDelta   `json:"delta,omitempty"`
	FinishReason string       `json:"finish_reason"`
	Logprobs     interface{}  `json:"logprobs,omitempty"`
}

// ChatDelta 流式增量内容
type ChatDelta struct {
	Role      string       `json:"role,omitempty"`
	Content   string       `json:"content,omitempty"`
	ToolCalls []ToolCall   `json:"tool_calls,omitempty"`
}

// ChatUsage Token 使用量
type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletionChunk 流式响应块
type ChatCompletionChunk struct {
	ID                string                 `json:"id"`
	Object            string                 `json:"object"`
	Created           int64                  `json:"created"`
	Model             string                 `json:"model"`
	Choices           []ChatCompletionChoice `json:"choices"`
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"`
}

// ResponsesRequest OpenAI Responses API 请求结构
type ResponsesRequest struct {
	Model       string                 `json:"model"`
	Input       interface{}            `json:"input"` // string 或结构化输入
	Instructions string                `json:"instructions,omitempty"`
	MaxTokens   *int                   `json:"max_tokens,omitempty"`
	Temperature *float64               `json:"temperature,omitempty"`
	TopP        *float64               `json:"top_p,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Tools       []ChatTool             `json:"tools,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ResponsesResponse OpenAI Responses API 响应结构
type ResponsesResponse struct {
	ID        string                 `json:"id"`
	Object    string                 `json:"object"`
	Created   int64                  `json:"created"`
	Model     string                 `json:"model"`
	Output    string                 `json:"output"`
	Usage     ChatUsage              `json:"usage"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ImageGenerationsRequest 图像生成请求
type ImageGenerationsRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              *int   `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	Style          string `json:"style,omitempty"`
	User           string `json:"user,omitempty"`
}

// ImageGenerationsResponse 图像生成响应
type ImageGenerationsResponse struct {
	Created int64      `json:"created"`
	Data    []ImageData `json:"data"`
}

// ImageData 图像数据
type ImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// EmbeddingsRequest Embeddings 请求
type EmbeddingsRequest struct {
	Model          string   `json:"model"`
	Input          interface{} `json:"input"` // string 或 []string
	EncodingFormat string   `json:"encoding_format,omitempty"`
	Dimensions     *int     `json:"dimensions,omitempty"`
	User           string   `json:"user,omitempty"`
}

// EmbeddingsResponse Embeddings 响应
type EmbeddingsResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  ChatUsage       `json:"usage"`
}

// EmbeddingData Embedding 数据
type EmbeddingData struct {
	Object    string      `json:"object"`
	Index     int         `json:"index"`
	Embedding interface{} `json:"embedding"` // []float32 或 base64 string
}

// ChatCompletions 处理 POST /v1/chat/completions
// OpenAI Chat Completions API 核心接口
func (h *OpenAIGatewayHandler) ChatCompletions(c *gin.Context) {
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
	var req ChatCompletionRequest
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

	// 复杂度分析（增强版，支持缓存和模型推荐）
	var complexityProfile *ComplexityProfile
	originalModel := req.Model

	if h.ComplexityService != nil {
		profile, err := h.ComplexityService.AnalyzeWithCache(c.Request.Context(), &ComplexityAnalyzeRequest{
			Model:     req.Model,
			Messages:  chatMessagesToGeneric(req.Messages),
			MaxTokens: getMaxTokens(req.MaxTokens),
			Stream:    req.Stream,
		})
		if err == nil && profile != nil {
			complexityProfile = profile

			// 如果推荐模型不为空且用户未禁止覆盖，使用推荐模型
			if profile.RecommendedModel != "" {
				allowOverride := c.GetHeader("X-Allow-Model-Override")
				if allowOverride == "" || allowOverride != "false" {
					req.Model = profile.RecommendedModel
				}
			}
		}
	}

	// 应用模型映射
	if h.ModelMappingService != nil {
		req.Model = h.ModelMappingService.ResolveMapping(req.Model)
	}

	// 确定目标平台（自动路由）
	targetPlatform := h.determineTargetPlatform(req.Model)

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
				Platform:          targetPlatform,
				UserID:            apiKey.UserID,
				APIKeyID:          apiKey.ID,
				RoutingTier:       getComplexityTier(complexityProfile),
				ExcludedAccountIDs: excluded,
			})
			if err != nil {
				return nil, err
			}

			// 根据目标平台构建请求
			if targetPlatform == "claude" {
				// 转换为 Claude 格式
				upstreamReq.URL = fmt.Sprintf("%s/v1/messages", account.BaseURL)
				upstreamReq.Headers = h.buildClaudeHeaders(account)
				convertedBody, convertErr := h.convertOpenAIToClaude(&req)
				if convertErr != nil {
					return nil, fmt.Errorf("请求格式转换失败: %w", convertErr)
				}
				upstreamReq.Body = convertedBody
			} else {
				// 保持 OpenAI 格式
				upstreamReq.URL = fmt.Sprintf("%s/v1/chat/completions", account.BaseURL)
				upstreamReq.Headers = h.buildOpenAIHeaders(account)
			}
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
		h.handleStreamResponse(c, resp, account, apiKey, startTime, req.Model, targetPlatform, complexityProfile, originalModel)
		return
	}

	// 处理非流式响应
	h.handleNonStreamResponse(c, resp, account, apiKey, startTime, req.Model, targetPlatform, complexityProfile, originalModel)
}

// Responses 处理 POST /v1/responses
// OpenAI Responses API（新版 API）
func (h *OpenAIGatewayHandler) Responses(c *gin.Context) {
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
	var req ResponsesRequest
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

	// 检查模型权限
	if !h.checkModelPermission(c, apiKey, req.Model) {
		ErrorResponse(c, http.StatusForbidden, "MODEL_NOT_ALLOWED", fmt.Sprintf("无权使用模型: %s", req.Model))
		return
	}

	// 选择上游账号
	account, err := h.AccountService.SelectAccount(c.Request.Context(), &AccountSelectRequest{
		Model:    req.Model,
		Platform: "openai",
		UserID:   apiKey.UserID,
		APIKeyID: apiKey.ID,
	})
	if err != nil {
		ErrorResponse(c, http.StatusServiceUnavailable, "NO_AVAILABLE_ACCOUNT", "暂无可用的上游账号")
		return
	}

	// 构建上游请求
	upstreamReq := &ProxyRequest{
		Method:    http.MethodPost,
		URL:       fmt.Sprintf("%s/v1/responses", account.BaseURL),
		Headers:   h.buildOpenAIHeaders(account),
		Body:      bodyBytes,
		Stream:    req.Stream,
		AccountID: account.ID,
		RequestID: requestID,
		Model:     req.Model,
		UserID:    apiKey.UserID,
		APIKeyID:  apiKey.ID,
	}

	// 发送请求到上游
	resp, err := h.ProxyService.DoRequest(c.Request.Context(), upstreamReq)
	if err != nil {
		ErrorResponse(c, http.StatusBadGateway, "UPSTREAM_ERROR", fmt.Sprintf("上游服务错误: %v", err))
		return
	}
	defer resp.Body.Close()

	// 处理响应
	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		ErrorResponse(c, http.StatusBadGateway, "UPSTREAM_ERROR", "读取上游响应失败")
		return
	}

	// 记录使用量
	latency := time.Since(startTime).Milliseconds()
	h.recordUsage(c, account, apiKey, req.Model, "openai", 0, 0, latency, "success", "")

	c.Data(resp.StatusCode, "application/json", bodyBytes)
}

// ImageGenerations 处理 POST /v1/images/generations
// OpenAI 图像生成 API
func (h *OpenAIGatewayHandler) ImageGenerations(c *gin.Context) {
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
	var req ImageGenerationsRequest
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
		// 默认使用 dall-e-3
		req.Model = "dall-e-3"
	}
	if req.Prompt == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_PROMPT", "缺少 prompt 参数")
		return
	}

	// 检查模型权限
	if !h.checkModelPermission(c, apiKey, req.Model) {
		ErrorResponse(c, http.StatusForbidden, "MODEL_NOT_ALLOWED", fmt.Sprintf("无权使用模型: %s", req.Model))
		return
	}

	// 选择上游账号
	account, err := h.AccountService.SelectAccount(c.Request.Context(), &AccountSelectRequest{
		Model:    req.Model,
		Platform: "openai",
		UserID:   apiKey.UserID,
		APIKeyID: apiKey.ID,
	})
	if err != nil {
		ErrorResponse(c, http.StatusServiceUnavailable, "NO_AVAILABLE_ACCOUNT", "暂无可用的上游账号")
		return
	}

	// 构建上游请求
	upstreamReq := &ProxyRequest{
		Method:    http.MethodPost,
		URL:       fmt.Sprintf("%s/v1/images/generations", account.BaseURL),
		Headers:   h.buildOpenAIHeaders(account),
		Body:      bodyBytes,
		Stream:    false,
		AccountID: account.ID,
		RequestID: requestID,
		Model:     req.Model,
		UserID:    apiKey.UserID,
		APIKeyID:  apiKey.ID,
	}

	// 发送请求到上游
	resp, err := h.ProxyService.DoRequest(c.Request.Context(), upstreamReq)
	if err != nil {
		ErrorResponse(c, http.StatusBadGateway, "UPSTREAM_ERROR", fmt.Sprintf("上游服务错误: %v", err))
		return
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		ErrorResponse(c, http.StatusBadGateway, "UPSTREAM_ERROR", "读取上游响应失败")
		return
	}

	// 记录使用量（图像生成按张计费）
	latency := time.Since(startTime).Milliseconds()
	h.recordUsage(c, account, apiKey, req.Model, "openai", 0, 0, latency, "success", "")

	c.Data(resp.StatusCode, "application/json", respBody)
}

// Embeddings 处理 POST /v1/embeddings
// OpenAI Embeddings API
func (h *OpenAIGatewayHandler) Embeddings(c *gin.Context) {
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
	var req EmbeddingsRequest
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
	if req.Input == nil {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_INPUT", "缺少 input 参数")
		return
	}

	// 检查模型权限
	if !h.checkModelPermission(c, apiKey, req.Model) {
		ErrorResponse(c, http.StatusForbidden, "MODEL_NOT_ALLOWED", fmt.Sprintf("无权使用模型: %s", req.Model))
		return
	}

	// 选择上游账号
	account, err := h.AccountService.SelectAccount(c.Request.Context(), &AccountSelectRequest{
		Model:    req.Model,
		Platform: "openai",
		UserID:   apiKey.UserID,
		APIKeyID: apiKey.ID,
	})
	if err != nil {
		ErrorResponse(c, http.StatusServiceUnavailable, "NO_AVAILABLE_ACCOUNT", "暂无可用的上游账号")
		return
	}

	// 构建上游请求
	upstreamReq := &ProxyRequest{
		Method:    http.MethodPost,
		URL:       fmt.Sprintf("%s/v1/embeddings", account.BaseURL),
		Headers:   h.buildOpenAIHeaders(account),
		Body:      bodyBytes,
		Stream:    false,
		AccountID: account.ID,
		RequestID: requestID,
		Model:     req.Model,
		UserID:    apiKey.UserID,
		APIKeyID:  apiKey.ID,
	}

	// 发送请求到上游
	resp, err := h.ProxyService.DoRequest(c.Request.Context(), upstreamReq)
	if err != nil {
		ErrorResponse(c, http.StatusBadGateway, "UPSTREAM_ERROR", fmt.Sprintf("上游服务错误: %v", err))
		return
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		ErrorResponse(c, http.StatusBadGateway, "UPSTREAM_ERROR", "读取上游响应失败")
		return
	}

	// 解析响应获取 token 使用量
	var embeddingsResp EmbeddingsResponse
	inputTokens := 0
	if err := json.Unmarshal(respBody, &embeddingsResp); err == nil {
		inputTokens = embeddingsResp.Usage.PromptTokens
	}

	// 记录使用量
	latency := time.Since(startTime).Milliseconds()
	h.recordUsage(c, account, apiKey, req.Model, "openai", inputTokens, 0, latency, "success", "")

	c.Data(resp.StatusCode, "application/json", respBody)
}

// handleStreamResponse 处理 SSE 流式响应
func (h *OpenAIGatewayHandler) handleStreamResponse(c *gin.Context, resp *http.Response, account *Account, apiKey *APIKeyContext, startTime time.Time, model, platform string, complexityProfile *ComplexityProfile, originalModel string) {
	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	writer := c.Writer
	reader := bufio.NewReader(resp.Body)

	var totalInputTokens, totalOutputTokens int

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

		// 处理 SSE 数据
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// 检查是否为结束标记
			if data == "[DONE]" {
				fmt.Fprintf(writer, "data: [DONE]\n\n")
				writer.Flush()
				break
			}

			// 解析 chunk 获取 token 使用量
			var chunk ChatCompletionChunk
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				// 可以在这里累积统计
			}

			// 转发给客户端
			fmt.Fprintf(writer, "data: %s\n\n", data)
			writer.Flush()
		}
	}

	// 记录使用量和计费
	latency := time.Since(startTime).Milliseconds()
	h.recordUsage(c, account, apiKey, model, platform, totalInputTokens, totalOutputTokens, latency, "success", "", complexityProfile, originalModel)
}

// handleNonStreamResponse 处理非流式响应
func (h *OpenAIGatewayHandler) handleNonStreamResponse(c *gin.Context, resp *http.Response, account *Account, apiKey *APIKeyContext, startTime time.Time, model, platform string, complexityProfile *ComplexityProfile, originalModel string) {
	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		ErrorResponse(c, http.StatusBadGateway, "UPSTREAM_ERROR", "读取上游响应失败")
		return
	}

	// 解析响应获取 token 使用量
	var chatResp ChatCompletionResponse
	inputTokens, outputTokens := 0, 0
	if err := json.Unmarshal(bodyBytes, &chatResp); err == nil {
		inputTokens = chatResp.Usage.PromptTokens
		outputTokens = chatResp.Usage.CompletionTokens
	}

	// 记录使用量和计费
	latency := time.Since(startTime).Milliseconds()
	h.recordUsage(c, account, apiKey, model, platform, inputTokens, outputTokens, latency, "success", "", complexityProfile, originalModel)

	// 返回响应
	c.Data(resp.StatusCode, "application/json", bodyBytes)
}

// determineTargetPlatform 根据模型名称确定目标平台
func (h *OpenAIGatewayHandler) determineTargetPlatform(model string) string {
	// Claude 模型
	if strings.HasPrefix(model, "claude") {
		return "claude"
	}
	// Gemini 模型
	if strings.HasPrefix(model, "gemini") {
		return "gemini"
	}
	// 默认为 OpenAI
	return "openai"
}

// convertOpenAIToClaude 将 OpenAI 格式转换为 Claude 格式
func (h *OpenAIGatewayHandler) convertOpenAIToClaude(req *ChatCompletionRequest) ([]byte, error) {
	// 构建 Claude 请求
	claudeReq := ClaudeMessagesRequest{
		Model:    req.Model,
		MaxTokens: 4096, // 默认值
		Stream:   req.Stream,
	}

	// 设置 max_tokens
	if req.MaxTokens != nil {
		claudeReq.MaxTokens = *req.MaxTokens
	}

	// 设置 temperature
	if req.Temperature != nil {
		claudeReq.Temperature = *req.Temperature
	}

	// 设置 top_p
	if req.TopP != nil {
		claudeReq.TopP = *req.TopP
	}

	// 转换消息格式
	claudeReq.Messages = make([]ClaudeMessage, 0, len(req.Messages))
	var systemPrompt string

	for _, msg := range req.Messages {
		// 提取 system 消息
		if msg.Role == "system" {
			// 解析 content
			var content string
			if err := json.Unmarshal(msg.Content, &content); err == nil {
				systemPrompt = content
			}
			continue
		}

		// 转换其他消息
		claudeMsg := ClaudeMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
		claudeReq.Messages = append(claudeReq.Messages, claudeMsg)
	}

	if systemPrompt != "" {
		claudeReq.System = systemPrompt
	}

	// 转换工具定义
	if len(req.Tools) > 0 {
		claudeReq.Tools = make([]ClaudeTool, 0, len(req.Tools))
		for _, tool := range req.Tools {
			if tool.Type == "function" {
				claudeTool := ClaudeTool{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					InputSchema: tool.Function.Parameters,
				}
				claudeReq.Tools = append(claudeReq.Tools, claudeTool)
			}
		}
	}

	return json.Marshal(claudeReq)
}

// buildOpenAIHeaders 构建 OpenAI API 请求头
func (h *OpenAIGatewayHandler) buildOpenAIHeaders(account *Account) map[string]string {
	return map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", account.APIKey),
	}
}

// buildClaudeHeaders 构建 Claude API 请求头
func (h *OpenAIGatewayHandler) buildClaudeHeaders(account *Account) map[string]string {
	return map[string]string{
		"Content-Type":      "application/json",
		"anthropic-version": "2023-06-01",
		"x-api-key":         account.APIKey,
	}
}

// checkModelPermission 检查模型使用权限
func (h *OpenAIGatewayHandler) checkModelPermission(c *gin.Context, apiKey *APIKeyContext, model string) bool {
	if len(apiKey.AllowedModels) == 0 {
		return true
	}

	for _, allowed := range apiKey.AllowedModels {
		if allowed == model || allowed == "*" {
			return true
		}
		if strings.HasSuffix(allowed, "*") {
			prefix := strings.TrimSuffix(allowed, "*")
			if strings.HasPrefix(model, prefix) {
				return true
			}
		}
	}

	return false
}

// recordUsage 记录使用量和计费
func (h *OpenAIGatewayHandler) recordUsage(c *gin.Context, account *Account, apiKey *APIKeyContext, model, platform string, inputTokens, outputTokens int, latencyMs int64, status, errorMsg string, complexityProfile *ComplexityProfile, originalModel string) {
	if h.BillingService == nil {
		return
	}

	if skipBilling, _ := c.Get(string(ctxkey.ContextKeySkipBilling)); skipBilling == true {
		return
	}

	cost := h.BillingService.CalculateCost(model, inputTokens, outputTokens)

	record := &UsageRecord{
		RequestID:        c.GetString(string(ctxkey.ContextKeyRequestID)),
		UserID:           apiKey.UserID,
		APIKeyID:         apiKey.ID,
		AccountID:        account.ID,
		Model:            model,
		Platform:         platform,
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

	go h.BillingService.RecordUsage(c.Request.Context(), record)
}

// RouterService 路由服务接口
type RouterService interface {
	// RouteRequest 路由请求到合适的平台
	RouteRequest(ctx interface{}, req *ChatCompletionRequest) (platform string, err error)
}

// ModelMappingService 模型映射服务接口
type ModelMappingService interface {
	// ResolveMapping 解析模型映射
	ResolveMapping(model string) string
}

// RegisterOpenAIHandlers 注册 OpenAI API 兼容路由到 HandlerGroup
func RegisterOpenAIHandlers(h *OpenAIGatewayHandler) map[string]gin.HandlerFunc {
	return map[string]gin.HandlerFunc{
		"ChatCompletions":   h.ChatCompletions,
		"Responses":         h.Responses,
		"ImageGenerations":  h.ImageGenerations,
		"Embeddings":        h.Embeddings,
	}
}

// OpenAIStreamProcessor OpenAI 流式处理器
type OpenAIStreamProcessor struct {
	writer  http.ResponseWriter
	flusher http.Flusher
}

// NewOpenAIStreamProcessor 创建 OpenAI 流式处理器
func NewOpenAIStreamProcessor(w http.ResponseWriter) *OpenAIStreamProcessor {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}
	return &OpenAIStreamProcessor{
		writer:  w,
		flusher: flusher,
	}
}

// WriteChunk 写入流式块
func (p *OpenAIStreamProcessor) WriteChunk(chunk *ChatCompletionChunk) error {
	data, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	fmt.Fprintf(p.writer, "data: %s\n\n", string(data))
	p.flusher.Flush()
	return nil
}

// WriteDone 写入结束标记
func (p *OpenAIStreamProcessor) WriteDone() {
	fmt.Fprintf(p.writer, "data: [DONE]\n\n")
	p.flusher.Flush()
}

// ParseOpenAIStreamChunk 解析 OpenAI 流式块
func ParseOpenAIStreamChunk(data string) (*ChatCompletionChunk, error) {
	var chunk ChatCompletionChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil, err
	}
	return &chunk, nil
}

// ConvertClaudeToOpenAI 将 Claude 响应转换为 OpenAI 格式
func ConvertClaudeToOpenAI(claudeResp *ClaudeMessagesResponse, model string) *ChatCompletionResponse {
	// 提取文本内容
	var content string
	for _, block := range claudeResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	// 构建 OpenAI 响应
	resp := &ChatCompletionResponse{
		ID:      claudeResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []ChatCompletionChoice{
			{
				Index: 0,
				Message: &ChatMessage{
					Role:    "assistant",
					Content: json.RawMessage(fmt.Sprintf(`"%s"`, content)),
				},
				FinishReason: "stop",
			},
		},
		Usage: ChatUsage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
	}

	// 设置 finish_reason
	if claudeResp.StopReason != nil {
		resp.Choices[0].FinishReason = *claudeResp.StopReason
	}

	return resp
}

// ConvertClaudeStreamToOpenAI 将 Claude 流式响应转换为 OpenAI 格式
func ConvertClaudeStreamToOpenAI(event *ClaudeStreamEvent, model string, id string) *ChatCompletionChunk {
	chunk := &ChatCompletionChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []ChatCompletionChoice{
			{
				Index: 0,
				Delta:  &ChatDelta{},
			},
		},
	}

	switch event.Type {
	case "content_block_start":
		// 开始内容块
		chunk.Choices[0].Delta.Role = "assistant"

	case "content_block_delta":
		// 增量内容
		if event.Delta != nil {
			chunk.Choices[0].Delta.Content = event.Delta.Text
		}

	case "content_block_stop":
		// 内容块结束

	case "message_stop":
		// 消息结束
		chunk.Choices[0].FinishReason = "stop"

	case "message_delta":
		// 消息增量（包含使用量）
		if event.Delta != nil && event.Delta.StopReason != "" {
			chunk.Choices[0].FinishReason = event.Delta.StopReason
		}
	}

	return chunk
}

// IsStreamingError 检查是否为流式错误
func IsStreamingError(data string) bool {
	return strings.Contains(data, `"error"`)
}

// ParseStreamingError 解析流式错误
func ParseStreamingError(data string) (string, string, error) {
	var errResp struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(data), &errResp); err != nil {
		return "", "", err
	}
	return errResp.Error.Type, errResp.Error.Message, nil
}

// CloneRequestBody 克隆请求体
func CloneRequestBody(body io.ReadCloser) ([]byte, io.ReadCloser, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, nil, err
	}
	return bodyBytes, io.NopCloser(bytes.NewBuffer(bodyBytes)), nil
}

// chatMessagesToGeneric 将 OpenAI Chat 消息转换为通用消息格式（用于复杂度分析）
func chatMessagesToGeneric(messages []ChatMessage) []GenericMessage {
	result := make([]GenericMessage, 0, len(messages))
	for _, msg := range messages {
		var content string
		if err := json.Unmarshal(msg.Content, &content); err == nil {
			result = append(result, GenericMessage{
				Role:    msg.Role,
				Content: content,
			})
		} else {
			result = append(result, GenericMessage{
				Role:    msg.Role,
				Content: string(msg.Content),
			})
		}
	}
	return result
}

// getMaxTokens 安全获取 MaxTokens 指针值
func getMaxTokens(maxTokens *int) int {
	if maxTokens == nil {
		return 0
	}
	return *maxTokens
}
