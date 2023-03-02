package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("group_id"),
		field.String("email").
			Unique(),
		field.String("nick_name"),
		field.Bytes("salt").
			Sensitive(),
		field.Bytes("verifier").
			Sensitive(),
		field.Uint64("storage"),
		field.Int64("score").
			Default(0),
		field.Enum("status").
			Values("non_activated", "active", "banned", "overuse_baned").
			Default("non_activated"),
	}
}

// Mixin of the User.
func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CreateTimeMixin{},
		UpdateTimeMixin{},
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner_group", Group.Type).
			Ref("users").
			Unique().
			// user cannot be created without its group
			Required().
			Field("group_id"),
	}
}

// Indexes of the User
func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email").Unique(),
		index.Fields("group_id"),
	}
}
