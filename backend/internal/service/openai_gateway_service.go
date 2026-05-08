// Package service 业务服务层
// 提供 OpenAI 网关服务
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

// OpenAIGatewayService OpenAI 网关服务接口
// 处理 OpenAI API 相关请求
type OpenAIGatewayService interface {
	// ChatCompletions 处理 Chat Completions API
	ChatCompletions(ctx context.Context, req *OpenAIChatCompletionRequest) (*OpenAIChatCompletionResponse, error)

	// ChatCompletionsStream 处理 Chat Completions API 流式请求
	ChatCompletionsStream(ctx context.Context, req *OpenAIChatCompletionRequest, callback func(event *SSEEvent) error) error

	// Responses 处理 OpenAI Responses API
	Responses(ctx context.Context, req *OpenAIResponsesRequest) (*OpenAIResponsesResponse, error)

	// ImagesGenerations 图片生成
	ImagesGenerations(ctx context.Context, req *OpenAIImagesRequest) (*OpenAIImagesResponse, error)

	// Embeddings 向量嵌入
	Embeddings(ctx context.Context, req *OpenAIEmbeddingsRequest) (*OpenAIEmbeddingsResponse, error)

	// ListModels 获取模型列表
	ListModels(ctx context.Context) (*OpenAIModelsResponse, error)

	// GetModel 获取模型信息
	GetModel(ctx context.Context, model string) (*OpenAIModelInfo, error)
}

// OpenAIChatCompletionRequest Chat Completions API 请求
type OpenAIChatCompletionRequest struct {
	Model            string                `json:"model"`
	Messages         []OpenAIChatMessage   `json:"messages"`
	Temperature      *float64              `json:"temperature,omitempty"`
	TopP             *float64              `json:"top_p,omitempty"`
	N                *int                  `json:"n,omitempty"`
	Stream           bool                  `json:"stream,omitempty"`
	StreamOptions    *OpenAIStreamOptions  `json:"stream_options,omitempty"`
	Stop             interface{}           `json:"stop,omitempty"`
	MaxTokens        *int                  `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int               `json:"max_completion_tokens,omitempty"`
	PresencePenalty  *float64              `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64              `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]float64    `json:"logit_bias,omitempty"`
	User             string                `json:"user,omitempty"`
	Tools            []OpenAITool          `json:"tools,omitempty"`
	ToolChoice       interface{}           `json:"tool_choice,omitempty"`
	ResponseFormat   *OpenAIResponseFormat `json:"response_format,omitempty"`
	Seed             *int                  `json:"seed,omitempty"`

	// 内部字段
	Account   *ent.Account `json:"-"`
	SessionID string       `json:"-"`
	RequestID string       `json:"-"`
	APIKeyID  int64        `json:"-"`
	UserID    int64        `json:"-"`
}

// OpenAIChatMessage Chat 消息
type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content interface{} `json:"content"` // 可以是 string 或 []OpenAIContentPart
	Name    string `json:"name,omitempty"`
}

// OpenAIContentPart 内容部分
type OpenAIContentPart struct {
	Type     string           `json:"type"` // text, image_url
	Text     string           `json:"text,omitempty"`
	ImageURL *OpenAIImageURL  `json:"image_url,omitempty"`
}

// OpenAIImageURL 图片 URL
type OpenAIImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // auto, low, high
}

// OpenAIStreamOptions 流式选项
type OpenAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// OpenAITool 工具定义
type OpenAITool struct {
	Type     string          `json:"type"` // function
	Function *OpenAIFunction `json:"function,omitempty"`
}

// OpenAIFunction 函数定义
type OpenAIFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// OpenAIResponseFormat 响应格式
type OpenAIResponseFormat struct {
	Type       string `json:"type"` // text, json_object, json_schema
	JSONSchema *OpenAIJSONSchema `json:"json_schema,omitempty"`
}

// OpenAIJSONSchema JSON Schema
type OpenAIJSONSchema struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict bool            `json:"strict,omitempty"`
}

// OpenAIChatCompletionResponse Chat Completions API 响应
type OpenAIChatCompletionResponse struct {
	ID                string                         `json:"id"`
	Object            string                         `json:"object"`
	Created           int64                          `json:"created"`
	Model             string                         `json:"model"`
	Choices           []OpenAIChatCompletionChoice   `json:"choices"`
	Usage             *OpenAIUsageInfo               `json:"usage,omitempty"`
	SystemFingerprint string                         `json:"system_fingerprint,omitempty"`
}

// OpenAIChatCompletionChoice 选择项
type OpenAIChatCompletionChoice struct {
	Index        int                   `json:"index"`
	Message      *OpenAIChatMessage    `json:"message,omitempty"`
	Delta        *OpenAIChatMessage    `json:"delta,omitempty"`
	FinishReason string                `json:"finish_reason"`
	Logprobs     *OpenAILogprobs       `json:"logprobs,omitempty"`
}

// OpenAILogprobs 对数概率
type OpenAILogprobs struct {
	Content []OpenAILogprobContent `json:"content,omitempty"`
}

// OpenAILogprobContent 对数概率内容
type OpenAILogprobContent struct {
	Token       string  `json:"token"`
	Logprob     float64 `json:"logprob"`
	Bytes       []byte  `json:"bytes,omitempty"`
	TopLogprobs []OpenAITopLogprob `json:"top_logprobs,omitempty"`
}

// OpenAITopLogprob 顶部对数概率
type OpenAITopLogprob struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []byte  `json:"bytes,omitempty"`
}

// OpenAIUsageInfo OpenAI 用量信息
type OpenAIUsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIResponsesRequest Responses API 请求
type OpenAIResponsesRequest struct {
	Model          string                 `json:"model"`
	Input          interface{}            `json:"input"` // string 或 []OpenAIChatMessage
	Instructions   string                 `json:"instructions,omitempty"`
	MaxOutputTokens int                   `json:"max_output_tokens,omitempty"`
	Temperature    *float64               `json:"temperature,omitempty"`
	TopP           *float64               `json:"top_p,omitempty"`
	Stream         bool                   `json:"stream,omitempty"`
	Tools          []OpenAITool           `json:"tools,omitempty"`
	ToolChoice     interface{}            `json:"tool_choice,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`

	// 内部字段
	Account   *ent.Account `json:"-"`
	SessionID string       `json:"-"`
	RequestID string       `json:"-"`
	APIKeyID  int64        `json:"-"`
	UserID    int64        `json:"-"`
}

