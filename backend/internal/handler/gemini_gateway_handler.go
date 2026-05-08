// Package handler 提供 MaaS-Router 的 HTTP 处理器
package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"maas-router/internal/pkg/ctxkey"
)

// GeminiGatewayHandler Gemini API 兼容网关 Handler
// 处理 Gemini API 格式的请求
type GeminiGatewayHandler struct {
	// AccountService 账号调度服务
	AccountService AccountService
	// BillingService 计费服务
	BillingService BillingService
	// ProxyService 代理转发服务
	ProxyService ProxyService
	// ModelService 模型信息服务
	ModelService ModelService
}

// NewGeminiGatewayHandler 创建 Gemini API 兼容网关 Handler
func NewGeminiGatewayHandler(
	accountService AccountService,
	billingService BillingService,
	proxyService ProxyService,
	modelService ModelService,
) *GeminiGatewayHandler {
	return &GeminiGatewayHandler{
		AccountService: accountService,
		BillingService: billingService,
		ProxyService:   proxyService,
		ModelService:   modelService,
	}
}

// GeminiGenerateContentRequest Gemini 生成内容请求
type GeminiGenerateContentRequest struct {
	Contents         []GeminiContent      `json:"contents"`
	Tools           []GeminiTool         `json:"tools,omitempty"`
	SafetySettings  []GeminiSafetySetting `json:"safetySettings,omitempty"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
	SystemInstruction *GeminiContent `json:"systemInstruction,omitempty"`
}

// GeminiContent Gemini 内容结构
type GeminiContent struct {
	Role  string          `json:"role,omitempty"`
	Parts []GeminiPart    `json:"parts"`
}

// GeminiPart Gemini 内容部分
type GeminiPart struct {
	Text         string                  `json:"text,omitempty"`
	InlineData  *GeminiInlineData       `json:"inlineData,omitempty"`
	FunctionCall *GeminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"`
}

// GeminiInlineData 内联数据（图片等）
type GeminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// GeminiFunctionCall 函数调用
type GeminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// GeminiFunctionResponse 函数响应
type GeminiFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

// GeminiTool Gemini 工具定义
type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
	CodeExecution        *GeminiCodeExecution        `json:"codeExecution,omitempty"`
}

// GeminiFunctionDeclaration 函数声明
type GeminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// GeminiCodeExecution 代码执行配置
type GeminiCodeExecution struct{}

// GeminiSafetySetting 安全设置
type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GeminiGenerationConfig 生成配置
type GeminiGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP           *float64 `json:"topP,omitempty"`
	TopK           *int     `json:"topK,omitempty"`
	CandidateCount *int     `json:"candidateCount,omitempty"`
	MaxOutputTokens *int    `json:"maxOutputTokens,omitempty"`
	StopSequences  []string `json:"stopSequences,omitempty"`
	ResponseMimeType string `json:"responseMimeType,omitempty"`
	ResponseSchema map[string]interface{} `json:"responseSchema,omitempty"`
}

// GeminiGenerateContentResponse Gemini 生成内容响应
type GeminiGenerateContentResponse struct {
	Candidates    []GeminiCandidate `json:"candidates"`
	PromptFeedback *GeminiPromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata *GeminiUsageMetadata `json:"usageMetadata,omitempty"`
}

// GeminiCandidate 候选结果
type GeminiCandidate struct {
	Content       GeminiContent        `json:"content"`
	FinishReason  string               `json:"finishReason,omitempty"`
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings,omitempty"`
	CitationMetadata *GeminiCitationMetadata `json:"citationMetadata,omitempty"`
}

// GeminiSafetyRating 安全评级
type GeminiSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// GeminiCitationMetadata 引用元数据
type GeminiCitationMetadata struct {
	CitationSources []GeminiCitationSource `json:"citationSources"`
}

