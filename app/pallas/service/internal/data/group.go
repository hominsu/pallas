package data

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/cache/v9"
	"golang.org/x/sync/singleflight"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/group"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/user"
	"github.com/hominsu/pallas/pkg/pagination"
)

var _ biz.GroupRepo = (*groupRepo)(nil)

const groupCacheKeyPrefix = "group_cache_key_"

type groupRepo struct {
	data *Data
	ck   map[string][]string
	sg   *singleflight.Group
	log  *log.Helper
}

// NewGroupRepo .
func NewGroupRepo(data *Data, logger log.Logger) biz.GroupRepo {
	gr := &groupRepo{
		data: data,
		sg:   &singleflight.Group{},
		log:  log.NewHelper(log.With(logger, "module", "data/group")),
	}
	gr.ck = make(map[string][]string)
	gr.ck["Get"] = []string{"get", "group", "id"}
	gr.ck["GetByName"] = []string{"get", "group", "name"}
	gr.ck["List"] = []string{"list", "group"}
	return gr
}

func (r *groupRepo) Create(ctx context.Context, group *biz.Group) (*biz.Group, error) {
	m := r.createBuilder(group)
	res, err := m.Save(ctx)
	switch {
	case err == nil:
		g, tErr := toGroup(res)
		if tErr != nil {
			return nil, v1.ErrorInternal("internal error: %v", tErr)
		}
		return g, nil
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorConflict("group already exists: %v", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorConflict("invalid argument: %v", err)
	default:
		return nil, v1.ErrorUnknown("unknown error: %v", err)
	}
}

func (r *groupRepo) Get(ctx context.Context, groupId int64, groupView biz.GroupView) (*biz.Group, error) {
	var (
		err error
		key string
		res any
	)
	switch groupView {
	case biz.GroupViewViewUnspecified, biz.GroupViewBasic:
		// key: group_cache_key_get_group_id:groupId
		key = r.cacheKey(strconv.FormatInt(groupId, 10), r.ck["Get"]...)
		res, err, _ = r.sg.Do(key, func() (any, error) {
			get := &ent.Group{}
			// get cache
			cErr := r.data.cache.Get(ctx, key, get)
			if cErr != nil && errors.Is(cErr, cache.ErrCacheMiss) { // cache miss
				// get from db
				get, cErr = r.data.db.Group.Get(ctx, groupId)
			}
			return get, cErr
		})
	case biz.GroupViewWithEdgeIds:
		// key: group_cache_key_get_group_id_edge_ids:groupId
		key = r.cacheKey(strconv.FormatInt(groupId, 10), append(r.ck["Get"], "edge_ids")...)
		res, err, _ = r.sg.Do(key, func() (any, error) {
			get := &ent.Group{}
			// get cache
			cErr := r.data.cache.Get(ctx, key, get)
			if cErr != nil && errors.Is(cErr, cache.ErrCacheMiss) { // cache miss
				// get from db
				get, cErr = r.data.db.Group.Query().
					Where(group.ID(groupId)).
					WithUsers(func(query *ent.UserQuery) {
						query.Select(user.FieldID)
						query.Select(user.FieldNickName)
						query.Select(user.FieldStatus)
					}).
					Only(ctx)
			}
			return get, cErr
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
			TTL:   r.data.conf.Cache.Ttl.AsDuration(),
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}
		return toGroup(res.(*ent.Group))
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFound("group not found: %v", err)
	default: // db error
		return nil, v1.ErrorUnknown("unknown error: %v", err)
	}
}

func (r *groupRepo) GetByName(ctx context.Context, name string, groupView biz.GroupView) (*biz.Group, error) {
	var (
		err error
		key string
		res any
	)
	switch groupView {
	case biz.GroupViewViewUnspecified, biz.GroupViewBasic:
		// key: group_cache_key_get_group_name:groupName
		key = r.cacheKey(name, r.ck["GetByName"]...)
		res, err, _ = r.sg.Do(key, func() (any, error) {
			get := &ent.Group{}
			// get cache
			cErr := r.data.cache.Get(ctx, key, get)
			if cErr != nil && errors.Is(cErr, cache.ErrCacheMiss) { // cache miss
				// get from db
				get, cErr = r.data.db.Group.Query().
					Where(group.NameEQ(name)).
					Only(ctx)
			}
			return get, cErr
		})
	case biz.GroupViewWithEdgeIds:
		// key: group_cache_key_get_group_name_edge_ids:groupName
		key = r.cacheKey(name, append(r.ck["GetByName"], "edge_ids")...)
		res, err, _ = r.sg.Do(key, func() (any, error) {
			get := &ent.Group{}
			// get cache
			cErr := r.data.cache.Get(ctx, key, get)
			if cErr != nil && errors.Is(cErr, cache.ErrCacheMiss) { // cache miss
				// get from db
				get, cErr = r.data.db.Group.Query().
					Where(group.NameEQ(name)).
					WithUsers(func(query *ent.UserQuery) {
						query.Select(user.FieldID)
						query.Select(user.FieldNickName)
						query.Select(user.FieldStatus)
					}).
					Only(ctx)
			}
			return get, cErr
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
			TTL:   r.data.conf.Cache.Ttl.AsDuration(),
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}
		return toGroup(res.(*ent.Group))
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFound("group not found: %v", err)
	default: // db error
		return nil, v1.ErrorUnknown("unknown error: %v", err)
	}
}

func (r *groupRepo) Update(ctx context.Context, group *biz.Group) (*biz.Group, error) {
	m := r.data.db.Group.UpdateOneID(group.Id)
	m.SetName(group.Name)
	m.SetMaxStorage(group.MaxStorage)
	m.SetShareEnabled(group.ShareEnable)
	m.SetSpeedLimit(group.SpeedLimit)
	for _, u := range group.Users {
		m.AddUserIDs(u.Id)
	}

	// update group
	res, err := m.Save(ctx)
	switch {
	case err == nil:
		// delete indexed cache
		if err = r.deleteCache(
			ctx,
			// key: group_cache_key_get_group_id:groupId
			r.cacheKey(strconv.FormatInt(group.Id, 10), r.ck["Get"]...),
			// key: group_cache_key_get_group_id_edge_ids:groupId
			r.cacheKey(strconv.FormatInt(group.Id, 10), append(r.ck["Get"], "edge_ids")...),
			// key: group_cache_key_get_group_name:groupName
			r.cacheKey(group.Name, r.ck["GetByName"]...),
			// key: group_cache_key_get_group_name_edge_ids:groupName
			r.cacheKey(group.Name, append(r.ck["GetByName"], "edge_ids")...),
		); err != nil {
			// TODO: delete again using the asynchronous queue
			r.log.Error(err)
		}
		// delete cache by scan redis
		if err = r.deleteKeysByScanPrefix(ctx,
			// match key: group_cache_key_list_group:pageSize_pageToken and
			// key: group_cache_key_list_group_edge_ids:pageSize_pageToken
			groupCacheKeyPrefix+strings.Join(r.ck["List"], "_"),
		); err != nil {
			// TODO: delete again using the asynchronous queue
			r.log.Error(err)
		}
		return toGroup(res)
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorConflict("group already exists: %v", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorConflict("invalid argument: %v", err)
	default:
		return nil, v1.ErrorUnknown("unknown error: %v", err)
	}
}

func (r *groupRepo) Delete(ctx context.Context, groupId int64) error {
	// get deleted group from db
	res, err := r.Get(ctx, groupId, biz.GroupViewBasic)
	if err != nil {
		return err
	}

	err = r.data.db.Group.DeleteOneID(res.Id).Exec(ctx)
	switch {
	case err == nil:
		// delete indexed cache
		if err = r.deleteCache(
			ctx,
			// key: group_cache_key_get_group_id:groupId
			r.cacheKey(strconv.FormatInt(res.Id, 10), r.ck["Get"]...),
			// key: group_cache_key_get_group_id_edge_ids:groupId
			r.cacheKey(strconv.FormatInt(res.Id, 10), append(r.ck["Get"], "edge_ids")...),
			// key: group_cache_key_get_group_name:groupName
			r.cacheKey(res.Name, r.ck["GetByName"]...),
			// key: group_cache_key_get_group_name_edge_ids:groupName
			r.cacheKey(res.Name, append(r.ck["GetByName"], "edge_ids")...),
		); err != nil {
			// TODO: delete again using the asynchronous queue
			r.log.Error(err)
		}
		// delete cache by scan redis
		if err = r.deleteKeysByScanPrefix(
			ctx,
			// match key: group_cache_key_list_group:pageSize_pageToken and
			// key: group_cache_key_list_group_edge_ids:pageSize_pageToken
			groupCacheKeyPrefix+strings.Join(r.ck["List"], "_"),
		); err != nil {
			// TODO: delete again using the asynchronous queue
			r.log.Error(err)
		}
		return nil
	case ent.IsNotFound(err):
		return v1.ErrorNotFound("group not found: %v", err)
	default:
		return v1.ErrorUnknown("unknown error: %v", err)
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
		token, pErr := pagination.DecodePageToken(pageToken)
		if pErr != nil {
			return nil, v1.ErrorInternal("decode page token err: %v", pErr)
		}
		listQuery = listQuery.Where(group.IDGTE(token))
	}

	var (
		err error
		key string
		res any
	)

	switch groupView {
	case biz.GroupViewViewUnspecified, biz.GroupViewBasic:
		// key: group_cache_key_list_group:pageSize_pageToken
		key = r.cacheKey(
			strings.Join([]string{strconv.FormatInt(int64(pageSize), 10), pageToken}, "_"),
			r.ck["List"]...,
		)
		res, err, _ = r.sg.Do(key, func() (any, error) {
			var entList []*ent.Group
			// get cache
			cErr := r.data.cache.GetSkippingLocalCache(ctx, key, &entList)
			if cErr != nil && errors.Is(cErr, cache.ErrCacheMiss) { // cache miss
				// get from db
				entList, cErr = listQuery.All(ctx)
			}
			return entList, cErr
		})
	case biz.GroupViewWithEdgeIds:
		// key: group_cache_key_list_group:pageSize_pageToken
		key = r.cacheKey(
			strings.Join([]string{strconv.FormatInt(int64(pageSize), 10), pageToken}, "_"),
			append(r.ck["List"], "edge_ids")...,
		)
		res, err, _ = r.sg.Do(key, func() (any, error) {
			var entList []*ent.Group
			// get cache
			cErr := r.data.cache.GetSkippingLocalCache(ctx, key, &entList)
			if cErr != nil && errors.Is(cErr, cache.ErrCacheMiss) { // cache miss
				// get from db
				entList, cErr = listQuery.
					WithUsers(func(query *ent.UserQuery) {
						query.Select(user.FieldID)
						query.Select(user.FieldNickName)
						query.Select(user.FieldStatus)
					}).
					All(ctx)
			}
			return entList, cErr
		})
	}
	switch {
	case err == nil: // db hit, set cache
		entList := res.([]*ent.Group)
		if err = r.data.cache.Set(&cache.Item{
			Ctx:            ctx,
			Key:            key,
			Value:          entList,
			TTL:            r.data.conf.Cache.Ttl.AsDuration(),
			SkipLocalCache: true,
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}

		// generate next page token
		var nextPageToken string
		if len(entList) == pageSize+1 {
			nextPageToken, err = pagination.EncodePageToken(entList[len(entList)-1].ID)
			if err != nil {
				return nil, v1.ErrorInternal("encode page token error: %v", err)
			}
			entList = entList[:len(entList)-1]
		}

		groupList, tErr := toGroupList(entList)
		if tErr != nil {
			return nil, v1.ErrorInternal("internal error: %s", tErr)
		}
		return &biz.GroupPage{
			Groups:        groupList,
			NextPageToken: nextPageToken,
		}, nil
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFound("group not found: %v", err)
	default: // error
		return nil, v1.ErrorUnknown("unknown error: %v", err)
	}
}

func (r *groupRepo) BatchCreate(ctx context.Context, groups []*biz.Group) ([]*biz.Group, error) {
	if len(groups) > biz.MaxBatchCreateSize {
		return nil, v1.ErrorInvalidArgument("batch size cannot be greater than %d", biz.MaxBatchCreateSize)
	}
	bulk := make([]*ent.GroupCreate, len(groups))
	for i, g := range groups {
		bulk[i] = r.createBuilder(g)
	}
	res, err := r.data.db.Group.CreateBulk(bulk...).Save(ctx)
	switch {
	case err == nil:
		groupList, tErr := toGroupList(res)
		if tErr != nil {
			return nil, v1.ErrorInternal("internal error: %s", tErr)
		}
		return groupList, nil
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorConflict("group already exists: %v", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorConflict("invalid argument: %v", err)
	default:
		return nil, v1.ErrorUnknown("unknown error: %v", err)
	}
}

func (r *groupRepo) createBuilder(group *biz.Group) *ent.GroupCreate {
	m := r.data.db.Group.Create()
	m.SetName(group.Name)
	m.SetMaxStorage(group.MaxStorage)
	m.SetShareEnabled(group.ShareEnable)
	m.SetSpeedLimit(group.SpeedLimit)
	for _, u := range group.Users {
		m.AddUserIDs(u.Id)
	}
	return m
}

func (r *groupRepo) cacheKey(unique string, a ...string) string {
	s := strings.Join(a, "_")
	return groupCacheKeyPrefix + s + ":" + unique
}

// deleteCache delete the cache both local cache and redis
func (r *groupRepo) deleteCache(ctx context.Context, key ...string) error {
	for _, k := range key {
		if err := r.data.cache.Delete(ctx, k); err != nil {
			return v1.ErrorCacheOperation("delete cache error: %v", err)
		}
	}
	return nil
}

// deleteKeysByScanPrefix delete the keys by scan the prefix on redis,
// notice that this function will not delete the keys on local cache
func (r *groupRepo) deleteKeysByScanPrefix(ctx context.Context, prefix ...string) error {
	for _, p := range prefix {
		iter := r.data.rdCmd.Scan(ctx, 0, p+"*", 0).Iterator()
		for iter.Next(ctx) {
			if err := r.data.rdCmd.Del(ctx, iter.Val()).Err(); err != nil {
				return v1.ErrorCacheOperation("delete group cache keys by scan prefix error: %v", err)
			}
		}
		if err := iter.Err(); err != nil {
			return v1.ErrorCacheOperation("delete group cache keys by scan prefix error: %v", err)
		}
	}
	return nil
}

func toGroup(e *ent.Group) (*biz.Group, error) {
	g := &biz.Group{}
	g.Id = e.ID
	g.Name = e.Name
	g.MaxStorage = e.MaxStorage
	g.ShareEnable = e.ShareEnabled
	g.SpeedLimit = e.SpeedLimit
	for _, edg := range e.Edges.Users {
		g.Users = append(g.Users, &biz.User{
			Id:       edg.ID,
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
