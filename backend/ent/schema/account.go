package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Account 上游AI账号模型（核心）
type Account struct {
	ent.Schema
}

// Annotations 上游账号表注解
func (Account) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "accounts"},
	}
}

// Fields 上游账号字段定义
func (Account) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(100).
			Comment("账号名称"),
		field.Enum("platform").
			Values("claude", "openai", "gemini", "self_hosted").
			Comment("平台类型: claude, openai, gemini, self_hosted"),
		field.Enum("account_type").
			Values("oauth", "api_key", "cookie").
			Comment("账号类型: oauth-OAuth授权, api_key-API密钥, cookie-Cookie认证"),
		field.JSON("credentials", map[string]interface{}{}).
			Sensitive().
			Comment("加密的凭证信息，JSON格式存储"),
		field.Enum("status").
			Values("active", "disabled", "unschedulable").
			Default("active").
			Comment("状态: active-正常, disabled-已禁用, unschedulable-不可调度"),
		field.Int("max_concurrency").
			Default(5).
			Comment("最大并发数"),
		field.Int("current_concurrency").
			Default(0).
			Comment("当前并发数"),
		field.Int("rpm_limit").
			Default(60).
			Comment("每分钟请求限制(RPM)"),
		field.Int64("total_requests").
			Default(0).
			Comment("总请求数"),
		field.Int64("error_count").
			Default(0).
			Comment("错误计数"),
		field.Time("last_used_at").
			Optional().
			Nillable().
			Comment("最后使用时间"),
		field.Time("last_error_at").
			Optional().
			Nillable().
			Comment("最后错误时间"),
		field.String("proxy_url").
			Optional().
			MaxLen(500).
			Comment("代理地址"),
		field.String("tls_fingerprint").
			Optional().
			MaxLen(100).
			Comment("TLS指纹标识"),
		field.JSON("extra", map[string]interface{}{}).
			Optional().
			Comment("扩展信息，JSON格式"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("更新时间"),
	}
}

// Edges 上游账号关系定义
func (Account) Edges() []ent.Edge {
	return []ent.Edge{
		// 账号属于多个分组（多对多，通过 account_group 关联表）
		edge.To("groups", Group.Type).
			Through("account_groups", AccountGroup.Type),
	}
}

// Indexes 上游账号索引定义
func (Account) Indexes() []ent.Index {
	return []ent.Index{
		// 普通索引：平台（用于按平台筛选账号）
		index.Fields("platform"),
		// 普通索引：状态（用于按状态筛选账号）
		index.Fields("status"),
		// 普通索引：账号类型（用于按类型筛选账号）
		index.Fields("account_type"),
		// 复合索引：平台和状态（最常用的查询组合）
		index.Fields("platform", "status"),
		// 复合索引：状态和最后使用时间（用于调度算法）
		index.Fields("status", "last_used_at"),
		// 普通索引：最后错误时间（用于错误监控）
		index.Fields("last_error_at"),
		// 普通索引：创建时间（用于排序）
		index.Fields("created_at"),
	}
}