// GeminiCitationSource 引用来源
type GeminiCitationSource struct {
	StartIndex int `json:"startIndex"`
	EndIndex   int `json:"endIndex"`
	URI        string `json:"uri"`
}

// GeminiPromptFeedback 提示反馈
type GeminiPromptFeedback struct {
	BlockReason   string               `json:"blockReason,omitempty"`
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings"`
}

// GeminiUsageMetadata 使用量元数据
type GeminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// GeminiModelInfo Gemini 模型信息
type GeminiModelInfo struct {
	Name                  string   `json:"name"`
	Version               string   `json:"version,omitempty"`
	DisplayName           string   `json:"displayName,omitempty"`
	Description           string   `json:"description,omitempty"`
	InputTokenLimit       int      `json:"inputTokenLimit,omitempty"`
	OutputTokenLimit      int      `json:"outputTokenLimit,omitempty"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods,omitempty"`
	Temperature           *float64 `json:"temperature,omitempty"`
	TopP                  *float64 `json:"topP,omitempty"`
	TopK                  *int     `json:"topK,omitempty"`
}

// GeminiListModelsResponse 模型列表响应
type GeminiListModelsResponse struct {
	Models    []GeminiModelInfo `json:"models"`
	NextPageToken string         `json:"nextPageToken,omitempty"`
}

// ListModelsBeta 处理 GET /v1beta/models
// 返回 Gemini 可用模型列表
func (h *GeminiGatewayHandler) ListModelsBeta(c *gin.Context) {
	// 获取 API Key 信息
	apiKeyInfo, exists := c.Get(string(ctxkey.ContextKeyAPIKey))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}
	apiKey := apiKeyInfo.(*APIKeyContext)

	// 获取查询参数
	pageSize := c.Query("pageSize")
	pageToken := c.Query("pageToken")

	// 选择上游账号
	account, err := h.AccountService.SelectAccount(c.Request.Context(), &AccountSelectRequest{
		Model:    "gemini",
		Platform: "gemini",
		UserID:   apiKey.UserID,
		APIKeyID: apiKey.ID,
	})
	if err != nil {
		ErrorResponse(c, http.StatusServiceUnavailable, "NO_AVAILABLE_ACCOUNT", "暂无可用的上游账号")
		return
	}

	// 构建上游请求 URL
	upstreamURL := fmt.Sprintf("%s/v1beta/models", account.BaseURL)
	if pageSize != "" {
		upstreamURL += fmt.Sprintf("?pageSize=%s", pageSize)
		if pageToken != "" {
			upstreamURL += fmt.Sprintf("&pageToken=%s", pageToken)
		}
	} else if pageToken != "" {
		upstreamURL += fmt.Sprintf("?pageToken=%s", pageToken)
	}

	// 构建代理请求
	upstreamReq := &ProxyRequest{
		Method:    http.MethodGet,
		URL:       upstreamURL,
		Headers:   h.buildGeminiHeaders(account),
		AccountID: account.ID,
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
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		ErrorResponse(c, http.StatusBadGateway, "UPSTREAM_ERROR", "读取上游响应失败")
		return
	}

	c.Data(resp.StatusCode, "application/json", bodyBytes)
}

