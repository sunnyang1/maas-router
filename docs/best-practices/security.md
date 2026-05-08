# MaaS-Router 安全最佳实践

本文档介绍 MaaS-Router 项目的安全最佳实践，包括 API Key 管理、JWT 安全、限流策略、审计日志和数据加密。

## 目录

- [API Key 管理](#api-key-管理)
- [JWT 安全](#jwt-安全)
- [限流策略](#限流策略)
- [审计日志](#审计日志)
- [数据加密](#数据加密)

## API Key 管理

### 1. API Key 生成策略

```go
// security/apikey.go
package security

import (
    "crypto/rand"
    "encoding/base64"
    "encoding/hex"
    "strings"
)

const (
    APIKeyPrefix = "maas_"
    APIKeyLength = 48 // 32字节随机数据 = 64字符base64
)

// GenerateAPIKey 生成安全的 API Key
func GenerateAPIKey() (string, string, error) {
    // 生成随机字节
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", "", err
    }
    
    // 生成密钥ID（前8字节）
    keyID := hex.EncodeToString(bytes[:8])
    
    // 生成密钥（后24字节）
    secret := base64.RawURLEncoding.EncodeToString(bytes[8:])
    
    // 组合完整 API Key
    fullKey := APIKeyPrefix + keyID + "_" + secret
    
    // 返回完整密钥和密钥ID（用于数据库索引）
    return fullKey, keyID, nil
}

// HashAPIKey 对 API Key 进行哈希存储
func HashAPIKey(apiKey string) string {
    // 使用 bcrypt 或 Argon2 进行哈希
    hash, _ := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
    return string(hash)
}

// VerifyAPIKey 验证 API Key
func VerifyAPIKey(apiKey, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(apiKey))
    return err == nil
}
```

### 2. API Key 存储

```sql
-- 用户表结构
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    api_key_id VARCHAR(16) UNIQUE NOT NULL,  -- 明文存储，用于索引
    api_key_hash VARCHAR(255) NOT NULL,       -- bcrypt 哈希
    api_key_prefix VARCHAR(8) NOT NULL,       -- 前缀用于日志显示
    api_key_created_at TIMESTAMP NOT NULL,
    api_key_last_used_at TIMESTAMP,
    api_key_expires_at TIMESTAMP,
    rate_limit_per_minute INT DEFAULT 1000,
    rate_limit_per_hour INT DEFAULT 10000,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 创建索引
CREATE INDEX idx_users_api_key_id ON users(api_key_id);
CREATE INDEX idx_users_email ON users(email);

-- API Key 使用日志
CREATE TABLE api_key_usage_logs (
    id BIGSERIAL PRIMARY KEY,
    api_key_id VARCHAR(16) NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    ip_address INET NOT NULL,
    user_agent TEXT,
    request_id VARCHAR(64),
    status_code INT,
    latency_ms INT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 分区
CREATE INDEX idx_api_key_usage_logs_key_created 
ON api_key_usage_logs(api_key_id, created_at);
```

### 3. API Key 中间件

```go
// middleware/apikey.go
package middleware

func APIKeyAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 获取 API Key
        apiKey := c.GetHeader("X-API-Key")
        if apiKey == "" {
            apiKey = c.Query("api_key")
        }
        
        if apiKey == "" {
            c.JSON(401, gin.H{"error": "API key required"})
            c.Abort()
            return
        }
        
        // 2. 解析 API Key
        keyID, err := extractKeyID(apiKey)
        if err != nil {
            c.JSON(401, gin.H{"error": "Invalid API key format"})
            c.Abort()
            return
        }
        
        // 3. 查询数据库验证
        user, err := userRepo.GetByAPIKeyID(c, keyID)
        if err != nil {
            c.JSON(401, gin.H{"error": "Invalid API key"})
            c.Abort()
            return
        }
        
        // 4. 验证哈希
        if !security.VerifyAPIKey(apiKey, user.APIKeyHash) {
            c.JSON(401, gin.H{"error": "Invalid API key"})
            c.Abort()
            return
        }
        
        // 5. 检查是否过期
        if user.APIKeyExpiresAt != nil && user.APIKeyExpiresAt.Before(time.Now()) {
            c.JSON(401, gin.H{"error": "API key expired"})
            c.Abort()
            return
        }
        
        // 6. 检查是否激活
        if !user.IsActive {
            c.JSON(401, gin.H{"error": "API key deactivated"})
            c.Abort()
            return
        }
        
        // 7. 记录使用日志（异步）
        go logAPIKeyUsage(user.ID, c.Request)
        
        // 8. 更新最后使用时间
        go userRepo.UpdateLastUsedAt(c, user.ID)
        
        // 9. 设置用户上下文
        c.Set("user_id", user.ID)
        c.Set("api_key_id", keyID)
        c.Set("rate_limit", user.RateLimitPerMinute)
        
        c.Next()
    }
}
```

### 4. API Key 轮换

```go
// 自动轮换策略
type APIKeyRotation struct {
    db     *sql.DB
    redis  *redis.Client
    logger *zap.Logger
}

func (r *APIKeyRotation) RotateAPIKey(ctx context.Context, userID int64) (*APIKey, error) {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()
    
    // 1. 生成新密钥
    newKey, keyID, err := security.GenerateAPIKey()
    if err != nil {
        return nil, err
    }
    
    // 2. 获取旧密钥信息
    var oldKeyID string
    err = tx.QueryRowContext(ctx, 
        "SELECT api_key_id FROM users WHERE id = $1", userID).Scan(&oldKeyID)
    if err != nil {
        return nil, err
    }
    
    // 3. 将旧密钥加入宽限期列表（30天）
    _, err = tx.ExecContext(ctx, `
        INSERT INTO api_key_grace_period (user_id, api_key_id, expires_at)
        VALUES ($1, $2, NOW() + INTERVAL '30 days')`,
        userID, oldKeyID)
    if err != nil {
        return nil, err
    }
    
    // 4. 更新用户新密钥
    _, err = tx.ExecContext(ctx, `
        UPDATE users 
        SET api_key_id = $1, 
            api_key_hash = $2,
            api_key_prefix = $3,
            api_key_created_at = NOW(),
            api_key_expires_at = NULL
        WHERE id = $4`,
        keyID, security.HashAPIKey(newKey), newKey[:8], userID)
    if err != nil {
        return nil, err
    }
    
    // 5. 提交事务
    if err := tx.Commit(); err != nil {
        return nil, err
    }
    
    // 6. 清除缓存
    r.redis.Del(ctx, fmt.Sprintf("user:api_key:%s", oldKeyID))
    
    return &APIKey{
        Key:       newKey,
        KeyID:     keyID,
        CreatedAt: time.Now(),
    }, nil
}

// 宽限期验证
func (r *APIKeyRotation) ValidateWithGracePeriod(ctx context.Context, apiKey string) (*User, error) {
    // 1. 尝试正常验证
    user, err := r.validateAPIKey(ctx, apiKey)
    if err == nil {
        return user, nil
    }
    
    // 2. 尝试宽限期验证
    keyID, _ := extractKeyID(apiKey)
    var graceUserID int64
    err = r.db.QueryRowContext(ctx, `
        SELECT user_id FROM api_key_grace_period 
        WHERE api_key_id = $1 AND expires_at > NOW()`,
        keyID).Scan(&graceUserID)
    
    if err != nil {
        return nil, ErrInvalidAPIKey
    }
    
    // 3. 获取用户信息
    return r.userRepo.GetByID(ctx, graceUserID)
}
```

## JWT 安全

### 1. JWT 配置

```yaml
# config.yaml
jwt:
  # 密钥配置
  secret: "${JWT_SECRET}"           # 从环境变量读取
  access_token_secret: "${JWT_ACCESS_SECRET}"
  refresh_token_secret: "${JWT_REFRESH_SECRET}"
  
  # 过期时间
  access_token_expire: "15m"        # Access Token 15分钟
  refresh_token_expire: "7d"        # Refresh Token 7天
  
  # 算法
  algorithm: "HS256"                # 或 RS256
  
  # 签发者
  issuer: "maas-router"
  audience: "maas-router-api"
  
  # 安全选项
  allow_multiple_logins: false      # 是否允许多设备登录
  token_blacklist_enabled: true     # 启用 Token 黑名单
  binding_ip: false                 # 是否绑定 IP
  binding_user_agent: false         # 是否绑定 User-Agent
```

### 2. JWT 实现

```go
// security/jwt.go
package security

import (
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

type JWTClaims struct {
    jwt.RegisteredClaims
    UserID    int64    `json:"user_id"`
    Email     string   `json:"email"`
    Role      string   `json:"role"`
    TokenID   string   `json:"jti"`      // 唯一 Token ID
    DeviceID  string   `json:"device_id"`
    IP        string   `json:"ip,omitempty"`
    UserAgent string   `json:"ua,omitempty"`
}

type JWTManager struct {
    accessSecret  []byte
    refreshSecret []byte
    issuer        string
    audience      string
    redis         *redis.Client
}

func (m *JWTManager) GenerateTokenPair(user *User, deviceID, ip, userAgent string) (*TokenPair, error) {
    tokenID := uuid.New().String()
    now := time.Now()
    
    // Access Token
    accessClaims := JWTClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            ID:        tokenID,
            Issuer:    m.issuer,
            Audience:  jwt.ClaimStrings{m.audience},
            Subject:   strconv.FormatInt(user.ID, 10),
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
        },
        UserID:    user.ID,
        Email:     user.Email,
        Role:      user.Role,
        TokenID:   tokenID,
        DeviceID:  deviceID,
        IP:        ip,
        UserAgent: userAgent,
    }
    
    accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
    accessString, err := accessToken.SignedString(m.accessSecret)
    if err != nil {
        return nil, err
    }
    
    // Refresh Token
    refreshClaims := jwt.RegisteredClaims{
        ID:        tokenID,
        Issuer:    m.issuer,
        Subject:   strconv.FormatInt(user.ID, 10),
        IssuedAt:  jwt.NewNumericDate(now),
        ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
    }
    
    refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
    refreshString, err := refreshToken.SignedString(m.refreshSecret)
    if err != nil {
        return nil, err
    }
    
    // 存储 Token 元数据到 Redis
    ctx := context.Background()
    m.redis.HSet(ctx, fmt.Sprintf("token:%s", tokenID), map[string]interface{}{
        "user_id":    user.ID,
        "device_id":  deviceID,
        "ip":         ip,
        "created_at": now.Unix(),
    })
    m.redis.Expire(ctx, fmt.Sprintf("token:%s", tokenID), 7*24*time.Hour)
    
    return &TokenPair{
        AccessToken:  accessString,
        RefreshToken: refreshString,
        TokenID:      tokenID,
        ExpiresIn:    900, // 15分钟
    }, nil
}

func (m *JWTManager) ValidateAccessToken(tokenString string, clientIP, userAgent string) (*JWTClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return m.accessSecret, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
        // 检查 Token 是否在黑名单
        ctx := context.Background()
        exists, _ := m.redis.Exists(ctx, fmt.Sprintf("token:blacklist:%s", claims.TokenID)).Result()
        if exists > 0 {
            return nil, ErrTokenBlacklisted
        }
        
        // 验证绑定信息
        if claims.IP != "" && claims.IP != clientIP {
            return nil, ErrTokenIPMismatch
        }
        
        return claims, nil
    }
    
    return nil, ErrInvalidToken
}

func (m *JWTManager) RevokeToken(tokenID string) error {
    ctx := context.Background()
    return m.redis.Set(ctx, fmt.Sprintf("token:blacklist:%s", tokenID), "1", 7*24*time.Hour).Err()
}

func (m *JWTManager) RevokeAllUserTokens(userID int64) error {
    // 实现撤销用户所有 Token 的逻辑
    ctx := context.Background()
    pattern := fmt.Sprintf("user:tokens:%d:*", userID)
    
    iter := m.redis.Scan(ctx, 0, pattern, 0).Iterator()
    for iter.Next(ctx) {
        tokenID := iter.Val()
        m.RevokeToken(tokenID)
    }
    
    return iter.Err()
}
```

### 3. JWT 中间件

```go
// middleware/jwt.go
func JWTAuth(jwtManager *security.JWTManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 获取 Token
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }
        
        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
            c.JSON(401, gin.H{"error": "Invalid authorization header format"})
            c.Abort()
            return
        }
        
        tokenString := parts[1]
        
        // 2. 验证 Token
        claims, err := jwtManager.ValidateAccessToken(
            tokenString,
            c.ClientIP(),
            c.Request.UserAgent(),
        )
        
        if err != nil {
            switch err {
            case security.ErrTokenExpired:
                c.JSON(401, gin.H{"error": "Token expired", "code": "TOKEN_EXPIRED"})
            case security.ErrTokenBlacklisted:
                c.JSON(401, gin.H{"error": "Token revoked", "code": "TOKEN_REVOKED"})
            default:
                c.JSON(401, gin.H{"error": "Invalid token"})
            }
            c.Abort()
            return
        }
        
        // 3. 设置上下文
        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)
        c.Set("role", claims.Role)
        c.Set("token_id", claims.TokenID)
        
        c.Next()
    }
}
```

## 限流策略

### 1. 限流配置

```yaml
ratelimit:
  # 全局配置
  enabled: true
  strategy: "token_bucket"  # token_bucket, sliding_window, fixed_window
  
  # 默认限制
  default:
    requests_per_second: 100
    burst: 150
  
  # 按用户类型
  by_tier:
    free:
      requests_per_minute: 60
      requests_per_hour: 1000
      requests_per_day: 10000
    pro:
      requests_per_minute: 600
      requests_per_hour: 10000
      requests_per_day: 100000
    enterprise:
      requests_per_minute: 6000
      requests_per_hour: 100000
      requests_per_day: 1000000
  
  # 按端点
  by_endpoint:
    "/v1/chat/completions":
      requests_per_minute: 100
      requests_per_hour: 1000
    "/v1/models":
      requests_per_minute: 1000
  
  # 按 IP
  by_ip:
    requests_per_minute: 30
    block_duration: "1h"
```

### 2. 限流实现

```go
// ratelimit/ratelimiter.go
package ratelimit

import (
    "context"
    "fmt"
    "time"
    
    "github.com/redis/go-redis/v9"
)

type RateLimiter struct {
    redis  *redis.Client
    config *Config
}

// Token Bucket 算法
func (r *RateLimiter) AllowTokenBucket(ctx context.Context, key string, rate, burst int) (bool, time.Duration, error) {
    luaScript := `
        local key = KEYS[1]
        local rate = tonumber(ARGV[1])
        local burst = tonumber(ARGV[2])
        local now = tonumber(ARGV[3])
        
        local bucket = redis.call('HMGET', key, 'tokens', 'last_update')
        local tokens = tonumber(bucket[1]) or burst
        local last_update = tonumber(bucket[2]) or now
        
        -- 计算新增令牌
        local delta = math.max(0, now - last_update)
        tokens = math.min(burst, tokens + delta * rate)
        
        if tokens >= 1 then
            tokens = tokens - 1
            redis.call('HMSET', key, 'tokens', tokens, 'last_update', now)
            redis.call('EXPIRE', key, 60)
            return {1, 0}
        else
            redis.call('HSET', key, 'last_update', now)
            local retry_after = math.ceil((1 - tokens) / rate)
            return {0, retry_after}
        end
    `
    
    now := time.Now().Unix()
    result, err := r.redis.Eval(ctx, luaScript, []string{key}, rate, burst, now).Result()
    if err != nil {
        return false, 0, err
    }
    
    values := result.([]interface{})
    allowed := values[0].(int64) == 1
    retryAfter := time.Duration(values[1].(int64)) * time.Second
    
    return allowed, retryAfter, nil
}

// Sliding Window 算法
func (r *RateLimiter) AllowSlidingWindow(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error) {
    now := time.Now().UnixNano() / 1e6 // 毫秒
    windowStart := now - int64(window.Milliseconds())
    
    pipe := r.redis.Pipeline()
    
    // 移除窗口外的请求记录
    pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
    
    // 获取当前窗口内的请求数
    countCmd := pipe.ZCard(ctx, key)
    
    // 添加当前请求
    pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
    
    // 设置过期时间
    pipe.Expire(ctx, key, window)
    
    _, err := pipe.Exec(ctx)
    if err != nil {
        return false, 0, err
    }
    
    count := countCmd.Val()
    
    if count < int64(limit) {
        return true, 0, nil
    }
    
    // 计算最早过期时间
    oldest, _ := r.redis.ZRange(ctx, key, 0, 0).Result()
    if len(oldest) > 0 {
        oldestTime := int64(0)
        fmt.Sscanf(oldest[0], "%d", &oldestTime)
        retryAfter := time.Duration(oldestTime+int64(window.Milliseconds())-now) * time.Millisecond
        return false, retryAfter, nil
    }
    
    return false, window, nil
}

// 中间件
func RateLimitMiddleware(limiter *ratelimit.RateLimiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 确定限流键
        var key string
        if userID, exists := c.Get("user_id"); exists {
            key = fmt.Sprintf("ratelimit:user:%d", userID)
        } else {
            key = fmt.Sprintf("ratelimit:ip:%s", c.ClientIP())
        }
        
        // 获取限流配置
        limit := 100 // 默认限制
        if tier, exists := c.Get("user_tier"); exists {
            switch tier {
            case "pro":
                limit = 600
            case "enterprise":
                limit = 6000
            }
        }
        
        // 检查限流
        allowed, retryAfter, err := limiter.AllowSlidingWindow(
            c.Request.Context(),
            key,
            limit,
            time.Minute,
        )
        
        if err != nil {
            c.JSON(500, gin.H{"error": "Rate limit check failed"})
            c.Abort()
            return
        }
        
        if !allowed {
            c.Header("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
            c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
            c.Header("X-RateLimit-Remaining", "0")
            c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(retryAfter).Unix()))
            c.JSON(429, gin.H{
                "error": "Rate limit exceeded",
                "retry_after": retryAfter.Seconds(),
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

## 审计日志

### 1. 审计日志配置

```yaml
audit:
  enabled: true
  level: "detailed"  # basic, detailed, full
  
  # 记录事件
  events:
    - "user.login"
    - "user.logout"
    - "user.api_key.created"
    - "user.api_key.rotated"
    - "user.api_key.revoked"
    - "request.inference"
    - "request.billing"
    - "admin.user.created"
    - "admin.user.deleted"
    - "admin.config.changed"
  
  # 存储配置
  storage:
    type: "database"  # database, file, elasticsearch
    retention_days: 365
    
  # 敏感字段脱敏
  sensitive_fields:
    - "password"
    - "api_key"
    - "token"
    - "credit_card"
    - "private_key"
```

### 2. 审计日志实现

```go
// audit/audit.go
package audit

import (
    "context"
    "encoding/json"
    "time"
    
    "github.com/google/uuid"
)

type AuditEvent struct {
    ID            string                 `json:"id"`
    Timestamp     time.Time              `json:"timestamp"`
    EventType     string                 `json:"event_type"`
    EventCategory string                 `json:"event_category"`
    Actor         Actor                  `json:"actor"`
    Target        Target                 `json:"target"`
    Action        string                 `json:"action"`
    Result        string                 `json:"result"`
    Details       map[string]interface{} `json:"details"`
    Context       Context                `json:"context"`
    Risk          Risk                   `json:"risk"`
}

type Actor struct {
    Type      string `json:"type"`       // user, system, api_key
    ID        string `json:"id"`
    Email     string `json:"email,omitempty"`
    IP        string `json:"ip"`
    UserAgent string `json:"user_agent"`
    APIKeyID  string `json:"api_key_id,omitempty"`
}

type Target struct {
    Type   string `json:"type"`
    ID     string `json:"id"`
    Name   string `json:"name,omitempty"`
}

type Context struct {
    RequestID   string `json:"request_id"`
    TraceID     string `json:"trace_id"`
    SessionID   string `json:"session_id,omitempty"`
}

type Risk struct {
    Level       string   `json:"level"`       // low, medium, high, critical
    Score       int      `json:"score"`       // 0-100
    Indicators  []string `json:"indicators"`
}

type Logger struct {
    storage Storage
    config  *Config
}

func (l *Logger) Log(ctx context.Context, eventType string, actor Actor, target Target, action string, result string, details map[string]interface{}) {
    event := AuditEvent{
        ID:            uuid.New().String(),
        Timestamp:     time.Now().UTC(),
        EventType:     eventType,
        EventCategory: getCategory(eventType),
        Actor:         actor,
        Target:        target,
        Action:        action,
        Result:        result,
        Details:       l.sanitize(details),
        Context: Context{
            RequestID: ctx.Value("request_id").(string),
            TraceID:   ctx.Value("trace_id").(string),
        },
        Risk: l.calculateRisk(eventType, actor, details),
    }
    
    // 异步写入
    go l.storage.Write(event)
    
    // 高风险事件实时告警
    if event.Risk.Level == "high" || event.Risk.Level == "critical" {
        go l.sendAlert(event)
    }
}

func (l *Logger) sanitize(details map[string]interface{}) map[string]interface{} {
    sanitized := make(map[string]interface{})
    for k, v := range details {
        if isSensitiveField(k) {
            sanitized[k] = "[REDACTED]"
        } else {
            sanitized[k] = v
        }
    }
    return sanitized
}

func (l *Logger) calculateRisk(eventType string, actor Actor, details map[string]interface{}) Risk {
    score := 0
    indicators := []string{}
    
    // 异常时间访问
    hour := time.Now().Hour()
    if hour < 6 || hour > 22 {
        score += 10
        indicators = append(indicators, "off_hours_access")
    }
    
    // 异地登录
    if eventType == "user.login" {
        if isUnusualLocation(actor.IP) {
            score += 20
            indicators = append(indicators, "unusual_location")
        }
    }
    
    // 大量请求
    if count, ok := details["request_count"].(int); ok && count > 10000 {
        score += 15
        indicators = append(indicators, "high_volume")
    }
    
    // 确定风险等级
    level := "low"
    if score >= 80 {
        level = "critical"
    } else if score >= 60 {
        level = "high"
    } else if score >= 30 {
        level = "medium"
    }
    
    return Risk{
        Level:      level,
        Score:      score,
        Indicators: indicators,
    }
}

// 中间件
func AuditMiddleware(logger *audit.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        // 只记录特定端点
        if !shouldAudit(c.Request.URL.Path) {
            return
        }
        
        actor := audit.Actor{
            Type:      "user",
            IP:        c.ClientIP(),
            UserAgent: c.Request.UserAgent(),
        }
        
        if userID, exists := c.Get("user_id"); exists {
            actor.ID = fmt.Sprintf("%d", userID)
        }
        if apiKeyID, exists := c.Get("api_key_id"); exists {
            actor.Type = "api_key"
            actor.APIKeyID = apiKeyID.(string)
        }
        
        target := audit.Target{
            Type: "endpoint",
            ID:   c.Request.URL.Path,
        }
        
        details := map[string]interface{}{
            "method":       c.Request.Method,
            "status_code":  c.Writer.Status(),
            "latency_ms":   time.Since(start).Milliseconds(),
            "request_size": c.Request.ContentLength,
        }
        
        result := "success"
        if c.Writer.Status() >= 400 {
            result = "failure"
        }
        
        logger.Log(c.Request.Context(), "request.api", actor, target, c.Request.Method, result, details)
    }
}
```

## 数据加密

### 1. 传输层加密 (TLS)

```yaml
server:
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/maas-router.crt"
    key_file: "/etc/ssl/private/maas-router.key"
    min_version: "1.2"
    cipher_suites:
      - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
      - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
      - "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"
    prefer_server_cipher_suites: true
    
  # HSTS
  hsts:
    enabled: true
    max_age: 31536000
    include_subdomains: true
    preload: true
```

### 2. 数据库字段加密

```go
// security/encryption.go
package security

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "io"
)

type FieldEncryptor struct {
    key []byte
}

func NewFieldEncryptor(key string) *FieldEncryptor {
    // 从环境变量或 KMS 获取密钥
    return &FieldEncryptor{
        key: []byte(key),
    }
}

func (e *FieldEncryptor) Encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *FieldEncryptor) Decrypt(ciphertext string) (string, error) {
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }
    
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return "", errors.New("ciphertext too short")
    }
    
    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }
    
    return string(plaintext), nil
}

