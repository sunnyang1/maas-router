// Package config 提供应用配置管理功能
// 使用 Viper 加载配置，支持 YAML 文件和环境变量覆盖
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config 应用全局配置结构体
type Config struct {
	Server     ServerConfig     `mapstructure:"server" json:"server" yaml:"server"`
	Database   DatabaseConfig   `mapstructure:"database" json:"database" yaml:"database"`
	Redis      RedisConfig      `mapstructure:"redis" json:"redis" yaml:"redis"`
	JWT        JWTConfig        `mapstructure:"jwt" json:"jwt" yaml:"jwt"`
	CORS       CORSConfig       `mapstructure:"cors" json:"cors" yaml:"cors"`
	Gateway    GatewayConfig    `mapstructure:"gateway" json:"gateway" yaml:"gateway"`
	Log        LogConfig        `mapstructure:"log" json:"log" yaml:"log"`
	JudgeAgent JudgeAgentConfig  `mapstructure:"judge_agent" json:"judge_agent" yaml:"judge_agent"`
	Complexity  ComplexityConfig `mapstructure:"complexity" json:"complexity" yaml:"complexity"`
	Billing    BillingConfig    `mapstructure:"billing" json:"billing" yaml:"billing"`
	OAuth      OAuthConfig      `mapstructure:"oauth" json:"oauth" yaml:"oauth"`
	Affiliate  AffiliateConfig  `mapstructure:"affiliate" json:"affiliate" yaml:"affiliate"`
	RedeemCode RedeemCodeConfig `mapstructure:"redeem_code" json:"redeem_code" yaml:"redeem_code"`
}

// ServerConfig HTTP 服务器配置
type ServerConfig struct {
	// 监听地址
	Host string `mapstructure:"host" json:"host" yaml:"host"`
	// 监听端口
	Port int `mapstructure:"port" json:"port" yaml:"port"`
	// 运行模式: Simple / Normal
	// Simple 模式下使用精简路由，Normal 模式下启用全部功能
	Mode string `mapstructure:"mode" json:"mode" yaml:"mode"`
	// 是否启用 HTTP/2 H2C（非 TLS 的 HTTP/2）
	EnableH2C bool `mapstructure:"enable_h2c" json:"enable_h2c" yaml:"enable_h2c"`
	// 优雅关闭超时时间（秒）
	ShutdownTimeout int `mapstructure:"shutdown_timeout" json:"shutdown_timeout" yaml:"shutdown_timeout"`
}

