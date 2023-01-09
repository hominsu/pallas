package data

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/cache/v8"
	"golang.org/x/sync/singleflight"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/group"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/user"
	"github.com/hominsu/pallas/pkg/pagination"
)

var _ biz.GroupRepo = (*groupRepo)(nil)

const groupCacheKey = "group_cache_key_"

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
		key string
		res interface{}
	)
	id := int(groupId)
	switch groupView {
	case biz.GroupViewViewUnspecified, biz.GroupViewBasic:
		// key: group_cache_key_get_group_id:groupId
		key = r.cacheKeyPrefix(strconv.FormatInt(groupId, 10), "get", "group", "id")
		res, err, _ = r.sg.Do(key, func() (interface{}, error) {
			get := &ent.Group{}
			// get cache
			er := r.data.cache.Get(ctx, key, get)
			if er != nil && errors.Is(er, cache.ErrCacheMiss) { // cache miss
				// get from db
				get, er = r.data.db.Group.Get(ctx, id)
			}
			return get, er
		})
	case biz.GroupViewWithEdgeIds:
		// key: group_cache_key_get_group_id:groupId
		key = r.cacheKeyPrefix(strconv.FormatInt(groupId, 10), "get", "group", "id", "edge_ids")
		res, err, _ = r.sg.Do(key, func() (interface{}, error) {
			get := &ent.Group{}
			// get cache
			er := r.data.cache.Get(ctx, key, get)
			if er != nil && errors.Is(er, cache.ErrCacheMiss) { // cache miss
				// get from db
				get, er = r.data.db.Group.Query().
					Where(group.ID(id)).
					WithUsers(func(query *ent.UserQuery) {
						query.Select(user.FieldID)
						query.Select(user.FieldNickName)
						query.Select(user.FieldStatus)
					}).
					Only(ctx)
			}
			return get, er
		})
	default:
		return nil, v1.ErrorInvalidArgument("invalid argument: unknown view")
	}
	switch {
	case err == nil: // db hit, set cache
		if err = r.data.cache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   key,
			Value: res.(*ent.Group),
			TTL:   r.data.conf.Redis.CacheExpiration.AsDuration(),
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}
		return toGroup(res.(*ent.Group))
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFoundError("not found: %s", err)
	default: // db error
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
	for _, u := range group.Users {
		m.AddUserIDs(int(u.Id))
	}

	// update group
	res, err := m.Save(ctx)
	switch {
	case err == nil:
		// delete id indexed cache
		if err = r.flushKeysByPrefix(
			ctx,
			// key: group_cache_key_get_group_id:groupId
			r.cacheKeyPrefix(strconv.FormatInt(group.Id, 10), "get", "group", "id"),
			// key: group_cache_key_get_group_id:groupId
			r.cacheKeyPrefix(strconv.FormatInt(group.Id, 10), "get", "group", "id", "edge_ids"),
			// match key: user_cache_key_list_user:pageSize_pageToken and key: user_cache_key_list_user_edge_ids:pageSize_pageToken
			groupCacheKey+"list_group",
		); err != nil {
			r.log.Error(err)
		}
		return toGroup(res)
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
		// delete id indexed cache
		if err = r.flushKeysByPrefix(
			ctx,
			// key: group_cache_key_get_group_id:groupId
			r.cacheKeyPrefix(strconv.FormatInt(groupId, 10), "get", "group", "id"),
			// key: group_cache_key_get_group_id:groupId
			r.cacheKeyPrefix(strconv.FormatInt(groupId, 10), "get", "group", "id", "edge_ids"),
			// match key: user_cache_key_list_user:pageSize_pageToken and key: user_cache_key_list_user_edge_ids:pageSize_pageToken
			groupCacheKey+"list_group",
		); err != nil {
			r.log.Error(err)
		}
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

	var (
		err error
		key string
		res interface{}
	)

	switch groupView {
	case biz.GroupViewViewUnspecified, biz.GroupViewBasic:
		// key: group_cache_key_list_group:pageSize_pageToken
		key = r.cacheKeyPrefix(strconv.FormatInt(int64(pageSize), 10)+pageToken, "list", "group")
		res, err, _ = r.sg.Do(key, func() (interface{}, error) {
			var entList []*ent.Group
			// get cache
			er := r.data.cache.Get(ctx, key, entList)
			if er != nil && errors.Is(er, cache.ErrCacheMiss) { // cache miss
				// get from db
				entList, er = listQuery.All(ctx)
			}
			return entList, er
		})
	case biz.GroupViewWithEdgeIds:
		// key: group_cache_key_list_group:pageSize_pageToken
		key = r.cacheKeyPrefix(strconv.FormatInt(int64(pageSize), 10)+pageToken, "list", "group", "edge_ids")
		res, err, _ = r.sg.Do(key, func() (interface{}, error) {
			var entList []*ent.Group
			// get cache
			er := r.data.cache.Get(ctx, key, entList)
			if er != nil && errors.Is(er, cache.ErrCacheMiss) { // cache miss
				// get from db
				entList, er = listQuery.
					WithUsers(func(query *ent.UserQuery) {
						query.Select(user.FieldID)
						query.Select(user.FieldNickName)
						query.Select(user.FieldStatus)
					}).
					All(ctx)
			}
			return entList, er
		})
	}
	switch {
	case err == nil: // db hit, set cache
		entList := res.([]*ent.Group)
		if err = r.data.cache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   key,
			Value: entList,
			TTL:   r.data.conf.Redis.CacheExpiration.AsDuration(),
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}

		// generate next page token
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
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFoundError("not found: %s", err)
	default: // error
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

func (r *groupRepo) cacheKeyPrefix(unique string, a ...string) string {
	s := strings.Join(a, "_")
	return groupCacheKey + s + ":" + unique
}

func (r *groupRepo) flushKeysByPrefix(ctx context.Context, prefix ...string) error {
	for _, p := range prefix {
		iter := r.data.rdCmd.Scan(ctx, 0, p+":*", 0).Iterator()
		for iter.Next(ctx) {
			if err := r.data.rdCmd.Del(ctx, iter.Val()).Err(); err != nil {
				return v1.ErrorInternalError("flush group cache keys by prefix error: %v", err)
			}
		}
		if err := iter.Err(); err != nil {
			return v1.ErrorInternalError("flush group cache keys by prefix error: %v", err)
		}
	}
	return nil
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
