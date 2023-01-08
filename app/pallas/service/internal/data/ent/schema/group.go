package schema

import (
	"entgo.io/contrib/entproto"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

// Group holds the schema definition for the Group entity.
type Group struct {
	ent.Schema
}

// Annotations of the Group
func (Group) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entproto.Message(),
		entproto.Service(),
	}
}

// Fields of the Group.
func (Group) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Unique().
			Annotations(entproto.Field(2)),
		field.Uint64("max_storage").
			Annotations(entproto.Field(3)),
		field.Bool("share_enabled").
			Annotations(entproto.Field(4)),
		field.Int("speed_limit").
			Annotations(entproto.Field(5)),
	}
}

// Mixin of the Group.
func (Group) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AnnotateFields(
			CreateTimeMixin{},
			entproto.Field(6),
		),
		mixin.AnnotateFields(
			UpdateTimeMixin{},
			entproto.Field(7),
		),
	}
}

// Edges of the Group.
func (Group) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("users", User.Type).
			Annotations(entproto.Field(8)),
	}
}

// Indexes of the Group
func (Group) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name"),
	}
}
