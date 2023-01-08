package schema

import (
	"entgo.io/contrib/entproto"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// Setting holds the schema definition for the Setting entity.
type Setting struct {
	ent.Schema
}

// Annotations of the Setting
func (Setting) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entproto.Message(),
		entproto.Service(),
	}
}

// Fields of the Setting.
func (Setting) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Unique().
			Annotations(entproto.Field(2)),
		field.String("value").
			Annotations(entproto.Field(3)),
		field.Enum("type").
			Values(
				"basic",
				"register",
				"login",
				"mail",
				"captcha",
				"pwa",
				"timeout",
				"upload",
				"share",
				"avatar",
				"payment",
				"score",
				"task",
				"auth",
				"cron").
			Annotations(
				entproto.Field(4),
				entproto.Enum(map[string]int32{
					"basic":    1,
					"register": 2,
					"login":    3,
					"mail":     4,
					"captcha":  5,
					"pwa":      6,
					"timeout":  7,
					"upload":   8,
					"share":    9,
					"avatar":   10,
					"payment":  11,
					"score":    12,
					"task":     13,
					"auth":     14,
					"cron":     15,
				}),
			),
	}
}

// Mixin of the Setting.
func (Setting) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AnnotateFields(
			CreateTimeMixin{},
			entproto.Field(5),
		),
		mixin.AnnotateFields(
			UpdateTimeMixin{},
			entproto.Field(6),
		),
	}
}

// Edges of the Setting.
func (Setting) Edges() []ent.Edge {
	return nil
}
