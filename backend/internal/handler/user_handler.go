// Package handler 提供 MaaS-Router 的 HTTP 处理器
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"maas-router/internal/pkg/ctxkey"
)

// UserHandler 用户 Handler
// 处理用户资料管理、密码修改等用户相关操作
type UserHandler struct {
	// UserService 用户服务
	UserService UserService
	// AuthService 认证服务
	AuthService AuthService
}

// NewUserHandler 创建用户 Handler
func NewUserHandler(
	userService UserService,
	authService AuthService,
) *UserHandler {
	return &UserHandler{
		UserService: userService,
		AuthService: authService,
	}
}

// UserProfileResponse 用户资料响应
type UserProfileResponse struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name,omitempty"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	Balance      float64   `json:"balance"`
	Concurrency  int       `json:"concurrency"`
	LastActiveAt *time.Time `json:"last_active_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UpdateProfileRequest 更新资料请求
type UpdateProfileRequest struct {
	Name string `json:"name,omitempty"`
}

// UpdatePasswordRequest 更新密码请求
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=64"`
}

// GetUserProfile 处理 GET /api/v1/user/profile
// 获取当前用户资料
func (h *UserHandler) GetUserProfile(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 获取用户信息
	user, err := h.UserService.GetByID(c.Request.Context(), userID.(string))
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "USER_NOT_FOUND", "用户不存在")
		return
	}

	c.JSON(http.StatusOK, UserProfileResponse{
		ID:           user.ID,
		Email:        user.Email,
		Name:         user.Name,
		Role:         user.Role,
		Status:       user.Status,
		Balance:      user.Balance,
		Concurrency:  5, // 默认并发数
		LastActiveAt: nil,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.CreatedAt,
	})
}

// UpdateUserProfile 处理 PUT /api/v1/user/profile
// 更新当前用户资料
func (h *UserHandler) UpdateUserProfile(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 解析请求
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求格式错误: "+err.Error())
		return
	}

	// 更新用户资料
	err := h.UserService.UpdateProfile(c.Request.Context(), userID.(string), &UpdateProfileParams{
		Name: req.Name,
	})
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "UPDATE_FAILED", "更新资料失败: "+err.Error())
		return
	}

	// 获取更新后的用户信息
	user, err := h.UserService.GetByID(c.Request.Context(), userID.(string))
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取用户信息失败")
		return
	}

	c.JSON(http.StatusOK, UserProfileResponse{
		ID:           user.ID,
		Email:        user.Email,
		Name:         user.Name,
		Role:         user.Role,
		Status:       user.Status,
		Balance:      user.Balance,
		Concurrency:  5,
		LastActiveAt: nil,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.CreatedAt,
	})
}

// UpdatePassword 处理 PUT /api/v1/user/password
// 修改当前用户密码
func (h *UserHandler) UpdatePassword(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 解析请求
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求格式错误: "+err.Error())
		return
	}

	// 获取用户信息
	user, err := h.UserService.GetByID(c.Request.Context(), userID.(string))
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "USER_NOT_FOUND", "用户不存在")
		return
	}

	// 验证旧密码
	_, err = h.AuthService.Authenticate(c.Request.Context(), user.Email, req.OldPassword)
	if err != nil {
		ErrorResponse(c, http.StatusUnauthorized, "INVALID_PASSWORD", "原密码错误")
		return
	}

	// 更新密码
	err = h.UserService.UpdatePassword(c.Request.Context(), userID.(string), req.NewPassword)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "PASSWORD_UPDATE_FAILED", "密码更新失败")
		return
	}

	// 使所有旧 Token 失效（增加 Token 版本）
	go h.UserService.IncrementTokenVersion(c.Request.Context(), userID.(string))

	c.JSON(http.StatusOK, gin.H{
		"message": "密码修改成功，请使用新密码重新登录",
	})
}

// DeleteAccountRequest 删除账户请求
type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required"`
}

// DeleteAccount 处理 DELETE /api/v1/user/account
// 删除当前用户账户（软删除）
func (h *UserHandler) DeleteAccount(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 解析请求
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求格式错误: "+err.Error())
		return
	}

	// 获取用户信息
	user, err := h.UserService.GetByID(c.Request.Context(), userID.(string))
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "USER_NOT_FOUND", "用户不存在")
		return
	}

	// 验证密码
	_, err = h.AuthService.Authenticate(c.Request.Context(), user.Email, req.Password)
	if err != nil {
		ErrorResponse(c, http.StatusUnauthorized, "INVALID_PASSWORD", "密码错误")
		return
	}

	// 软删除用户账户
	err = h.UserService.DeleteUser(c.Request.Context(), userID.(string))
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "DELETE_FAILED", "删除账户失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "账户已删除",
	})
}

// GetUserStats 获取用户统计数据
func (h *UserHandler) GetUserStats(c *gin.Context) {
	// 从 Context 获取用户 ID
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	// 获取用户统计数据
	stats, err := h.UserService.GetUserStats(c.Request.Context(), userID.(string))
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取统计数据失败")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// RegisterUserHandlers 注册用户路由到 HandlerGroup
func RegisterUserHandlers(h *UserHandler) map[string]gin.HandlerFunc {
	return map[string]gin.HandlerFunc{
		"GetUserProfile":   h.GetUserProfile,
		"UpdateUserProfile": h.UpdateUserProfile,
		"UpdatePassword":   h.UpdatePassword,
	}
}

// ===== 扩展接口定义 =====

// UpdateProfileParams 更新资料参数
type UpdateProfileParams struct {
	Name string
}

// UserStats 用户统计数据
type UserStats struct {
	TotalRequests   int64   `json:"total_requests"`
	TotalTokens     int64   `json:"total_tokens"`
	TotalCost       float64 `json:"total_cost"`
	ActiveAPIKeys   int     `json:"active_api_keys"`
	RequestsToday   int64   `json:"requests_today"`
	CostToday       float64 `json:"cost_today"`
	RequestsMonth   int64   `json:"requests_month"`
	CostMonth       float64 `json:"cost_month"`
}

// 扩展 UserService 接口
type UserServiceExt interface {
	UserService
	// UpdateProfile 更新用户资料
	UpdateProfile(ctx interface{}, userID string, params *UpdateProfileParams) error
	// DeleteUser 删除用户（软删除）
	DeleteUser(ctx interface{}, userID string) error
	// GetUserStats 获取用户统计数据
	GetUserStats(ctx interface{}, userID string) (*UserStats, error)
}