// GetModel 处理 GET /v1beta/models/:model
// 返回指定模型信息
func (h *GeminiGatewayHandler) GetModel(c *gin.Context) {
	// 获取 API Key 信息
	apiKeyInfo, exists := c.Get(string(ctxkey.ContextKeyAPIKey))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}
	apiKey := apiKeyInfo.(*APIKeyContext)

	// 获取模型名称
	modelName := c.Param("model")
	if modelName == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_MODEL", "缺少模型名称")
		return
	}

	// 选择上游账号
	account, err := h.AccountService.SelectAccount(c.Request.Context(), &AccountSelectRequest{
		Model:    modelName,
		Platform: "gemini",
		UserID:   apiKey.UserID,
		APIKeyID: apiKey.ID,
	})
	if err != nil {
		ErrorResponse(c, http.StatusServiceUnavailable, "NO_AVAILABLE_ACCOUNT", "暂无可用的上游账号")
		return
	}

	// 构建上游请求 URL
	upstreamURL := fmt.Sprintf("%s/v1beta/models/%s", account.BaseURL, modelName)

	// 构建代理请求
	upstreamReq := &ProxyRequest{
		Method:    http.MethodGet,
		URL:       upstreamURL,
		Headers:   h.buildGeminiHeaders(account),
		AccountID: account.ID,
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
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		ErrorResponse(c, http.StatusBadGateway, "UPSTREAM_ERROR", "读取上游响应失败")
		return
	}

	c.Data(resp.StatusCode, "application/json", bodyBytes)
}

// ModelAction 处理 POST /v1beta/models/*modelAction
// 支持 generateContent, streamGenerateContent, countTokens 等操作
func (h *GeminiGatewayHandler) ModelAction(c *gin.Context) {
	startTime := time.Now()
	requestID := c.GetString(string(ctxkey.ContextKeyRequestID))

	// 获取 API Key 信息
	apiKeyInfo, exists := c.Get(string(ctxkey.ContextKeyAPIKey))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}
	apiKey := apiKeyInfo.(*APIKeyContext)

	// 解析路径获取模型和操作
	// 路径格式: /v1beta/models/{model}:{action}
	path := c.Param("modelAction")
	modelName, action := h.parseModelAction(path)

	if modelName == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_MODEL", "缺少模型名称")
		return
	}

	if action == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_ACTION", "缺少操作类型")
		return
	}

	// 检查模型权限
	if !h.checkModelPermission(c, apiKey, modelName) {
		ErrorResponse(c, http.StatusForbidden, "MODEL_NOT_ALLOWED", fmt.Sprintf("无权使用模型: %s", modelName))
		return
	}

	// 解析请求体
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "无法读取请求体")
		return
	}

	// 选择上游账号
	account, err := h.AccountService.SelectAccount(c.Request.Context(), &AccountSelectRequest{
		Model:    modelName,
		Platform: "gemini",
		UserID:   apiKey.UserID,
		APIKeyID: apiKey.ID,
	})
	if err != nil {
		ErrorResponse(c, http.StatusServiceUnavailable, "NO_AVAILABLE_ACCOUNT", "暂无可用的上游账号")
		return
	}

	// 根据操作类型处理
	switch action {
	case "generateContent":
		h.handleGenerateContent(c, account, apiKey, modelName, bodyBytes, requestID, startTime)
	case "streamGenerateContent":
		h.handleStreamGenerateContent(c, account, apiKey, modelName, bodyBytes, requestID, startTime)
	case "countTokens":
		h.handleCountTokens(c, account, apiKey, modelName, bodyBytes)
	default:
		ErrorResponse(c, http.StatusBadRequest, "UNSUPPORTED_ACTION", fmt.Sprintf("不支持的操作: %s", action))
	}
}

