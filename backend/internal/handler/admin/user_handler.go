// Package admin 提供管理员相关的 HTTP 处理器
package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"maas-router/internal/repository"
)

// UserHandler 用户管理 Handler
type UserHandler struct {
	userRepo  *repository.UserRepository
	usageRepo *repository.UsageRecordRepository
}

// NewUserHandler 创建用户管理 Handler
func NewUserHandler(
	userRepo *repository.UserRepository,
	usageRepo *repository.UsageRecordRepository,
) *UserHandler {
	return &UserHandler{
		userRepo:  userRepo,
		usageRepo: usageRepo,
	}
}

// UserListRequest 用户列表请求
type UserListRequest struct {
	// 页码
	Page int `form:"page" binding:"min=1"`
	// 每页数量
	PageSize int `form:"page_size" binding:"min=1,max=100"`
	// 搜索关键词（邮箱或昵称）
	Keyword string `form:"keyword"`
	// 用户角色筛选
	Role string `form:"role"`
	// 用户状态筛选
	Status string `form:"status"`
	// 排序字段
	SortBy string `form:"sort_by"`
	// 排序方向
	SortOrder string `form:"sort_order"`
}

// UserListResponse 用户列表响应
type UserListResponse struct {
	// 用户列表
	List []*UserInfo `json:"list"`
	// 总数
	Total int64 `json:"total"`
	// 当前页码
	Page int `json:"page"`
	// 每页数量
	PageSize int `json:"page_size"`
}

// UserInfo 用户信息
type UserInfo struct {
	// 用户ID
	ID int64 `json:"id"`
	// 邮箱
	Email string `json:"email"`
	// 昵称
	Name string `json:"name"`
	// 角色
	Role string `json:"role"`
	// 状态
	Status string `json:"status"`
	// 余额
	Balance float64 `json:"balance"`
	// 并发限制
	Concurrency int `json:"concurrency"`
	// 最后活跃时间
	LastActiveAt *string `json:"last_active_at,omitempty"`
	// 创建时间
	CreatedAt string `json:"created_at"`
	// 更新时间
	UpdatedAt string `json:"updated_at"`
	// API Key 数量
	APIKeyCount int64 `json:"api_key_count"`
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	// 邮箱
	Email string `json:"email" binding:"required,email,max=255"`
	// 密码
	Password string `json:"password" binding:"required,min=6,max=72"`
	// 昵称
	Name string `json:"name" binding:"max=100"`
	// 角色
	Role string `json:"role" binding:"omitempty,oneof=user admin"`
	// 初始余额
	Balance float64 `json:"balance" binding:"min=0"`
	// 并发限制
	Concurrency int `json:"concurrency" binding:"min=1,max=1000"`
}

// CreateUserResponse 创建用户响应
type CreateUserResponse struct {
	// 用户ID
	ID int64 `json:"id"`
	// 邮箱
	Email string `json:"email"`
	// 昵称
	Name string `json:"name"`
	// 角色
	Role string `json:"role"`
	// 状态
	Status string `json:"status"`
	// 余额
	Balance float64 `json:"balance"`
	// 并发限制
	Concurrency int `json:"concurrency"`
	// 创建时间
	CreatedAt string `json:"created_at"`
}

// UserDetailResponse 用户详情响应
type UserDetailResponse struct {
	// 基本信息
	UserInfo
	// 使用统计
	UsageStats UserUsageStats `json:"usage_stats"`
}

