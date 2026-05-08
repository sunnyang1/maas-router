// Package service 业务服务层
// 提供 Claude 网关服务
package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"maas-router/ent"
	"maas-router/internal/config"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// ClaudeGatewayService Claude 网关服务接口
// 处理 Claude Messages API 相关请求
type ClaudeGatewayService interface {
	// Messages 处理 Claude Messages API
	Messages(ctx context.Context, req *ClaudeMessagesRequest) (*ClaudeMessagesResponse, error)

	// MessagesStream 处理 Claude Messages API 流式请求
	MessagesStream(ctx context.Context, req *ClaudeMessagesRequest, callback func(event *SSEEvent) error) error

	// CountTokens Token 计数
	CountTokens(ctx context.Context, req *ClaudeCountTokensRequest) (*ClaudeCountTokensResponse, error)

	// ListModels 获取模型列表
	ListModels(ctx context.Context) (*ClaudeModelsResponse, error)

	// GetUsage 获取用量
	GetUsage(ctx context.Context, apiKeyID int64) (*ClaudeUsageResponse, error)
}

// ClaudeMessagesRequest Claude Messages API 请求
type ClaudeMessagesRequest struct {
	Model       string                   `json:"model"`
	Messages    []ClaudeMessage          `json:"messages"`
	MaxTokens   int                      `json:"max_tokens"`
	Temperature float64                  `json:"temperature,omitempty"`
	TopP        float64                  `json:"top_p,omitempty"`
	TopK        int                      `json:"top_k,omitempty"`
	Stream      bool                     `json:"stream,omitempty"`
	System      string                   `json:"system,omitempty"`
	Tools       []ClaudeTool             `json:"tools,omitempty"`
	ToolChoice  *ClaudeToolChoice        `json:"tool_choice,omitempty"`
	Metadata    *ClaudeMetadata          `json:"metadata,omitempty"`
	StopSequences []string               `json:"stop_sequences,omitempty"`

	// 内部字段
	Account    *ent.Account `json:"-"`
	SessionID  string       `json:"-"`
	RequestID  string       `json:"-"`
	APIKeyID   int64        `json:"-"`
	UserID     int64        `json:"-"`
}

// ClaudeMessage Claude 消息
type ClaudeMessage struct {
	Role    string        `json:"role"`
	Content interface{}   `json:"content"` // 可以是 string 或 []ClaudeContentBlock
}

// ClaudeContentBlock Claude 内容块
type ClaudeContentBlock struct {
	Type string `json:"type"` // text, image, tool_use, tool_result
	Text string `json:"text,omitempty"`
	
	// 图片相关
	Source *ClaudeImageSource `json:"source,omitempty"`
	
	// 工具相关
	ID       string          `json:"id,omitempty"`
	Name     string          `json:"name,omitempty"`
	Input    json.RawMessage `json:"input,omitempty"`
	ToolUseID string         `json:"tool_use_id,omitempty"`
	Content  string          `json:"content,omitempty"`
	IsError  bool            `json:"is_error,omitempty"`
}

// ClaudeImageSource 图片源
type ClaudeImageSource struct {
	Type      string `json:"type"`       // base64, url
	MediaType string `json:"media_type"` // image/jpeg, image/png, image/gif, image/webp
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
}

// ClaudeTool Claude 工具定义
type ClaudeTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ClaudeToolChoice 工具选择
type ClaudeToolChoice struct {
	Type string `json:"type"` // auto, any, tool
	Name string `json:"name,omitempty"`
}

// ClaudeMetadata 元数据
type ClaudeMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

// ClaudeMessagesResponse Claude Messages API 响应
type ClaudeMessagesResponse struct {
	ID           string                `json:"id"`
	Type         string                `json:"type"`
	Role         string                `json:"role"`
	Content      []ClaudeContentBlock  `json:"content"`
	Model        string                `json:"model"`
	StopReason   string                `json:"stop_reason,omitempty"`
	StopSequence string                `json:"stop_sequence,omitempty"`
	Usage        ClaudeUsageInfo       `json:"usage"`
}

