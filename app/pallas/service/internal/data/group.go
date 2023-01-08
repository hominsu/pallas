package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/sync/singleflight"

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
	sg   *singleflight.Group
	log  *log.Helper
}

// NewGroupRepo .
func NewGroupRepo(data *Data, logger log.Logger) biz.GroupRepo {
	return &groupRepo{
		data: data,
		sg:   &singleflight.Group{},
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
		g, er := toGroup(res)
		if er != nil {
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
		res interface{}
	)
	id := int(groupId)
	switch groupView {
	case biz.GroupViewViewUnspecified, biz.GroupViewBasic:
		res, err, _ = r.sg.Do(fmt.Sprintf("get_group_by_id_%d", id),
			func() (interface{}, error) {
				get, er := r.data.db.Group.Get(ctx, id)
				switch {
				case er == nil:
					return toGroup(get)
				case ent.IsNotFound(er):
					return nil, v1.ErrorNotFoundError("not found: %s", er)
				default:
					return nil, v1.ErrorUnknownError("unknown error: %s", er)
				}
			})
	case biz.GroupViewWithEdgeIds:
		res, err, _ = r.sg.Do(fmt.Sprintf("get_group_by_id_%d_with_edge_ids", id),
			func() (interface{}, error) {
				get, er := r.data.db.Group.Query().
					Where(group.ID(id)).
					WithUsers(func(query *ent.UserQuery) {
						query.Select(user.FieldID)
						query.Select(user.FieldNickName)
						query.Select(user.FieldStatus)
					}).
					Only(ctx)
				switch {
				case er == nil:
					return toGroup(get)
				case ent.IsNotFound(er):
					return nil, v1.ErrorNotFoundError("not found: %s", er)
				default:
					return nil, v1.ErrorUnknownError("unknown error: %s", er)
				}
			})
	default:
		return nil, v1.ErrorInvalidArgument("invalid argument: unknown view")
	}
	if err != nil {
		return nil, err
	}
	return res.(*biz.Group), nil
}

func (r *groupRepo) Update(ctx context.Context, group *biz.Group) (*biz.Group, error) {
	m := r.data.db.Group.UpdateOneID(int(group.Id))
	m.SetName(group.Name)
	m.SetMaxStorage(group.MaxStorage)
	m.SetShareEnabled(group.ShareEnable)
	m.SetSpeedLimit(int(group.SpeedLimit))
	m.SetUpdatedAt(time.Now())
	for _, u := range group.Users {
		m.AddUserIDs(int(u.Id))
	}
	res, err := m.Save(ctx)
	switch {
	case err == nil:
		g, er := toGroup(res)
		if er != nil {
			return nil, v1.ErrorInternalError("internal error: %s", er)
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
	id := int(groupId)
	err = r.data.db.Group.DeleteOneID(id).Exec(ctx)
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

func (r *groupRepo) List(
	ctx context.Context,
	pageSize int,
	pageToken string,
	groupView biz.GroupView,
) (*biz.GroupPage, error) {
	// list groups
	var (
		err     error
		entList []*ent.Group
	)
	listQuery := r.data.db.Group.Query().
		Order(ent.Asc(group.FieldID)).
		Limit(pageSize + 1)
	if pageToken != "" {
		token, er := pagination.DecodePageToken(pageToken)
		if er != nil {
			return nil, v1.ErrorDecodePageTokenError("%s", er)
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
		groupList, er := toGroupList(entList)
		if er != nil {
			return nil, v1.ErrorInternalError("internal error: %s", er)
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
	for i, g := range groups {
		var err error
		bulk[i], err = r.createBuilder(g)
		if err != nil {
			return nil, v1.ErrorInternalError("internal error: %s", err)
		}
	}
	res, err := r.data.db.Group.CreateBulk(bulk...).Save(ctx)
	switch {
	case err == nil:
		groupList, er := toGroupList(res)
		if er != nil {
			return nil, v1.ErrorInternalError("internal error: %s", er)
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
	for _, u := range group.Users {
		m.AddUserIDs(int(u.Id))
	}
	return m, nil
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
	groupList := make([]*biz.Group, len(e))
	for i, entEntity := range e {
		g, err := toGroup(entEntity)
		if err != nil {
			return nil, errors.New("convert to groupList error")
		}
		groupList[i] = g
	}
	return groupList, nil
}