// OpenAIResponsesResponse Responses API 响应
type OpenAIResponsesResponse struct {
	ID           string                    `json:"id"`
	Object       string                    `json:"object"`
	CreatedAt    int64                     `json:"created_at"`
	Status       string                    `json:"status"`
	Model        string                    `json:"model"`
	Output       []OpenAIResponseOutput    `json:"output"`
	Usage        *OpenAIUsageInfo          `json:"usage,omitempty"`
}

// OpenAIResponseOutput 响应输出
type OpenAIResponseOutput struct {
	Type    string          `json:"type"` // message, function_call
	ID      string          `json:"id,omitempty"`
	Role    string          `json:"role,omitempty"`
	Content []OpenAIContentPart `json:"content,omitempty"`
	Name    string          `json:"name,omitempty"`
	Arguments string        `json:"arguments,omitempty"`
}

// OpenAIImagesRequest 图片生成请求
type OpenAIImagesRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              *int   `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	Style          string `json:"style,omitempty"`
	User           string `json:"user,omitempty"`

	// 内部字段
	Account   *ent.Account `json:"-"`
	SessionID string       `json:"-"`
	RequestID string       `json:"-"`
	APIKeyID  int64        `json:"-"`
	UserID    int64        `json:"-"`
}

// OpenAIImagesResponse 图片生成响应
type OpenAIImagesResponse struct {
	Created int64               `json:"created"`
	Data    []OpenAIImageData   `json:"data"`
}

// OpenAIImageData 图片数据
type OpenAIImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// OpenAIEmbeddingsRequest 向量嵌入请求
type OpenAIEmbeddingsRequest struct {
	Model          string   `json:"model"`
	Input          interface{} `json:"input"` // string 或 []string
	EncodingFormat string   `json:"encoding_format,omitempty"`
	Dimensions     *int     `json:"dimensions,omitempty"`
	User           string   `json:"user,omitempty"`

	// 内部字段
	Account   *ent.Account `json:"-"`
	SessionID string       `json:"-"`
	RequestID string       `json:"-"`
	APIKeyID  int64        `json:"-"`
	UserID    int64        `json:"-"`
}

// OpenAIEmbeddingsResponse 向量嵌入响应
type OpenAIEmbeddingsResponse struct {
	Object string                  `json:"object"`
	Data   []OpenAIEmbeddingData   `json:"data"`
	Model  string                  `json:"model"`
	Usage  *OpenAIUsageInfo        `json:"usage"`
}

// OpenAIEmbeddingData 嵌入数据
type OpenAIEmbeddingData struct {
	Object    string      `json:"object"`
	Index     int         `json:"index"`
	Embedding []float64   `json:"embedding"`
}

// OpenAIModelsResponse 模型列表响应
type OpenAIModelsResponse struct {
	Object string             `json:"object"`
	Data   []OpenAIModelInfo  `json:"data"`
}

// OpenAIModelInfo 模型信息
type OpenAIModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// openaiGatewayService OpenAI 网关服务实现
type openaiGatewayService struct {
	db             *ent.Client
	redis          *redis.Client
	cfg            *config.Config
	logger         *zap.Logger
	accountService AccountService
	billingService BillingService
	httpClient     *http.Client
}

// NewOpenAIGatewayService 创建 OpenAI 网关服务实例
func NewOpenAIGatewayService(
	db *ent.Client,
	redis *redis.Client,
	cfg *config.Config,
	logger *zap.Logger,
	accountService AccountService,
	billingService BillingService,
) OpenAIGatewayService {
	return &openaiGatewayService{
		db:             db,
		redis:          redis,
		cfg:            cfg,
		logger:         logger,
		accountService: accountService,
		billingService: billingService,
		httpClient:     &http.Client{Timeout: time.Duration(cfg.Gateway.UpstreamTimeout) * time.Second},
	}
}

// ChatCompletions 处理 Chat Completions API
func (s *openaiGatewayService) ChatCompletions(ctx context.Context, req *OpenAIChatCompletionRequest) (*OpenAIChatCompletionResponse, error) {
	// 1. 选择账号
	account, err := s.accountService.SelectAccount(ctx, "openai", req.Model, req.SessionID)
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
	upstreamURL := "https://api.openai.com/v1/chat/completions"
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
	var response OpenAIChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 8. 记录用量
	latency := time.Since(startTime)
	s.recordUsage(ctx, req, &response, latency)

	s.logger.Info("OpenAI Chat Completions 请求成功",
		zap.String("request_id", req.RequestID),
		zap.String("model", req.Model),
		zap.Int64("account_id", account.ID),
		zap.Duration("latency", latency))

	return &response, nil
}

// ChatCompletionsStream 处理 Chat Completions API 流式请求
func (s *openaiGatewayService) ChatCompletionsStream(ctx context.Context, req *OpenAIChatCompletionRequest, callback func(event *SSEEvent) error) error {
	// 强制设置流式标志
	req.Stream = true

	// 1. 选择账号
	account, err := s.accountService.SelectAccount(ctx, "openai", req.Model, req.SessionID)
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
	upstreamURL := "https://api.openai.com/v1/chat/completions"
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
	var totalUsage OpenAIUsageInfo
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行
		if line == "" {
			continue
		}

		// 解析 SSE 数据
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// 检查是否是结束标记
			if data == "[DONE]" {
				event := &SSEEvent{
					Event: "done",
					Data:  json.RawMessage(`"[DONE]"`),
				}
				callback(event)
				break
			}

			event := &SSEEvent{
				Event: "message",
				Data:  json.RawMessage(data),
			}

			// 解析事件以提取用量信息
			var streamResp OpenAIChatCompletionResponse
			if err := json.Unmarshal(event.Data, &streamResp); err == nil {
				if streamResp.Usage != nil {
					totalUsage = *streamResp.Usage
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

	s.logger.Info("OpenAI Chat Completions Stream 请求成功",
		zap.String("request_id", req.RequestID),
		zap.String("model", req.Model),
		zap.Int64("account_id", account.ID),
		zap.Duration("latency", latency))

	return nil
}

// Responses 处理 OpenAI Responses API
func (s *openaiGatewayService) Responses(ctx context.Context, req *OpenAIResponsesRequest) (*OpenAIResponsesResponse, error) {
	// 1. 选择账号
	account, err := s.accountService.SelectAccount(ctx, "openai", req.Model, req.SessionID)
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
	upstreamURL := "https://api.openai.com/v1/responses"
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
		return nil, fmt.Errorf("上游返回错误: %d - %s", resp.StatusCode, string(body))
	}

	// 7. 解析响应
	var response OpenAIResponsesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 8. 记录用量
	latency := time.Since(startTime)
	s.logger.Info("OpenAI Responses 请求成功",
		zap.String("request_id", req.RequestID),
		zap.String("model", req.Model),
		zap.Int64("account_id", account.ID),
		zap.Duration("latency", latency))

	return &response, nil
}

// ImagesGenerations 图片生成
func (s *openaiGatewayService) ImagesGenerations(ctx context.Context, req *OpenAIImagesRequest) (*OpenAIImagesResponse, error) {
	// 1. 选择账号
	account, err := s.accountService.SelectAccount(ctx, "openai", req.Model, req.SessionID)
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
	upstreamURL := "https://api.openai.com/v1/images/generations"
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
		return nil, fmt.Errorf("上游返回错误: %d - %s", resp.StatusCode, string(body))
	}

	// 7. 解析响应
	var response OpenAIImagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 8. 记录用量
	latency := time.Since(startTime)
	s.logger.Info("OpenAI Images Generations 请求成功",
		zap.String("request_id", req.RequestID),
		zap.String("model", req.Model),
		zap.Int64("account_id", account.ID),
		zap.Duration("latency", latency))

	return &response, nil
}

// Embeddings 向量嵌入
func (s *openaiGatewayService) Embeddings(ctx context.Context, req *OpenAIEmbeddingsRequest) (*OpenAIEmbeddingsResponse, error) {
	// 1. 选择账号
	account, err := s.accountService.SelectAccount(ctx, "openai", req.Model, req.SessionID)
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
	upstreamURL := "https://api.openai.com/v1/embeddings"
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
		return nil, fmt.Errorf("上游返回错误: %d - %s", resp.StatusCode, string(body))
	}

	// 7. 解析响应
	var response OpenAIEmbeddingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 8. 记录用量
	latency := time.Since(startTime)
	s.logger.Info("OpenAI Embeddings 请求成功",
		zap.String("request_id", req.RequestID),
		zap.String("model", req.Model),
		zap.Int64("account_id", account.ID),
		zap.Duration("latency", latency))

	return &response, nil
}

// ListModels 获取模型列表
func (s *openaiGatewayService) ListModels(ctx context.Context) (*OpenAIModelsResponse, error) {
	// 选择账号
	account, err := s.accountService.SelectAccount(ctx, "openai", "", "")
	if err != nil {
		return nil, fmt.Errorf("选择账号失败: %w", err)
	}

	// 构建请求
	upstreamURL := "https://api.openai.com/v1/models"
	httpReq, err := http.NewRequestWithContext(ctx, "GET", upstreamURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	s.setRequestHeaders(httpReq, account)

	// 发送请求
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求上游失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("上游返回错误: %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response OpenAIModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &response, nil
}

// GetModel 获取模型信息
func (s *openaiGatewayService) GetModel(ctx context.Context, model string) (*OpenAIModelInfo, error) {
	// 选择账号
	account, err := s.accountService.SelectAccount(ctx, "openai", model, "")
	if err != nil {
		return nil, fmt.Errorf("选择账号失败: %w", err)
	}

	// 构建请求
	upstreamURL := fmt.Sprintf("https://api.openai.com/v1/models/%s", model)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", upstreamURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	s.setRequestHeaders(httpReq, account)

	// 发送请求
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求上游失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("上游返回错误: %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response OpenAIModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &response, nil
}

// setRequestHeaders 设置请求头
func (s *openaiGatewayService) setRequestHeaders(req *http.Request, account *ent.Account) {
	// 设置基础请求头
	req.Header.Set("Content-Type", "application/json")

	// 设置认证
	credentials := account.Credentials
	if credentials != nil {
		if apiKey, ok := credentials["api_key"].(string); ok {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
	}
}

// recordUsage 记录用量
func (s *openaiGatewayService) recordUsage(ctx context.Context, req *OpenAIChatCompletionRequest, resp *OpenAIChatCompletionResponse, latency time.Duration) {
	if resp.Usage == nil {
		return
	}

	// 计算费用
	cost := s.calculateChatCost(req.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	// 记录到计费服务
	if s.billingService != nil && req.UserID > 0 {
		record := &UsageRecord{
			RequestID:        req.RequestID,
			UserID:           req.UserID,
			APIKeyID:         req.APIKeyID,
			AccountID:        req.Account.ID,
			Model:            req.Model,
			Platform:         "openai",
			PromptTokens:     int32(resp.Usage.PromptTokens),
			CompletionTokens: int32(resp.Usage.CompletionTokens),
			TotalTokens:      int32(resp.Usage.TotalTokens),
			Cost:             cost,
			LatencyMs:        int32(latency.Milliseconds()),
			Status:           "success",
		}
		s.billingService.RecordUsage(ctx, record)
	}
}

// recordStreamUsage 记录流式请求用量
func (s *openaiGatewayService) recordStreamUsage(ctx context.Context, req *OpenAIChatCompletionRequest, usage *OpenAIUsageInfo, latency time.Duration) {
	if usage.TotalTokens == 0 {
		return
	}

	// 计算费用
	cost := s.calculateChatCost(req.Model, usage.PromptTokens, usage.CompletionTokens)

	// 记录到计费服务
	if s.billingService != nil && req.UserID > 0 {
		record := &UsageRecord{
			RequestID:        req.RequestID,
			UserID:           req.UserID,
			APIKeyID:         req.APIKeyID,
			AccountID:        req.Account.ID,
			Model:            req.Model,
			Platform:         "openai",
			PromptTokens:     int32(usage.PromptTokens),
			CompletionTokens: int32(usage.CompletionTokens),
			TotalTokens:      int32(usage.TotalTokens),
			Cost:             cost,
			LatencyMs:        int32(latency.Milliseconds()),
			Status:           "success",
		}
		s.billingService.RecordUsage(ctx, record)
	}
}

// calculateChatCost 计算 Chat 费用
func (s *openaiGatewayService) calculateChatCost(model string, inputTokens, outputTokens int) float64 {
	// OpenAI 定价（美元/百万 Token）
	pricing := map[string]struct {
		Input  float64
		Output float64
	}{
		"gpt-4o":                    {Input: 2.5, Output: 10},
		"gpt-4o-2024-11-20":         {Input: 2.5, Output: 10},
		"gpt-4o-2024-08-06":         {Input: 2.5, Output: 10},
		"gpt-4o-2024-05-13":         {Input: 5, Output: 15},
		"gpt-4o-mini":               {Input: 0.15, Output: 0.6},
		"gpt-4o-mini-2024-07-18":    {Input: 0.15, Output: 0.6},
		"gpt-4-turbo":               {Input: 10, Output: 30},
		"gpt-4-turbo-2024-04-09":    {Input: 10, Output: 30},
		"gpt-4":                     {Input: 30, Output: 60},
		"gpt-4-32k":                 {Input: 60, Output: 120},
		"gpt-3.5-turbo":             {Input: 0.5, Output: 1.5},
		"gpt-3.5-turbo-0125":        {Input: 0.5, Output: 1.5},
		"o1-preview":                {Input: 15, Output: 60},
		"o1-mini":                   {Input: 3, Output: 12},
	}

	price, ok := pricing[model]
	if !ok {
		// 默认使用 GPT-4o-mini 定价
		price = pricing["gpt-4o-mini"]
	}

	inputCost := float64(inputTokens) * price.Input / 1_000_000
	outputCost := float64(outputTokens) * price.Output / 1_000_000

	return inputCost + outputCost
}

// ValidateRequest 验证请求参数
func (s *openaiGatewayService) ValidateRequest(req *OpenAIChatCompletionRequest) error {
	if req.Model == "" {
		return fmt.Errorf("model 不能为空")
	}
	if len(req.Messages) == 0 {
		return fmt.Errorf("messages 不能为空")
	}
	return nil
}

// GetModelMapping 获取模型映射
func (s *openaiGatewayService) GetModelMapping(requestedModel string) string {
	// 模型别名映射
	mapping := map[string]string{
		"gpt-4":          "gpt-4-turbo",
		"gpt-4-turbo":    "gpt-4-turbo-2024-04-09",
		"gpt-3.5-turbo":  "gpt-3.5-turbo-0125",
	}

	if mapped, ok := mapping[requestedModel]; ok {
		return mapped
	}
	return requestedModel
}

// OpenAIErrorResponse OpenAI 错误响应
type OpenAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code,omitempty"`
	} `json:"error"`
}