// ClaudeUsageInfo Claude 用量信息
type ClaudeUsageInfo struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ClaudeCountTokensRequest Token 计数请求
type ClaudeCountTokensRequest struct {
	Model    string          `json:"model"`
	Messages []ClaudeMessage `json:"messages"`
	System   string          `json:"system,omitempty"`
	Tools    []ClaudeTool    `json:"tools,omitempty"`
}

// ClaudeCountTokensResponse Token 计数响应
type ClaudeCountTokensResponse struct {
	InputTokens int `json:"input_tokens"`
}

// ClaudeModelsResponse Claude 模型列表响应
type ClaudeModelsResponse struct {
	Data []ClaudeModelInfo `json:"data"`
}

// ClaudeModelInfo Claude 模型信息
type ClaudeModelInfo struct {
	ID         string `json:"id"`
	Object     string `json:"object"`
	Created    int64  `json:"created"`
	OwnedBy    string `json:"owned_by"`
}

// ClaudeUsageResponse Claude 用量响应
type ClaudeUsageResponse struct {
	TotalTokens int64   `json:"total_tokens"`
	TotalCost   float64 `json:"total_cost"`
	PeriodStart string  `json:"period_start"`
	PeriodEnd   string  `json:"period_end"`
}

// SSEEvent SSE 事件
type SSEEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// ClaudeStreamEvent Claude 流式事件
type ClaudeStreamEvent struct {
	Type         string               `json:"type"`
	Index        int                  `json:"index,omitempty"`
	Delta        *ClaudeStreamDelta   `json:"delta,omitempty"`
	ContentBlock *ClaudeContentBlock  `json:"content_block,omitempty"`
	Message      *ClaudeMessagesResponse `json:"message,omitempty"`
	Usage        *ClaudeUsageInfo     `json:"usage,omitempty"`
}