// handleGenerateContent 处理非流式生成内容
func (h *GeminiGatewayHandler) handleGenerateContent(c *gin.Context, account *Account, apiKey *APIKeyContext, modelName string, bodyBytes []byte, requestID string, startTime time.Time) {
	// 构建上游请求 URL
	upstreamURL := fmt.Sprintf("%s/v1beta/models/%s:generateContent", account.BaseURL, modelName)

	// 构建代理请求
	upstreamReq := &ProxyRequest{
		Method:    http.MethodPost,
		URL:       upstreamURL,
		Headers:   h.buildGeminiHeaders(account),
		Body:      bodyBytes,
		Stream:    false,
		AccountID: account.ID,
		RequestID: requestID,
		Model:     modelName,
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
	var geminiResp GeminiGenerateContentResponse
	inputTokens, outputTokens := 0, 0
	if err := json.Unmarshal(respBody, &geminiResp); err == nil {
		if geminiResp.UsageMetadata != nil {
			inputTokens = geminiResp.UsageMetadata.PromptTokenCount
			outputTokens = geminiResp.UsageMetadata.CandidatesTokenCount
		}
	}

	// 记录使用量
	latency := time.Since(startTime).Milliseconds()
	h.recordUsage(c, account, apiKey, modelName, inputTokens, outputTokens, latency, "success", "")

	c.Data(resp.StatusCode, "application/json", respBody)
}

// handleStreamGenerateContent 处理流式生成内容
func (h *GeminiGatewayHandler) handleStreamGenerateContent(c *gin.Context, account *Account, apiKey *APIKeyContext, modelName string, bodyBytes []byte, requestID string, startTime time.Time) {
	// 构建上游请求 URL（流式）
	upstreamURL := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse", account.BaseURL, modelName)

	// 构建代理请求
	upstreamReq := &ProxyRequest{
		Method:    http.MethodPost,
		URL:       upstreamURL,
		Headers:   h.buildGeminiHeaders(account),
		Body:      bodyBytes,
		Stream:    true,
		AccountID: account.ID,
		RequestID: requestID,
		Model:     modelName,
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

			// 解析事件获取 token 使用量
			var geminiResp GeminiGenerateContentResponse
			if err := json.Unmarshal([]byte(data), &geminiResp); err == nil {
				if geminiResp.UsageMetadata != nil {
					totalInputTokens = geminiResp.UsageMetadata.PromptTokenCount
					totalOutputTokens = geminiResp.UsageMetadata.CandidatesTokenCount
				}
			}

			// 转发给客户端
			fmt.Fprintf(writer, "data: %s\n\n", data)
			writer.Flush()
		}
	}

	// 记录使用量
	latency := time.Since(startTime).Milliseconds()
	h.recordUsage(c, account, apiKey, modelName, totalInputTokens, totalOutputTokens, latency, "success", "")
}

// handleCountTokens 处理 Token 计数
func (h *GeminiGatewayHandler) handleCountTokens(c *gin.Context, account *Account, apiKey *APIKeyContext, modelName string, bodyBytes []byte) {
	// 构建上游请求 URL
	upstreamURL := fmt.Sprintf("%s/v1beta/models/%s:countTokens", account.BaseURL, modelName)

	// 构建代理请求
	upstreamReq := &ProxyRequest{
		Method:    http.MethodPost,
		URL:       upstreamURL,
		Headers:   h.buildGeminiHeaders(account),
		Body:      bodyBytes,
		Stream:    false,
		AccountID: account.ID,
		Model:     modelName,
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
	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		ErrorResponse(c, http.StatusBadGateway, "UPSTREAM_ERROR", "读取上游响应失败")
		return
	}

	c.Data(resp.StatusCode, "application/json", bodyBytes)
}

// parseModelAction 解析模型和操作
// 输入格式: /models/gemini-pro:generateContent 或 /gemini-pro:generateContent
func (h *GeminiGatewayHandler) parseModelAction(path string) (model, action string) {
	// 移除前导斜杠
	path = strings.TrimPrefix(path, "/")

	// 移除 models/ 前缀（如果有）
	path = strings.TrimPrefix(path, "models/")

	// 使用正则匹配模型名和操作
	re := regexp.MustCompile(`^([^:]+):(.+)$`)
	matches := re.FindStringSubmatch(path)

	if len(matches) == 3 {
		return matches[1], matches[2]
	}

	// 如果没有操作，只返回模型名
	if path != "" && !strings.Contains(path, ":") {
		return path, ""
	}

	return "", ""
}

// buildGeminiHeaders 构建 Gemini API 请求头
func (h *GeminiGatewayHandler) buildGeminiHeaders(account *Account) map[string]string {
	return map[string]string{
		"Content-Type":  "application/json",
		"x-goog-api-key": account.APIKey,
	}
}

