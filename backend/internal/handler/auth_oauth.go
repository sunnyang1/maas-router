// Package handler OAuth认证处理器
// 支持 GitHub、Google、微信等第三方登录
package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"maas-router/ent"
	"maas-router/internal/config"
	"maas-router/internal/pkg/ctxkey"
)

// OAuthHandler OAuth认证处理器
type OAuthHandler struct {
	UserService  UserService
	AuthService  AuthService
	Config       *config.Config
	Logger       *zap.Logger
	DB           *ent.Client
}

// NewOAuthHandler 创建OAuth处理器
func NewOAuthHandler(
	userService UserService,
	authService AuthService,
	cfg *config.Config,
	logger *zap.Logger,
	db *ent.Client,
) *OAuthHandler {
	return &OAuthHandler{
		UserService: userService,
		AuthService: authService,
		Config:      cfg,
		Logger:      logger,
		DB:          db,
	}
}

// OAuthConfig OAuth配置
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	Scopes       []string
}

// OAuthUserInfo OAuth用户信息
type OAuthUserInfo struct {
	Provider   string
	ProviderID string
	Email      string
	Name       string
	AvatarURL  string
	RawData    map[string]interface{}
}

// ============ GitHub OAuth ============

// GitHubAuth GitHub登录入口
// GET /api/v1/auth/github
func (h *OAuthHandler) GitHubAuth(c *gin.Context) {
	state := generateOAuthState()
	// 存储state到cookie，用于回调验证
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)

	cfg := h.getGitHubConfig()
	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		cfg.AuthURL,
		cfg.ClientID,
		url.QueryEscape(cfg.RedirectURL),
		url.QueryEscape("user:email"),
		state,
	)
	c.Redirect(http.StatusFound, authURL)
}

// GitHubCallback GitHub登录回调
// GET /api/v1/auth/github/callback
func (h *OAuthHandler) GitHubCallback(c *gin.Context) {
	// 验证state
	state := c.Query("state")
	cookieState, _ := c.Cookie("oauth_state")
	if state == "" || state != cookieState {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_STATE", "无效的state参数")
		return
	}
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	code := c.Query("code")
	if code == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_CODE", "缺少授权码")
		return
	}

	// 交换access token
	cfg := h.getGitHubConfig()
	token, err := h.exchangeGitHubToken(code, cfg)
	if err != nil {
		h.Logger.Error("GitHub token交换失败", zap.Error(err))
		ErrorResponse(c, http.StatusInternalServerError, "TOKEN_EXCHANGE_FAILED", "获取访问令牌失败")
		return
	}

	// 获取用户信息
	userInfo, err := h.getGitHubUserInfo(token)
	if err != nil {
		h.Logger.Error("GitHub用户信息获取失败", zap.Error(err))
		ErrorResponse(c, http.StatusInternalServerError, "USER_INFO_FAILED", "获取用户信息失败")
		return
	}

	// 处理登录/注册
	h.handleOAuthLogin(c, userInfo)
}

func (h *OAuthHandler) getGitHubConfig() *OAuthConfig {
	return &OAuthConfig{
		ClientID:     h.Config.OAuth.GitHub.ClientID,
		ClientSecret: h.Config.OAuth.GitHub.ClientSecret,
		RedirectURL:  h.Config.OAuth.GitHub.RedirectURL,
		AuthURL:      "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		UserInfoURL:  "https://api.github.com/user",
	}
}

func (h *OAuthHandler) exchangeGitHubToken(code string, cfg *OAuthConfig) (string, error) {
	data := url.Values{}
	data.Set("client_id", cfg.ClientID)
	data.Set("client_secret", cfg.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", cfg.RedirectURL)

	req, err := http.NewRequest("POST", cfg.TokenURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.URL.RawQuery = data.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("GitHub OAuth error: %s", result.Error)
	}
	return result.AccessToken, nil
}

func (h *OAuthHandler) getGitHubUserInfo(token string) (*OAuthUserInfo, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var user struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	// 如果email为空，获取primary email
	email := user.Email
	if email == "" {
		email = h.getGitHubPrimaryEmail(token)
	}

	var rawData map[string]interface{}
	json.Unmarshal(body, &rawData)

	return &OAuthUserInfo{
		Provider:   "github",
		ProviderID: fmt.Sprintf("%d", user.ID),
		Email:      email,
		Name:       user.Name,
		AvatarURL:  user.AvatarURL,
		RawData:    rawData,
	}, nil
}

func (h *OAuthHandler) getGitHubPrimaryEmail(token string) string {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return ""
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	if len(emails) > 0 {
		return emails[0].Email
	}
	return ""
}

// ============ Google OAuth ============

// GoogleAuth Google登录入口
// GET /api/v1/auth/google
func (h *OAuthHandler) GoogleAuth(c *gin.Context) {
	state := generateOAuthState()
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)

	cfg := h.getGoogleConfig()
	scope := url.QueryEscape("openid email profile")
	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		cfg.AuthURL,
		cfg.ClientID,
		url.QueryEscape(cfg.RedirectURL),
		scope,
		state,
	)
	c.Redirect(http.StatusFound, authURL)
}

