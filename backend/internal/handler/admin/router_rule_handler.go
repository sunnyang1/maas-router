// Package admin 提供管理员相关的 HTTP 处理器
package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"maas-router/internal/repository"
)

// RouterRuleHandler 路由规则管理 Handler
type RouterRuleHandler struct {
	routerRuleRepo *repository.RouterRuleRepository
}

// NewRouterRuleHandler 创建路由规则管理 Handler
func NewRouterRuleHandler(
	routerRuleRepo *repository.RouterRuleRepository,
) *RouterRuleHandler {
	return &RouterRuleHandler{
		routerRuleRepo: routerRuleRepo,
	}
}

// RouterRuleListRequest 路由规则列表请求
type RouterRuleListRequest struct {
	// 页码
	Page int `form:"page" binding:"min=1"`
	// 每页数量
	PageSize int `form:"page_size" binding:"min=1,max=100"`
	// 是否只查询启用的规则
	ActiveOnly bool `form:"active_only"`
	// 排序字段
	SortBy string `form:"sort_by"`
	// 排序方向
	SortOrder string `form:"sort_order"`
}

// RouterRuleListResponse 路由规则列表响应
type RouterRuleListResponse struct {
	// 规则列表
	List []*RouterRuleInfo `json:"list"`
	// 总数
	Total int64 `json:"total"`
	// 当前页码
	Page int `json:"page"`
	// 每页数量
	PageSize int `json:"page_size"`
}

// RouterRuleInfo 路由规则信息
type RouterRuleInfo struct {
	// 规则ID
	ID int64 `json:"id"`
	// 规则名称
	Name string `json:"name"`
	// 规则描述
	Description string `json:"description,omitempty"`
	// 优先级
	Priority int `json:"priority"`
	// 匹配条件
	Condition map[string]interface{} `json:"condition"`
	// 执行动作
	Action map[string]interface{} `json:"action"`
	// 是否启用
	IsActive bool `json:"is_active"`
	// 创建时间
	CreatedAt string `json:"created_at"`
	// 更新时间
	UpdatedAt string `json:"updated_at"`
}

// CreateRouterRuleRequest 创建路由规则请求
type CreateRouterRuleRequest struct {
	// 规则名称
	Name string `json:"name" binding:"required,max=100"`
	// 规则描述
	Description string `json:"description" binding:"max=500"`
	// 优先级
	Priority int `json:"priority" binding:"min=-1000,max=1000"`
	// 匹配条件
	Condition map[string]interface{} `json:"condition" binding:"required"`
	// 执行动作
	Action map[string]interface{} `json:"action" binding:"required"`
	// 是否启用
	IsActive bool `json:"is_active"`
}

// CreateRouterRuleResponse 创建路由规则响应
type CreateRouterRuleResponse struct {
	// 规则ID
	ID int64 `json:"id"`
	// 规则名称
	Name string `json:"name"`
	// 优先级
	Priority int `json:"priority"`
	// 是否启用
	IsActive bool `json:"is_active"`
	// 创建时间
	CreatedAt string `json:"created_at"`
}

// UpdateRouterRuleRequest 更新路由规则请求
type UpdateRouterRuleRequest struct {
	// 规则名称
	Name string `json:"name" binding:"omitempty,max=100"`
	// 规则描述
	Description string `json:"description" binding:"omitempty,max=500"`
	// 优先级
	Priority int `json:"priority" binding:"omitempty,min=-1000,max=1000"`
	// 匹配条件
	Condition map[string]interface{} `json:"condition"`
	// 执行动作
	Action map[string]interface{} `json:"action"`
	// 是否启用
	IsActive *bool `json:"is_active"`
}

