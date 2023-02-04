package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Setting holds the schema definition for the Setting entity.
type Setting struct {
	ent.Schema
}

// Fields of the Setting.
func (Setting) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Unique(),
		field.String("value"),
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
				"cron"),
	}
}

// Mixin of the Setting.
func (Setting) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CreateTimeMixin{},
		UpdateTimeMixin{},
	}
}

// Edges of the Setting.
func (Setting) Edges() []ent.Edge {
	return nil
}
