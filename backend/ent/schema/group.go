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

// Group 分组/渠道模型
type Group struct {
	ent.Schema
}

// Annotations 分组表注解
func (Group) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "groups"},
	}
}

// Fields 分组字段定义
func (Group) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(100).
			Comment("分组名称"),
		field.String("description").
			Optional().
			MaxLen(500).
			Comment("分组描述"),
		field.Enum("platform").
			Values("claude", "openai", "gemini", "self_hosted").
			Comment("平台类型: claude, openai, gemini, self_hosted"),
		field.Enum("billing_mode").
			Values("balance", "subscription").
			Default("balance").
			Comment("计费模式: balance-余额计费, subscription-订阅计费"),
		field.Float("rate_multiplier").
			Default(1.0).
			Precision(10).
			Scale(4).
			Comment("费率倍率"),
		field.Int("rpm_override").
			Optional().
			Nillable().
			Comment("每分钟请求限制覆盖值，为空则使用账号默认值"),
		field.JSON("model_mapping", map[string]string{}).
			Optional().
			Comment("模型映射关系，JSON格式，键为请求模型，值为实际转发模型"),
		field.Int("priority").
			Default(0).
			Comment("优先级，数值越大优先级越高"),
		field.Int("weight").
			Default(100).
			Comment("权重，用于负载均衡"),
		field.Enum("status").
			Values("active", "inactive").
			Default("active").
			Comment("状态: active-启用, inactive-停用"),
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

// Edges 分组关系定义
func (Group) Edges() []ent.Edge {
	return []ent.Edge{
		// 分组包含多个账号（多对多，通过 account_group 关联表）
		edge.To("accounts", Account.Type).
			Through("account_groups", AccountGroup.Type),
	}
}

// Indexes 分组索引定义
func (Group) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("platform"),
		index.Fields("status"),
		index.Fields("priority"),
	}
}
