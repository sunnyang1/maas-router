// Package schema 定义支付订单的数据库模型
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PaymentOrder 支付订单实体
type PaymentOrder struct {
	ent.Schema
}

// PaymentStatus 支付状态
type PaymentStatus string

const (
	// PaymentStatusPending 待支付
	PaymentStatusPending PaymentStatus = "pending"
	// PaymentStatusProcessing 处理中
	PaymentStatusProcessing PaymentStatus = "processing"
	// PaymentStatusSuccess 支付成功
	PaymentStatusSuccess PaymentStatus = "success"
	// PaymentStatusFailed 支付失败
	PaymentStatusFailed PaymentStatus = "failed"
	// PaymentStatusCancelled 已取消
	PaymentStatusCancelled PaymentStatus = "cancelled"
	// PaymentStatusRefunded 已退款
	PaymentStatusRefunded PaymentStatus = "refunded"
	// PaymentStatusPartialRefunded 部分退款
	PaymentStatusPartialRefunded PaymentStatus = "partial_refunded"
)

// PaymentProvider 支付提供商
type PaymentProvider string

const (
	// ProviderStripe Stripe 支付
	ProviderStripe PaymentProvider = "stripe"
	// ProviderAlipay 支付宝
	ProviderAlipay PaymentProvider = "alipay"
	// ProviderWechat 微信支付
	ProviderWechat PaymentProvider = "wechat"
	// ProviderUnknown 未知
	ProviderUnknown PaymentProvider = "unknown"
)

// Fields 定义支付订单字段
func (PaymentOrder) Fields() []ent.Field {
	return []ent.Field{
		// 主键ID
		field.String("id").
			Unique().
			Immutable().
			Comment("支付订单ID，全局唯一"),

		// 关联订单
		field.String("order_id").
			NotEmpty().
			Comment("业务订单ID"),

		// 关联用户
		field.String("user_id").
			NotEmpty().
			Comment("用户ID"),

		// 支付金额
		field.Int64("amount").
			Positive().
			Comment("支付金额，单位：分"),

		// 货币类型
		field.String("currency").
			Default("CNY").
			Comment("货币代码：CNY、USD、EUR等"),

		// 支付提供商
		field.String("provider").
			NotEmpty().
			Comment("支付提供商：stripe、alipay、wechat"),

		// 支付状态
		field.String("status").
			Default(string(PaymentStatusPending)).
			Comment("支付状态：pending、processing、success、failed、cancelled、refunded"),

		// 第三方支付单号
		field.String("third_party_id").
			Optional().
			Nillable().
			Comment("第三方支付平台订单号"),

		// 商品描述
		field.String("description").
			Optional().
			Nillable().
			Comment("商品描述"),

		// 支付URL
		field.String("payment_url").
			Optional().
			Nillable().
			Comment("支付跳转URL"),

		// 支付参数（JSON格式，用于JSAPI/APP支付）
		field.JSON("payment_params", map[string]interface{}{}).
			Optional().
			Comment("支付参数，JSON格式"),

		// 回调URL
		field.String("notify_url").
			Optional().
			Nillable().
			Comment("异步回调URL"),

		// 返回URL
		field.String("return_url").
			Optional().
			Nillable().
			Comment("支付完成后跳转URL"),

		// 过期时间
		field.Time("expire_at").
			Optional().
			Nillable().
			Comment("订单过期时间"),

		// 支付时间
		field.Time("paid_at").
			Optional().
			Nillable().
			Comment("实际支付时间"),

		// 回调时间
		field.Time("notified_at").
			Optional().
			Nillable().
			Comment("收到回调通知时间"),

		// 回调原始数据
		field.Text("notify_data").
			Optional().
			Nillable().
			Comment("回调原始数据"),

		// 错误信息
		field.String("error_message").
			Optional().
			Nillable().
			Comment("错误信息"),

		// 附加元数据
		field.JSON("metadata", map[string]string{}).
			Optional().
			Default(map[string]string{}).
			Comment("附加元数据"),

		// 客户端IP
		field.String("client_ip").
			Optional().
			Nillable().
			Comment("客户端IP地址"),

		// 用户代理
		field.String("user_agent").
			Optional().
			Nillable().
			Comment("用户代理信息"),

		// 创建时间
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),

		// 更新时间
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("更新时间"),
	}
}

