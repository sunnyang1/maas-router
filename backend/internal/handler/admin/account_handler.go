// Package admin 提供管理员相关的 HTTP 处理器
package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"maas-router/internal/repository"
)

// AccountHandler 账号管理 Handler
type AccountHandler struct {
	accountRepo *repository.AccountRepository
	groupRepo   *repository.GroupRepository
}

// NewAccountHandler 创建账号管理 Handler
func NewAccountHandler(
	accountRepo *repository.AccountRepository,
	groupRepo *repository.GroupRepository,
) *AccountHandler {
	return &AccountHandler{
		accountRepo: accountRepo,
		groupRepo:   groupRepo,
	}
}

// AccountListRequest 账号列表请求
type AccountListRequest struct {
	// 页码
	Page int `form:"page" binding:"min=1"`
	// 每页数量
	PageSize int `form:"page_size" binding:"min=1,max=100"`
	// 平台筛选
	Platform string `form:"platform"`
	// 状态筛选
	Status string `form:"status"`
	// 账号类型筛选
	AccountType string `form:"account_type"`
	// 搜索关键词
	Keyword string `form:"keyword"`
	// 排序字段
	SortBy string `form:"sort_by"`
	// 排序方向
	SortOrder string `form:"sort_order"`
}

// AccountListResponse 账号列表响应
type AccountListResponse struct {
	// 账号列表
	List []*AccountInfo `json:"list"`
	// 总数
	Total int64 `json:"total"`
	// 当前页码
	Page int `json:"page"`
	// 每页数量
	PageSize int `json:"page_size"`
}

// AccountInfo 账号信息
type AccountInfo struct {
	// 账号ID
	ID int64 `json:"id"`
	// 账号名称
	Name string `json:"name"`
	// 平台类型
	Platform string `json:"platform"`
	// 账号类型
	AccountType string `json:"account_type"`
	// 状态
	Status string `json:"status"`
	// 最大并发数
	MaxConcurrency int `json:"max_concurrency"`
	// 当前并发数
	CurrentConcurrency int `json:"current_concurrency"`
	// RPM限制
	RPMLimit int `json:"rpm_limit"`
	// 总请求数
	TotalRequests int64 `json:"total_requests"`
	// 错误计数
	ErrorCount int64 `json:"error_count"`
	// 代理地址
	ProxyURL string `json:"proxy_url,omitempty"`
	// 最后使用时间
	LastUsedAt *string `json:"last_used_at,omitempty"`
	// 最后错误时间
	LastErrorAt *string `json:"last_error_at,omitempty"`
	// 创建时间
	CreatedAt string `json:"created_at"`
	// 更新时间
	UpdatedAt string `json:"updated_at"`
	// 所属分组ID列表
	GroupIDs []int64 `json:"group_ids,omitempty"`
}

// CreateAccountRequest 创建账号请求
type CreateAccountRequest struct {
	// 账号名称
	Name string `json:"name" binding:"required,max=100"`
	// 平台类型
	Platform string `json:"platform" binding:"required,oneof=claude openai gemini self_hosted"`
	// 账号类型
	AccountType string `json:"account_type" binding:"required,oneof=oauth api_key cookie"`
	// 凭证信息（加密存储）
	Credentials map[string]interface{} `json:"credentials" binding:"required"`
	// 最大并发数
	MaxConcurrency int `json:"max_concurrency" binding:"min=1,max=1000"`
	// RPM限制
	RPMLimit int `json:"rpm_limit" binding:"min=1,max=10000"`
	// 代理地址
	ProxyURL string `json:"proxy_url" binding:"max=500"`
	// TLS指纹
	TLSFingerprint string `json:"tls_fingerprint" binding:"max=100"`
	// 扩展信息
	Extra map[string]interface{} `json:"extra"`
	// 所属分组ID列表
	GroupIDs []int64 `json:"group_ids"`
}

// CreateAccountResponse 创建账号响应
type CreateAccountResponse struct {
	// 账号ID
	ID int64 `json:"id"`
	// 账号名称
	Name string `json:"name"`
	// 平台类型
	Platform string `json:"platform"`
	// 账号类型
	AccountType string `json:"account_type"`
	// 状态
	Status string `json:"status"`
	// 创建时间
	CreatedAt string `json:"created_at"`
}

// AccountDetailResponse 账号详情响应
type AccountDetailResponse struct {
	AccountInfo
	// 凭证信息（脱敏）
	Credentials map[string]interface{} `json:"credentials,omitempty"`
	// 扩展信息
	Extra map[string]interface{} `json:"extra,omitempty"`
	// TLS指纹
	TLSFingerprint string `json:"tls_fingerprint,omitempty"`
}