// GoogleCallback Google登录回调
// GET /api/v1/auth/google/callback
func (h *OAuthHandler) GoogleCallback(c *gin.Context) {
	state := c.Query("state")
	cookieState, _ := c.Cookie("oauth_state")
	if state == "" || state != cookieState {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_STATE", "无效的state参数")
		return
	}
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	code := c.Query("code")
	if code == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_CODE", "缺少授权码")
		return
	}

	cfg := h.getGoogleConfig()
	token, err := h.exchangeGoogleToken(code, cfg)
	if err != nil {
		h.Logger.Error("Google token交换失败", zap.Error(err))
		ErrorResponse(c, http.StatusInternalServerError, "TOKEN_EXCHANGE_FAILED", "获取访问令牌失败")
		return
	}

	userInfo, err := h.getGoogleUserInfo(token)
	if err != nil {
		h.Logger.Error("Google用户信息获取失败", zap.Error(err))
		ErrorResponse(c, http.StatusInternalServerError, "USER_INFO_FAILED", "获取用户信息失败")
		return
	}

	h.handleOAuthLogin(c, userInfo)
}

func (h *OAuthHandler) getGoogleConfig() *OAuthConfig {
	return &OAuthConfig{
		ClientID:     h.Config.OAuth.Google.ClientID,
		ClientSecret: h.Config.OAuth.Google.ClientSecret,
		RedirectURL:  h.Config.OAuth.Google.RedirectURL,
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
	}
}

func (h *OAuthHandler) exchangeGoogleToken(code string, cfg *OAuthConfig) (string, error) {
	data := url.Values{}
	data.Set("client_id", cfg.ClientID)
	data.Set("client_secret", cfg.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", cfg.RedirectURL)
	data.Set("grant_type", "authorization_code")

	resp, err := http.PostForm(cfg.TokenURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("Google OAuth error: %s", result.Error)
	}
	return result.AccessToken, nil
}

func (h *OAuthHandler) getGoogleUserInfo(token string) (*OAuthUserInfo, error) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var user struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	var rawData map[string]interface{}
	json.Unmarshal(body, &rawData)

	return &OAuthUserInfo{
		Provider:   "google",
		ProviderID: user.ID,
		Email:      user.Email,
		Name:       user.Name,
		AvatarURL:  user.Picture,
		RawData:    rawData,
	}, nil
}

// ============ 微信 OAuth ============

// WeChatAuth 微信登录入口
// GET /api/v1/auth/wechat
func (h *OAuthHandler) WeChatAuth(c *gin.Context) {
	state := generateOAuthState()
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)

	cfg := h.getWeChatConfig()
	authURL := fmt.Sprintf("https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s#wechat_redirect",
		cfg.ClientID,
		url.QueryEscape(cfg.RedirectURL),
		state,
	)
	c.Redirect(http.StatusFound, authURL)
}

// WeChatCallback 微信登录回调
// GET /api/v1/auth/wechat/callback
func (h *OAuthHandler) WeChatCallback(c *gin.Context) {
	state := c.Query("state")
	cookieState, _ := c.Cookie("oauth_state")
	if state == "" || state != cookieState {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_STATE", "无效的state参数")
		return
	}
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	code := c.Query("code")
	if code == "" {
		ErrorResponse(c, http.StatusBadRequest, "MISSING_CODE", "缺少授权码")
		return
	}

	cfg := h.getWeChatConfig()
	accessToken, openID, err := h.exchangeWeChatToken(code, cfg)
	if err != nil {
		h.Logger.Error("微信token交换失败", zap.Error(err))
		ErrorResponse(c, http.StatusInternalServerError, "TOKEN_EXCHANGE_FAILED", "获取访问令牌失败")
		return
	}

	userInfo, err := h.getWeChatUserInfo(accessToken, openID)
	if err != nil {
		h.Logger.Error("微信用户信息获取失败", zap.Error(err))
		ErrorResponse(c, http.StatusInternalServerError, "USER_INFO_FAILED", "获取用户信息失败")
		return
	}

	h.handleOAuthLogin(c, userInfo)
}

