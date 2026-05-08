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

// AuthIdentity 用户第三方认证身份模型
// 存储用户与第三方平台（GitHub、Google、微信等）的绑定关系
type AuthIdentity struct {
	ent.Schema
}

// Annotations 第三方认证身份表注解
func (AuthIdentity) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "auth_identities"},
	}
}

// Fields 第三方认证身份字段定义
func (AuthIdentity) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Comment("关联的用户ID"),
		field.Enum("provider").
			Values("github", "google", "wechat").
			Comment("OAuth提供商: github-GitHub, google-Google, wechat-微信"),
		field.String("provider_user_id").
			MaxLen(255).
			Comment("第三方平台的用户唯一标识"),
		field.String("email").
			Optional().
			MaxLen(255).
			Comment("第三方平台提供的邮箱地址"),
		field.String("name").
			Optional().
			MaxLen(100).
			Comment("第三方平台提供的用户名"),
		field.String("avatar_url").
			Optional().
			MaxLen(500).
			Comment("第三方平台提供的头像URL"),
		field.String("access_token").
			Optional().
			Sensitive().
			MaxLen(1000).
			Comment("访问令牌（加密存储）"),
		field.String("refresh_token").
			Optional().
			Sensitive().
			MaxLen(1000).
			Comment("刷新令牌（加密存储）"),
		field.Time("token_expires_at").
			Optional().
			Nillable().
			Comment("令牌过期时间"),
		field.JSON("raw_data", map[string]interface{}{}).
			Optional().
			Comment("第三方平台返回的原始数据"),
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

// Edges 第三方认证身份关系定义
func (AuthIdentity) Edges() []ent.Edge {
	return []ent.Edge{
		// 认证身份属于一个用户
		edge.From("user", User.Type).
			Ref("auth_identities").
			Unique().
			Required().
			Field("user_id"),
	}
}

// Indexes 第三方认证身份索引定义
func (AuthIdentity) Indexes() []ent.Index {
	return []ent.Index{
		// 同一提供商下，provider_user_id 唯一
		index.Fields("provider", "provider_user_id").Unique(),
		index.Fields("user_id"),
		index.Fields("provider"),
	}
}
