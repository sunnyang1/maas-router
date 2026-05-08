// Package handler 提供 MaaS-Router 的 HTTP 处理器
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"maas-router/internal/pkg/ctxkey"
)

// AuthHandler 认证 Handler
// 处理用户注册、登录、Token 刷新、密码重置等认证相关操作
type AuthHandler struct {
	// AuthService 认证服务
	AuthService AuthService
	// UserService 用户服务
	UserService UserService
	// EmailService 邮件服务
	EmailService EmailService
	// JWTConfig JWT 配置
	JWTConfig *JWTConfig
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret     string
	ExpiresIn  time.Duration
	Issuer     string
}

// NewAuthHandler 创建认证 Handler
func NewAuthHandler(
	authService AuthService,
	userService UserService,
	emailService EmailService,
	jwtConfig *JWTConfig,
) *AuthHandler {
	return &AuthHandler{
		AuthService:  authService,
		UserService:  userService,
		EmailService: emailService,
		JWTConfig:    jwtConfig,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=64"`
	Name     string `json:"name,omitempty"`
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	User  *UserInfo `json:"user"`
	Token *TokenInfo `json:"token,omitempty"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	User  *UserInfo  `json:"user"`
	Token *TokenInfo `json:"token"`
}

// TokenInfo Token 信息
type TokenInfo struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // 秒
	TokenType    string `json:"token_type"`
}

// RefreshTokenRequest 刷新 Token 请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ForgotPasswordRequest 忘记密码请求
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=8,max=64"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name,omitempty"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}

// Register 处理 POST /api/v1/auth/register
// 用户注册接口
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求格式错误: "+err.Error())
		return
	}

	// 检查邮箱是否已注册
	exists, err := h.UserService.EmailExists(c.Request.Context(), req.Email)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "检查邮箱失败")
		return
	}
	if exists {
		ErrorResponse(c, http.StatusConflict, "EMAIL_EXISTS", "该邮箱已被注册")
		return
	}

	// 创建用户
	user, err := h.UserService.CreateUser(c.Request.Context(), &CreateUserParams{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Role:     "user",
	})
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "CREATE_USER_FAILED", "创建用户失败: "+err.Error())
		return
	}

	// 生成 Token（可选，注册后自动登录）
	token, err := h.AuthService.GenerateToken(c.Request.Context(), user.ID, user.Email, user.Role, user.TokenVersion)
	if err != nil {
		// 用户创建成功但 Token 生成失败，仍然返回成功
		c.JSON(http.StatusCreated, RegisterResponse{
			User: &UserInfo{
				ID:        user.ID,
				Email:     user.Email,
				Name:      user.Name,
				Role:      user.Role,
				Status:    user.Status,
				Balance:   user.Balance,
				CreatedAt: user.CreatedAt,
			},
		})
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{
		User: &UserInfo{
			ID:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			Role:      user.Role,
			Status:    user.Status,
			Balance:   user.Balance,
			CreatedAt: user.CreatedAt,
		},
		Token: &TokenInfo{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			ExpiresIn:    token.ExpiresIn,
			TokenType:    "Bearer",
		},
	})
}

// Login 处理 POST /api/v1/auth/login
// 用户登录接口
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求格式错误: "+err.Error())
		return
	}

	// 验证用户凭证
	user, err := h.AuthService.Authenticate(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		ErrorResponse(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "邮箱或密码错误")
		return
	}

	// 检查用户状态
	if user.Status != "active" {
		ErrorResponse(c, http.StatusForbidden, "USER_SUSPENDED", "用户账户已被暂停")
		return
	}

	// 生成 Token
	token, err := h.AuthService.GenerateToken(c.Request.Context(), user.ID, user.Email, user.Role, user.TokenVersion)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "TOKEN_GENERATION_FAILED", "生成 Token 失败")
		return
	}

	// 更新最后活跃时间
	go h.UserService.UpdateLastActive(c.Request.Context(), user.ID)

	c.JSON(http.StatusOK, LoginResponse{
		User: &UserInfo{
			ID:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			Role:      user.Role,
			Status:    user.Status,
			Balance:   user.Balance,
			CreatedAt: user.CreatedAt,
		},
		Token: &TokenInfo{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			ExpiresIn:    token.ExpiresIn,
			TokenType:    "Bearer",
		},
	})
}

// RefreshToken 处理 POST /api/v1/auth/refresh
// 刷新 Token 接口
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求格式错误: "+err.Error())
		return
	}

	// 验证并刷新 Token
	token, err := h.AuthService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		ErrorResponse(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "无效的刷新 Token")
		return
	}

	c.JSON(http.StatusOK, TokenInfo{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    token.ExpiresIn,
		TokenType:    "Bearer",
	})
}

// Logout 处理 POST /api/v1/auth/logout
// 用户登出接口（可选：将 Token 加入黑名单）
func (h *AuthHandler) Logout(c *gin.Context) {
	// 获取当前用户信息（可选）
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
		return
	}

	// 可选：将 Token 加入黑名单
	// 这需要 Token 黑名单服务支持
	// h.AuthService.InvalidateToken(c.Request.Context(), userID.(string))

	// 更新最后活跃时间
	go h.UserService.UpdateLastActive(c.Request.Context(), userID.(string))

	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}

// ForgotPassword 处理 POST /api/v1/auth/forgot-password
// 忘记密码接口，发送重置密码邮件
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求格式错误: "+err.Error())
		return
	}

	// 检查邮箱是否存在
	user, err := h.UserService.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		// 为了安全，不暴露邮箱是否存在的信息
		c.JSON(http.StatusOK, gin.H{
			"message": "如果该邮箱已注册，您将收到重置密码的邮件",
		})
		return
	}

	// 生成重置密码 Token
	resetToken, err := h.AuthService.GeneratePasswordResetToken(c.Request.Context(), user.ID)
	if err != nil {
		// 仍然返回成功，避免暴露错误信息
		c.JSON(http.StatusOK, gin.H{
			"message": "如果该邮箱已注册，您将收到重置密码的邮件",
		})
		return
	}

	// 发送重置密码邮件
	go func() {
		_ = h.EmailService.SendPasswordResetEmail(c.Request.Context(), user.Email, resetToken)
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "如果该邮箱已注册，您将收到重置密码的邮件",
	})
}

// ResetPassword 处理 POST /api/v1/auth/reset-password
// 重置密码接口
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求格式错误: "+err.Error())
		return
	}

	// 验证重置 Token
	userID, err := h.AuthService.ValidatePasswordResetToken(c.Request.Context(), req.Token)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_RESET_TOKEN", "无效或已过期的重置 Token")
		return
	}

	// 更新密码
	err = h.UserService.UpdatePassword(c.Request.Context(), userID, req.Password)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "PASSWORD_UPDATE_FAILED", "密码更新失败")
		return
	}

	// 使旧 Token 失效（增加 Token 版本）
	go h.UserService.IncrementTokenVersion(c.Request.Context(), userID)

	c.JSON(http.StatusOK, gin.H{
		"message": "密码重置成功，请使用新密码登录",
	})
}

// RegisterAuthHandlers 注册认证路由到 HandlerGroup
func RegisterAuthHandlers(h *AuthHandler) map[string]gin.HandlerFunc {
	return map[string]gin.HandlerFunc{
		"Register":       h.Register,
		"Login":          h.Login,
		"RefreshToken":   h.RefreshToken,
		"Logout":         h.Logout,
		"ForgotPassword": h.ForgotPassword,
		"ResetPassword":  h.ResetPassword,
	}
}

// ===== 服务接口定义 =====

// AuthService 认证服务接口
type AuthService interface {
	// Authenticate 验证用户凭证
	Authenticate(ctx interface{}, email, password string) (*User, error)
	// GenerateToken 生成 Token
	GenerateToken(ctx interface{}, userID, email, role string, tokenVersion int) (*Token, error)
	// RefreshToken 刷新 Token
	RefreshToken(ctx interface{}, refreshToken string) (*Token, error)
	// GeneratePasswordResetToken 生成密码重置 Token
	GeneratePasswordResetToken(ctx interface{}, userID string) (string, error)
	// ValidatePasswordResetToken 验证密码重置 Token
	ValidatePasswordResetToken(ctx interface{}, token string) (string, error)
}

// Token Token 信息
type Token struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// User 用户信息
type User struct {
	ID           string
	Email        string
	Name         string
	Role         string
	Status       string
	Balance      float64
	TokenVersion int
	CreatedAt    time.Time
}

// UserService 用户服务接口
type UserService interface {
	// CreateUser 创建用户
	CreateUser(ctx interface{}, params *CreateUserParams) (*User, error)
	// GetByEmail 根据邮箱获取用户
	GetByEmail(ctx interface{}, email string) (*User, error)
	// GetByID 根据 ID 获取用户
	GetByID(ctx interface{}, id string) (*User, error)
	// EmailExists 检查邮箱是否存在
	EmailExists(ctx interface{}, email string) (bool, error)
	// UpdatePassword 更新密码
	UpdatePassword(ctx interface{}, userID, newPassword string) error
	// UpdateLastActive 更新最后活跃时间
	UpdateLastActive(ctx interface{}, userID string) error
	// IncrementTokenVersion 增加 Token 版本
	IncrementTokenVersion(ctx interface{}, userID string) error
}

// CreateUserParams 创建用户参数
type CreateUserParams struct {
	Email    string
	Password string
	Name     string
	Role     string
}

// EmailService 邮件服务接口
type EmailService interface {
	// SendPasswordResetEmail 发送密码重置邮件
	SendPasswordResetEmail(ctx interface{}, email, token string) error
	// SendVerificationEmail 发送验证邮件
	SendVerificationEmail(ctx interface{}, email, token string) error
}

// VerifyEmail 处理邮箱验证（可选接口）
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_TOKEN", "缺少验证 Token")
		return
	}

	// 验证 Token 并标记邮箱已验证
	userID, err := h.AuthService.ValidatePasswordResetToken(c.Request.Context(), token)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_TOKEN", "无效或已过期的验证 Token")
		return
	}

	// 标记邮箱已验证
	// err = h.UserService.MarkEmailVerified(c.Request.Context(), userID)
	// if err != nil {
	// 	ErrorResponse(c, http.StatusInternalServerError, "VERIFICATION_FAILED", "邮箱验证失败")
	// 	return
	// }

	c.JSON(http.StatusOK, gin.H{
		"message": "邮箱验证成功",
		"user_id": userID,
	})
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=64"`
}

// ChangePassword 修改密码（需要已登录）
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	var req ChangePasswordRequest
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

	// 使所有旧 Token 失效
	go h.UserService.IncrementTokenVersion(c.Request.Context(), userID.(string))

	c.JSON(http.StatusOK, gin.H{
		"message": "密码修改成功，请重新登录",
	})
}

// GetCurrentUser 获取当前用户信息
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权的请求")
		return
	}

	user, err := h.UserService.GetByID(c.Request.Context(), userID.(string))
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "USER_NOT_FOUND", "用户不存在")
		return
	}

	c.JSON(http.StatusOK, UserInfo{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		Status:    user.Status,
		Balance:   user.Balance,
		CreatedAt: user.CreatedAt,
	})
}