// ParseOpenAIError 解析 OpenAI 错误
func ParseOpenAIError(body []byte) (*OpenAIErrorResponse, error) {
	var errResp OpenAIErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return nil, err
	}
	return &errResp, nil
}

// ConvertToClaudeFormat 将 OpenAI 请求转换为 Claude 格式
func (s *openaiGatewayService) ConvertToClaudeFormat(req *OpenAIChatCompletionRequest) *ClaudeMessagesRequest {
	// 转换消息
	claudeMessages := make([]ClaudeMessage, 0, len(req.Messages))
	var system string

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			// 系统消息单独处理
			if content, ok := msg.Content.(string); ok {
				system = content
			}
			continue
		}

		claudeMsg := ClaudeMessage{
			Role: msg.Role,
		}

		// 处理内容
		switch content := msg.Content.(type) {
		case string:
			claudeMsg.Content = content
		case []interface{}:
			// 多模态内容
			blocks := make([]ClaudeContentBlock, 0)
			for _, part := range content {
				if p, ok := part.(map[string]interface{}); ok {
					block := ClaudeContentBlock{}
					if t, ok := p["type"].(string); ok {
						block.Type = t
					}
					if t, ok := p["text"].(string); ok {
						block.Text = t
					}
					blocks = append(blocks, block)
				}
			}
			claudeMsg.Content = blocks
		}

		claudeMessages = append(claudeMessages, claudeMsg)
	}

	// 构建请求
	claudeReq := &ClaudeMessagesRequest{
		Model:    s.mapModelToClaude(req.Model),
		Messages: claudeMessages,
		System:   system,
	}

	// 设置参数
	if req.MaxTokens != nil {
		claudeReq.MaxTokens = *req.MaxTokens
	} else if req.MaxCompletionTokens != nil {
		claudeReq.MaxTokens = *req.MaxCompletionTokens
	} else {
		claudeReq.MaxTokens = 4096 // 默认值
	}

	if req.Temperature != nil {
		claudeReq.Temperature = *req.Temperature
	}
	if req.TopP != nil {
		claudeReq.TopP = *req.TopP
	}
	claudeReq.Stream = req.Stream

	return claudeReq
}

// mapModelToClaude 将 OpenAI 模型映射到 Claude 模型
func (s *openaiGatewayService) mapModelToClaude(openaiModel string) string {
	mapping := map[string]string{
		"gpt-4":          "claude-3-opus-20240229",
		"gpt-4-turbo":    "claude-3-5-sonnet-20241022",
		"gpt-4o":         "claude-3-5-sonnet-20241022",
		"gpt-4o-mini":    "claude-3-5-haiku-20241022",
		"gpt-3.5-turbo":  "claude-3-haiku-20240307",
	}

	if mapped, ok := mapping[openaiModel]; ok {
		return mapped
	}
	return "claude-3-5-sonnet-20241022" // 默认
}
