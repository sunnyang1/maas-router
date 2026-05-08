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

// RedeemCode 卡密充值码模型
// 用于生成和管理充值卡密，支持批量生成和一次性使用
type RedeemCode struct {
	ent.Schema
}

// Annotations 卡密表注解
func (RedeemCode) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "redeem_codes"},
	}
}

// Fields 卡密字段定义
func (RedeemCode) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Unique().
			NotEmpty().
			MaxLen(64).
			Comment("卡密代码，唯一标识"),
		field.Float("amount").
			Precision(18).
			Scale(6).
			Comment("卡密面额（元）"),
		field.Enum("status").
			Values("unused", "used", "expired", "disabled").
			Default("unused").
			Comment("卡密状态: unused-未使用, used-已使用, expired-已过期, disabled-已禁用"),
		field.Int64("created_by").
			Optional().
			Comment("创建者管理员ID"),
		field.String("batch_no").
			Optional().
			MaxLen(64).
			Comment("批次号，用于批量生成"),
		field.Time("expires_at").
			Optional().
			Nillable().
			Comment("过期时间，null表示永不过期"),
		field.Time("used_at").
			Optional().
			Nillable().
			Comment("使用时间"),
		field.Int64("used_by").
			Optional().
			Comment("使用用户ID"),
		field.String("remark").
			Optional().
			MaxLen(255).
			Comment("备注信息"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
	}
}

// Edges 卡密关系定义
func (RedeemCode) Edges() []ent.Edge {
	return []ent.Edge{
		// 卡密被某个用户使用
		edge.From("user", User.Type).
			Ref("redeemed_codes").
			Unique().
			Field("used_by"),
	}
}

// Indexes 卡密索引定义
func (RedeemCode) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code").Unique(),
		index.Fields("status"),
		index.Fields("batch_no"),
		index.Fields("created_at"),
	}
}
