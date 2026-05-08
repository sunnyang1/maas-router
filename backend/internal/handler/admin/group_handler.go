// Package admin 提供管理员相关的 HTTP 处理器
package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"maas-router/internal/repository"
)

// GroupHandler 分组管理 Handler
type GroupHandler struct {
	groupRepo   *repository.GroupRepository
	accountRepo *repository.AccountRepository
}

// NewGroupHandler 创建分组管理 Handler
func NewGroupHandler(
	groupRepo *repository.GroupRepository,
	accountRepo *repository.AccountRepository,
) *GroupHandler {
	return &GroupHandler{
		groupRepo:   groupRepo,
		accountRepo: accountRepo,
	}
}

// GroupListRequest 分组列表请求
type GroupListRequest struct {
	// 页码
	Page int `form:"page" binding:"min=1"`
	// 每页数量
	PageSize int `form:"page_size" binding:"min=1,max=100"`
	// 平台筛选
	Platform string `form:"platform"`
	// 状态筛选
	Status string `form:"status"`
	// 搜索关键词
	Keyword string `form:"keyword"`
	// 排序字段
	SortBy string `form:"sort_by"`
	// 排序方向
	SortOrder string `form:"sort_order"`
}

// GroupListResponse 分组列表响应
type GroupListResponse struct {
	// 分组列表
	List []*GroupInfo `json:"list"`
	// 总数
	Total int64 `json:"total"`
	// 当前页码
	Page int `json:"page"`
	// 每页数量
	PageSize int `json:"page_size"`
}

// GroupInfo 分组信息
type GroupInfo struct {
	// 分组ID
	ID int64 `json:"id"`
	// 分组名称
	Name string `json:"name"`
	// 分组描述
	Description string `json:"description,omitempty"`
	// 平台类型
	Platform string `json:"platform"`
	// 计费模式
	BillingMode string `json:"billing_mode"`
	// 费率倍率
	RateMultiplier float64 `json:"rate_multiplier"`
	// RPM覆盖值
	RPMOverride *int `json:"rpm_override,omitempty"`
	// 模型映射
	ModelMapping map[string]string `json:"model_mapping,omitempty"`
	// 优先级
	Priority int `json:"priority"`
	// 权重
	Weight int `json:"weight"`
	// 状态
	Status string `json:"status"`
	// 账号数量
	AccountCount int64 `json:"account_count"`
	// 创建时间
	CreatedAt string `json:"created_at"`
	// 更新时间
	UpdatedAt string `json:"updated_at"`
}

// CreateGroupRequest 创建分组请求
type CreateGroupRequest struct {
	// 分组名称
	Name string `json:"name" binding:"required,max=100"`
	// 分组描述
	Description string `json:"description" binding:"max=500"`
	// 平台类型
	Platform string `json:"platform" binding:"required,oneof=claude openai gemini self_hosted"`
	// 计费模式
	BillingMode string `json:"billing_mode" binding:"omitempty,oneof=balance subscription"`
	// 费率倍率
	RateMultiplier float64 `json:"rate_multiplier" binding:"min=0.001,max=100"`
	// RPM覆盖值
	RPMOverride *int `json:"rpm_override" binding:"omitempty,min=1,max=10000"`
	// 模型映射
	ModelMapping map[string]string `json:"model_mapping"`
	// 优先级
	Priority int `json:"priority" binding:"min=-1000,max=1000"`
	// 权重
	Weight int `json:"weight" binding:"min=1,max=1000"`
	// 状态
	Status string `json:"status" binding:"omitempty,oneof=active inactive"`
	// 关联账号ID列表
	AccountIDs []int64 `json:"account_ids"`
}

// CreateGroupResponse 创建分组响应
type CreateGroupResponse struct {
	// 分组ID
	ID int64 `json:"id"`
	// 分组名称
	Name string `json:"name"`
	// 平台类型
	Platform string `json:"platform"`
	// 状态
	Status string `json:"status"`
	// 创建时间
	CreatedAt string `json:"created_at"`
}

// GroupDetailResponse 分组详情响应
type GroupDetailResponse struct {
	GroupInfo
	// 关联账号列表
	Accounts []*GroupAccountInfo `json:"accounts,omitempty"`
}

// GroupAccountInfo 分组中的账号信息
type GroupAccountInfo struct {
	// 账号ID
	ID int64 `json:"id"`
	// 账号名称
	Name string `json:"name"`
	// 平台类型
	Platform string `json:"platform"`
	// 状态
	Status string `json:"status"`
	// 最大并发数
	MaxConcurrency int `json:"max_concurrency"`
	// 当前并发数
	CurrentConcurrency int `json:"current_concurrency"`
}

