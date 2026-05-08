package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// AccountGroup 账号-分组关联模型
type AccountGroup struct {
	ent.Schema
}

// Annotations 账号-分组关联表注解
func (AccountGroup) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "account_groups"},
	}
}

// Fields 账号-分组关联字段定义
func (AccountGroup) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_id").
			Comment("上游账号ID"),
		field.Int64("group_id").
			Comment("分组ID"),
	}
}

// Edges 账号-分组关联关系定义
func (AccountGroup) Edges() []ent.Edge {
	return []ent.Edge{
		// 关联到上游账号
		edge.To("account", Account.Type).
			Unique().
			Required().
			Field("account_id"),
		// 关联到分组
		edge.To("group", Group.Type).
			Unique().
			Required().
			Field("group_id"),
	}
}
