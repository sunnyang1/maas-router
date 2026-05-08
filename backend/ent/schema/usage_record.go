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

// UsageRecord 使用记录模型
type UsageRecord struct {
	ent.Schema
}

// Annotations 使用记录表注解
func (UsageRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "usage_records"},
	}
}

// Fields 使用记录字段定义
func (UsageRecord) Fields() []ent.Field {
	return []ent.Field{
		field.String("request_id").
			Unique().
			NotEmpty().
			MaxLen(64).
			Comment("请求唯一标识"),
		field.Int64("user_id").
			Comment("用户ID"),
		field.Int64("api_key_id").
			Optional().
			Nillable().
			Comment("API Key ID"),
		field.Int64("account_id").
			Optional().
			Nillable().
			Comment("上游账号ID"),
		field.Int64("group_id").
			Optional().
			Nillable().
			Comment("分组ID"),
		field.String("model").
			NotEmpty().
			MaxLen(100).
			Comment("使用的模型名称"),
		field.String("platform").
			NotEmpty().
			MaxLen(50).
			Comment("平台类型"),
		field.Int32("prompt_tokens").
			Default(0).
			Comment("输入Token数量"),
		field.Int32("completion_tokens").
			Default(0).
			Comment("输出Token数量"),
		field.Int32("total_tokens").
			Default(0).
			Comment("总Token数量"),
		field.Int32("latency_ms").
			Optional().
			Nillable().
			Comment("请求总延迟（毫秒）"),
		field.Int32("first_token_ms").
			Optional().
			Nillable().
			Comment("首Token延迟（毫秒）"),
		field.Float("cost").
			Default(0).
			Precision(18).
			Scale(6).
			Comment("请求费用"),
		field.Enum("status").
			Values("success", "failed", "timeout").
			Comment("请求状态: success-成功, failed-失败, timeout-超时"),
		field.String("error_message").
			Optional().
			MaxLen(2000).
			Comment("错误信息"),
		field.String("client_ip").
			Optional().
			MaxLen(45).
			Comment("客户端IP地址"),
		field.String("user_agent").
			Optional().
			MaxLen(500).
			Comment("客户端User-Agent"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
		// 复杂度分析字段
		field.Float("complexity_score").
			Optional().
			Default(0).
			Comment("复杂度评分 φ(x) ∈ [0,1]"),
		field.String("complexity_level").
			Optional().
			MaxLen(20).
			Comment("复杂度级别: simple/medium/complex/expert"),
		field.String("routing_tier").
			Optional().
			MaxLen(20).
			Comment("路由层级: economy/standard/premium"),
		field.String("complexity_model").
			Optional().
			MaxLen(100).
			Comment("复杂度引擎推荐的模型"),
		field.Float("cost_saving_ratio").
			Optional().
			Default(0).
			Comment("成本节省比例"),
		field.String("quality_risk").
			Optional().
			MaxLen(20).
			Comment("质量风险: low/medium/high"),
		field.Bool("was_upgraded").
			Optional().
			Default(false).
			Comment("是否被自动升级"),
	}
}

// Edges 使用记录关系定义
func (UsageRecord) Edges() []ent.Edge {
	return []ent.Edge{
		// 使用记录关联到用户
		edge.From("user", User.Type).
			Ref("usage_records").
			Unique().
			Field("user_id"),
		// 使用记录关联到API Key
		edge.From("api_key", APIKey.Type).
			Ref("usage_records").
			Unique().
			Field("api_key_id"),
	}
}

// Indexes 使用记录索引定义
func (UsageRecord) Indexes() []ent.Index {
	return []ent.Index{
		// 唯一索引：请求ID（用于幂等性检查）
		index.Fields("request_id").Unique(),
		// 普通索引：用户ID（用于查询用户使用记录）
		index.Fields("user_id"),
		// 普通索引：创建时间（用于按时间范围查询）
		index.Fields("created_at"),
		// 复合索引：用户ID和创建时间（最常用的查询组合）
		index.Fields("user_id", "created_at"),
		// 普通索引：API Key ID（用于统计 API Key 使用）
		index.Fields("api_key_id"),
		// 普通索引：账号ID（用于统计账号使用）
		index.Fields("account_id"),
		// 普通索引：分组ID（用于统计分组使用）
		index.Fields("group_id"),
		// 普通索引：平台（用于按平台统计）
		index.Fields("platform"),
		// 普通索引：状态（用于按状态筛选）
		index.Fields("status"),
		// 复合索引：平台和创建时间（用于平台统计）
		index.Fields("platform", "created_at"),
		// 普通索引：模型（用于按模型统计）
		index.Fields("model"),
		// 复杂度分析索引
		index.Fields("complexity_level"),
		index.Fields("routing_tier", "created_at"),
		index.Fields("complexity_level", "routing_tier"),
	}
}