// UpdateGroupRequest 更新分组请求
type UpdateGroupRequest struct {
	// 分组名称
	Name string `json:"name" binding:"omitempty,max=100"`
	// 分组描述
	Description string `json:"description" binding:"omitempty,max=500"`
	// 计费模式
	BillingMode string `json:"billing_mode" binding:"omitempty,oneof=balance subscription"`
	// 费率倍率
	RateMultiplier float64 `json:"rate_multiplier" binding:"omitempty,min=0.001,max=100"`
	// RPM覆盖值
	RPMOverride *int `json:"rpm_override" binding:"omitempty,min=1,max=10000"`
	// 模型映射
	ModelMapping map[string]string `json:"model_mapping"`
	// 优先级
	Priority int `json:"priority" binding:"omitempty,min=-1000,max=1000"`
	// 权重
	Weight int `json:"weight" binding:"omitempty,min=1,max=1000"`
	// 状态
	Status string `json:"status" binding:"omitempty,oneof=active inactive"`
}

// AddAccountsRequest 添加账号到分组请求
type AddAccountsRequest struct {
	// 账号ID列表
	AccountIDs []int64 `json:"account_ids" binding:"required,min=1"`
}

// RemoveAccountsRequest 从分组移除账号请求
type RemoveAccountsRequest struct {
	// 账号ID列表
	AccountIDs []int64 `json:"account_ids" binding:"required,min=1"`
}

// List 获取分组列表
// GET /api/v1/admin/groups
func (h *GroupHandler) List(c *gin.Context) {
	var req GroupListRequest
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
	filter := repository.GroupListFilter{
		Platform:  req.Platform,
		Status:    req.Status,
		Keyword:   req.Keyword,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	// 查询分组列表
	groups, total, err := h.groupRepo.List(c.Request.Context(), filter, req.Page, req.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "查询分组列表失败",
			},
		})
		return
	}

	// 转换为响应格式
	list := make([]*GroupInfo, 0, len(groups))
	for _, g := range groups {
		// 获取分组账号数量
		accountCount, _ := h.groupRepo.CountAccounts(c.Request.Context(), g.ID)

		list = append(list, &GroupInfo{
			ID:             g.ID,
			Name:           g.Name,
			Description:    g.Description,
			Platform:       string(g.Platform),
			BillingMode:    string(g.BillingMode),
			RateMultiplier: g.RateMultiplier,
			RPMOverride:    g.RPMOverride,
			ModelMapping:   g.ModelMapping,
			Priority:       g.Priority,
			Weight:         g.Weight,
			Status:         string(g.Status),
			AccountCount:   accountCount,
			CreatedAt:      g.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:      g.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, GroupListResponse{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

// Create 创建分组
// POST /api/v1/admin/groups
func (h *GroupHandler) Create(c *gin.Context) {
	var req CreateGroupRequest
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
	billingMode := "balance"
	if req.BillingMode != "" {
		billingMode = req.BillingMode
	}
	rateMultiplier := 1.0
	if req.RateMultiplier > 0 {
		rateMultiplier = req.RateMultiplier
	}
	priority := 0
	if req.Priority != 0 {
		priority = req.Priority
	}
	weight := 100
	if req.Weight > 0 {
		weight = req.Weight
	}
	status := "active"
	if req.Status != "" {
		status = req.Status
	}

	// 创建分组
	group, err := h.groupRepo.Create(c.Request.Context(), &repository.CreateGroupInput{
		Name:           req.Name,
		Description:    req.Description,
		Platform:       req.Platform,
		BillingMode:    billingMode,
		RateMultiplier: rateMultiplier,
		RPMOverride:    req.RPMOverride,
		ModelMapping:   req.ModelMapping,
		Priority:       priority,
		Weight:         weight,
		Status:         status,
		AccountIDs:     req.AccountIDs,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "创建分组失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, CreateGroupResponse{
		ID:        group.ID,
		Name:      group.Name,
		Platform:  string(group.Platform),
		Status:    string(group.Status),
		CreatedAt: group.CreatedAt.Format("2006-01-02 15:04:05"),
	})
}

// Get 获取分组详情
// GET /api/v1/admin/groups/:id
func (h *GroupHandler) Get(c *gin.Context) {
	// 解析分组ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的分组ID",
			},
		})
		return
	}

	// 查询分组
	group, err := h.groupRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "GROUP_NOT_FOUND",
				"message": "分组不存在",
			},
		})
		return
	}

	// 获取分组账号数量
	accountCount, _ := h.groupRepo.CountAccounts(c.Request.Context(), id)

	// 获取分组关联的账号
	accounts, _ := h.groupRepo.GetAccounts(c.Request.Context(), id)

	// 转换账号信息
	accountInfos := make([]*GroupAccountInfo, 0, len(accounts))
	for _, acc := range accounts {
		accountInfos = append(accountInfos, &GroupAccountInfo{
			ID:                 acc.ID,
			Name:               acc.Name,
			Platform:           string(acc.Platform),
			Status:             string(acc.Status),
			MaxConcurrency:     acc.MaxConcurrency,
			CurrentConcurrency: acc.CurrentConcurrency,
		})
	}

	c.JSON(http.StatusOK, GroupDetailResponse{
		GroupInfo: GroupInfo{
			ID:             group.ID,
			Name:           group.Name,
			Description:    group.Description,
			Platform:       string(group.Platform),
			BillingMode:    string(group.BillingMode),
			RateMultiplier: group.RateMultiplier,
			RPMOverride:    group.RPMOverride,
			ModelMapping:   group.ModelMapping,
			Priority:       group.Priority,
			Weight:         group.Weight,
			Status:         string(group.Status),
			AccountCount:   accountCount,
			CreatedAt:      group.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:      group.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
		Accounts: accountInfos,
	})
}

