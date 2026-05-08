// Package handler 提供 MaaS-Router 的 HTTP 处理器
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"maas-router/internal/pkg/ctxkey"
)

// APIKeyHandler API Key 管理 Handler
// 处理 API Key 的创建、查询、更新、删除等操作
type APIKeyHandler struct {
	// APIKeyService API Key 服务
	APIKeyService APIKeyService
	// UserService 用户服务
	UserService UserService
}

// NewAPIKeyHandler 创建 API Key 管理 Handler
func NewAPIKeyHandler(
	apiKeyService APIKeyService,
	userService UserService,
) *APIKeyHandler {
	return &APIKeyHandler{
		APIKeyService: apiKeyService,
		UserService:   userService,
	}
}

// APIKeyInfo API Key 信息
type APIKeyInfo struct {
	ID            string     `json:"id"`
	Name          string     `json:"name,omitempty"`
	KeyPrefix     string     `json:"key_prefix"`
	Status        string     `json:"status"`
	DailyLimit    *float64   `json:"daily_limit,omitempty"`
	MonthlyLimit  *float64   `json:"monthly_limit,omitempty"`
	AllowedModels []string   `json:"allowed_models,omitempty"`
	IPWhitelist   []string   `json:"ip_whitelist,omitempty"`
	IPBlacklist   []string   `json:"ip_blacklist,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
	Name          string   `json:"name,omitempty"`
	DailyLimit    *float64 `json:"daily_limit,omitempty"`
	MonthlyLimit  *float64 `json:"monthly_limit,omitempty"`
	AllowedModels []string `json:"allowed_models,omitempty"`
	IPWhitelist   []string `json:"ip_whitelist,omitempty"`
	IPBlacklist   []string `json:"ip_blacklist,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKeyResponse 创建 API Key 响应
type CreateAPIKeyResponse struct {
	APIKey *APIKeyInfo `json:"api_key"`
	Key    string      `json:"key"` // 完整的 API Key，仅在创建时返回一次
}

// UpdateAPIKeyRequest 更新 API Key 请求
type UpdateAPIKeyRequest struct {
	Name          string     `json:"name,omitempty"`
	Status        string     `json:"status,omitempty"`
	DailyLimit    *float64   `json:"daily_limit,omitempty"`
	MonthlyLimit  *float64   `json:"monthly_limit,omitempty"`
	AllowedModels []string   `json:"allowed_models,omitempty"`
	IPWhitelist   []string   `json:"ip_whitelist,omitempty"`
	IPBlacklist   []string   `json:"ip_blacklist,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

// ListAPIKeysRequest 列表请求
type ListAPIKeysRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Status   string `form:"status" binding:"omitempty,oneof=active revoked expired"`
}

// ListAPIKeysResponse 列表响应
type ListAPIKeysResponse struct {
	Data       []APIKeyInfo `json:"data"`
	Total      int64        `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
	TotalPages int          `json:"total_pages"`
}

// ListKeys 处理 GET /api/v1/keys
// 获取当前用户的 API Key 列表
func (h *APIKeyHandler) ListKeys(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 解析请求参数
	var req ListAPIKeysRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// 获取 API Key 列表
	keys, total, err := h.APIKeyService.ListByUser(c.Request.Context(), userID.(string), &ListAPIKeysParams{
		Page:     req.Page,
		PageSize: req.PageSize,
		Status:   req.Status,
	})
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取 API Key 列表失败")
		return
	}

	// 转换为响应格式
	data := make([]APIKeyInfo, 0, len(keys))
	for _, key := range keys {
		data = append(data, *convertToAPIKeyInfo(key))
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, ListAPIKeysResponse{
		Data:       data,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	})
}

// CreateKey 处理 POST /api/v1/keys
// 创建新的 API Key
func (h *APIKeyHandler) CreateKey(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 解析请求
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求格式错误: "+err.Error())
		return
	}

	// 创建 API Key
	key, rawKey, err := h.APIKeyService.Create(c.Request.Context(), &CreateAPIKeyParams{
		UserID:        userID.(string),
		Name:          req.Name,
		DailyLimit:    req.DailyLimit,
		MonthlyLimit:  req.MonthlyLimit,
		AllowedModels: req.AllowedModels,
		IPWhitelist:   req.IPWhitelist,
		IPBlacklist:   req.IPBlacklist,
		ExpiresAt:     req.ExpiresAt,
	})
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "CREATE_FAILED", "创建 API Key 失败: "+err.Error())
		return
	}

	c.JSON(http.StatusCreated, CreateAPIKeyResponse{
		APIKey: convertToAPIKeyInfo(key),
		Key:    rawKey,
	})
}