// checkModelPermission 检查模型使用权限
func (h *GeminiGatewayHandler) checkModelPermission(c *gin.Context, apiKey *APIKeyContext, model string) bool {
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
func (h *GeminiGatewayHandler) recordUsage(c *gin.Context, account *Account, apiKey *APIKeyContext, model string, inputTokens, outputTokens int, latencyMs int64, status, errorMsg string) {
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
		Platform:         "gemini",
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

	go h.BillingService.RecordUsage(c.Request.Context(), record)
}

// RegisterGeminiHandlers 注册 Gemini API 兼容路由到 HandlerGroup
func RegisterGeminiHandlers(h *GeminiGatewayHandler) map[string]gin.HandlerFunc {
	return map[string]gin.HandlerFunc{
		"ListModelsBeta": h.ListModelsBeta,
		"GetModel":       h.GetModel,
		"ModelAction":    h.ModelAction,
	}
}

// ConvertGeminiToOpenAI 将 Gemini 响应转换为 OpenAI 格式
func ConvertGeminiToOpenAI(geminiResp *GeminiGenerateContentResponse, model string) *ChatCompletionResponse {
	// 提取文本内容
	var content string
	if len(geminiResp.Candidates) > 0 {
		for _, part := range geminiResp.Candidates[0].Content.Parts {
			content += part.Text
		}
	}

	// 构建 OpenAI 响应
	resp := &ChatCompletionResponse{
		ID:      fmt.Sprintf("gemini-%d", time.Now().UnixNano()),
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
	}

	// 设置使用量
	if geminiResp.UsageMetadata != nil {
		resp.Usage = ChatUsage{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		}
	}

	// 设置 finish_reason
	if len(geminiResp.Candidates) > 0 && geminiResp.Candidates[0].FinishReason != "" {
		finishReason := geminiResp.Candidates[0].FinishReason
		// 转换 Gemini 的 finish_reason 到 OpenAI 格式
		switch finishReason {
		case "STOP":
			resp.Choices[0].FinishReason = "stop"
		case "MAX_TOKENS":
			resp.Choices[0].FinishReason = "length"
		case "SAFETY":
			resp.Choices[0].FinishReason = "content_filter"
		case "RECITATION":
			resp.Choices[0].FinishReason = "content_filter"
		default:
			resp.Choices[0].FinishReason = finishReason
		}
	}

	return resp
}

// ConvertOpenAIToGemini 将 OpenAI 请求转换为 Gemini 格式
func ConvertOpenAIToGemini(req *ChatCompletionRequest) (*GeminiGenerateContentRequest, error) {
	geminiReq := &GeminiGenerateContentRequest{
		Contents: make([]GeminiContent, 0, len(req.Messages)),
	}

	// 转换消息
	var systemPrompt string
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			// 提取 system 消息
			var content string
			if err := json.Unmarshal(msg.Content, &content); err == nil {
				systemPrompt = content
			}
			continue
		}

		// 转换角色
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		// 解析内容
		var textContent string
		var parts []GeminiPart

		// 尝试解析为字符串
		if err := json.Unmarshal(msg.Content, &textContent); err == nil {
			parts = []GeminiPart{{Text: textContent}}
		} else {
			// 尝试解析为多模态内容
			var contentParts []ContentPart
			if err := json.Unmarshal(msg.Content, &contentParts); err == nil {
				parts = make([]GeminiPart, 0, len(contentParts))
				for _, cp := range contentParts {
					if cp.Type == "text" {
						parts = append(parts, GeminiPart{Text: cp.Text})
					} else if cp.Type == "image_url" && cp.ImageURL != nil {
						// 解析 base64 图片数据
						if strings.HasPrefix(cp.ImageURL.URL, "data:") {
							// 格式: data:image/png;base64,xxxxx
							parts := strings.SplitN(cp.ImageURL.URL, ",", 2)
							if len(parts) == 2 {
								mimeType := strings.TrimPrefix(parts[0], "data:")
								mimeType = strings.TrimSuffix(mimeType, ";base64")
								parts = append(parts, GeminiPart{
									InlineData: &GeminiInlineData{
										MimeType: mimeType,
										Data:     parts[1],
									},
								})
							}
						}
					}
				}
			}
		}

		geminiReq.Contents = append(geminiReq.Contents, GeminiContent{
			Role:  role,
			Parts: parts,
		})
	}

	// 设置 system instruction
	if systemPrompt != "" {
		geminiReq.SystemInstruction = &GeminiContent{
			Parts: []GeminiPart{{Text: systemPrompt}},
		}
	}

	// 转换生成配置
	if req.MaxTokens != nil || req.Temperature != nil || req.TopP != nil {
		geminiReq.GenerationConfig = &GeminiGenerationConfig{}
		if req.MaxTokens != nil {
			geminiReq.GenerationConfig.MaxOutputTokens = req.MaxTokens
		}
		if req.Temperature != nil {
			geminiReq.GenerationConfig.Temperature = req.Temperature
		}
		if req.TopP != nil {
			geminiReq.GenerationConfig.TopP = req.TopP
		}
	}

	// 转换工具
	if len(req.Tools) > 0 {
		geminiReq.Tools = make([]GeminiTool, 0, 1)
		tool := GeminiTool{
			FunctionDeclarations: make([]GeminiFunctionDeclaration, 0, len(req.Tools)),
		}
		for _, t := range req.Tools {
			if t.Type == "function" {
				decl := GeminiFunctionDeclaration{
					Name:        t.Function.Name,
					Description: t.Function.Description,
				}
				// 解析 parameters
				if t.Function.Parameters != nil {
					var params map[string]interface{}
					if err := json.Unmarshal(t.Function.Parameters, &params); err == nil {
						decl.Parameters = params
					}
				}
				tool.FunctionDeclarations = append(tool.FunctionDeclarations, decl)
			}
		}
		if len(tool.FunctionDeclarations) > 0 {
			geminiReq.Tools = append(geminiReq.Tools, tool)
		}
	}

	return geminiReq, nil
}