// List 获取路由规则列表
// GET /api/v1/admin/router-rules
func (h *RouterRuleHandler) List(c *gin.Context) {
	var req RouterRuleListRequest
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
	filter := repository.RouterRuleListFilter{
		ActiveOnly: req.ActiveOnly,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
	}

	// 查询规则列表
	rules, total, err := h.routerRuleRepo.List(c.Request.Context(), filter, req.Page, req.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "查询路由规则列表失败",
			},
		})
		return
	}

	// 转换为响应格式
	list := make([]*RouterRuleInfo, 0, len(rules))
	for _, r := range rules {
		list = append(list, &RouterRuleInfo{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Priority:    r.Priority,
			Condition:   r.Condition,
			Action:      r.Action,
			IsActive:    r.IsActive,
			CreatedAt:   r.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   r.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, RouterRuleListResponse{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

// Create 创建路由规则
// POST /api/v1/admin/router-rules
func (h *RouterRuleHandler) Create(c *gin.Context) {
	var req CreateRouterRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "请求参数无效: " + err.Error(),
			},
		})
		return
	}

	// 创建路由规则
	rule, err := h.routerRuleRepo.Create(c.Request.Context(), &repository.CreateRouterRuleInput{
		Name:        req.Name,
		Description: req.Description,
		Priority:    req.Priority,
		Condition:   req.Condition,
		Action:      req.Action,
		IsActive:    req.IsActive,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "创建路由规则失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, CreateRouterRuleResponse{
		ID:        rule.ID,
		Name:      rule.Name,
		Priority:  rule.Priority,
		IsActive:  rule.IsActive,
		CreatedAt: rule.CreatedAt.Format("2006-01-02 15:04:05"),
	})
}

// Get 获取路由规则详情
// GET /api/v1/admin/router-rules/:id
func (h *RouterRuleHandler) Get(c *gin.Context) {
	// 解析规则ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的规则ID",
			},
		})
		return
	}

	// 查询规则
	rule, err := h.routerRuleRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "RULE_NOT_FOUND",
				"message": "路由规则不存在",
			},
		})
		return
	}

	c.JSON(http.StatusOK, RouterRuleInfo{
		ID:          rule.ID,
		Name:        rule.Name,
		Description: rule.Description,
		Priority:    rule.Priority,
		Condition:   rule.Condition,
		Action:      rule.Action,
		IsActive:    rule.IsActive,
		CreatedAt:   rule.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   rule.UpdatedAt.Format("2006-01-02 15:04:05"),
	})
}

// Update 更新路由规则
// PUT /api/v1/admin/router-rules/:id
func (h *RouterRuleHandler) Update(c *gin.Context) {
	// 解析规则ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的规则ID",
			},
		})
		return
	}

	var req UpdateRouterRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "请求参数无效: " + err.Error(),
			},
		})
		return
	}

	// 检查规则是否存在
	_, err = h.routerRuleRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "RULE_NOT_FOUND",
				"message": "路由规则不存在",
			},
		})
		return
	}

	// 更新规则
	updateInput := &repository.UpdateRouterRuleInput{}
	if req.Name != "" {
		updateInput.Name = &req.Name
	}
	if req.Description != "" {
		updateInput.Description = &req.Description
	}
	if req.Priority != 0 {
		updateInput.Priority = &req.Priority
	}
	if req.Condition != nil {
		updateInput.Condition = req.Condition
	}
	if req.Action != nil {
		updateInput.Action = req.Action
	}
	if req.IsActive != nil {
		updateInput.IsActive = req.IsActive
	}

	updatedRule, err := h.routerRuleRepo.Update(c.Request.Context(), id, updateInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "更新路由规则失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, RouterRuleInfo{
		ID:          updatedRule.ID,
		Name:        updatedRule.Name,
		Description: updatedRule.Description,
		Priority:    updatedRule.Priority,
		Condition:   updatedRule.Condition,
		Action:      updatedRule.Action,
		IsActive:    updatedRule.IsActive,
		CreatedAt:   updatedRule.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   updatedRule.UpdatedAt.Format("2006-01-02 15:04:05"),
	})
}

// Delete 删除路由规则
// DELETE /api/v1/admin/router-rules/:id
func (h *RouterRuleHandler) Delete(c *gin.Context) {
	// 解析规则ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的规则ID",
			},
		})
		return
	}

	// 检查规则是否存在
	_, err = h.routerRuleRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "RULE_NOT_FOUND",
				"message": "路由规则不存在",
			},
		})
		return
	}

	// 删除规则
	err = h.routerRuleRepo.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "删除路由规则失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "路由规则已删除",
	})
}
