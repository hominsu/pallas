package schema

import (
	"time"

	"entgo.io/contrib/entproto"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Annotations of the User
func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entproto.Message(),
		entproto.Service(),
	}
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Int("group_id").
			Annotations(entproto.Field(2)),
		field.String("email").
			Unique().
			Annotations(entproto.Field(3)),
		field.String("nick_name").
			Annotations(entproto.Field(4)),
		field.String("password_hash").
			Sensitive().
			Annotations(entproto.Field(5)),
		field.Uint64("storage").
			Annotations(entproto.Field(6)),
		field.Int("score").
			Default(0).
			Annotations(entproto.Field(7)),
		field.Enum("status").
			Values("non_activated", "active", "banned", "overuse_baned").
			Default("non_activated").
			Annotations(
				entproto.Field(8),
				entproto.Enum(map[string]int32{
					"non_activated": 0,
					"active":        1,
					"banned":        2,
					"overuse_baned": 3,
				}),
			),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Annotations(entproto.Field(9)),
		field.Time("updated_at").
			Default(time.Now).
			Annotations(entproto.Field(10)),
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
			Field("group_id").
			Annotations(entproto.Field(11)),
	}
}

// Indexes of the User
func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email").Unique(),
		index.Fields("group_id"),
	}
}