// Edges 定义支付订单关系
func (PaymentOrder) Edges() []ent.Edge {
	return []ent.Edge{
		// 关联用户（假设有User实体）
		edge.From("user", User.Type).
			Ref("payments").
			Field("user_id").
			Unique(),

		// 关联退款记录
		edge.To("refunds", Refund.Type).
			Comment("关联的退款记录"),
	}
}

// Indexes 定义支付订单索引
func (PaymentOrder) Indexes() []ent.Index {
	return []ent.Index{
		// 订单ID索引
		index.Fields("order_id"),

		// 用户ID索引
		index.Fields("user_id"),

		// 状态索引
		index.Fields("status"),

		// 支付提供商索引
		index.Fields("provider"),

		// 第三方支付单号索引
		index.Fields("third_party_id").
			Unique(),

		// 复合索引：用户+状态
		index.Fields("user_id", "status"),

		// 复合索引：订单+提供商
		index.Fields("order_id", "provider"),
	}
}

// Mixin 支付订单混合
func (PaymentOrder) Mixin() []ent.Mixin {
	return []ent.Mixin{
		// 可以添加审计日志等混合
	}
}

// Hooks 支付订单钩子
func (PaymentOrder) Hooks() []ent.Hook {
	return []ent.Hook{
		// 可以添加状态变更钩子
	}
}

// Annotations 支付订单注解
func (PaymentOrder) Annotations() []schema.Annotation {
	return []schema.Annotation{
		// 表名注解
		entsql.Annotation{
			Table: "payment_orders",
		},
	}
}

// Policy 支付订单权限策略
func (PaymentOrder) Policy() ent.Policy {
	return nil
}

// ============================================
// Refund 退款记录实体
// ============================================

// Refund 退款记录
type Refund struct {
	ent.Schema
}

// RefundStatus 退款状态
type RefundStatus string

const (
	// RefundStatusPending 退款处理中
	RefundStatusPending RefundStatus = "pending"
	// RefundStatusSuccess 退款成功
	RefundStatusSuccess RefundStatus = "success"
	// RefundStatusFailed 退款失败
	RefundStatusFailed RefundStatus = "failed"
	// RefundStatusClosed 退款关闭
	RefundStatusClosed RefundStatus = "closed"
)

// Fields 定义退款记录字段
func (Refund) Fields() []ent.Field {
	return []ent.Field{
		// 主键ID
		field.String("id").
			Unique().
			Immutable().
			Comment("退款记录ID"),

		// 关联支付订单
		field.String("payment_id").
			NotEmpty().
			Comment("关联的支付订单ID"),

		// 退款单号
		field.String("refund_no").
			NotEmpty().
			Unique().
			Comment("退款单号，用于幂等"),

		// 第三方退款单号
		field.String("third_party_refund_id").
			Optional().
			Nillable().
			Comment("第三方支付平台退款单号"),

		// 退款金额
		field.Int64("amount").
			Positive().
			Comment("退款金额，单位：分"),

		// 退款原因
		field.String("reason").
			Optional().
			Nillable().
			Comment("退款原因"),

		// 退款状态
		field.String("status").
			Default(string(RefundStatusPending)).
			Comment("退款状态"),

		// 退款时间
		field.Time("refunded_at").
			Optional().
			Nillable().
			Comment("实际退款时间"),

		// 错误信息
		field.String("error_message").
			Optional().
			Nillable().
			Comment("退款失败原因"),

		// 操作人
		field.String("operator_id").
			Optional().
			Nillable().
			Comment("操作人ID"),

		// 创建时间
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),

		// 更新时间
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("更新时间"),
	}
}

// Edges 定义退款记录关系
func (Refund) Edges() []ent.Edge {
	return []ent.Edge{
		// 关联支付订单
		edge.From("payment", PaymentOrder.Type).
			Ref("refunds").
			Field("payment_id").
			Unique().
			Required(),
	}
}

// Indexes 定义退款记录索引
func (Refund) Indexes() []ent.Index {
	return []ent.Index{
		// 支付订单ID索引
		index.Fields("payment_id"),

		// 退款单号索引
		index.Fields("refund_no").
			Unique(),

		// 状态索引
		index.Fields("status"),
	}
}

// Annotations 退款记录注解
func (Refund) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table: "refunds",
		},
	}
}

// ============================================
// User 用户实体（简化定义，用于关联）
// ============================================

// User 用户实体
type User struct {
	ent.Schema
}

// Fields 定义用户字段
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.String("email").
			Unique(),
		field.Time("created_at").
			Default(time.Now),
	}
}

// Edges 定义用户关系
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("payments", PaymentOrder.Type).
			Comment("用户的支付记录"),
	}
}