func (h *OAuthHandler) getWeChatConfig() *OAuthConfig {
	return &OAuthConfig{
		ClientID:     h.Config.OAuth.WeChat.AppID,
		ClientSecret: h.Config.OAuth.WeChat.AppSecret,
		RedirectURL:  h.Config.OAuth.WeChat.RedirectURL,
	}
}

func (h *OAuthHandler) exchangeWeChatToken(code string, cfg *OAuthConfig) (string, string, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		cfg.ClientID, cfg.ClientSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		OpenID      string `json:"openid"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}
	if result.ErrCode != 0 {
		return "", "", fmt.Errorf("WeChat OAuth error: %s", result.ErrMsg)
	}
	return result.AccessToken, result.OpenID, nil
}

func (h *OAuthHandler) getWeChatUserInfo(accessToken, openID string) (*OAuthUserInfo, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s",
		accessToken, openID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var user struct {
		OpenID     string   `json:"openid"`
		Nickname   string   `json:"nickname"`
		HeadImgURL string   `json:"headimgurl"`
		Privilege  []string `json:"privilege"`
		UnionID    string   `json:"unionid"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	var rawData map[string]interface{}
	json.Unmarshal(body, &rawData)

	// 微信不直接提供邮箱，使用unionid或openid生成一个占位邮箱
	email := ""
	if user.UnionID != "" {
		email = fmt.Sprintf("wechat_%s@wechat.user", user.UnionID)
	} else {
		email = fmt.Sprintf("wechat_%s@wechat.user", user.OpenID)
	}

	return &OAuthUserInfo{
		Provider:   "wechat",
		ProviderID: user.OpenID,
		Email:      email,
		Name:       user.Nickname,
		AvatarURL:  user.HeadImgURL,
		RawData:    rawData,
	}, nil
}

// ============ 通用处理逻辑 ============

// handleOAuthLogin 处理OAuth登录/注册
func (h *OAuthHandler) handleOAuthLogin(c *gin.Context, userInfo *OAuthUserInfo) {
	ctx := c.Request.Context()

	// 查找是否已存在该OAuth身份
	identity, err := h.DB.AuthIdentity.Query().
		Where(
			ent.AuthIdentityProviderEQ(ent.AuthIdentityProvider(userInfo.Provider)),
			ent.AuthIdentityProviderUserID(userInfo.ProviderID),
		).
		WithUser().
		Only(ctx)

	if err == nil {
		// 已存在，更新token信息并登录
		user := identity.Edges.User
		token, err := h.AuthService.GenerateToken(ctx, user.ID, user.Email, user.Role, user.TokenVersion)
		if err != nil {
			ErrorResponse(c, http.StatusInternalServerError, "TOKEN_GENERATION_FAILED", "生成Token失败")
			return
		}

		// 更新OAuth token信息（如果有）
		// ...

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
		return
	}

	if !ent.IsNotFound(err) {
		h.Logger.Error("查询OAuth身份失败", zap.Error(err))
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "系统错误")
		return
	}

	// 不存在，检查邮箱是否已注册
	var user *ent.User
	if userInfo.Email != "" && userInfo.Email != fmt.Sprintf("wechat_%s@wechat.user", userInfo.ProviderID) {
		existingUser, err := h.DB.User.Query().
			Where(ent.UserEmail(userInfo.Email)).
			Only(ctx)
		if err == nil {
			user = existingUser
		}
	}

	// 创建新用户
	if user == nil {
		// 生成随机密码
		randomPassword := generateRandomPassword(16)
		passwordHash, _ := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)

		// 生成邀请码
		inviteCode := generateInviteCode()

		newUser, err := h.DB.User.Create().
			SetEmail(userInfo.Email).
			SetPasswordHash(string(passwordHash)).
			SetName(userInfo.Name).
			SetInviteCode(inviteCode).
			SetRole(ent.UserRoleUser).
			SetStatus(ent.UserStatusActive).
			SetBalance(0).
			SetAffiliateBalance(0).
			SetTotalAffiliateEarnings(0).
			SetInviteCount(0).
			SetConcurrency(5).
			SetTokenVersion(1).
			Save(ctx)
		if err != nil {
			h.Logger.Error("创建用户失败", zap.Error(err))
			ErrorResponse(c, http.StatusInternalServerError, "CREATE_USER_FAILED", "创建用户失败")
			return
		}
		user = newUser
	}

	// 创建OAuth身份记录
	_, err = h.DB.AuthIdentity.Create().
		SetUserID(user.ID).
		SetProvider(ent.AuthIdentityProvider(userInfo.Provider)).
		SetProviderUserID(userInfo.ProviderID).
		SetEmail(userInfo.Email).
		SetName(userInfo.Name).
		SetAvatarURL(userInfo.AvatarURL).
		SetRawData(userInfo.RawData).
		Save(ctx)
	if err != nil {
		h.Logger.Error("创建OAuth身份失败", zap.Error(err))
		// 继续登录流程，不阻断
	}

	// 生成Token
	token, err := h.AuthService.GenerateToken(ctx, user.ID, user.Email, user.Role, user.TokenVersion)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "TOKEN_GENERATION_FAILED", "生成Token失败")
		return
	}

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