// ClaudeStreamDelta 流式增量
type ClaudeStreamDelta struct {
	Type       string `json:"type,omitempty"`
	Text       string `json:"text,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
}

// claudeGatewayService Claude 网关服务实现
type claudeGatewayService struct {
	db             *ent.Client
	redis          *redis.Client
	cfg            *config.Config
	logger         *zap.Logger
	accountService AccountService
	billingService BillingService
	httpClient     *http.Client
}

// NewClaudeGatewayService 创建 Claude 网关服务实例
func NewClaudeGatewayService(
	db *ent.Client,
	redis *redis.Client,
	cfg *config.Config,
	logger *zap.Logger,
	accountService AccountService,
	billingService BillingService,
) ClaudeGatewayService {
	return &claudeGatewayService{
		db:             db,
		redis:          redis,
		cfg:            cfg,
		logger:         logger,
		accountService: accountService,
		billingService: billingService,
		httpClient:     &http.Client{Timeout: time.Duration(cfg.Gateway.UpstreamTimeout) * time.Second},
	}
}

// Messages 处理 Claude Messages API
func (s *claudeGatewayService) Messages(ctx context.Context, req *ClaudeMessagesRequest) (*ClaudeMessagesResponse, error) {
	// 1. 选择账号
	account, err := s.accountService.SelectAccount(ctx, "claude", req.Model, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("选择账号失败: %w", err)
	}
	req.Account = account

	// 2. 增加并发计数
	defer s.accountService.DecrementConcurrency(ctx, account.ID)
	if err := s.accountService.IncrementConcurrency(ctx, account.ID); err != nil {
		return nil, fmt.Errorf("增加并发计数失败: %w", err)
	}

	// 3. 构建上游请求
	upstreamURL := "https://api.anthropic.com/v1/messages"
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", upstreamURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 4. 设置请求头
	s.setRequestHeaders(httpReq, account)

	// 5. 发送请求
	startTime := time.Now()
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		s.accountService.RecordError(ctx, account.ID, err)
		return nil, fmt.Errorf("请求上游失败: %w", err)
	}
	defer resp.Body.Close()

	// 6. 检查响应状态
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("上游返回错误: %d - %s", resp.StatusCode, string(body))
		s.accountService.RecordError(ctx, account.ID, err)
		return nil, err
	}

	// 7. 解析响应
	var response ClaudeMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 8. 记录用量
	latency := time.Since(startTime)
	s.recordUsage(ctx, req, &response, latency)

	s.logger.Info("Claude Messages 请求成功",
		zap.String("request_id", req.RequestID),
		zap.String("model", req.Model),
		zap.Int64("account_id", account.ID),
		zap.Duration("latency", latency),
		zap.Int("input_tokens", response.Usage.InputTokens),
		zap.Int("output_tokens", response.Usage.OutputTokens))

	return &response, nil
}

// MessagesStream 处理 Claude Messages API 流式请求
func (s *claudeGatewayService) MessagesStream(ctx context.Context, req *ClaudeMessagesRequest, callback func(event *SSEEvent) error) error {
	// 强制设置流式标志
	req.Stream = true

	// 1. 选择账号
	account, err := s.accountService.SelectAccount(ctx, "claude", req.Model, req.SessionID)
	if err != nil {
		return fmt.Errorf("选择账号失败: %w", err)
	}
	req.Account = account

	// 2. 增加并发计数
	defer s.accountService.DecrementConcurrency(ctx, account.ID)
	if err := s.accountService.IncrementConcurrency(ctx, account.ID); err != nil {
		return fmt.Errorf("增加并发计数失败: %w", err)
	}

	// 3. 构建上游请求
	upstreamURL := "https://api.anthropic.com/v1/messages"
	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", upstreamURL, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 4. 设置请求头
	s.setRequestHeaders(httpReq, account)
	httpReq.Header.Set("Accept", "text/event-stream")

	// 5. 发送请求
	startTime := time.Now()
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		s.accountService.RecordError(ctx, account.ID, err)
		return fmt.Errorf("请求上游失败: %w", err)
	}
	defer resp.Body.Close()

	// 6. 检查响应状态
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("上游返回错误: %d - %s", resp.StatusCode, string(body))
		s.accountService.RecordError(ctx, account.ID, err)
		return err
	}

	// 7. 处理 SSE 流
	var totalUsage ClaudeUsageInfo
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行
		if line == "" {
			continue
		}

		// 解析 SSE 事件
		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(line, "event: ")

			// 读取下一行数据
			if !scanner.Scan() {
				break
			}
			dataLine := scanner.Text()
			if !strings.HasPrefix(dataLine, "data: ") {
				continue
			}
			data := strings.TrimPrefix(dataLine, "data: ")

			event := &SSEEvent{
				Event: eventType,
				Data:  json.RawMessage(data),
			}

			// 解析事件以提取用量信息
			var streamEvent ClaudeStreamEvent
			if err := json.Unmarshal(event.Data, &streamEvent); err == nil {
				if streamEvent.Type == "message_delta" && streamEvent.Usage != nil {
					totalUsage.OutputTokens = streamEvent.Usage.OutputTokens
				}
				if streamEvent.Type == "message_start" && streamEvent.Message != nil {
					totalUsage.InputTokens = streamEvent.Message.Usage.InputTokens
				}
			}

			// 调用回调函数
			if err := callback(event); err != nil {
				return fmt.Errorf("回调处理失败: %w", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取流失败: %w", err)
	}

	// 8. 记录用量
	latency := time.Since(startTime)
	s.recordStreamUsage(ctx, req, &totalUsage, latency)

	s.logger.Info("Claude Messages Stream 请求成功",
		zap.String("request_id", req.RequestID),
		zap.String("model", req.Model),
		zap.Int64("account_id", account.ID),
		zap.Duration("latency", latency),
		zap.Int("input_tokens", totalUsage.InputTokens),
		zap.Int("output_tokens", totalUsage.OutputTokens))

	return nil
}

// CountTokens Token 计数
func (s *claudeGatewayService) CountTokens(ctx context.Context, req *ClaudeCountTokensRequest) (*ClaudeCountTokensResponse, error) {
	// 1. 选择账号
	account, err := s.accountService.SelectAccount(ctx, "claude", req.Model, "")
	if err != nil {
		return nil, fmt.Errorf("选择账号失败: %w", err)
	}

	// 2. 构建上游请求
	upstreamURL := "https://api.anthropic.com/v1/messages/count_tokens"
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", upstreamURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 3. 设置请求头
	s.setRequestHeaders(httpReq, account)

	// 4. 发送请求
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求上游失败: %w", err)
	}
	defer resp.Body.Close()

	// 5. 检查响应状态
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("上游返回错误: %d - %s", resp.StatusCode, string(body))
	}

	// 6. 解析响应
	var response ClaudeCountTokensResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &response, nil
}

// ListModels 获取模型列表
func (s *claudeGatewayService) ListModels(ctx context.Context) (*ClaudeModelsResponse, error) {
	// Claude 没有官方的模型列表 API，返回硬编码的模型列表
	models := []ClaudeModelInfo{
		{ID: "claude-3-opus-20240229", Object: "model", Created: 1707859200, OwnedBy: "anthropic"},
		{ID: "claude-3-sonnet-20240229", Object: "model", Created: 1707859200, OwnedBy: "anthropic"},
		{ID: "claude-3-haiku-20240307", Object: "model", Created: 1709760000, OwnedBy: "anthropic"},
		{ID: "claude-3-5-sonnet-20240620", Object: "model", Created: 1718841600, OwnedBy: "anthropic"},
		{ID: "claude-3-5-sonnet-20241022", Object: "model", Created: 1729555200, OwnedBy: "anthropic"},
		{ID: "claude-3-5-haiku-20241022", Object: "model", Created: 1729555200, OwnedBy: "anthropic"},
		{ID: "claude-3-opus-latest", Object: "model", Created: 1729555200, OwnedBy: "anthropic"},
		{ID: "claude-3-5-sonnet-latest", Object: "model", Created: 1729555200, OwnedBy: "anthropic"},
		{ID: "claude-3-5-haiku-latest", Object: "model", Created: 1729555200, OwnedBy: "anthropic"},
	}

	return &ClaudeModelsResponse{Data: models}, nil
}

// GetUsage 获取用量
func (s *claudeGatewayService) GetUsage(ctx context.Context, apiKeyID int64) (*ClaudeUsageResponse, error) {
	// 从 Redis 或数据库获取用量统计
	// 这里返回模拟数据
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	return &ClaudeUsageResponse{
		TotalTokens: 0,
		TotalCost:   0,
		PeriodStart: startOfMonth.Format("2006-01-02"),
		PeriodEnd:   now.Format("2006-01-02"),
	}, nil
}

// setRequestHeaders 设置请求头
func (s *claudeGatewayService) setRequestHeaders(req *http.Request, account *ent.Account) {
	// 设置基础请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	// 设置认证
	credentials := account.Credentials
	if credentials != nil {
		if apiKey, ok := credentials["api_key"].(string); ok {
			req.Header.Set("x-api-key", apiKey)
		}
	}
}

// recordUsage 记录用量
func (s *claudeGatewayService) recordUsage(ctx context.Context, req *ClaudeMessagesRequest, resp *ClaudeMessagesResponse, latency time.Duration) {
	// 计算费用
	cost := s.calculateCost(req.Model, resp.Usage.InputTokens, resp.Usage.OutputTokens)

	// 记录到计费服务
	if s.billingService != nil && req.UserID > 0 {
		record := &UsageRecord{
			RequestID:        req.RequestID,
			UserID:           req.UserID,
			APIKeyID:         req.APIKeyID,
			AccountID:        req.Account.ID,
			Model:            req.Model,
			Platform:         "claude",
			PromptTokens:     int32(resp.Usage.InputTokens),
			CompletionTokens: int32(resp.Usage.OutputTokens),
			TotalTokens:      int32(resp.Usage.InputTokens + resp.Usage.OutputTokens),
			Cost:             cost,
			LatencyMs:        int32(latency.Milliseconds()),
			Status:           "success",
		}
		s.billingService.RecordUsage(ctx, record)
	}
}

// recordStreamUsage 记录流式请求用量
func (s *claudeGatewayService) recordStreamUsage(ctx context.Context, req *ClaudeMessagesRequest, usage *ClaudeUsageInfo, latency time.Duration) {
	// 计算费用
	cost := s.calculateCost(req.Model, usage.InputTokens, usage.OutputTokens)

	// 记录到计费服务
	if s.billingService != nil && req.UserID > 0 {
		record := &UsageRecord{
			RequestID:        req.RequestID,
			UserID:           req.UserID,
			APIKeyID:         req.APIKeyID,
			AccountID:        req.Account.ID,
			Model:            req.Model,
			Platform:         "claude",
			PromptTokens:     int32(usage.InputTokens),
			CompletionTokens: int32(usage.OutputTokens),
			TotalTokens:      int32(usage.InputTokens + usage.OutputTokens),
			Cost:             cost,
			LatencyMs:        int32(latency.Milliseconds()),
			Status:           "success",
		}
		s.billingService.RecordUsage(ctx, record)
	}
}

// calculateCost 计算费用
func (s *claudeGatewayService) calculateCost(model string, inputTokens, outputTokens int) float64 {
	// Claude 定价（美元/百万 Token）
	pricing := map[string]struct {
		Input  float64
		Output float64
	}{
		"claude-3-opus-20240229":     {Input: 15, Output: 75},
		"claude-3-opus-latest":       {Input: 15, Output: 75},
		"claude-3-sonnet-20240229":   {Input: 3, Output: 15},
		"claude-3-haiku-20240307":    {Input: 0.25, Output: 1.25},
		"claude-3-5-sonnet-20240620": {Input: 3, Output: 15},
		"claude-3-5-sonnet-20241022": {Input: 3, Output: 15},
		"claude-3-5-sonnet-latest":   {Input: 3, Output: 15},
		"claude-3-5-haiku-20241022":  {Input: 0.8, Output: 4},
		"claude-3-5-haiku-latest":    {Input: 0.8, Output: 4},
	}

	price, ok := pricing[model]
	if !ok {
		// 默认使用 Sonnet 定价
		price = pricing["claude-3-5-sonnet-20241022"]
	}

	inputCost := float64(inputTokens) * price.Input / 1_000_000
	outputCost := float64(outputTokens) * price.Output / 1_000_000

	return inputCost + outputCost
}

// ParseClaudeError 解析 Claude 错误响应
type ClaudeErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// ParseClaudeError 解析 Claude 错误
func ParseClaudeError(body []byte) (*ClaudeErrorResponse, error) {
	var errResp ClaudeErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return nil, err
	}
	return &errResp, nil
}

// ConvertToOpenAIFormat 将 Claude 响应转换为 OpenAI 格式
func (s *claudeGatewayService) ConvertToOpenAIFormat(resp *ClaudeMessagesResponse, model string) map[string]interface{} {
	// 提取文本内容
	var content string
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return map[string]interface{}{
		"id":      resp.ID,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": resp.StopReason,
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     resp.Usage.InputTokens,
			"completion_tokens": resp.Usage.OutputTokens,
			"total_tokens":      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

// ValidateRequest 验证请求参数
func (s *claudeGatewayService) ValidateRequest(req *ClaudeMessagesRequest) error {
	if req.Model == "" {
		return fmt.Errorf("model 不能为空")
	}
	if len(req.Messages) == 0 {
		return fmt.Errorf("messages 不能为空")
	}
	if req.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens 必须大于 0")
	}
	return nil
}

// GetModelMapping 获取模型映射
func (s *claudeGatewayService) GetModelMapping(requestedModel string) string {
	// 模型别名映射
	mapping := map[string]string{
		"claude-3-opus":   "claude-3-opus-20240229",
		"claude-3-sonnet": "claude-3-sonnet-20240229",
		"claude-3-haiku":  "claude-3-haiku-20240307",
		"claude-3.5-sonnet": "claude-3-5-sonnet-20241022",
		"claude-3.5-haiku":  "claude-3-5-haiku-20241022",
	}

	if mapped, ok := mapping[requestedModel]; ok {
		return mapped
	}
	return requestedModel
}
