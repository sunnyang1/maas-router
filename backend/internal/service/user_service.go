// Package service 业务服务层
// 提供用户服务
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"maas-router/ent"
	"maas-router/internal/cache"
	"maas-router/internal/config"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserService 用户服务接口
// 处理用户注册、登录、资料管理等
type UserService interface {
	// Register 注册
	Register(ctx context.Context, email, password, name string) (*ent.User, error)

	// Login 登录
	Login(ctx context.Context, email, password string) (*LoginResponse, error)

	// RefreshToken 刷新 Token
	RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error)

	// Logout 登出
	Logout(ctx context.Context, userID int64) error

	// GetProfile 获取资料
	GetProfile(ctx context.Context, userID int64) (*ent.User, error)

	// UpdateProfile 更新资料
	UpdateProfile(ctx context.Context, userID int64, data *UpdateProfileRequest) error

	// ChangePassword 修改密码
	ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error

	// GetByID 根据 ID 获取用户
	GetByID(ctx context.Context, userID int64) (*ent.User, error)

	// GetByEmail 根据邮箱获取用户
	GetByEmail(ctx context.Context, email string) (*ent.User, error)

	// UpdateBalance 更新余额
	UpdateBalance(ctx context.Context, userID int64, amount float64) error

	// ListUsers 获取用户列表
	ListUsers(ctx context.Context, page, pageSize int) ([]*ent.User, int, error)

	// UpdateStatus 更新用户状态
	UpdateStatus(ctx context.Context, userID int64, status string) error

	// DeleteUser 删除用户
	DeleteUser(ctx context.Context, userID int64) error
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"` // 秒
	User         *UserInfo `json:"user"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID        int64   `json:"id"`
	Email     string  `json:"email"`
	Name      string  `json:"name"`
	Role      string  `json:"role"`
	Status    string  `json:"status"`
	Balance   float64 `json:"balance"`
	CreatedAt string  `json:"created_at"`
}

// UpdateProfileRequest 更新资料请求
type UpdateProfileRequest struct {
	Name string `json:"name,omitempty"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// JWTClaims JWT 声明