// 使用示例
type User struct {
    ID            int64  `db:"id"`
    Email         string `db:"email"`
    APIKeyID      string `db:"api_key_id"`
    // 敏感字段加密存储
    PhoneEncrypted string `db:"phone_encrypted"`
    AddressEncrypted string `db:"address_encrypted"`
}

func (u *User) SetPhone(encryptor *FieldEncryptor, phone string) error {
    encrypted, err := encryptor.Encrypt(phone)
    if err != nil {
        return err
    }
    u.PhoneEncrypted = encrypted
    return nil
}

func (u *User) GetPhone(encryptor *FieldEncryptor) (string, error) {
    return encryptor.Decrypt(u.PhoneEncrypted)
}
```

### 3. KMS 集成

```go
// security/kms.go
package security

import (
    "github.com/aws/aws-sdk-go-v2/service/kms"
)

type KMSClient struct {
    client *kms.Client
    keyID  string
}

func (k *KMSClient) GenerateDataKey() ([]byte, []byte, error) {
    result, err := k.client.GenerateDataKey(context.Background(), &kms.GenerateDataKeyInput{
        KeyId:   aws.String(k.keyID),
        KeySpec: types.DataKeySpecAes256,
    })
    if err != nil {
        return nil, nil, err
    }
    
    // 明文密钥用于加密
    plaintextKey := result.Plaintext
    
    // 加密后的密钥用于存储
    encryptedKey := result.CiphertextBlob
    
    return plaintextKey, encryptedKey, nil
}

