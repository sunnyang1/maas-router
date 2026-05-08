package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Announcement 公告模型
type Announcement struct {
	ent.Schema
}

// Annotations 公告表注解
func (Announcement) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "announcements"},
	}
}

// Fields 公告字段定义
func (Announcement) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			NotEmpty().
			MaxLen(200).
			Comment("公告标题"),
		field.Text("content").
			Comment("公告内容"),
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

// Edges 公告关系定义
func (Announcement) Edges() []ent.Edge {
	return []ent.Edge{
		// 公告暂无直接关联
	}
}
