package data

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/cache/v8"
	"golang.org/x/sync/singleflight"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/setting"
)

var _ biz.SettingRepo = (*settingRepo)(nil)

const settingCacheKey = "setting_cache_key_"

type settingRepo struct {
	data *Data
	sg   *singleflight.Group
	log  *log.Helper
}

// NewSettingRepo .
func NewSettingRepo(data *Data, logger log.Logger) biz.SettingRepo {
	return &settingRepo{
		data: data,
		sg:   &singleflight.Group{},
		log:  log.NewHelper(log.With(logger, "module", "data/setting")),
	}
}

func (r *settingRepo) Create(ctx context.Context, s *biz.Setting) (*biz.Setting, error) {
	m, err := r.createBuilder(s)
	if err != nil {
		return nil, v1.ErrorInternalError("internal error: %s", err)
	}
	res, err := m.Save(ctx)
	switch {
	case err != nil:
		set, er := toSetting(res)
		if er != nil {
			return nil, v1.ErrorInternalError("internal error: %s", err)
		}
		return set, nil
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorAlreadyExistsError("setting already exists: %s", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorInvalidArgument("invalid argument: %s", err)
	default:
		r.log.Errorf("unknown err: %v", err)
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *settingRepo) Get(ctx context.Context, id int64) (*biz.Setting, error) {
	// key: setting_cache_key_get_setting_id:settingId
	key := r.cacheKeyPrefix(strconv.FormatInt(id, 10), "get", "setting", "id")
	res, err, _ := r.sg.Do(key, func() (interface{}, error) {
		get := &ent.Setting{}
		// get cache
		err := r.data.cache.Get(ctx, key, get)
		if err != nil && errors.Is(err, cache.ErrCacheMiss) { // cache miss
			// get from db
			get, err = r.data.db.Setting.Get(ctx, int(id))
		}
		return get, err
	})
	switch {
	case err == nil: // db hit, set cache
		if err = r.data.cache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   key,
			Value: res.(*ent.Setting),
			TTL:   r.data.conf.Redis.CacheExpiration.AsDuration(),
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}
		return toSetting(res.(*ent.Setting))
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFoundError("not found: %s", err)
	default: // error
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *settingRepo) GetByName(ctx context.Context, name string) (*biz.Setting, error) {
	// key: setting_cache_key_get_setting_id:settingId
	key := r.cacheKeyPrefix(name, "get", "setting", "name")
	res, err, _ := r.sg.Do(key, func() (interface{}, error) {
		get := &ent.Setting{}
		// get cache
		err := r.data.cache.Get(ctx, key, get)
		if err != nil && errors.Is(err, cache.ErrCacheMiss) { // cache miss
			// get from db
			get, err = r.data.db.Setting.Query().
				Where(setting.NameEQ(name)).
				Only(ctx)
		}
		return get, err
	})
	switch {
	case err == nil: // db hit, set cache
		if err = r.data.cache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   key,
			Value: res.(*ent.User),
			TTL:   r.data.conf.Redis.CacheExpiration.AsDuration(),
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}
		return toSetting(res.(*ent.Setting))
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFoundError("not found: %s", err)
	default: // error
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *settingRepo) Update(ctx context.Context, s *biz.Setting) (*biz.Setting, error) {
	m := r.data.db.Setting.UpdateOneID(int(s.Id))
	m.SetName(s.Name)
	m.SetValue(s.Value)
	m.SetType(toEntSettingType(s.Type))
	res, err := m.Save(ctx)
	if err != nil {
		return nil, err
	}
	switch {
	case err == nil:
		// delete indexed cache
		if err = r.deleteCache(
			ctx,
			// key: setting_cache_key_get_setting_id:settingId
			r.cacheKeyPrefix(strconv.FormatInt(int64(res.ID), 10), "get", "setting", "id"),
			// key: setting_cache_key_get_setting_id:settingId
			r.cacheKeyPrefix(res.Name, "get", "setting", "name"),
			// key: setting_cache_key_list_group:all
			r.cacheKeyPrefix("all", "list", "group"),
			// key: setting_cache_key_list_group_type:settingType
			r.cacheKeyPrefix(res.Type.String(), "list", "group", "type"),
		); err != nil {
			r.log.Error(err)
		}
		return toSetting(res)
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorAlreadyExistsError("setting already exists: %s", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorInvalidArgument("invalid argument: %s", err)
	default:
		r.log.Errorf("unknown err: %v", err)
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *settingRepo) Delete(ctx context.Context, id int64) error {
	// get deleted setting from db
	res, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	err = r.data.db.Setting.DeleteOneID(int(id)).Exec(ctx)
	switch {
	case err == nil:
		// delete indexed cache
		if err = r.deleteCache(
			ctx,
			// key: setting_cache_key_get_setting_id:settingId
			r.cacheKeyPrefix(strconv.FormatInt(id, 10), "get", "setting", "id"),
			// key: setting_cache_key_get_setting_id:settingId
			r.cacheKeyPrefix(res.Name, "get", "setting", "name"),
			// key: setting_cache_key_list_group:all
			r.cacheKeyPrefix("all", "list", "group"),
			// key: setting_cache_key_list_group_type:settingType
			r.cacheKeyPrefix(res.Type.String(), "list", "group", "type"),
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

func (r *settingRepo) List(ctx context.Context) ([]*biz.Setting, error) {
	// key: setting_cache_key_list_group:all
	key := r.cacheKeyPrefix("all", "list", "group")
	res, err, _ := r.sg.Do(key, func() (interface{}, error) {
		var entList []*ent.Setting
		// get cache
		err := r.data.cache.Get(ctx, key, &entList)
		if err != nil && errors.Is(err, cache.ErrCacheMiss) { // cache miss
			// get from db
			entList, err = r.data.db.Setting.Query().All(ctx)
		}
		return entList, err
	})

	switch {
	case err == nil: // db hit, set cache
		entList := res.([]*ent.Setting)
		if err = r.data.cache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   key,
			Value: entList,
			TTL:   r.data.conf.Redis.CacheExpiration.AsDuration(),
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}
		settingList, er := toSettingsList(entList)
		if er != nil {
			return nil, v1.ErrorInternalError("internal error: %s", er)
		}
		return settingList, nil
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFoundError("not found: %s", err)
	default: // error
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *settingRepo) ListByType(ctx context.Context, t biz.SettingType) ([]*biz.Setting, error) {
	// key: setting_cache_key_list_group_type:settingType
	key := r.cacheKeyPrefix(t.String(), "list", "group", "type")
	res, err, _ := r.sg.Do(key, func() (interface{}, error) {
		var entList []*ent.Setting
		// get cache
		err := r.data.cache.Get(ctx, key, &entList)
		if err != nil && errors.Is(err, cache.ErrCacheMiss) { // cache miss
			// get from db
			entList, err = r.data.db.Setting.Query().
				Where(setting.TypeEQ(toEntSettingType(t))).
				All(ctx)
		}
		return entList, err
	})
	switch {
	case err == nil: // db hit, set cache
		entList := res.([]*ent.Setting)
		if err = r.data.cache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   key,
			Value: entList,
			TTL:   r.data.conf.Redis.CacheExpiration.AsDuration(),
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}
		settingList, er := toSettingsList(entList)
		if er != nil {
			return nil, v1.ErrorInternalError("internal error: %s", er)
		}
		return settingList, nil
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFoundError("not found: %s", err)
	default: // error
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *settingRepo) BatchCreate(ctx context.Context, settings []*biz.Setting) ([]*biz.Setting, error) {
	if len(settings) > biz.MaxBatchCreateSize {
		return nil, v1.ErrorInvalidArgument("batch size cannot be greater than %d", biz.MaxBatchCreateSize)
	}
	bulk := make([]*ent.SettingCreate, len(settings))
	for i, s := range settings {
		var err error
		bulk[i], err = r.createBuilder(s)
		if err != nil {
			return nil, v1.ErrorInternalError("internal error: %s", err)
		}
	}
	res, err := r.data.db.Setting.CreateBulk(bulk...).Save(ctx)
	switch {
	case err == nil:
		settingList, er := toSettingsList(res)
		if er != nil {
			return nil, v1.ErrorInternalError("internal error: %s", er)
		}
		return settingList, nil
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorAlreadyExistsError("setting already exists: %s", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorInvalidArgument("invalid argument: %s", err)
	default:
		r.log.Errorf("unknown err: %v", err)
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *settingRepo) createBuilder(setting *biz.Setting) (*ent.SettingCreate, error) {
	m := r.data.db.Setting.Create()
	m.SetName(setting.Name)
	m.SetValue(setting.Value)
	m.SetType(toEntSettingType(setting.Type))
	return m, nil
}

func (r *settingRepo) cacheKeyPrefix(unique string, a ...string) string {
	s := strings.Join(a, "_")
	return settingCacheKey + s + ":" + unique
}

// deleteCache delete the cache both local cache and redis
func (r *settingRepo) deleteCache(ctx context.Context, key ...string) error {
	for _, k := range key {
		if err := r.data.cache.Delete(ctx, k); err != nil {
			return v1.ErrorInternalError("delete cache error: %v", err)
		}
	}
	return nil
}

func toSettingType(e setting.Type) biz.SettingType { return biz.SettingType(e) }

func toEntSettingType(s biz.SettingType) setting.Type { return setting.Type(s) }

func toSetting(e *ent.Setting) (*biz.Setting, error) {
	s := &biz.Setting{}
	s.Id = int64(e.ID)
	s.Name = e.Name
	s.Value = e.Value
	s.Type = toSettingType(e.Type)
	return s, nil
}

func toSettingsList(e []*ent.Setting) ([]*biz.Setting, error) {
	settingList := make([]*biz.Setting, len(e))
	for i, entEntity := range e {
		s, err := toSetting(entEntity)
		if err != nil {
			return nil, err
		}
		settingList[i] = s
	}
	return settingList, nil
}