func (k *KMSClient) DecryptDataKey(encryptedKey []byte) ([]byte, error) {
    result, err := k.client.Decrypt(context.Background(), &kms.DecryptInput{
        CiphertextBlob: encryptedKey,
        KeyId:          aws.String(k.keyID),
    })
    if err != nil {
        return nil, err
    }
    
    return result.Plaintext, nil
}
```

### 4. 密钥管理最佳实践

```yaml
# 密钥轮换策略
key_rotation:
  # 数据加密密钥
  data_encryption_key:
    rotation_period: "90d"
    automatic: true
    
  # API Key
  api_key:
    max_age: "365d"
    warning_before: "30d"
    
  # JWT 密钥
  jwt_secret:
    rotation_period: "180d"
    grace_period: "7d"  # 新旧密钥同时有效的宽限期
```

## 安全检查清单

- [ ] 所有 API 端点都经过身份验证
- [ ] API Key 使用安全随机数生成
- [ ] API Key 哈希存储，永不明文保存
- [ ] JWT 使用短有效期 Access Token + Refresh Token 机制
- [ ] JWT 支持撤销和黑名单
- [ ] 实施分级限流策略
- [ ] 所有敏感数据加密存储
- [ ] 传输层强制 TLS 1.2+
- [ ] 启用 HSTS
- [ ] 完整的审计日志
- [ ] 定期密钥轮换
- [ ] 安全响应头配置
- [ ] CORS 严格配置
- [ ] SQL 注入防护
- [ ] XSS 防护
- [ ] CSRF 防护