// GetKey 处理 GET /api/v1/keys/:id
// 获取指定 API Key 详情
func (h *APIKeyHandler) GetKey(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 获取 API Key ID
	keyID := c.Param("id")
	if keyID == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_ID", "缺少 API Key ID")
		return
	}

	// 获取 API Key
	key, err := h.APIKeyService.GetByID(c.Request.Context(), keyID)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", "API Key 不存在")
		return
	}

	// 验证所有权
	if key.UserID != userID.(string) {
		ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "无权访问此 API Key")
		return
	}

	c.JSON(http.StatusOK, convertToAPIKeyInfo(key))
}

// UpdateKey 处理 PUT /api/v1/keys/:id
// 更新指定 API Key
func (h *APIKeyHandler) UpdateKey(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 获取 API Key ID
	keyID := c.Param("id")
	if keyID == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_ID", "缺少 API Key ID")
		return
	}

	// 解析请求
	var req UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求格式错误: "+err.Error())
		return
	}

	// 获取现有 API Key
	key, err := h.APIKeyService.GetByID(c.Request.Context(), keyID)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", "API Key 不存在")
		return
	}

	// 验证所有权
	if key.UserID != userID.(string) {
		ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "无权修改此 API Key")
		return
	}

	// 更新 API Key
	updatedKey, err := h.APIKeyService.Update(c.Request.Context(), &UpdateAPIKeyParams{
		ID:            keyID,
		Name:          req.Name,
		Status:        req.Status,
		DailyLimit:    req.DailyLimit,
		MonthlyLimit:  req.MonthlyLimit,
		AllowedModels: req.AllowedModels,
		IPWhitelist:   req.IPWhitelist,
		IPBlacklist:   req.IPBlacklist,
		ExpiresAt:     req.ExpiresAt,
	})
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "UPDATE_FAILED", "更新 API Key 失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, convertToAPIKeyInfo(updatedKey))
}

// DeleteKey 处理 DELETE /api/v1/keys/:id
// 删除（撤销）指定 API Key
func (h *APIKeyHandler) DeleteKey(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 获取 API Key ID
	keyID := c.Param("id")
	if keyID == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_ID", "缺少 API Key ID")
		return
	}

	// 获取现有 API Key
	key, err := h.APIKeyService.GetByID(c.Request.Context(), keyID)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", "API Key 不存在")
		return
	}

	// 验证所有权
	if key.UserID != userID.(string) {
		ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "无权删除此 API Key")
		return
	}

	// 删除（撤销）API Key
	err = h.APIKeyService.Delete(c.Request.Context(), keyID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "DELETE_FAILED", "删除 API Key 失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API Key 已删除",
	})
}

// RegenerateKey 重新生成 API Key
func (h *APIKeyHandler) RegenerateKey(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 获取 API Key ID
	keyID := c.Param("id")
	if keyID == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_ID", "缺少 API Key ID")
		return
	}

	// 获取现有 API Key
	key, err := h.APIKeyService.GetByID(c.Request.Context(), keyID)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", "API Key 不存在")
		return
	}

	// 验证所有权
	if key.UserID != userID.(string) {
		ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "无权操作此 API Key")
		return
	}

	// 重新生成 Key
	newKey, rawKey, err := h.APIKeyService.Regenerate(c.Request.Context(), keyID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "REGENERATE_FAILED", "重新生成 API Key 失败")
		return
	}

	c.JSON(http.StatusOK, CreateAPIKeyResponse{
		APIKey: convertToAPIKeyInfo(newKey),
		Key:    rawKey,
	})
}

// GetKeyUsage 获取 API Key 使用统计
func (h *APIKeyHandler) GetKeyUsage(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 获取 API Key ID
	keyID := c.Param("id")
	if keyID == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_ID", "缺少 API Key ID")
		return
	}

	// 获取现有 API Key
	key, err := h.APIKeyService.GetByID(c.Request.Context(), keyID)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", "API Key 不存在")
		return
	}

	// 验证所有权
	if key.UserID != userID.(string) {
		ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "无权访问此 API Key")
		return
	}

	// 获取使用统计
	usage, err := h.APIKeyService.GetUsage(c.Request.Context(), keyID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取使用统计失败")
		return
	}

	c.JSON(http.StatusOK, usage)
}