// UserUsageStats 用户使用统计
type UserUsageStats struct {
	// 今日请求数
	TodayRequests int64 `json:"today_requests"`
	// 今日费用
	TodayCost float64 `json:"today_cost"`
	// 本月请求数
	MonthRequests int64 `json:"month_requests"`
	// 本月费用
	MonthCost float64 `json:"month_cost"`
	// 总请求数
	TotalRequests int64 `json:"total_requests"`
	// 总费用
	TotalCost float64 `json:"total_cost"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	// 昵称
	Name string `json:"name" binding:"omitempty,max=100"`
	// 角色
	Role string `json:"role" binding:"omitempty,oneof=user admin"`
	// 状态
	Status string `json:"status" binding:"omitempty,oneof=active suspended deleted"`
	// 并发限制
	Concurrency int `json:"concurrency" binding:"omitempty,min=1,max=1000"`
}

// AdjustBalanceRequest 调整余额请求
type AdjustBalanceRequest struct {
	// 调整金额（正数为增加，负数为减少）
	Amount float64 `json:"amount" binding:"required"`
	// 调整原因
	Reason string `json:"reason" binding:"required,max=500"`
}

// AdjustBalanceResponse 调整余额响应
type AdjustBalanceResponse struct {
	// 调整前余额
	BeforeBalance float64 `json:"before_balance"`
	// 调整后余额
	AfterBalance float64 `json:"after_balance"`
	// 调整金额
	Amount float64 `json:"amount"`
}

// List 获取用户列表
// GET /api/v1/admin/users
func (h *UserHandler) List(c *gin.Context) {
	var req UserListRequest
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
	filter := repository.UserListFilter{
		Keyword:   req.Keyword,
		Role:      req.Role,
		Status:    req.Status,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	// 查询用户列表
	users, total, err := h.userRepo.List(c.Request.Context(), filter, req.Page, req.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "查询用户列表失败",
			},
		})
		return
	}

	// 转换为响应格式
	list := make([]*UserInfo, 0, len(users))
	for _, u := range users {
		var lastActiveAt *string
		if u.LastActiveAt != nil {
			t := u.LastActiveAt.Format("2006-01-02 15:04:05")
			lastActiveAt = &t
		}

		// 获取用户的 API Key 数量
		apiKeyCount, _ := h.userRepo.CountAPIKeys(c.Request.Context(), u.ID)

		list = append(list, &UserInfo{
			ID:           u.ID,
			Email:        u.Email,
			Name:         u.Name,
			Role:         string(u.Role),
			Status:       string(u.Status),
			Balance:      u.Balance,
			Concurrency:  u.Concurrency,
			LastActiveAt: lastActiveAt,
			CreatedAt:    u.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    u.UpdatedAt.Format("2006-01-02 15:04:05"),
			APIKeyCount:  apiKeyCount,
		})
	}

	c.JSON(http.StatusOK, UserListResponse{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

// Create 创建用户
// POST /api/v1/admin/users
func (h *UserHandler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "请求参数无效: " + err.Error(),
			},
		})
		return
	}

	// 检查邮箱是否已存在
	exists, err := h.userRepo.ExistsByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "检查邮箱失败",
			},
		})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{
			"error": gin.H{
				"code":    "EMAIL_EXISTS",
				"message": "邮箱已被注册",
			},
		})
		return
	}

	// 设置默认值
	role := "user"
	if req.Role != "" {
		role = req.Role
	}
	concurrency := 5
	if req.Concurrency > 0 {
		concurrency = req.Concurrency
	}

	// 创建用户
	user, err := h.userRepo.Create(c.Request.Context(), &repository.CreateUserInput{
		Email:       strings.ToLower(req.Email),
		Password:    req.Password,
		Name:        req.Name,
		Role:        role,
		Balance:     req.Balance,
		Concurrency: concurrency,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "创建用户失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, CreateUserResponse{
		ID:          user.ID,
		Email:       user.Email,
		Name:        user.Name,
		Role:        string(user.Role),
		Status:      string(user.Status),
		Balance:     user.Balance,
		Concurrency: user.Concurrency,
		CreatedAt:   user.CreatedAt.Format("2006-01-02 15:04:05"),
	})
}

// Get 获取用户详情
// GET /api/v1/admin/users/:id
func (h *UserHandler) Get(c *gin.Context) {
	// 解析用户ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的用户ID",
			},
		})
		return
	}

	// 查询用户
	user, err := h.userRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "用户不存在",
			},
		})
		return
	}

	// 获取用户使用统计
	usageStats, err := h.usageRepo.GetUserStats(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取用户统计失败",
			},
		})
		return
	}

	// 获取用户的 API Key 数量
	apiKeyCount, _ := h.userRepo.CountAPIKeys(c.Request.Context(), id)

	var lastActiveAt *string
	if user.LastActiveAt != nil {
		t := user.LastActiveAt.Format("2006-01-02 15:04:05")
		lastActiveAt = &t
	}

	c.JSON(http.StatusOK, UserDetailResponse{
		UserInfo: UserInfo{
			ID:           user.ID,
			Email:        user.Email,
			Name:         user.Name,
			Role:         string(user.Role),
			Status:       string(user.Status),
			Balance:      user.Balance,
			Concurrency:  user.Concurrency,
			LastActiveAt: lastActiveAt,
			CreatedAt:    user.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    user.UpdatedAt.Format("2006-01-02 15:04:05"),
			APIKeyCount:  apiKeyCount,
		},
		UsageStats: UserUsageStats{
			TodayRequests: usageStats.TodayRequests,
			TodayCost:     usageStats.TodayCost,
			MonthRequests: usageStats.MonthRequests,
			MonthCost:     usageStats.MonthCost,
			TotalRequests: usageStats.TotalRequests,
			TotalCost:     usageStats.TotalCost,
		},
	})
}

// Update 更新用户
// PUT /api/v1/admin/users/:id
func (h *UserHandler) Update(c *gin.Context) {
	// 解析用户ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的用户ID",
			},
		})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "请求参数无效: " + err.Error(),
			},
		})
		return
	}

	// 检查用户是否存在
	user, err := h.userRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "用户不存在",
			},
		})
		return
	}

	// 更新用户
	updateInput := &repository.UpdateUserInput{}
	if req.Name != "" {
		updateInput.Name = &req.Name
	}
	if req.Role != "" {
		updateInput.Role = &req.Role
	}
	if req.Status != "" {
		updateInput.Status = &req.Status
	}
	if req.Concurrency > 0 {
		updateInput.Concurrency = &req.Concurrency
	}

	updatedUser, err := h.userRepo.Update(c.Request.Context(), id, updateInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "更新用户失败: " + err.Error(),
			},
		})
		return
	}

	var lastActiveAt *string
	if updatedUser.LastActiveAt != nil {
		t := updatedUser.LastActiveAt.Format("2006-01-02 15:04:05")
		lastActiveAt = &t
	}

	c.JSON(http.StatusOK, UserInfo{
		ID:           updatedUser.ID,
		Email:        updatedUser.Email,
		Name:         updatedUser.Name,
		Role:         string(updatedUser.Role),
		Status:       string(updatedUser.Status),
		Balance:      updatedUser.Balance,
		Concurrency:  updatedUser.Concurrency,
		LastActiveAt: lastActiveAt,
		CreatedAt:    updatedUser.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    updatedUser.UpdatedAt.Format("2006-01-02 15:04:05"),
	})
}

// Delete 删除用户
// DELETE /api/v1/admin/users/:id
func (h *UserHandler) Delete(c *gin.Context) {
	// 解析用户ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的用户ID",
			},
		})
		return
	}

	// 检查用户是否存在
	_, err = h.userRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "用户不存在",
			},
		})
		return
	}

	// 软删除用户（将状态设为 deleted）
	err = h.userRepo.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "删除用户失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "用户已删除",
	})
}

// AdjustBalance 调整用户余额
// POST /api/v1/admin/users/:id/balance
func (h *UserHandler) AdjustBalance(c *gin.Context) {
	// 解析用户ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "无效的用户ID",
			},
		})
		return
	}

	var req AdjustBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "请求参数无效: " + err.Error(),
			},
		})
		return
	}

	// 检查用户是否存在
	user, err := h.userRepo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "用户不存在",
			},
		})
		return
	}

	// 记录调整前余额
	beforeBalance := user.Balance

	// 调整余额
	afterBalance, err := h.userRepo.AdjustBalance(c.Request.Context(), id, req.Amount, req.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BALANCE_ADJUST_FAILED",
				"message": "余额调整失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, AdjustBalanceResponse{
		BeforeBalance: beforeBalance,
		AfterBalance:  afterBalance,
		Amount:        req.Amount,
	})
}