// Update 更新分组
// PUT /api/v1/admin/groups/:id
func (h *GroupHandler) Update(c *gin.Context) {
	// 解析分组ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的分组ID",
			},
		})
		return
	}

	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "请求参数无效: " + err.Error(),
			},
		})
		return
	}

	// 检查分组是否存在
	_, err = h.groupRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "GROUP_NOT_FOUND",
				"message": "分组不存在",
			},
		})
		return
	}

	// 更新分组
	updateInput := &repository.UpdateGroupInput{}
	if req.Name != "" {
		updateInput.Name = &req.Name
	}
	if req.Description != "" {
		updateInput.Description = &req.Description
	}
	if req.BillingMode != "" {
		updateInput.BillingMode = &req.BillingMode
	}
	if req.RateMultiplier > 0 {
		updateInput.RateMultiplier = &req.RateMultiplier
	}
	if req.RPMOverride != nil {
		updateInput.RPMOverride = req.RPMOverride
	}
	if req.ModelMapping != nil {
		updateInput.ModelMapping = req.ModelMapping
	}
	if req.Priority != 0 {
		updateInput.Priority = &req.Priority
	}
	if req.Weight > 0 {
		updateInput.Weight = &req.Weight
	}
	if req.Status != "" {
		updateInput.Status = &req.Status
	}

	updatedGroup, err := h.groupRepo.Update(c.Request.Context(), id, updateInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "更新分组失败: " + err.Error(),
			},
		})
		return
	}

	// 获取分组账号数量
	accountCount, _ := h.groupRepo.CountAccounts(c.Request.Context(), id)

	c.JSON(http.StatusOK, GroupInfo{
		ID:             updatedGroup.ID,
		Name:           updatedGroup.Name,
		Description:    updatedGroup.Description,
		Platform:       string(updatedGroup.Platform),
		BillingMode:    string(updatedGroup.BillingMode),
		RateMultiplier: updatedGroup.RateMultiplier,
		RPMOverride:    updatedGroup.RPMOverride,
		ModelMapping:   updatedGroup.ModelMapping,
		Priority:       updatedGroup.Priority,
		Weight:         updatedGroup.Weight,
		Status:         string(updatedGroup.Status),
		AccountCount:   accountCount,
		CreatedAt:      updatedGroup.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:      updatedGroup.UpdatedAt.Format("2006-01-02 15:04:05"),
	})
}

// Delete 删除分组
// DELETE /api/v1/admin/groups/:id
func (h *GroupHandler) Delete(c *gin.Context) {
	// 解析分组ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的分组ID",
			},
		})
		return
	}

	// 检查分组是否存在
	_, err = h.groupRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "GROUP_NOT_FOUND",
				"message": "分组不存在",
			},
		})
		return
	}

	// 删除分组
	err = h.groupRepo.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "删除分组失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "分组已删除",
	})
}

// AddAccounts 添加账号到分组
// POST /api/v1/admin/groups/:id/accounts
func (h *GroupHandler) AddAccounts(c *gin.Context) {
	// 解析分组ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的分组ID",
			},
		})
		return
	}

	var req AddAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "请求参数无效: " + err.Error(),
			},
		})
		return
	}

	// 检查分组是否存在
	_, err = h.groupRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "GROUP_NOT_FOUND",
				"message": "分组不存在",
			},
		})
		return
	}

	// 添加账号到分组
	err = h.groupRepo.AddAccounts(c.Request.Context(), id, req.AccountIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "添加账号失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "账号已添加到分组",
	})
}

// RemoveAccounts 从分组移除账号
// DELETE /api/v1/admin/groups/:id/accounts
func (h *GroupHandler) RemoveAccounts(c *gin.Context) {
	// 解析分组ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的分组ID",
			},
		})
		return
	}

	var req RemoveAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "请求参数无效: " + err.Error(),
			},
		})
		return
	}

	// 检查分组是否存在
	_, err = h.groupRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "GROUP_NOT_FOUND",
				"message": "分组不存在",
			},
		})
		return
	}

	// 从分组移除账号
	err = h.groupRepo.RemoveAccounts(c.Request.Context(), id, req.AccountIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "移除账号失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "账号已从分组移除",
	})
}
