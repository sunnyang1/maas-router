package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// RouterRule 路由规则模型
type RouterRule struct {
	ent.Schema
}

// Annotations 路由规则表注解
func (RouterRule) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "router_rules"},
	}
}

// Fields 路由规则字段定义
func (RouterRule) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(100).
			Comment("规则名称"),
		field.String("description").
			Optional().
			MaxLen(500).
			Comment("规则描述"),
		field.Int("priority").
			Default(0).
			Comment("优先级，数值越大优先级越高"),
		field.JSON("condition", map[string]interface{}{}).
			Comment("匹配条件，JSON格式"),
		field.JSON("action", map[string]interface{}{}).
			Comment("执行动作，JSON格式"),
		field.Bool("is_active").
			Default(true).
			Comment("是否启用"),
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

// Edges 路由规则关系定义
func (RouterRule) Edges() []ent.Edge {
	return []ent.Edge{
		// 路由规则暂无直接关联
	}
}

// Indexes 路由规则索引定义
func (RouterRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("priority"),
		index.Fields("is_active"),
	}
}
