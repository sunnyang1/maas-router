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

// AffiliateRecord 返利记录模型
// 记录每次返利的详细信息，包括来源用户、金额、类型等
type AffiliateRecord struct {
	ent.Schema
}

// Annotations 返利记录表注解
func (AffiliateRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "affiliate_records"},
	}
}

// Fields 返利记录字段定义
func (AffiliateRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Comment("获得返利的用户ID"),
		field.Int64("from_user_id").
			Comment("产生返利的来源用户ID"),
		field.Float("amount").
			Precision(18).
			Scale(6).
			Comment("返利金额"),
		field.Enum("type").
			Values("register", "recharge", "consumption", "withdrawal").
			Comment("返利类型: register-注册奖励, recharge-充值奖励, consumption-消费提成, withdrawal-提现"),
		field.Float("source_amount").
			Optional().
			Precision(18).
			Scale(6).
			Comment("来源金额（如充值金额、消费金额）"),
		field.Float("rate").
			Optional().
			Precision(5).
			Scale(4).
			Comment("返利比例（如 0.1 表示 10%）"),
		field.Enum("status").
			Values("pending", "confirmed", "cancelled").
			Default("pending").
			Comment("返利状态: pending-待确认, confirmed-已确认, cancelled-已取消"),
		field.String("description").
			Optional().
			MaxLen(255).
			Comment("描述信息"),
		field.Time("confirmed_at").
			Optional().
			Nillable().
			Comment("确认时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
	}
}

// Edges 返利记录关系定义
func (AffiliateRecord) Edges() []ent.Edge {
	return []ent.Edge{
		// 返利记录属于一个用户
		edge.From("user", User.Type).
			Ref("affiliate_records").
			Unique().
			Required().
			Field("user_id"),
	}
}

// Indexes 返利记录索引定义
func (AffiliateRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("from_user_id"),
		index.Fields("type"),
		index.Fields("status"),
		index.Fields("created_at"),
	}
}
