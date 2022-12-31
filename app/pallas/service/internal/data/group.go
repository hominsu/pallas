package data

import (
	"context"
	"errors"
	"time"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/group"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/user"
	"github.com/hominsu/pallas/pkg/pagination"
)

var _ biz.GroupRepo = (*groupRepo)(nil)

type groupRepo struct {
	data *Data
	log  *log.Helper
}

func toGroup(e *ent.Group) (*biz.Group, error) {
	g := &biz.Group{}
	g.Id = int64(e.ID)
	g.Name = e.Name
	g.MaxStorage = e.MaxStorage
	g.ShareEnable = e.ShareEnabled
	g.SpeedLimit = int64(e.SpeedLimit)
	for _, edg := range e.Edges.Users {
		g.Users = append(g.Users, &biz.User{
			Id:       int64(edg.ID),
			NickName: edg.NickName,
			Status:   toUserStatus(edg.Status),
		})
	}
	return g, nil
}

func toGroupList(e []*ent.Group) ([]*biz.Group, error) {
	var groupList []*biz.Group
	for _, entEntity := range e {
		group, err := toGroup(entEntity)
		if err != nil {
			return nil, errors.New("convert to groupList error")
		}
		groupList = append(groupList, group)
	}
	return groupList, nil
}

// NewGroupRepo .
func NewGroupRepo(data *Data, logger log.Logger) biz.GroupRepo {
	return &groupRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "data/group")),
	}
}

func (r *groupRepo) Create(ctx context.Context, group *biz.Group) (*biz.Group, error) {
	m, err := r.createBuilder(group)
	if err != nil {
		return nil, v1.ErrorInternalError("internal error: %s", err)
	}
	res, err := m.Save(ctx)
	switch {
	case err == nil:
		g, err := toGroup(res)
		if err != nil {
			return nil, v1.ErrorInternalError("internal error: %s", err)
		}
		return g, nil
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorAlreadyExistsError("group already exists: %s", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorInvalidArgument("invalid argument: %s", err)
	default:
		r.log.Errorf("unknown err: %v", err)
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *groupRepo) Get(ctx context.Context, groupId int64, groupView biz.GroupView) (*biz.Group, error) {
	var (
		err error
		get *ent.Group
	)
	id := int(groupId)
	switch groupView {
	case biz.GroupViewViewUnspecified, biz.GroupViewBasic:
		get, err = r.data.db.Group.Get(ctx, id)
	case biz.GroupViewWithEdgeIds:
		get, err = r.data.db.Group.Query().
			Where(group.ID(id)).
			WithUsers(func(query *ent.UserQuery) {
				query.Select(user.FieldID)
				query.Select(user.FieldNickName)
				query.Select(user.FieldStatus)
			}).
			Only(ctx)
	default:
		return nil, v1.ErrorInvalidArgument("invalid argument: unknown view")
	}
	switch {
	case err == nil:
		return toGroup(get)
	case ent.IsNotFound(err):
		return nil, v1.ErrorNotFoundError("not found: %s", err)
	default:
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *groupRepo) Update(ctx context.Context, group *biz.Group) (*biz.Group, error) {
	m := r.data.db.Group.UpdateOneID(int(group.Id))
	m.SetName(group.Name)
	m.SetMaxStorage(group.MaxStorage)
	m.SetShareEnabled(group.ShareEnable)
	m.SetSpeedLimit(int(group.SpeedLimit))
	m.SetUpdatedAt(time.Now())
	for _, user := range group.Users {
		m.AddUserIDs(int(user.Id))
	}
	res, err := m.Save(ctx)
	switch {
	case err == nil:
		g, err := toGroup(res)
		if err != nil {
			return nil, v1.ErrorInternalError("internal error: %s", err)
		}
		return g, nil
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorAlreadyExistsError("group already exists: %s", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorInvalidArgument("invalid argument: %s", err)
	default:
		r.log.Errorf("unknown err: %v", err)
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *groupRepo) Delete(ctx context.Context, groupId int64) error {
	var err error
	err = r.data.db.Group.DeleteOneID(int(groupId)).Exec(ctx)
	switch {
	case err == nil:
		return nil
	case ent.IsNotFound(err):
		return v1.ErrorNotFoundError("not found: %s", err)
	default:
		r.log.Errorf("unknown err: %v", err)
		return v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *groupRepo) List(ctx context.Context, pageSize int, pageToken string, groupView biz.GroupView) (*biz.GroupPage, error) {
	var (
		err     error
		entList []*ent.Group
	)
	listQuery := r.data.db.Group.Query().
		Order(ent.Asc(group.FieldID)).
		Limit(pageSize + 1)
	if pageToken != "" {
		token, err := pagination.DecodePageToken(pageToken)
		if err != nil {
			return nil, v1.ErrorDecodePageTokenError("%s", err)
		}
		listQuery = listQuery.Where(group.IDGTE(token))
	}
	switch groupView {
	case biz.GroupViewViewUnspecified, biz.GroupViewBasic:
		entList, err = listQuery.All(ctx)
	case biz.GroupViewWithEdgeIds:
		entList, err = listQuery.
			WithUsers(func(query *ent.UserQuery) {
				query.Select(user.FieldID)
				query.Select(user.FieldNickName)
				query.Select(user.FieldStatus)
			}).
			All(ctx)
	}
	switch {
	case err == nil:
		var nextPageToken string
		if len(entList) == pageSize+1 {
			nextPageToken, err = pagination.EncodePageToken(entList[len(entList)-1].ID)
			if err != nil {
				return nil, v1.ErrorEncodePageTokenError("%s", err)
			}
			entList = entList[:len(entList)-1]
		}
		groupList, err := toGroupList(entList)
		if err != nil {
			return nil, v1.ErrorInternalError("internal error: %s", err)
		}
		return &biz.GroupPage{
			Groups:        groupList,
			NextPageToken: nextPageToken,
		}, nil
	default:
		r.log.Errorf("unknown err: %v", err)
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *groupRepo) BatchCreate(ctx context.Context, groups []*biz.Group) ([]*biz.Group, error) {
	if len(groups) > biz.MaxBatchCreateSize {
		return nil, v1.ErrorInvalidArgument("batch size cannot be greater than %d", biz.MaxBatchCreateSize)
	}
	bulk := make([]*ent.GroupCreate, len(groups))
	for i, group := range groups {
		var err error
		bulk[i], err = r.createBuilder(group)
		if err != nil {
			return nil, v1.ErrorInternalError("internal error: %s", err)
		}
	}
	res, err := r.data.db.Group.CreateBulk(bulk...).Save(ctx)
	switch {
	case err == nil:
		groupList, err := toGroupList(res)
		if err != nil {
			return nil, v1.ErrorInternalError("internal error: %s", err)
		}
		return groupList, nil
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorAlreadyExistsError("group already exists: %s", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorInvalidArgument("invalid argument: %s", err)
	default:
		r.log.Errorf("unknown err: %v", err)
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *groupRepo) createBuilder(group *biz.Group) (*ent.GroupCreate, error) {
	m := r.data.db.Group.Create()
	m.SetName(group.Name)
	m.SetMaxStorage(group.MaxStorage)
	m.SetShareEnabled(group.ShareEnable)
	m.SetSpeedLimit(int(group.SpeedLimit))
	now := time.Now()
	m.SetCreatedAt(now)
	m.SetUpdatedAt(now)
	for _, user := range group.Users {
		m.AddUserIDs(int(user.Id))
	}
	return m, nil
}
