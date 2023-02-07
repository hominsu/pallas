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
	"github.com/hominsu/pallas/pkg/srp"
)

var _ biz.UserRepo = (*userRepo)(nil)

const userCacheKeyPrefix = "user_cache_key_"

type userRepo struct {
	data *Data
	ck   map[string][]string
	sg   *singleflight.Group
	log  *log.Helper
}

// NewUserRepo .
func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	ur := &userRepo{
		data: data,
		sg:   &singleflight.Group{},
		log:  log.NewHelper(log.With(logger, "module", "data/user")),
	}
	ur.ck["Get"] = []string{"get", "user", "id"}
	ur.ck["GetByEmail"] = []string{"get", "user", "email"}
	ur.ck["List"] = []string{"list", "user"}
	ur.ck["IsAdminUser"] = []string{"is", "admin", "user", "id"}
	return ur
}

func (r *userRepo) Create(ctx context.Context, user *biz.User) (*biz.User, error) {
	m, err := r.createBuilder(user)
	if err != nil {
		return nil, v1.ErrorInternalError("internal error: %s", err)
	}
	res, err := m.Save(ctx)
	switch {
	case err == nil:
		u, er := toUser(res)
		if er != nil {
			return nil, v1.ErrorInternalError("internal error: %s", er)
		}
		return u, nil
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorAlreadyExistsError("user already exists: %s", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorInvalidArgument("invalid argument: %s", err)
	default:
		r.log.Errorf("unknown err: %v", err)
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *userRepo) Get(ctx context.Context, userId int64, userView biz.UserView) (*biz.User, error) {
	var (
		err error
		key string
		res any
	)
	id := int(userId)
	switch userView {
	case biz.UserViewViewUnspecified, biz.UserViewBasic:
		// key: user_cache_key_get_user_id:userId
		key = r.cacheKey(strconv.FormatInt(userId, 10), r.ck["Get"]...)
		res, err, _ = r.sg.Do(key, func() (any, error) {
			get := &ent.User{}
			// get cache
			er := r.data.cache.Get(ctx, key, get)
			if er != nil && errors.Is(er, cache.ErrCacheMiss) { // cache miss
				// get from db
				get, er = r.data.db.User.Get(ctx, id)
			}
			return get, er
		})
	case biz.UserViewWithEdgeIds:
		// key: user_cache_key_get_user_id_edge_ids:userId
		key = r.cacheKey(strconv.FormatInt(userId, 10), append(r.ck["Get"], "edge_ids")...)
		res, err, _ = r.sg.Do(key, func() (any, error) {
			get := &ent.User{}
			// get cache
			er := r.data.cache.Get(ctx, key, get)
			if er != nil && errors.Is(er, cache.ErrCacheMiss) { // cache miss
				// get from db
				get, er = r.data.db.User.Query().
					Where(user.ID(id)).
					WithOwnerGroup(func(query *ent.GroupQuery) {
						query.Select(group.FieldID)
						query.Select(group.FieldName)
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
			Value: res.(*ent.User),
			TTL:   r.data.conf.Cache.Ttl.AsDuration(),
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}
		return toUser(res.(*ent.User))
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFoundError("not found: %s", err)
	default: // error
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *userRepo) GetByEmail(ctx context.Context, email string, userView biz.UserView) (*biz.User, error) {
	var (
		err error
		key string
		res any
	)
	switch userView {
	case biz.UserViewViewUnspecified, biz.UserViewBasic:
		// key: user_cache_key_get_user_email:userEmail
		key = r.cacheKey(email, r.ck["GetByEmail"]...)
		res, err, _ = r.sg.Do(key, func() (any, error) {
			get := &ent.User{}
			// get cache
			er := r.data.cache.Get(ctx, key, get)
			if er != nil && errors.Is(er, cache.ErrCacheMiss) { // cache miss
				// get from db
				get, er = r.data.db.User.Query().Where(user.EmailEQ(email)).Only(ctx)
			}
			return get, er
		})
	case biz.UserViewWithEdgeIds:
		// key: user_cache_key_get_user_email_edge_ids:userEmail
		key = r.cacheKey(email, append(r.ck["GetByEmail"], "edge_ids")...)
		res, err, _ = r.sg.Do(key, func() (any, error) {
			get := &ent.User{}
			// get cache
			er := r.data.cache.Get(ctx, key, get)
			if er != nil && errors.Is(er, cache.ErrCacheMiss) { // cache miss
				// get from db
				get, er = r.data.db.User.Query().
					Where(user.EmailEQ(email)).
					WithOwnerGroup(func(query *ent.GroupQuery) {
						query.Select(group.FieldID)
						query.Select(group.FieldName)
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
			Value: res.(*ent.User),
			TTL:   r.data.conf.Cache.Ttl.AsDuration(),
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}
		return toUser(res.(*ent.User))
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFoundError("not found: %s", err)
	default: // error
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *userRepo) Update(ctx context.Context, user *biz.User) (*biz.User, error) {
	m := r.data.db.User.UpdateOneID(int(user.Id))
	m.SetEmail(user.Email)
	m.SetNickName(user.NickName)
	m.SetStorage(user.Storage)
	m.SetScore(int(user.Score))
	m.SetStatus(toEntUserStatus(user.Status))
	if user.OwnerGroup != nil {
		m.SetOwnerGroupID(int(user.OwnerGroup.Id))
	}

	// update user
	res, err := m.Save(ctx)
	switch {
	case err == nil:
		// delete indexed cache
		if err = r.deleteCache(
			ctx,
			// key: user_cache_key_get_user_id:userId
			r.cacheKey(strconv.FormatInt(int64(res.ID), 10), r.ck["Get"]...),
			// key: user_cache_key_get_user_id_edge_ids:userId
			r.cacheKey(strconv.FormatInt(int64(res.ID), 10), append(r.ck["Get"], "edge_ids")...),
			// key: user_cache_key_get_user:userEmail
			r.cacheKey(res.Email, r.ck["GetByEmail"]...),
			// key: user_cache_key_get_user_edge_ids:userEmail
			r.cacheKey(res.Email, append(r.ck["GetByEmail"], "edge_ids")...),
			// key: user_cache_key_is_admin_user_id:userId
			r.cacheKey(strconv.FormatInt(int64(res.ID), 10), r.ck["IsAdminUser"]...),
		); err != nil {
			r.log.Error(err)
		}

		// delete cache by scan redis
		if err = r.deleteKeysByScanPrefix(
			ctx,
			// match key: user_cache_key_list_user:pageSize_pageToken and key: user_cache_key_list_user_edge_ids:pageSize_pageToken
			userCacheKeyPrefix+strings.Join(r.ck["List"], "_"),
		); err != nil {
			r.log.Error(err)
		}
		return toUser(res)
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorAlreadyExistsError("user already exists: %s", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorInvalidArgument("invalid argument: %s", err)
	default:
		r.log.Errorf("unknown err: %v", err)
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *userRepo) Delete(ctx context.Context, userId int64) error {
	// get deleted user from db
	res, err := r.Get(ctx, userId, biz.UserViewBasic)
	if err != nil {
		return err
	}

	// delete user
	err = r.data.db.User.DeleteOneID(int(userId)).Exec(ctx)
	switch {
	case err == nil:
		// delete indexed cache
		if err = r.deleteCache(
			ctx,
			// key: user_cache_key_get_user_id:userId
			r.cacheKey(strconv.FormatInt(userId, 10), r.ck["Get"]...),
			// key: user_cache_key_get_user_id_edge_ids:userId
			r.cacheKey(strconv.FormatInt(userId, 10), append(r.ck["Get"], "edge_ids")...),
			// key: user_cache_key_get_user:userEmail
			r.cacheKey(res.Email, r.ck["GetByEmail"]...),
			// key: user_cache_key_get_user_edge_ids:userEmail
			r.cacheKey(res.Email, append(r.ck["GetByEmail"], "edge_ids")...),
			// key: user_cache_key_is_admin_user_id:userId
			r.cacheKey(strconv.FormatInt(userId, 10), r.ck["IsAdminUser"]...),
		); err != nil {
			r.log.Error(err)
		}

		// delete cache by scan redis
		if err = r.deleteKeysByScanPrefix(
			ctx,
			// match key: user_cache_key_list_user:pageSize_pageToken and key: user_cache_key_list_user_edge_ids:pageSize_pageToken
			userCacheKeyPrefix+strings.Join(r.ck["List"], "_"),
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

func (r *userRepo) List(
	ctx context.Context,
	pageSize int,
	pageToken string,
	userView biz.UserView,
) (*biz.UserPage, error) {
	// list users
	listQuery := r.data.db.User.Query().
		Order(ent.Asc(user.FieldID)).
		Limit(pageSize + 1)
	if pageToken != "" {
		token, er := pagination.DecodePageToken(pageToken)
		if er != nil {
			return nil, v1.ErrorDecodePageTokenError("%s", er)
		}
		listQuery = listQuery.Where(user.IDGTE(token))
	}

	var (
		err error
		key string
		res any
	)

	switch userView {
	case biz.UserViewViewUnspecified, biz.UserViewBasic:
		// key: user_cache_key_list_user:pageSize_pageToken
		key = r.cacheKey(
			strings.Join([]string{strconv.FormatInt(int64(pageSize), 10), pageToken}, "_"),
			r.ck["List"]...,
		)
		res, err, _ = r.sg.Do(key, func() (any, error) {
			var entList []*ent.User
			// get cache
			er := r.data.cache.GetSkippingLocalCache(ctx, key, &entList)
			if er != nil && errors.Is(er, cache.ErrCacheMiss) { // cache miss
				// get from db
				entList, er = listQuery.All(ctx)
			}
			return entList, er
		})
	case biz.UserViewWithEdgeIds:
		// key: user_cache_key_list_user_edge_ids:pageSize_pageToken
		key = r.cacheKey(
			strings.Join([]string{strconv.FormatInt(int64(pageSize), 10), pageToken}, "_"),
			append(r.ck["List"], "edge_ids")...,
		)
		res, err, _ = r.sg.Do(key, func() (any, error) {
			var entList []*ent.User
			// get cache
			er := r.data.cache.GetSkippingLocalCache(ctx, key, &entList)
			if er != nil && errors.Is(er, cache.ErrCacheMiss) { // cache miss
				// get from db
				entList, er = listQuery.WithOwnerGroup(func(query *ent.GroupQuery) {
					query.Select(group.FieldID)
					query.Select(group.FieldName)
				}).All(ctx)
			}
			return entList, er
		})
	default:
		return nil, v1.ErrorInvalidArgument("invalid argument: unknown view")
	}
	switch {
	case err == nil: // db hit, set cache
		entList := res.([]*ent.User)
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
				return nil, v1.ErrorEncodePageTokenError("%s", err)
			}
			entList = entList[:len(entList)-1]
		}

		userList, er := toUserList(entList)
		if er != nil {
			return nil, v1.ErrorInternalError("internal error: %s", er)
		}
		return &biz.UserPage{
			Users:         userList,
			NextPageToken: nextPageToken,
		}, nil
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFoundError("not found: %s", err)
	default: // error
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *userRepo) BatchCreate(ctx context.Context, users []*biz.User) ([]*biz.User, error) {
	if len(users) > biz.MaxBatchCreateSize {
		return nil, v1.ErrorInvalidArgument("batch size cannot be greater than %d", biz.MaxBatchCreateSize)
	}
	bulk := make([]*ent.UserCreate, len(users))
	for i, u := range users {
		var err error
		bulk[i], err = r.createBuilder(u)
		if err != nil {
			return nil, v1.ErrorInternalError("internal error: %s", err)
		}
	}
	res, err := r.data.db.User.CreateBulk(bulk...).Save(ctx)
	switch {
	case err == nil:
		userList, er := toUserList(res)
		if er != nil {
			return nil, v1.ErrorInternalError("internal error: %s", er)
		}
		return userList, nil
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorAlreadyExistsError("user already exists: %s", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorInvalidArgument("invalid argument: %s", err)
	default:
		r.log.Errorf("unknown err: %v", err)
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *userRepo) createBuilder(user *biz.User) (*ent.UserCreate, error) {
	m := r.data.db.User.Create()
	m.SetEmail(user.Email)
	m.SetNickName(user.NickName)
	m.SetSalt(user.Salt)
	m.SetVerifier(user.Verifier)
	m.SetStorage(user.Storage)
	m.SetScore(int(user.Score))
	m.SetStatus(toEntUserStatus(user.Status))
	if user.OwnerGroup != nil {
		m.SetOwnerGroupID(int(user.OwnerGroup.Id))
	}
	return m, nil
}

func (r *userRepo) IsAdminUser(ctx context.Context, userId int64) (bool, error) {
	// key: user_cache_key_is_admin_user_id:userId
	key := r.cacheKey(strconv.FormatInt(userId, 10), r.ck["IsAdminUser"]...)
	var res bool
	// get cache
	err := r.data.cache.Get(ctx, key, res)
	if err != nil && errors.Is(err, cache.ErrCacheMiss) { // cache miss
		// get from db
		res, err = r.data.db.User.Query().
			Where(user.ID(int(userId))).
			WithOwnerGroup(func(query *ent.GroupQuery) {
				query.Where(group.NameEQ("Admin"))
			}).
			Exist(ctx)
	}

	switch {
	case err == nil: // db hit, set cache
		if err = r.data.cache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   key,
			Value: res,
			TTL:   r.data.conf.Cache.Ttl.AsDuration(),
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}
		return res, nil
	default: // error
		return false, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *userRepo) CacheSRPServer(ctx context.Context, email string, server *srp.Server) error {
	err := r.data.cache.Set(&cache.Item{
		Ctx:   ctx,
		Key:   email,
		Value: server,
		TTL:   r.data.conf.Cache.SrpTtl.AsDuration(),
	})
	if err != nil {
		r.log.Errorf("cache error: %v", err)
		return v1.ErrorInternalError("cache srp error")
	}
	return nil
}

func (r *userRepo) GetSRPServer(ctx context.Context, email string) (*srp.Server, error) {
	get := &srp.Server{}
	// get cache
	err := r.data.cache.Get(ctx, email, get)
	if err != nil && errors.Is(err, cache.ErrCacheMiss) { // cache miss
		return nil, v1.ErrorSrpExpiration("srp cache expired")
	}
	return get, nil
}

func (r *userRepo) cacheKey(unique string, a ...string) string {
	s := strings.Join(a, "_")
	return userCacheKeyPrefix + s + ":" + unique
}

// deleteCache delete the cache both local cache and redis
func (r *userRepo) deleteCache(ctx context.Context, key ...string) error {
	for _, k := range key {
		if err := r.data.cache.Delete(ctx, k); err != nil {
			return v1.ErrorInternalError("delete cache error: %v", err)
		}
	}
	return nil
}

// deleteKeysByScanPrefix delete the keys by scan the prefix on redis,
// notice that this function will not delete the keys on local cache
func (r *userRepo) deleteKeysByScanPrefix(ctx context.Context, prefix ...string) error {
	for _, p := range prefix {
		iter := r.data.rdCmd.Scan(ctx, 0, p+":*", 0).Iterator()
		for iter.Next(ctx) {
			if err := r.data.rdCmd.Del(ctx, iter.Val()).Err(); err != nil {
				return v1.ErrorInternalError("delete user cache keys by scan prefix error: %v", err)
			}
		}
		if err := iter.Err(); err != nil {
			return v1.ErrorInternalError("delete user cache keys by scan prefix error: %v", err)
		}
	}
	return nil
}

func toUserStatus(e user.Status) biz.UserStatus { return biz.UserStatus(e) }

func toEntUserStatus(u biz.UserStatus) user.Status { return user.Status(u) }

func toUser(e *ent.User) (*biz.User, error) {
	u := &biz.User{}
	u.Id = int64(e.ID)
	u.GroupId = int64(e.GroupID)
	u.Email = e.Email
	u.NickName = e.NickName
	u.Salt = e.Salt
	u.Verifier = e.Verifier
	u.Storage = e.Storage
	u.Score = int64(e.Score)
	u.Status = toUserStatus(e.Status)
	u.CreateAt = e.CreatedAt
	u.UpdateAt = e.UpdatedAt
	if edg := e.Edges.OwnerGroup; edg != nil {
		u.OwnerGroup = &biz.Group{
			Id:   int64(edg.ID),
			Name: edg.Name,
		}
	}
	return u, nil
}

func toUserList(e []*ent.User) ([]*biz.User, error) {
	userList := make([]*biz.User, len(e))
	for i, entEntity := range e {
		u, err := toUser(entEntity)
		if err != nil {
			return nil, errors.New("convert to userList error")
		}
		userList[i] = u
	}
	return userList, nil
}
