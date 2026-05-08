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

// APIKey API Key模型
type APIKey struct {
	ent.Schema
}

// Annotations API Key表注解
func (APIKey) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "api_keys"},
	}
}

// Fields API Key字段定义
func (APIKey) Fields() []ent.Field {
	return []ent.Field{
		field.String("key_hash").
			NotEmpty().
			MaxLen(255).
			Comment("API Key的哈希值，用于安全验证"),
		field.String("key_prefix").
			NotEmpty().
			MaxLen(8).
			Comment("API Key前8位，用于展示和识别"),
		field.String("name").
			Optional().
			MaxLen(100).
			Comment("API Key名称/备注"),
		field.Enum("status").
			Values("active", "revoked", "expired").
			Default("active").
			Comment("状态: active-正常, revoked-已撤销, expired-已过期"),
		field.Float("daily_limit").
			Optional().
			Nillable().
			Precision(18).
			Scale(6).
			Comment("每日使用额度限制"),
		field.Float("monthly_limit").
			Optional().
			Nillable().
			Precision(18).
			Scale(6).
			Comment("每月使用额度限制"),
		field.JSON("allowed_models", []string{}).
			Optional().
			Comment("允许使用的模型列表，JSON数组格式"),
		field.JSON("ip_whitelist", []string{}).
			Optional().
			Comment("IP白名单，JSON数组格式"),
		field.JSON("ip_blacklist", []string{}).
			Optional().
			Comment("IP黑名单，JSON数组格式"),
		field.Time("expires_at").
			Optional().
			Nillable().
			Comment("过期时间"),
		field.Time("last_used_at").
			Optional().
			Nillable().
			Comment("最后使用时间"),
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

// Edges API Key关系定义
func (APIKey) Edges() []ent.Edge {
	return []ent.Edge{
		// API Key属于一个用户
		edge.From("user", User.Type).
			Ref("api_keys").
			Unique().
			Required().
			Field("user_id"),
		// API Key拥有多条使用记录
		edge.To("usage_records", UsageRecord.Type),
	}
}

// Indexes API Key索引定义
func (APIKey) Indexes() []ent.Index {
	return []ent.Index{
		// 唯一索引：Key Hash（用于验证 API Key）
		index.Fields("key_hash").Unique(),
		// 普通索引：用户ID（用于查询用户的 API Keys）
		index.Fields("user_id"),
		// 普通索引：状态（用于按状态筛选）
		index.Fields("status"),
		// 复合索引：用户ID和状态（最常用的查询组合）
		index.Fields("user_id", "status"),
		// 普通索引：过期时间（用于清理过期 Key）
		index.Fields("expires_at"),
		// 普通索引：最后使用时间（用于统计）
		index.Fields("last_used_at"),
		// 普通索引：创建时间（用于排序）
		index.Fields("created_at"),
	}
}