// DatabaseConfig PostgreSQL 数据库配置
type DatabaseConfig struct {
	// 数据库主机地址
	Host string `mapstructure:"host" json:"host" yaml:"host"`
	// 数据库端口
	Port int `mapstructure:"port" json:"port" yaml:"port"`
	// 数据库用户名
	User string `mapstructure:"user" json:"user" yaml:"user"`
	// 数据库密码
	Password string `mapstructure:"password" json:"password" yaml:"password"`
	// 数据库名称
	DBName string `mapstructure:"dbname" json:"dbname" yaml:"dbname"`
	// SSL 模式: disable / require / verify-ca / verify-full
	SSLMode string `mapstructure:"sslmode" json:"sslmode" yaml:"sslmode"`
	// 最大空闲连接数
	MaxIdleConns int `mapstructure:"max_idle_conns" json:"max_idle_conns" yaml:"max_idle_conns"`
	// 最大打开连接数
	MaxOpenConns int `mapstructure:"max_open_conns" json:"max_open_conns" yaml:"max_open_conns"`
	// 连接最大存活时间（秒）
	ConnMaxLifetime int `mapstructure:"conn_max_lifetime" json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	// 连接最大空闲时间（秒）
	ConnMaxIdleTime int `mapstructure:"conn_max_idle_time" json:"conn_max_idle_time" yaml:"conn_max_idle_time"`
	// 连接超时时间（秒）
	ConnTimeout int `mapstructure:"conn_timeout" json:"conn_timeout" yaml:"conn_timeout"`
	// 是否启用连接池预热
	EnableWarmup bool `mapstructure:"enable_warmup" json:"enable_warmup" yaml:"enable_warmup"`
	// 预热连接数
	WarmupConns int `mapstructure:"warmup_conns" json:"warmup_conns" yaml:"warmup_conns"`
}

// DSN 生成 PostgreSQL 数据源名称
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// RedisConfig Redis 缓存配置
type RedisConfig struct {
	// Redis 主机地址
	Host string `mapstructure:"host" json:"host" yaml:"host"`
	// Redis 端口
	Port int `mapstructure:"port" json:"port" yaml:"port"`
	// Redis 密码（可选）
	Password string `mapstructure:"password" json:"password" yaml:"password"`
	// 数据库编号
	DB int `mapstructure:"db" json:"db" yaml:"db"`
	// 连接池大小
	PoolSize int `mapstructure:"pool_size" json:"pool_size" yaml:"pool_size"`
	// 最小空闲连接数
	MinIdleConns int `mapstructure:"min_idle_conns" json:"min_idle_conns" yaml:"min_idle_conns"`
	// 连接超时时间（秒）
	ConnTimeout int `mapstructure:"conn_timeout" json:"conn_timeout" yaml:"conn_timeout"`
	// 读取超时时间（秒）
	ReadTimeout int `mapstructure:"read_timeout" json:"read_timeout" yaml:"read_timeout"`
	// 写入超时时间（秒）
	WriteTimeout int `mapstructure:"write_timeout" json:"write_timeout" yaml:"write_timeout"`
	// 连接最大重试次数
	MaxRetries int `mapstructure:"max_retries" json:"max_retries" yaml:"max_retries"`
	// 连接重试间隔（毫秒）
	RetryBackoff int `mapstructure:"retry_backoff" json:"retry_backoff" yaml:"retry_backoff"`
}

// Addr 生成 Redis 连接地址
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// JWTConfig JWT 认证配置
type JWTConfig struct {
	// 签名密钥
	Secret string `mapstructure:"secret" json:"secret" yaml:"secret"`
	// Token 过期时间（小时）
	ExpireHours int `mapstructure:"expire_hours" json:"expire_hours" yaml:"expire_hours"`
	// Refresh Token 过期时间（小时）
	RefreshExpireHours int `mapstructure:"refresh_expire_hours" json:"refresh_expire_hours" yaml:"refresh_expire_hours"`
	// 签发者
	Issuer string `mapstructure:"issuer" json:"issuer" yaml:"issuer"`
}

// CORSConfig 跨域资源共享配置
type CORSConfig struct {
	// 是否启用 CORS
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	// 允许的源列表，* 表示全部允许
	AllowOrigins []string `mapstructure:"allow_origins" json:"allow_origins" yaml:"allow_origins"`
	// 允许的 HTTP 方法
	AllowMethods []string `mapstructure:"allow_methods" json:"allow_methods" yaml:"allow_methods"`
	// 允许的请求头
	AllowHeaders []string `mapstructure:"allow_headers" json:"allow_headers" yaml:"allow_headers"`
	// 是否允许携带凭证
	AllowCredentials bool `mapstructure:"allow_credentials" json:"allow_credentials" yaml:"allow_credentials"`
	// 预检请求缓存时间（秒）
	MaxAge int `mapstructure:"max_age" json:"max_age" yaml:"max_age"`
}

// GatewayConfig API 网关配置
type GatewayConfig struct {
	// 请求体最大大小（MB）
	MaxRequestBodyMB int `mapstructure:"max_request_body_mb" json:"max_request_body_mb" yaml:"max_request_body_mb"`
	// 请求超时时间（秒）
	RequestTimeout int `mapstructure:"request_timeout" json:"request_timeout" yaml:"request_timeout"`
	// 上游服务超时时间（秒）
	UpstreamTimeout int `mapstructure:"upstream_timeout" json:"upstream_timeout" yaml:"upstream_timeout"`
	// 是否启用请求压缩
	EnableCompression bool `mapstructure:"enable_compression" json:"enable_compression" yaml:"enable_compression"`
	// HTTP 连接池配置
	HTTPPool HTTPPoolConfig `mapstructure:"http_pool" json:"http_pool" yaml:"http_pool"`
}

// HTTPPoolConfig HTTP 连接池配置
type HTTPPoolConfig struct {
	// 最大空闲连接数
	MaxIdleConns int `mapstructure:"max_idle_conns" json:"max_idle_conns" yaml:"max_idle_conns"`
	// 每个主机的最大空闲连接数
	MaxIdleConnsPerHost int `mapstructure:"max_idle_conns_per_host" json:"max_idle_conns_per_host" yaml:"max_idle_conns_per_host"`
	// 空闲连接超时时间（秒）
	IdleConnTimeout int `mapstructure:"idle_conn_timeout" json:"idle_conn_timeout" yaml:"idle_conn_timeout"`
	// TLS 握手超时时间（秒）
	TLSHandshakeTimeout int `mapstructure:"tls_handshake_timeout" json:"tls_handshake_timeout" yaml:"tls_handshake_timeout"`
	// 是否禁用 Keep-Alive
	DisableKeepAlives bool `mapstructure:"disable_keep_alives" json:"disable_keep_alives" yaml:"disable_keep_alives"`
	// 是否禁用压缩
	DisableCompression bool `mapstructure:"disable_compression" json:"disable_compression" yaml:"disable_compression"`
	// 响应头超时时间（秒）
	ResponseHeaderTimeout int `mapstructure:"response_header_timeout" json:"response_header_timeout" yaml:"response_header_timeout"`
	// 期望 100-Continue 超时时间（秒）
	ExpectContinueTimeout int `mapstructure:"expect_continue_timeout" json:"expect_continue_timeout" yaml:"expect_continue_timeout"`
	// 最大重试次数
	MaxRetries int `mapstructure:"max_retries" json:"max_retries" yaml:"max_retries"`
	// 重试间隔（毫秒）
	RetryInterval int `mapstructure:"retry_interval" json:"retry_interval" yaml:"retry_interval"`
}

// LogConfig 日志配置
type LogConfig struct {
	// 日志级别: debug / info / warn / error
	Level string `mapstructure:"level" json:"level" yaml:"level"`
	// 日志文件路径（为空则输出到 stdout）
	FilePath string `mapstructure:"file_path" json:"file_path" yaml:"file_path"`
	// 日志文件最大大小（MB）
	MaxSizeMB int `mapstructure:"max_size_mb" json:"max_size_mb" yaml:"max_size_mb"`
	// 保留的旧日志文件最大数量
	MaxBackups int `mapstructure:"max_backups" json:"max_backups" yaml:"max_backups"`
	// 保留旧日志文件的最大天数
	MaxAgeDays int `mapstructure:"max_age_days" json:"max_age_days" yaml:"max_age_days"`
	// 是否压缩旧日志文件
	Compress bool `mapstructure:"compress" json:"compress" yaml:"compress"`
	// 是否使用 JSON 格式输出
	JSONFormat bool `mapstructure:"json_format" json:"json_format" yaml:"json_format"`
}

// JudgeAgentConfig 智能路由 Agent 配置
type JudgeAgentConfig struct {
	// Agent 服务地址
	Addr string `mapstructure:"addr" json:"addr" yaml:"addr"`
	// 请求超时时间（毫秒）
	TimeoutMs int `mapstructure:"timeout_ms" json:"timeout_ms" yaml:"timeout_ms"`
	// 最大重试次数
	MaxRetries int `mapstructure:"max_retries" json:"max_retries" yaml:"max_retries"`
	// 连接池大小
	PoolSize int `mapstructure:"pool_size" json:"pool_size" yaml:"pool_size"`
	// 是否启用 Agent
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
}

// ComplexityConfig 智能推理资源优化引擎配置
type ComplexityConfig struct {
	// 是否启用复杂度分析
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	// 分析模式: local（本地）/ remote（远程）/ hybrid（混合）
	Mode string `mapstructure:"mode" json:"mode" yaml:"mode"`
	// 远程分析服务地址
	RemoteAddr string `mapstructure:"remote_addr" json:"remote_addr" yaml:"remote_addr"`
	// 请求超时时间（毫秒）
	TimeoutMs int `mapstructure:"timeout_ms" json:"timeout_ms" yaml:"timeout_ms"`
	// 最大重试次数
	MaxRetries int `mapstructure:"max_retries" json:"max_retries" yaml:"max_retries"`
	// 分析失败时是否回退到 Judge Agent
	FallbackToJudge bool `mapstructure:"fallback_to_judge" json:"fallback_to_judge" yaml:"fallback_to_judge"`
	// 缓存 TTL（秒）
	CacheTTLSec int `mapstructure:"cache_ttl_sec" json:"cache_ttl_sec" yaml:"cache_ttl_sec"`
	// 模型层级配置
	ModelTiers []ModelTierConfig `mapstructure:"model_tiers" json:"model_tiers" yaml:"model_tiers"`
	// 特征提取配置
	Features FeatureConfig `mapstructure:"features" json:"features" yaml:"features"`
	// 质量守卫配置
	QualityGuard QualityGuardConfig `mapstructure:"quality_guard" json:"quality_guard" yaml:"quality_guard"`
}

// ModelTierConfig 模型层级配置
type ModelTierConfig struct {
	// 层级名称: economy, standard, advanced, premium
	Name string `mapstructure:"name" json:"name" yaml:"name"`
	// 模型名称
	Model string `mapstructure:"model" json:"model" yaml:"model"`
	// 复杂度阈值 [0, 1]，score <= threshold 时使用该层级
	Threshold float64 `mapstructure:"threshold" json:"threshold" yaml:"threshold"`
	// 每 token 成本（美元）
	CostPerToken float64 `mapstructure:"cost_per_token" json:"cost_per_token" yaml:"cost_per_token"`
	// 回退模型
	FallbackModel string `mapstructure:"fallback_model" json:"fallback_model" yaml:"fallback_model"`
}

// FeatureConfig 特征提取配置
type FeatureConfig struct {
	// 最大 token 计数上限（用于归一化）
	MaxTokenCount int `mapstructure:"max_token_count" json:"max_token_count" yaml:"max_token_count"`
	// 最大句子数上限（用于归一化）
	MaxSentenceCount int `mapstructure:"max_sentence_count" json:"max_sentence_count" yaml:"max_sentence_count"`
	// 最大上下文大小（字符数，用于归一化）
	MaxContextSize int `mapstructure:"max_context_size" json:"max_context_size" yaml:"max_context_size"`
	// 最大历史消息数（用于归一化）
	MaxHistoryLength int `mapstructure:"max_history_length" json:"max_history_length" yaml:"max_history_length"`
	// 自定义专业术语列表
	CustomTechnicalTerms []string `mapstructure:"custom_technical_terms" json:"custom_technical_terms" yaml:"custom_technical_terms"`
}

// QualityGuardConfig 质量守卫配置
type QualityGuardConfig struct {
	// 是否启用质量守卫
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	// 最低质量通过率阈值
	MinQualityPassRate float64 `mapstructure:"min_quality_pass_rate" json:"min_quality_pass_rate" yaml:"min_quality_pass_rate"`
	// 自动升级阈值（质量风险达到此值时自动升级模型）
	AutoUpgradeThreshold string `mapstructure:"auto_upgrade_threshold" json:"auto_upgrade_threshold" yaml:"auto_upgrade_threshold"`
	// 反馈采样率 [0, 1]
	FeedbackSampleRate float64 `mapstructure:"feedback_sample_rate" json:"feedback_sample_rate" yaml:"feedback_sample_rate"`
	// 统计窗口大小（秒）
	StatsWindowSec int `mapstructure:"stats_window_sec" json:"stats_window_sec" yaml:"stats_window_sec"`
}

// BillingConfig 计费系统配置
type BillingConfig struct {
	// 计费服务地址
	Addr string `mapstructure:"addr" json:"addr" yaml:"addr"`
	// 请求超时时间（毫秒）
	TimeoutMs int `mapstructure:"timeout_ms" json:"timeout_ms" yaml:"timeout_ms"`
	// 是否启用计费
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	// 计费周期（秒），0 表示实时计费
	BillingCycleSec int `mapstructure:"billing_cycle_sec" json:"billing_cycle_sec" yaml:"billing_cycle_sec"`
	// 最低计费单位（元）
	MinChargeUnit float64 `mapstructure:"min_charge_unit" json:"min_charge_unit" yaml:"min_charge_unit"`
}

// OAuthConfig OAuth认证配置
type OAuthConfig struct {
	// GitHub OAuth配置
	GitHub OAuthProviderConfig `mapstructure:"github" json:"github" yaml:"github"`
	// Google OAuth配置
	Google OAuthProviderConfig `mapstructure:"google" json:"google" yaml:"google"`
	// 微信OAuth配置
	WeChat WeChatOAuthConfig `mapstructure:"wechat" json:"wechat" yaml:"wechat"`
}

// OAuthProviderConfig OAuth提供商配置
type OAuthProviderConfig struct {
	// 客户端ID
	ClientID string `mapstructure:"client_id" json:"client_id" yaml:"client_id"`
	// 客户端密钥
	ClientSecret string `mapstructure:"client_secret" json:"client_secret" yaml:"client_secret"`
	// 回调URL
	RedirectURL string `mapstructure:"redirect_url" json:"redirect_url" yaml:"redirect_url"`
	// 是否启用
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
}

// WeChatOAuthConfig 微信OAuth配置
type WeChatOAuthConfig struct {
	// 应用ID
	AppID string `mapstructure:"app_id" json:"app_id" yaml:"app_id"`
	// 应用密钥
	AppSecret string `mapstructure:"app_secret" json:"app_secret" yaml:"app_secret"`
	// 回调URL
	RedirectURL string `mapstructure:"redirect_url" json:"redirect_url" yaml:"redirect_url"`
	// 是否启用
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
}

// AffiliateConfig 邀请返利配置
type AffiliateConfig struct {
	// 是否启用邀请返利
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	// 注册奖励金额（元）
	RegisterReward float64 `mapstructure:"register_reward" json:"register_reward" yaml:"register_reward"`
	// 充值返利比例（如0.1表示10%）
	RechargeRate float64 `mapstructure:"recharge_rate" json:"recharge_rate" yaml:"recharge_rate"`
	// 消费返利比例（如0.05表示5%）
	ConsumptionRate float64 `mapstructure:"consumption_rate" json:"consumption_rate" yaml:"consumption_rate"`
	// 最低提现金额
	MinWithdrawal float64 `mapstructure:"min_withdrawal" json:"min_withdrawal" yaml:"min_withdrawal"`
}

// RedeemCodeConfig 卡密充值配置
type RedeemCodeConfig struct {
	// 是否启用卡密充值
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	// 卡密前缀
	CodePrefix string `mapstructure:"code_prefix" json:"code_prefix" yaml:"code_prefix"`
	// 卡密长度
	CodeLength int `mapstructure:"code_length" json:"code_length" yaml:"code_length"`
}

// LoadConfig 加载应用配置
// 按优先级: 命令行指定配置文件 > 环境变量 > 默认值
// 环境变量使用 MAAS_ROUTER_ 前缀，层级用下划线分隔，例如: MAAS_ROUTER_SERVER_PORT=8080
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 配置文件设置
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// 默认搜索路径
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/maas-router")
	}

	// 环境变量配置
	// 使用 MAAS_ROUTER_ 作为前缀，例如 MAAS_ROUTER_SERVER_PORT
	v.SetEnvPrefix("MAAS_ROUTER")
	// 将配置键中的点号替换为下划线，适配环境变量
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
		// 配置文件不存在时使用默认值和环境变量
	}

	// 解析配置到结构体
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &cfg, nil
}

// setDefaults 设置配置默认值
func setDefaults(v *viper.Viper) {
	// 服务器默认配置
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "normal")
	v.SetDefault("server.enable_h2c", true)
	v.SetDefault("server.shutdown_timeout", 30)

	// 数据库默认配置
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	// 数据库密码必须通过环境变量 MAAS_ROUTER_DATABASE_PASSWORD 或配置文件设置，不应使用默认值
	v.SetDefault("database.password", "")
	v.SetDefault("database.dbname", "maas_router")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.conn_max_lifetime", 3600)
	v.SetDefault("database.conn_max_idle_time", 600)
	v.SetDefault("database.conn_timeout", 10)
	v.SetDefault("database.enable_warmup", true)
	v.SetDefault("database.warmup_conns", 5)

	// Redis 默认配置
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 100)
	v.SetDefault("redis.min_idle_conns", 10)
	v.SetDefault("redis.conn_timeout", 5)
	v.SetDefault("redis.read_timeout", 3)
	v.SetDefault("redis.write_timeout", 3)
	v.SetDefault("redis.max_retries", 3)
	v.SetDefault("redis.retry_backoff", 100)

	// JWT 默认配置
	// JWT Secret 必须通过环境变量 MAAS_ROUTER_JWT_SECRET 或配置文件设置
	// 生产环境禁止使用空 Secret，服务启动时会进行校验
	v.SetDefault("jwt.secret", "")
	v.SetDefault("jwt.expire_hours", 24)
	v.SetDefault("jwt.refresh_expire_hours", 168)
	v.SetDefault("jwt.issuer", "maas-router")

	// CORS 默认配置
	// 注意：生产环境必须配置具体的允许域名，禁止使用通配符 "*"
	// 当 AllowCredentials 为 true 时，AllowOrigins 不可使用 "*"，否则会导致 CORS 策略矛盾
	v.SetDefault("cors.enabled", true)
	v.SetDefault("cors.allow_origins", []string{"http://localhost:3000", "http://localhost:8000"})
	v.SetDefault("cors.allow_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"})
	v.SetDefault("cors.allow_headers", []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"})
	v.SetDefault("cors.allow_credentials", false)
	v.SetDefault("cors.max_age", 86400)

	// 网关默认配置
	v.SetDefault("gateway.max_request_body_mb", 10)
	v.SetDefault("gateway.request_timeout", 120)
	v.SetDefault("gateway.upstream_timeout", 300)
	v.SetDefault("gateway.enable_compression", true)
	// HTTP 连接池默认配置
	v.SetDefault("gateway.http_pool.max_idle_conns", 100)
	v.SetDefault("gateway.http_pool.max_idle_conns_per_host", 10)
	v.SetDefault("gateway.http_pool.idle_conn_timeout", 90)
	v.SetDefault("gateway.http_pool.tls_handshake_timeout", 10)
	v.SetDefault("gateway.http_pool.disable_keep_alives", false)
	v.SetDefault("gateway.http_pool.disable_compression", false)
	v.SetDefault("gateway.http_pool.response_header_timeout", 30)
	v.SetDefault("gateway.http_pool.expect_continue_timeout", 1)
	v.SetDefault("gateway.http_pool.max_retries", 3)
	v.SetDefault("gateway.http_pool.retry_interval", 100)

	// 日志默认配置
	v.SetDefault("log.level", "info")
	v.SetDefault("log.file_path", "")
	v.SetDefault("log.max_size_mb", 100)
	v.SetDefault("log.max_backups", 10)
	v.SetDefault("log.max_age_days", 30)
	v.SetDefault("log.compress", true)
	v.SetDefault("log.json_format", true)

	// JudgeAgent 默认配置
	v.SetDefault("judge_agent.addr", "http://localhost:8001")
	v.SetDefault("judge_agent.timeout_ms", 5000)
	v.SetDefault("judge_agent.max_retries", 3)
	v.SetDefault("judge_agent.pool_size", 10)
	v.SetDefault("judge_agent.enabled", true)

	// Complexity 默认配置
	v.SetDefault("complexity.enabled", false)
	v.SetDefault("complexity.mode", "local")
	v.SetDefault("complexity.remote_addr", "http://localhost:8003")
	v.SetDefault("complexity.timeout_ms", 3000)
	v.SetDefault("complexity.max_retries", 2)
	v.SetDefault("complexity.fallback_to_judge", true)
	v.SetDefault("complexity.cache_ttl_sec", 3600)
	v.SetDefault("complexity.model_tiers", []map[string]interface{}{
		{"name": "economy", "model": "claude-3-5-haiku-20241022", "threshold": 0.25, "cost_per_token": 0.0000008, "fallback_model": ""},
		{"name": "standard", "model": "claude-3-5-sonnet-20241022", "threshold": 0.50, "cost_per_token": 0.000003, "fallback_model": "claude-3-5-haiku-20241022"},
		{"name": "advanced", "model": "claude-3-opus-20240229", "threshold": 0.75, "cost_per_token": 0.000015, "fallback_model": "claude-3-5-sonnet-20241022"},
		{"name": "premium", "model": "claude-3-opus-20240229", "threshold": 1.00, "cost_per_token": 0.000015, "fallback_model": "claude-3-5-sonnet-20241022"},
	})
	v.SetDefault("complexity.features.max_token_count", 500)
	v.SetDefault("complexity.features.max_sentence_count", 20)
	v.SetDefault("complexity.features.max_context_size", 50000)
	v.SetDefault("complexity.features.max_history_length", 20)
	v.SetDefault("complexity.features.custom_technical_terms", []string{})
	v.SetDefault("complexity.quality_guard.enabled", true)
	v.SetDefault("complexity.quality_guard.min_quality_pass_rate", 0.85)
	v.SetDefault("complexity.quality_guard.auto_upgrade_threshold", "high")
	v.SetDefault("complexity.quality_guard.feedback_sample_rate", 0.1)
	v.SetDefault("complexity.quality_guard.stats_window_sec", 3600)

	// 计费默认配置
	v.SetDefault("billing.addr", "http://localhost:8002")
	v.SetDefault("billing.timeout_ms", 3000)
	v.SetDefault("billing.enabled", true)
	v.SetDefault("billing.billing_cycle_sec", 0)
	v.SetDefault("billing.min_charge_unit", 0.001)

	// OAuth默认配置
	v.SetDefault("oauth.github.enabled", false)
	v.SetDefault("oauth.github.redirect_url", "http://localhost:3000/auth/github/callback")
	v.SetDefault("oauth.google.enabled", false)
	v.SetDefault("oauth.google.redirect_url", "http://localhost:3000/auth/google/callback")
	v.SetDefault("oauth.wechat.enabled", false)
	v.SetDefault("oauth.wechat.redirect_url", "http://localhost:3000/auth/wechat/callback")

	// 邀请返利默认配置
	v.SetDefault("affiliate.enabled", true)
	v.SetDefault("affiliate.register_reward", 5.0)
	v.SetDefault("affiliate.recharge_rate", 0.1)
	v.SetDefault("affiliate.consumption_rate", 0.05)
	v.SetDefault("affiliate.min_withdrawal", 10.0)

	// 卡密充值默认配置
	v.SetDefault("redeem_code.enabled", true)
	v.SetDefault("redeem_code.code_prefix", "MR")
	v.SetDefault("redeem_code.code_length", 16)
}

// Validate 校验配置的安全性和完整性
// 应在服务启动前调用，如果校验失败则拒绝启动
func (c *Config) Validate() error {
	// JWT Secret 安全校验
	if c.JWT.Secret == "" {
		// 开发模式下允许空 Secret，但输出警告
		if c.Server.Mode == "debug" || c.Server.Mode == "dev" {
			fmt.Fprintln(os.Stderr, "[WARN] JWT Secret 未设置，仅允许在开发模式使用，生产环境必须通过环境变量 MAAS_ROUTER_JWT_SECRET 设置")
			return nil
		}
		return fmt.Errorf("JWT Secret 未设置，生产环境必须通过环境变量 MAAS_ROUTER_JWT_SECRET 或配置文件设置安全密钥（至少32字符）")
	}
	if len(c.JWT.Secret) < 32 {
		fmt.Fprintf(os.Stderr, "[WARN] JWT Secret 长度不足32字符（当前: %d），建议使用更长的密钥以提高安全性\n", len(c.JWT.Secret))
	}

	// 数据库密码校验
	if c.Database.Password == "" {
		fmt.Fprintln(os.Stderr, "[WARN] 数据库密码未设置，请通过环境变量 MAAS_ROUTER_DATABASE_PASSWORD 或配置文件设置")
	}

	return nil
}

// BuildLogger 根据配置构建 zap 日志实例
// 使用 lumberjack 实现日志轮转
// Wire 注入入口：接收完整 Config，提取 Log 配置
func BuildLogger(cfg *Config) (*zap.Logger, error) {
	return buildLoggerFromLogConfig(&cfg.Log)
}

// buildLoggerFromLogConfig 根据日志配置构建 zap 日志实例
func buildLoggerFromLogConfig(cfg *LogConfig) (*zap.Logger, error) {
	// 解析日志级别
	var zapLevel zapcore.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// 编码器配置
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder

	// 选择编码器
	var encoder zapcore.Encoder
	if cfg.JSONFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 配置日志输出
	var writeSyncer zapcore.WriteSyncer
	if cfg.FilePath != "" {
		// 使用 lumberjack 实现日志轮转
		lumberJackLogger := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
			Compress:   cfg.Compress,
		}
		writeSyncer = zapcore.AddSync(lumberJackLogger)
	} else {
		// 输出到标准输出
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	// 创建核心
	core := zapcore.NewCore(encoder, writeSyncer, zapLevel)

	// 创建 logger
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, nil
}