// UpdateAccountRequest 更新账号请求
type UpdateAccountRequest struct {
	// 账号名称
	Name string `json:"name" binding:"omitempty,max=100"`
	// 凭证信息
	Credentials map[string]interface{} `json:"credentials"`
	// 状态
	Status string `json:"status" binding:"omitempty,oneof=active disabled unschedulable"`
	// 最大并发数
	MaxConcurrency int `json:"max_concurrency" binding:"omitempty,min=1,max=1000"`
	// RPM限制
	RPMLimit int `json:"rpm_limit" binding:"omitempty,min=1,max=10000"`
	// 代理地址
	ProxyURL string `json:"proxy_url" binding:"omitempty,max=500"`
	// TLS指纹
	TLSFingerprint string `json:"tls_fingerprint" binding:"omitempty,max=100"`
	// 扩展信息
	Extra map[string]interface{} `json:"extra"`
}

// TestAccountResponse 测试账号响应
type TestAccountResponse struct {
	// 是否成功
	Success bool `json:"success"`
	// 响应时间（毫秒）
	LatencyMs int64 `json:"latency_ms"`
	// 错误信息
	Error string `json:"error,omitempty"`
	// 测试详情
	Details map[string]interface{} `json:"details,omitempty"`
}

// RefreshTokenResponse 刷新Token响应
type RefreshTokenResponse struct {
	// 是否成功
	Success bool `json:"success"`
	// 新Token过期时间
	ExpiresAt *string `json:"expires_at,omitempty"`
	// 错误信息
	Error string `json:"error,omitempty"`
}

// List 获取账号列表
// GET /api/v1/admin/accounts
func (h *AccountHandler) List(c *gin.Context) {
	var req AccountListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "请求参数无效: " + err.Error(),
			},
		})
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// 构建查询条件
	filter := repository.AccountListFilter{
		Platform:    req.Platform,
		Status:      req.Status,
		AccountType: req.AccountType,
		Keyword:     req.Keyword,
		SortBy:      req.SortBy,
		SortOrder:   req.SortOrder,
	}

	// 查询账号列表
	accounts, total, err := h.accountRepo.List(c.Request.Context(), filter, req.Page, req.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "查询账号列表失败",
			},
		})
		return
	}

	// 转换为响应格式
	list := make([]*AccountInfo, 0, len(accounts))
	for _, acc := range accounts {
		info := h.convertToAccountInfo(acc)
		list = append(list, info)
	}

	c.JSON(http.StatusOK, AccountListResponse{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

// Create 创建账号
// POST /api/v1/admin/accounts
func (h *AccountHandler) Create(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "请求参数无效: " + err.Error(),
			},
		})
		return
	}

	// 设置默认值
	maxConcurrency := 5
	if req.MaxConcurrency > 0 {
		maxConcurrency = req.MaxConcurrency
	}
	rpmLimit := 60
	if req.RPMLimit > 0 {
		rpmLimit = req.RPMLimit
	}

	// 创建账号
	account, err := h.accountRepo.Create(c.Request.Context(), &repository.CreateAccountInput{
		Name:            req.Name,
		Platform:        req.Platform,
		AccountType:     req.AccountType,
		Credentials:     req.Credentials,
		MaxConcurrency:  maxConcurrency,
		RPMLimit:        rpmLimit,
		ProxyURL:        req.ProxyURL,
		TLSFingerprint:  req.TLSFingerprint,
		Extra:           req.Extra,
		GroupIDs:        req.GroupIDs,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "创建账号失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, CreateAccountResponse{
		ID:          account.ID,
		Name:        account.Name,
		Platform:    string(account.Platform),
		AccountType: string(account.AccountType),
		Status:      string(account.Status),
		CreatedAt:   account.CreatedAt.Format("2006-01-02 15:04:05"),
	})
}

// Get 获取账号详情
// GET /api/v1/admin/accounts/:id
func (h *AccountHandler) Get(c *gin.Context) {
	// 解析账号ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的账号ID",
			},
		})
		return
	}

	// 查询账号
	account, err := h.accountRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "ACCOUNT_NOT_FOUND",
				"message": "账号不存在",
			},
		})
		return
	}

	// 获取账号所属分组
	groupIDs, _ := h.accountRepo.GetGroupIDs(c.Request.Context(), id)

	info := h.convertToAccountInfo(account)
	info.GroupIDs = groupIDs

	c.JSON(http.StatusOK, AccountDetailResponse{
		AccountInfo:    *info,
		Credentials:    account.Credentials,
		Extra:          account.Extra,
		TLSFingerprint: account.TLSFingerprint,
	})
}

