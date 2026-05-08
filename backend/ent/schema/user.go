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

// User 用户模型
type User struct {
	ent.Schema
}

// Annotations 用户表注解
func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "users"},
	}
}

// Fields 用户字段定义
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("email").
			Unique().
			NotEmpty().
			MaxLen(255).
			Comment("用户邮箱，唯一标识"),
		field.String("password_hash").
			NotEmpty().
			MaxLen(255).
			Sensitive().
			Comment("密码哈希值"),
		field.String("name").
			Optional().
			MaxLen(100).
			Comment("用户昵称"),
		field.Enum("role").
			Values("user", "admin").
			Default("user").
			Comment("用户角色: user-普通用户, admin-管理员"),
		field.Enum("status").
			Values("active", "suspended", "deleted").
			Default("active").
			Comment("用户状态: active-正常, suspended-已暂停, deleted-已删除"),
		field.Float("balance").
			Default(0).
			Precision(18).
			Scale(6).
			Comment("账户余额"),
		field.Int("concurrency").
			Default(5).
			Comment("并发请求限制数"),
		field.Int("token_version").
			Default(1).
			Comment("Token版本号，修改密码后递增使旧Token失效"),
		field.Time("last_active_at").
			Optional().
			Nillable().
			Comment("最后活跃时间"),
		// ===== 邀请返利系统字段 =====
		field.String("invite_code").
			Unique().
			Optional().
			MaxLen(20).
			Comment("用户邀请码，唯一标识"),
		field.Int64("invited_by").
			Optional().
			Nillable().
			Comment("邀请人用户ID"),
		field.Float("affiliate_balance").
			Default(0).
			Precision(18).
			Scale(6).
			Comment("返利余额（可提现）"),
		field.Float("total_affiliate_earnings").
			Default(0).
			Precision(18).
			Scale(6).
			Comment("累计返利收益"),
		field.Int("invite_count").
			Default(0).
			Comment("成功邀请人数"),
		// ===== 用户分组字段 =====
		field.String("group_name").
			Default("default").
			Optional().
			MaxLen(64).
			Comment("用户所属分组名称"),
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

// Edges 用户关系定义
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		// 用户拥有多个API Key
		edge.To("api_keys", APIKey.Type),
		// 用户拥有多条使用记录
		edge.To("usage_records", UsageRecord.Type),
		// 用户拥有多个支付订单
		edge.To("payment_orders", PaymentOrder.Type),
		// 用户拥有多个第三方认证身份
		edge.To("auth_identities", AuthIdentity.Type),
		// 用户使用的卡密记录
		edge.To("redeemed_codes", RedeemCode.Type),
		// 邀请人（自引用关系）
		edge.From("inviter", User.Type).
			Ref("invitees").
			Unique().
			Field("invited_by"),
		// 被邀请的用户列表
		edge.To("invitees", User.Type).
			From("inviter"),
		// 用户的返利记录
		edge.To("affiliate_records", AffiliateRecord.Type),
	}
}

// Indexes 用户索引定义
func (User) Indexes() []ent.Index {
	return []ent.Index{
		// 唯一索引：邮箱
		index.Fields("email").Unique(),
		// 普通索引：状态（用于按状态筛选用户）
		index.Fields("status"),
		// 普通索引：角色（用于按角色筛选用户）
		index.Fields("role"),
		// 唯一索引：邀请码
		index.Fields("invite_code").Unique(),
		// 普通索引：邀请人ID
		index.Fields("invited_by"),
		// 复合索引：状态和最后活跃时间（用于查询活跃用户）
		index.Fields("status", "last_active_at"),
		// 复合索引：角色和状态（用于管理员查询）
		index.Fields("role", "status"),
		// 普通索引：分组名称（用于按分组筛选用户）
		index.Fields("group_name"),
		// 普通索引：创建时间（用于排序）
		index.Fields("created_at"),
	}
}