// RegisterAPIKeyHandlers 注册 API Key 路由到 HandlerGroup
func RegisterAPIKeyHandlers(h *APIKeyHandler) map[string]gin.HandlerFunc {
	return map[string]gin.HandlerFunc{
		"ListKeys":  h.ListKeys,
		"CreateKey": h.CreateKey,
		"GetKey":    h.GetKey,
		"UpdateKey": h.UpdateKey,
		"DeleteKey": h.DeleteKey,
	}
}

// ===== 辅助函数 =====

func convertToAPIKeyInfo(key *APIKey) *APIKeyInfo {
	return &APIKeyInfo{
		ID:            key.ID,
		Name:          key.Name,
		KeyPrefix:     key.KeyPrefix,
		Status:        key.Status,
		DailyLimit:    key.DailyLimit,
		MonthlyLimit:  key.MonthlyLimit,
		AllowedModels: key.AllowedModels,
		IPWhitelist:   key.IPWhitelist,
		IPBlacklist:   key.IPBlacklist,
		ExpiresAt:     key.ExpiresAt,
		LastUsedAt:    key.LastUsedAt,
		CreatedAt:     key.CreatedAt,
		UpdatedAt:     key.UpdatedAt,
	}
}

// ===== 服务接口定义 =====

// APIKey API Key 数据结构
type APIKey struct {
	ID            string
	UserID        string
	Name          string
	KeyPrefix     string
	KeyHash       string
	Status        string
	DailyLimit    *float64
	MonthlyLimit  *float64
	AllowedModels []string
	IPWhitelist   []string
	IPBlacklist   []string
	ExpiresAt     *time.Time
	LastUsedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// APIKeyService API Key 服务接口
type APIKeyService interface {
	// ListByUser 获取用户的 API Key 列表
	ListByUser(ctx interface{}, userID string, params *ListAPIKeysParams) ([]*APIKey, int64, error)
	// Create 创建 API Key
	Create(ctx interface{}, params *CreateAPIKeyParams) (*APIKey, string, error)
	// GetByID 根据 ID 获取 API Key
	GetByID(ctx interface{}, id string) (*APIKey, error)
	// Update 更新 API Key
	Update(ctx interface{}, params *UpdateAPIKeyParams) (*APIKey, error)
	// Delete 删除（撤销）API Key
	Delete(ctx interface{}, id string) error
	// Regenerate 重新生成 API Key
	Regenerate(ctx interface{}, id string) (*APIKey, string, error)
	// GetUsage 获取 API Key 使用统计
	GetUsage(ctx interface{}, id string) (*APIKeyUsage, error)
}

// ListAPIKeysParams 列表参数
type ListAPIKeysParams struct {
	Page     int
	PageSize int
	Status   string
}

// CreateAPIKeyParams 创建参数
type CreateAPIKeyParams struct {
	UserID        string
	Name          string
	DailyLimit    *float64
	MonthlyLimit  *float64
	AllowedModels []string
	IPWhitelist   []string
	IPBlacklist   []string
	ExpiresAt     *time.Time
}

// UpdateAPIKeyParams 更新参数
type UpdateAPIKeyParams struct {
	ID            string
	Name          string
	Status        string
	DailyLimit    *float64
	MonthlyLimit  *float64
	AllowedModels []string
	IPWhitelist   []string
	IPBlacklist   []string
	ExpiresAt     *time.Time
}

// APIKeyUsage API Key 使用统计
type APIKeyUsage struct {
	TotalRequests   int64   `json:"total_requests"`
	TotalTokens     int64   `json:"total_tokens"`
	TotalCost       float64 `json:"total_cost"`
	RequestsToday   int64   `json:"requests_today"`
	TokensToday     int64   `json:"tokens_today"`
	CostToday       float64 `json:"cost_today"`
	RequestsMonth   int64   `json:"requests_month"`
	TokensMonth     int64   `json:"tokens_month"`
	CostMonth       float64 `json:"cost_month"`
	UsedDailyLimit  float64 `json:"used_daily_limit"`
	UsedMonthlyLimit float64 `json:"used_monthly_limit"`
}