type JWTClaims struct {
	UserID       int64  `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	TokenVersion int    `json:"token_version"`
	jwt.RegisteredClaims
}

// userService 用户服务实现
type userService struct {
	db        *ent.Client
	redis     *redis.Client
	cache     cache.Cache
	cacheKey  *cache.CacheKey
	cfg       *config.Config
	logger    *zap.Logger
}

// NewUserService 创建用户服务实例
func NewUserService(
	db *ent.Client,
	redis *redis.Client,
	cfg *config.Config,
	logger *zap.Logger,
) UserService {
	// 创建统一缓存实例
	c := cache.NewCacheFromClient(redis, logger, "maas")
	return &userService{
		db:       db,
		redis:    redis,
		cache:    c,
		cacheKey: cache.NewCacheKey("maas"),
		cfg:      cfg,
		logger:   logger,
	}
}

// Register 注册
func (s *userService) Register(ctx context.Context, email, password, name string) (*ent.User, error) {
	// 验证邮箱格式
	if !isValidEmail(email) {
		return nil, fmt.Errorf("邮箱格式不正确")
	}

	// 验证密码强度
	if !isValidPassword(password) {
		return nil, fmt.Errorf("密码长度至少 8 位，需包含字母和数字")
	}

	// 检查邮箱是否已注册
	exists, err := s.db.User.Query().
		Where(ent.UserEmail(email)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("检查邮箱失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("邮箱已被注册")
	}

	// 哈希密码
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码哈希失败: %w", err)
	}

	// 创建用户
	user, err := s.db.User.Create().
		SetEmail(email).
		SetPasswordHash(string(passwordHash)).
		SetName(name).
		SetRole(ent.UserRoleUser).
		SetStatus(ent.UserStatusActive).
		SetBalance(0).
		SetConcurrency(5).
		SetTokenVersion(1).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	s.logger.Info("用户注册成功",
		zap.Int64("user_id", user.ID),
		zap.String("email", email))

	return user, nil
}

// Login 登录
func (s *userService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	// 查找用户
	user, err := s.db.User.Query().
		Where(ent.UserEmail(email)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("邮箱或密码错误")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 检查用户状态
	if user.Status != ent.UserStatusActive {
		return nil, fmt.Errorf("账号已被禁用")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("邮箱或密码错误")
	}

	// 生成 Token
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("生成刷新令牌失败: %w", err)
	}

	// 更新最后活跃时间
	_, err = s.db.User.UpdateOneID(user.ID).
		SetLastActiveAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Warn("更新最后活跃时间失败", zap.Error(err))
	}

	// 缓存刷新 Token
	s.cacheRefreshToken(ctx, user.ID, refreshToken)

	s.logger.Info("用户登录成功",
		zap.Int64("user_id", user.ID),
		zap.String("email", email))

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.JWT.ExpireHours * 3600),
		User: &UserInfo{
			ID:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			Role:      string(user.Role),
			Status:    string(user.Status),
			Balance:   user.Balance,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}

// RefreshToken 刷新 Token
func (s *userService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// 解析刷新 Token
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("无效的刷新令牌")
	}

	// 检查缓存中的刷新 Token
	cachedToken, err := s.redis.Get(ctx, s.getRefreshTokenKey(claims.UserID)).Result()
	if err != nil || cachedToken != refreshToken {
		return nil, fmt.Errorf("刷新令牌已过期或已被撤销")
	}

	// 获取用户信息
	user, err := s.db.User.Get(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在")
	}

	// 检查 Token 版本
	if user.TokenVersion != claims.TokenVersion {
		return nil, fmt.Errorf("令牌已失效，请重新登录")
	}

	// 检查用户状态
	if user.Status != ent.UserStatusActive {
		return nil, fmt.Errorf("账号已被禁用")
	}

	// 生成新的 Token
	newAccessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %w", err)
	}

	newRefreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("生成刷新令牌失败: %w", err)
	}

	// 更新缓存的刷新 Token
	s.cacheRefreshToken(ctx, user.ID, newRefreshToken)

	s.logger.Info("Token 刷新成功",
		zap.Int64("user_id", user.ID))

	return &LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.JWT.ExpireHours * 3600),
		User: &UserInfo{
			ID:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			Role:      string(user.Role),
			Status:    string(user.Status),
			Balance:   user.Balance,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}

// Logout 登出
func (s *userService) Logout(ctx context.Context, userID int64) error {
	// 删除缓存的刷新 Token
	key := s.getRefreshTokenKey(userID)
	if err := s.redis.Del(ctx, key).Err(); err != nil {
		s.logger.Warn("删除刷新令牌缓存失败", zap.Error(err))
	}

	s.logger.Info("用户登出成功", zap.Int64("user_id", userID))
	return nil
}

// GetProfile 获取资料（带缓存）
func (s *userService) GetProfile(ctx context.Context, userID int64) (*ent.User, error) {
	// 尝试从缓存获取
	cacheKey := s.cacheKey.User(userID)
	var cachedUser ent.User
	if err := s.cache.GetObject(ctx, cacheKey, &cachedUser); err == nil {
		s.logger.Debug("从缓存获取用户资料", zap.Int64("user_id", userID))
		return &cachedUser, nil
	}

	// 从数据库获取
	user, err := s.db.User.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户资料失败: %w", err)
	}

	// 写入缓存
	if err := s.cache.SetObject(ctx, cacheKey, user, cache.CommonCacheTTL.Medium); err != nil {
		s.logger.Warn("缓存用户资料失败", zap.Error(err))
	}

	return user, nil
}

// UpdateProfile 更新资料
func (s *userService) UpdateProfile(ctx context.Context, userID int64, data *UpdateProfileRequest) error {
	// 构建更新
	update := s.db.User.UpdateOneID(userID)

	if data.Name != "" {
		update = update.SetName(data.Name)
	}

	// 执行更新
	_, err := update.Save(ctx)
	if err != nil {
		return fmt.Errorf("更新资料失败: %w", err)
	}

	// 清除用户缓存
	cacheKey := s.cacheKey.User(userID)
	if err := s.cache.Delete(ctx, cacheKey); err != nil {
		s.logger.Warn("清除用户缓存失败", zap.Error(err))
	}

	s.logger.Info("用户资料更新成功",
		zap.Int64("user_id", userID))

	return nil
}

// ChangePassword 修改密码
func (s *userService) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	// 获取用户
	user, err := s.db.User.Get(ctx, userID)
	if err != nil {
		return fmt.Errorf("获取用户失败: %w", err)
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("旧密码错误")
	}

	// 验证新密码强度
	if !isValidPassword(newPassword) {
		return fmt.Errorf("密码长度至少 8 位，需包含字母和数字")
	}

	// 哈希新密码
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %w", err)
	}

	// 更新密码和 Token 版本
	_, err = s.db.User.UpdateOneID(userID).
		SetPasswordHash(string(passwordHash)).
		AddTokenVersion(1).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}

	// 删除缓存的刷新 Token，强制重新登录
	s.redis.Del(ctx, s.getRefreshTokenKey(userID))

	s.logger.Info("密码修改成功",
		zap.Int64("user_id", userID))

	return nil
}

// GetByID 根据 ID 获取用户（带缓存）
func (s *userService) GetByID(ctx context.Context, userID int64) (*ent.User, error) {
	// 尝试从缓存获取
	cacheKey := s.cacheKey.User(userID)
	var cachedUser ent.User
	if err := s.cache.GetObject(ctx, cacheKey, &cachedUser); err == nil {
		return &cachedUser, nil
	}

	// 从数据库获取
	user, err := s.db.User.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	if err := s.cache.SetObject(ctx, cacheKey, user, cache.CommonCacheTTL.Medium); err != nil {
		s.logger.Warn("缓存用户信息失败", zap.Error(err))
	}

	return user, nil
}

// GetByEmail 根据邮箱获取用户（带缓存）
func (s *userService) GetByEmail(ctx context.Context, email string) (*ent.User, error) {
	// 尝试从缓存获取
	cacheKey := s.cacheKey.UserByEmail(email)
	var cachedUser ent.User
	if err := s.cache.GetObject(ctx, cacheKey, &cachedUser); err == nil {
		return &cachedUser, nil
	}

	// 从数据库获取
	user, err := s.db.User.Query().
		Where(ent.UserEmail(email)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	if err := s.cache.SetObject(ctx, cacheKey, user, cache.CommonCacheTTL.Medium); err != nil {
		s.logger.Warn("缓存用户邮箱映射失败", zap.Error(err))
	}

	return user, nil
}

// UpdateBalance 更新余额
func (s *userService) UpdateBalance(ctx context.Context, userID int64, amount float64) error {
	_, err := s.db.User.UpdateOneID(userID).
		AddBalance(amount).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("更新余额失败: %w", err)
	}

	// 清除用户缓存和余额缓存
	cacheKey := s.cacheKey.User(userID)
	balanceKey := s.cacheKey.Balance(userID)
	if err := s.cache.Delete(ctx, cacheKey, balanceKey); err != nil {
		s.logger.Warn("清除用户缓存失败", zap.Error(err))
	}

	s.logger.Info("余额更新成功",
		zap.Int64("user_id", userID),
		zap.Float64("amount", amount))

	return nil
}

// ListUsers 获取用户列表
func (s *userService) ListUsers(ctx context.Context, page, pageSize int) ([]*ent.User, int, error) {
	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询总数
	total, err := s.db.User.Query().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询用户总数失败: %w", err)
	}

	// 查询用户列表
	users, err := s.db.User.Query().
		Offset(offset).
		Limit(pageSize).
		Order(ent.Desc(ent.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询用户列表失败: %w", err)
	}

	return users, total, nil
}

// UpdateStatus 更新用户状态
func (s *userService) UpdateStatus(ctx context.Context, userID int64, status string) error {
	// 验证状态值
	validStatuses := map[string]bool{
		"active":    true,
		"suspended": true,
		"deleted":   true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("无效的用户状态: %s", status)
	}

	_, err := s.db.User.UpdateOneID(userID).
		SetStatus(ent.UserStatus(status)).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("更新用户状态失败: %w", err)
	}

	s.logger.Info("用户状态更新成功",
		zap.Int64("user_id", userID),
		zap.String("status", status))

	return nil
}

// DeleteUser 删除用户
func (s *userService) DeleteUser(ctx context.Context, userID int64) error {
	// 软删除：更新状态为 deleted
	_, err := s.db.User.UpdateOneID(userID).
		SetStatus(ent.UserStatusDeleted).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}

	// 删除相关缓存
	s.redis.Del(ctx, s.getRefreshTokenKey(userID))
	s.redis.Del(ctx, fmt.Sprintf("user:balance:%d", userID))

	s.logger.Info("用户删除成功",
		zap.Int64("user_id", userID))

	return nil
}

// generateAccessToken 生成访问令牌
func (s *userService) generateAccessToken(user *ent.User) (string, error) {
	now := time.Now()
	claims := &JWTClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         string(user.Role),
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.cfg.JWT.Issuer,
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.cfg.JWT.ExpireHours) * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}

// generateRefreshToken 生成刷新令牌
func (s *userService) generateRefreshToken(user *ent.User) (string, error) {
	now := time.Now()
	claims := &JWTClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         string(user.Role),
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.cfg.JWT.Issuer,
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.cfg.JWT.RefreshExpireHours) * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}

// parseToken 解析 Token
func (s *userService) parseToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("无效的签名方法")
		}
		return []byte(s.cfg.JWT.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("无效的令牌")
}

// getRefreshTokenKey 获取刷新 Token 的缓存键
func (s *userService) getRefreshTokenKey(userID int64) string {
	return fmt.Sprintf("user:refresh_token:%d", userID)
}

// cacheRefreshToken 缓存刷新 Token
func (s *userService) cacheRefreshToken(ctx context.Context, userID int64, refreshToken string) {
	key := s.getRefreshTokenKey(userID)
	expiration := time.Duration(s.cfg.JWT.RefreshExpireHours) * time.Hour
	s.redis.Set(ctx, key, refreshToken, expiration)
}

// ValidateToken 验证访问令牌
func (s *userService) ValidateToken(tokenString string) (*JWTClaims, error) {
	claims, err := s.parseToken(tokenString)
	if err != nil {
		return nil, err
	}

	// 检查令牌是否过期
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("令牌已过期")
	}

	return claims, nil
}

// ValidateTokenWithVersion 验证令牌并检查版本
func (s *userService) ValidateTokenWithVersion(ctx context.Context, tokenString string) (*JWTClaims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// 获取用户检查 Token 版本
	user, err := s.db.User.Get(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在")
	}

	if user.TokenVersion != claims.TokenVersion {
		return nil, fmt.Errorf("令牌已失效")
	}

	if user.Status != ent.UserStatusActive {
		return nil, fmt.Errorf("账号已被禁用")
	}

	return claims, nil
}

// GenerateRandomPassword 生成随机密码
func GenerateRandomPassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

// isValidEmail 验证邮箱格式
func isValidEmail(email string) bool {
	if len(email) < 5 || len(email) > 255 {
		return false
	}
	// 简单验证：包含 @ 且 @ 不在首尾
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			if atIndex != -1 {
				return false // 多个 @
			}
			atIndex = i
		}
	}
	return atIndex > 0 && atIndex < len(email)-1
}

// isValidPassword 验证密码强度
func isValidPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasLetter := false
	hasDigit := false

	for _, c := range password {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			hasLetter = true
		}
		if c >= '0' && c <= '9' {
			hasDigit = true
		}
	}

	return hasLetter && hasDigit
}

// IsAdmin 检查用户是否是管理员
func (s *userService) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	user, err := s.db.User.Get(ctx, userID)
	if err != nil {
		return false, err
	}
	return user.Role == ent.UserRoleAdmin, nil
}

// SetAdmin 设置用户为管理员
func (s *userService) SetAdmin(ctx context.Context, userID int64, isAdmin bool) error {
	role := ent.UserRoleUser
	if isAdmin {
		role = ent.UserRoleAdmin
	}

	_, err := s.db.User.UpdateOneID(userID).
		SetRole(role).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("设置管理员失败: %w", err)
	}

	s.logger.Info("用户角色更新成功",
		zap.Int64("user_id", userID),
		zap.String("role", string(role)))

	return nil
}

// GetActiveUserCount 获取活跃用户数量
func (s *userService) GetActiveUserCount(ctx context.Context) (int, error) {
	return s.db.User.Query().
		Where(
			ent.UserStatusEQ(ent.UserStatusActive),
			ent.UserLastActiveAtGTE(time.Now().AddDate(0, 0, -30)),
		).
		Count(ctx)
}

// errors 变量
var (
	ErrUserNotFound      = errors.New("用户不存在")
	ErrInvalidPassword   = errors.New("密码错误")
	ErrUserSuspended     = errors.New("账号已被暂停")
	ErrEmailExists       = errors.New("邮箱已被注册")
)