// ============ OAuth绑定/解绑 ============

// BindOAuthRequest 绑定OAuth请求
type BindOAuthRequest struct {
	Provider string `json:"provider" binding:"required,oneof=github google wechat"`
	Code     string `json:"code" binding:"required"`
}

// BindOAuth 绑定OAuth账号
// POST /api/v1/user/oauth/bind
func (h *OAuthHandler) BindOAuth(c *gin.Context) {
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权")
		return
	}

	var req BindOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误")
		return
	}

	ctx := c.Request.Context()
	uid := userID.(int64)

	// 检查是否已绑定
	exists_binding, err := h.DB.AuthIdentity.Query().
		Where(
			ent.AuthIdentityUserID(uid),
			ent.AuthIdentityProviderEQ(ent.AuthIdentityProvider(req.Provider)),
		).
		Exist(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "系统错误")
		return
	}
	if exists_binding {
		ErrorResponse(c, http.StatusConflict, "ALREADY_BOUND", "该账号类型已绑定")
		return
	}

	// 获取用户信息（简化处理，实际需要实现各平台的token交换）
	// 这里返回成功提示，实际实现需要根据具体流程调整
	c.JSON(http.StatusOK, gin.H{
		"message": "请使用对应的OAuth登录接口完成绑定",
	})
}

// UnbindOAuth 解绑OAuth账号
// DELETE /api/v1/user/oauth/:provider
func (h *OAuthHandler) UnbindOAuth(c *gin.Context) {
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权")
		return
	}

	provider := c.Param("provider")
	if provider != "github" && provider != "google" && provider != "wechat" {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_PROVIDER", "无效的提供商")
		return
	}

	ctx := c.Request.Context()
	uid := userID.(int64)

	// 删除绑定记录
	_, err := h.DB.AuthIdentity.Delete().
		Where(
			ent.AuthIdentityUserID(uid),
			ent.AuthIdentityProviderEQ(ent.AuthIdentityProvider(provider)),
		).
		Exec(ctx)
	if err != nil {
		h.Logger.Error("解绑OAuth失败", zap.Error(err))
		ErrorResponse(c, http.StatusInternalServerError, "UNBIND_FAILED", "解绑失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "解绑成功",
	})
}

// GetOAuthBindings 获取用户的OAuth绑定列表
// GET /api/v1/user/oauth
func (h *OAuthHandler) GetOAuthBindings(c *gin.Context) {
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权")
		return
	}

	ctx := c.Request.Context()
	uid := userID.(int64)

	identities, err := h.DB.AuthIdentity.Query().
		Where(ent.AuthIdentityUserID(uid)).
		All(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询失败")
		return
	}

	bindings := make([]gin.H, 0, len(identities))
	for _, id := range identities {
		bindings = append(bindings, gin.H{
			"provider":    id.Provider,
			"name":        id.Name,
			"email":       id.Email,
			"avatar_url":  id.AvatarURL,
			"created_at":  id.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"bindings": bindings,
	})
}

// ============ 辅助函数 ============

// generateOAuthState 生成随机的OAuth state参数
func generateOAuthState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// generateRandomPassword 生成随机密码
func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

// generateInviteCode 生成邀请码
func generateInviteCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}