// Update 更新账号
// PUT /api/v1/admin/accounts/:id
func (h *AccountHandler) Update(c *gin.Context) {
	// 解析账号ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的账号ID",
			},
		})
		return
	}

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "请求参数无效: " + err.Error(),
			},
		})
		return
	}

	// 检查账号是否存在
	_, err = h.accountRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "ACCOUNT_NOT_FOUND",
				"message": "账号不存在",
			},
		})
		return
	}

	// 更新账号
	updateInput := &repository.UpdateAccountInput{}
	if req.Name != "" {
		updateInput.Name = &req.Name
	}
	if req.Credentials != nil {
		updateInput.Credentials = req.Credentials
	}
	if req.Status != "" {
		updateInput.Status = &req.Status
	}
	if req.MaxConcurrency > 0 {
		updateInput.MaxConcurrency = &req.MaxConcurrency
	}
	if req.RPMLimit > 0 {
		updateInput.RPMLimit = &req.RPMLimit
	}
	if req.ProxyURL != "" {
		updateInput.ProxyURL = &req.ProxyURL
	}
	if req.TLSFingerprint != "" {
		updateInput.TLSFingerprint = &req.TLSFingerprint
	}
	if req.Extra != nil {
		updateInput.Extra = req.Extra
	}

	updatedAccount, err := h.accountRepo.Update(c.Request.Context(), id, updateInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "更新账号失败: " + err.Error(),
			},
		})
		return
	}

	info := h.convertToAccountInfo(updatedAccount)
	c.JSON(http.StatusOK, info)
}

// Delete 删除账号
// DELETE /api/v1/admin/accounts/:id
func (h *AccountHandler) Delete(c *gin.Context) {
	// 解析账号ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的账号ID",
			},
		})
		return
	}

	// 检查账号是否存在
	_, err = h.accountRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "ACCOUNT_NOT_FOUND",
				"message": "账号不存在",
			},
		})
		return
	}

	// 删除账号
	err = h.accountRepo.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "删除账号失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "账号已删除",
	})
}

// Test 测试账号
// POST /api/v1/admin/accounts/:id/test
func (h *AccountHandler) Test(c *gin.Context) {
	// 解析账号ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的账号ID",
			},
		})
		return
	}

	// 查询账号
	account, err := h.accountRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "ACCOUNT_NOT_FOUND",
				"message": "账号不存在",
			},
		})
		return
	}

	// TODO: 实现真实的账号测试逻辑
	// 这里需要根据平台类型调用对应的测试接口
	// 目前返回模拟结果
	c.JSON(http.StatusOK, TestAccountResponse{
		Success:   true,
		LatencyMs: 150,
		Details: map[string]interface{}{
			"platform":    account.Platform,
			"account_type": account.AccountType,
			"test_method": "connection_test",
		},
	})
}

// RefreshToken 刷新Token
// POST /api/v1/admin/accounts/:id/refresh
func (h *AccountHandler) RefreshToken(c *gin.Context) {
	// 解析账号ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的账号ID",
			},
		})
		return
	}

	// 查询账号
	account, err := h.accountRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "ACCOUNT_NOT_FOUND",
				"message": "账号不存在",
			},
		})
		return
	}

	// TODO: 实现真实的Token刷新逻辑
	// 这里需要根据平台类型调用对应的刷新接口
	// 目前返回模拟结果
	expiresAt := "2099-12-31 23:59:59"
	c.JSON(http.StatusOK, RefreshTokenResponse{
		Success:   true,
		ExpiresAt: &expiresAt,
	})
}

// convertToAccountInfo 将账号实体转换为响应格式
func (h *AccountHandler) convertToAccountInfo(acc *repository.Account) *AccountInfo {
	var lastUsedAt, lastErrorAt *string
	if acc.LastUsedAt != nil {
		t := acc.LastUsedAt.Format("2006-01-02 15:04:05")
		lastUsedAt = &t
	}
	if acc.LastErrorAt != nil {
		t := acc.LastErrorAt.Format("2006-01-02 15:04:05")
		lastErrorAt = &t
	}

	return &AccountInfo{
		ID:                 acc.ID,
		Name:               acc.Name,
		Platform:           string(acc.Platform),
		AccountType:        string(acc.AccountType),
		Status:             string(acc.Status),
		MaxConcurrency:     acc.MaxConcurrency,
		CurrentConcurrency: acc.CurrentConcurrency,
		RPMLimit:           acc.RPMLimit,
		TotalRequests:      acc.TotalRequests,
		ErrorCount:         acc.ErrorCount,
		ProxyURL:           acc.ProxyURL,
		LastUsedAt:         lastUsedAt,
		LastErrorAt:        lastErrorAt,
		CreatedAt:          acc.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:          acc.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