// GeminiStreamProcessor Gemini 流式处理器
type GeminiStreamProcessor struct {
	writer  http.ResponseWriter
	flusher http.Flusher
}

// NewGeminiStreamProcessor 创建 Gemini 流式处理器
func NewGeminiStreamProcessor(w http.ResponseWriter) *GeminiStreamProcessor {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}
	return &GeminiStreamProcessor{
		writer:  w,
		flusher: flusher,
	}
}

// WriteResponse 写入 Gemini 响应
func (p *GeminiStreamProcessor) WriteResponse(resp *GeminiGenerateContentResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	fmt.Fprintf(p.writer, "data: %s\n\n", string(data))
	p.flusher.Flush()
	return nil
}

// ParseGeminiStreamResponse 解析 Gemini 流式响应
func ParseGeminiStreamResponse(data string) (*GeminiGenerateContentResponse, error) {
	var resp GeminiGenerateContentResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// IsGeminiError 检查是否为 Gemini 错误响应
func IsGeminiError(data string) bool {
	return strings.Contains(data, `"error"`)
}

// ParseGeminiError 解析 Gemini 错误
func ParseGeminiError(data string) (int, string, string, error) {
	var errResp struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(data), &errResp); err != nil {
		return 0, "", "", err
	}
	return errResp.Error.Code, errResp.Error.Status, errResp.Error.Message, nil
}

// GetGeminiModelName 从完整模型名中提取简短名称
// 输入: "models/gemini-pro" 或 "gemini-pro"
// 输出: "gemini-pro"
func GetGeminiModelName(fullName string) string {
	return strings.TrimPrefix(fullName, "models/")
}
