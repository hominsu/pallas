package data

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/cache/v9"
	"golang.org/x/sync/singleflight"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/setting"
)

var _ biz.SettingRepo = (*settingRepo)(nil)

const settingCacheKeyPrefix = "setting_cache_key_"

type settingRepo struct {
	data *Data
	ck   map[string][]string
	sg   *singleflight.Group
	log  *log.Helper
}

// NewSettingRepo .
func NewSettingRepo(data *Data, logger log.Logger) biz.SettingRepo {
	sr := &settingRepo{
		data: data,
		sg:   &singleflight.Group{},
		log:  log.NewHelper(log.With(logger, "module", "data/setting")),
	}
	sr.ck = make(map[string][]string)
	sr.ck["Get"] = []string{"get", "setting", "id"}
	sr.ck["GetByName"] = []string{"get", "setting", "name"}
	sr.ck["List"] = []string{"list", "group"}
	sr.ck["ListByType"] = []string{"list", "group", "type"}
	return sr
}

func (r *settingRepo) Create(ctx context.Context, s *biz.Setting) (*biz.Setting, error) {
	m := r.createBuilder(s)
	res, err := m.Save(ctx)
	switch {
	case err != nil:
		set, tErr := toSetting(res)
		if tErr != nil {
			return nil, v1.ErrorInternal("internal error: %v", err)
		}
		return set, nil
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorConflict("setting already exists: %v", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorConflict("invalid argument: %v", err)
	default:
		return nil, v1.ErrorConflict("unknown error: %v", err)
	}
}

func (r *settingRepo) Get(ctx context.Context, id int64) (*biz.Setting, error) {
	// key: setting_cache_key_get_setting_id:settingId
	key := r.cacheKey(strconv.FormatInt(id, 10), r.ck["Get"]...)
	res, err, _ := r.sg.Do(key, func() (any, error) {
		get := &ent.Setting{}
		// get cache
		err := r.data.cache.Get(ctx, key, get)
		if err != nil && errors.Is(err, cache.ErrCacheMiss) { // cache miss
			// get from db
			get, err = r.data.db.Setting.Get(ctx, id)
		}
		return get, err
	})
	switch {
	case err == nil: // db hit, set cache
		if err = r.data.cache.Set(&cache.Item{
			Ctx:            ctx,
			Key:            key,
			Value:          res.(*ent.Setting),
			TTL:            r.data.conf.Cache.Ttl.AsDuration(),
			SkipLocalCache: true,
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}
		return toSetting(res.(*ent.Setting))
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFound("setting not found: %v", err)
	default: // error
		return nil, v1.ErrorUnknown("unknown error: %v", err)
	}
}

func (r *settingRepo) GetByName(ctx context.Context, name string) (*biz.Setting, error) {
	// key: setting_cache_key_get_setting_name:settingName
	key := r.cacheKey(name, r.ck["GetByName"]...)
	res, err, _ := r.sg.Do(key, func() (any, error) {
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
			Ctx:            ctx,
			Key:            key,
			Value:          res.(*ent.User),
			TTL:            r.data.conf.Cache.Ttl.AsDuration(),
			SkipLocalCache: true,
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}
		return toSetting(res.(*ent.Setting))
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFound("setting not found: %v", err)
	default: // error
		return nil, v1.ErrorUnknown("unknown error: %v", err)
	}
}

func (r *settingRepo) Update(ctx context.Context, s *biz.Setting) (*biz.Setting, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, v1.ErrorInternal("create transactional client error: %v", err)
	}
	defer func() {
		if v := recover(); v != nil {
			if rErr := tx.Rollback(); rErr != nil {
				r.log.Warnf("rollback failed, err: %v", rErr)
			}
			panic(v)
		}
	}()

	m := tx.Setting.UpdateOneID(s.Id)
	if s.Name != nil {
		m.SetName(*s.Name)
	}
	if s.Value != nil {
		m.SetValue(*s.Value)
	}
	if s.Type != nil {
		m.SetType(toEntSettingType(*s.Type))
	}
	res, err := m.Save(ctx)

	switch {
	case err == nil:
		if cErr := tx.Commit(); cErr != nil {
			return nil, v1.ErrorInternal("failed commits the transaction, err: %v", cErr)
		}
		// delete indexed cache
		if err = r.deleteCache(
			ctx,
			// key: setting_cache_key_get_setting_id:settingId
			r.cacheKey(strconv.FormatInt(res.ID, 10), r.ck["Get"]...),
			// key: setting_cache_key_get_setting_name:settingName
			r.cacheKey(res.Name, r.ck["GetByName"]...),
			// key: setting_cache_key_list_group:all
			r.cacheKey("all", r.ck["List"]...),
			// key: setting_cache_key_list_group_type:settingType
			r.cacheKey(res.Type.String(), r.ck["ListByType"]...),
		); err != nil {
			// TODO: delete again using the asynchronous queue
			r.log.Error(err)
		}
		return toSetting(res)
	default:
		if rErr := tx.Rollback(); rErr != nil {
			return nil, v1.ErrorInternal("rollback failed, err: %v",
				fmt.Errorf("%w: rolling back transaction: %v", err, rErr))
		}
		switch {
		case ent.IsNotFound(err): // db miss
			return nil, v1.ErrorNotFound("setting not found: %v", err)
		default: // error
			return nil, v1.ErrorUnknown("unknown error: %v", err)
		}
	}
}

func (r *settingRepo) Delete(ctx context.Context, id int64) error {
	// get deleted setting from db
	res, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	err = r.data.db.Setting.DeleteOneID(id).Exec(ctx)
	switch {
	case err == nil:
		// delete indexed cache
		if err = r.deleteCache(
			ctx,
			// key: setting_cache_key_get_setting_id:settingId
			r.cacheKey(strconv.FormatInt(id, 10), r.ck["Get"]...),
			// key: setting_cache_key_get_setting_name:settingName
			r.cacheKey(*res.Name, r.ck["GetByName"]...),
			// key: setting_cache_key_list_group:all
			r.cacheKey("all", r.ck["List"]...),
			// key: setting_cache_key_list_group_type:settingType
			r.cacheKey(res.Type.String(), r.ck["ListByType"]...),
		); err != nil {
			// TODO: delete again using the asynchronous queue
			r.log.Error(err)
		}
		return nil
	case ent.IsNotFound(err):
		return v1.ErrorNotFound("setting not found: %v", err)
	default:
		return v1.ErrorUnknown("unknown error: %v", err)
	}
}

func (r *settingRepo) List(ctx context.Context) (map[biz.SettingName]*biz.Setting, error) {
	// key: setting_cache_key_list_group:all
	key := r.cacheKey("all", r.ck["List"]...)
	res, err, _ := r.sg.Do(key, func() (any, error) {
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
			Ctx:            ctx,
			Key:            key,
			Value:          entList,
			TTL:            r.data.conf.Cache.Ttl.AsDuration(),
			SkipLocalCache: true,
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}
		settingMap, tErr := toSettingsMap(entList)
		if tErr != nil {
			return nil, v1.ErrorInternal("internal error: %v", tErr)
		}
		return settingMap, nil
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFound("setting not found: %v", err)
	default: // error
		return nil, v1.ErrorUnknown("unknown error: %v", err)
	}
}

func (r *settingRepo) ListByType(ctx context.Context, t biz.SettingType) (map[biz.SettingName]*biz.Setting, error) {
	// key: setting_cache_key_list_group_type:settingType
	key := r.cacheKey(t.String(), r.ck["ListByType"]...)
	res, err, _ := r.sg.Do(key, func() (any, error) {
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
			Ctx:            ctx,
			Key:            key,
			Value:          entList,
			TTL:            r.data.conf.Cache.Ttl.AsDuration(),
			SkipLocalCache: true,
		}); err != nil {
			r.log.Errorf("cache error: %v", err)
		}
		settingMap, tErr := toSettingsMap(entList)
		if tErr != nil {
			return nil, v1.ErrorInternal("internal error: %v", tErr)
		}
		return settingMap, nil
	case ent.IsNotFound(err): // db miss
		return nil, v1.ErrorNotFound("setting not found: %v", err)
	default: // error
		return nil, v1.ErrorUnknown("unknown error: %v", err)
	}
}

func (r *settingRepo) BatchCreate(ctx context.Context, settings []*biz.Setting) ([]*biz.Setting, error) {
	if len(settings) > biz.MaxBatchCreateSize {
		return nil, v1.ErrorBatchSize("batch size cannot be greater than %d", biz.MaxBatchCreateSize)
	}
	bulk := make([]*ent.SettingCreate, len(settings))
	for i, s := range settings {
		bulk[i] = r.createBuilder(s)
	}
	res, err := r.data.db.Setting.CreateBulk(bulk...).Save(ctx)
	switch {
	case err == nil:
		settingList, tErr := toSettingsList(res)
		if tErr != nil {
			return nil, v1.ErrorInternal("internal error: %v", tErr)
		}
		return settingList, nil
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorConflict("setting already exists: %v", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorConflict("invalid argument: %v", err)
	default:
		return nil, v1.ErrorUnknown("unknown error: %v", err)
	}
}

func (r *settingRepo) BatchUpsert(ctx context.Context, settings []*biz.Setting) error {
	if len(settings) > biz.MaxBatchUpdateSize {
		return v1.ErrorBatchSize("batch size cannot be greater than %d", biz.MaxBatchUpdateSize)
	}

	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return v1.ErrorInternal("create transactional client error: %v", err)
	}
	defer func() {
		if v := recover(); v != nil {
			if rErr := tx.Rollback(); rErr != nil {
				r.log.Warnf("rollback failed, err: %v", rErr)
			}
			panic(v)
		}
	}()

	bulk := make([]*ent.SettingCreate, len(settings))
	for i, s := range settings {
		bulk[i] = r.createTxBuilder(tx, s)
	}
	err = r.data.db.Setting.CreateBulk(bulk...).OnConflict().UpdateValue().Exec(ctx)
	switch {
	case err == nil:
		if cErr := tx.Commit(); cErr != nil {
			return v1.ErrorInternal("failed commits the transaction, err: %v", cErr)
		}
		// delete cache by scan redis
		if err = r.deleteKeysByScanPrefix(ctx,
			// match key with prefix: setting_cache_key_
			settingCacheKeyPrefix,
		); err != nil {
			// TODO: delete again using the asynchronous queue
			r.log.Error(err)
		}
		return nil
	default:
		if rErr := tx.Rollback(); rErr != nil {
			return v1.ErrorInternal("rollback failed, err: %v",
				fmt.Errorf("%w: rolling back transaction: %v", err, rErr))
		}
		return v1.ErrorUnknown("unknown error: %v", err)
	}
}

func (r *settingRepo) createBuilder(s *biz.Setting) *ent.SettingCreate {
	m := r.data.db.Setting.Create()
	if s.Name != nil {
		m.SetName(*s.Name)
	}
	if s.Value != nil {
		m.SetValue(*s.Value)
	}
	if s.Type != nil {
		m.SetType(toEntSettingType(*s.Type))
	}
	return m
}

func (r *settingRepo) createTxBuilder(tx *ent.Tx, s *biz.Setting) *ent.SettingCreate {
	m := tx.Setting.Create()
	if s.Name != nil {
		m.SetName(*s.Name)
	}
	if s.Value != nil {
		m.SetValue(*s.Value)
	}
	if s.Type != nil {
		m.SetType(toEntSettingType(*s.Type))
	}
	return m
}

func (r *settingRepo) cacheKey(unique string, a ...string) string {
	s := strings.Join(a, "_")
	return settingCacheKeyPrefix + s + ":" + unique
}

// deleteCache delete the cache both local cache and redis
func (r *settingRepo) deleteCache(ctx context.Context, key ...string) error {
	for _, k := range key {
		if err := r.data.cache.Delete(ctx, k); err != nil {
			return v1.ErrorCacheOperation("delete cache error: %v", err)
		}
	}
	return nil
}

// deleteKeysByScanPrefix delete the keys by scan the prefix on redis,
// notice that this function will not delete the keys on local cache
func (r *settingRepo) deleteKeysByScanPrefix(ctx context.Context, prefix ...string) error {
	for _, p := range prefix {
		iter := r.data.rdCmd.Scan(ctx, 0, p+"*", 0).Iterator()
		for iter.Next(ctx) {
			if err := r.data.rdCmd.Del(ctx, iter.Val()).Err(); err != nil {
				return v1.ErrorCacheOperation("delete setting cache keys by scan prefix error: %v", err)
			}
		}
		if err := iter.Err(); err != nil {
			return v1.ErrorCacheOperation("delete setting cache keys by scan prefix error: %v", err)
		}
	}
	return nil
}

func toSettingsMap(e []*ent.Setting) (map[biz.SettingName]*biz.Setting, error) {
	settingMap := make(map[biz.SettingName]*biz.Setting, len(e))
	for _, entEntity := range e {
		s, err := toSetting(entEntity)
		if err != nil {
			return nil, err
		}
		settingMap[biz.SettingName(*s.Name)] = s
	}
	return settingMap, nil
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

func toSetting(e *ent.Setting) (*biz.Setting, error) {
	s := &biz.Setting{}
	s.Id = e.ID
	s.Name = &e.Name
	s.Value = &e.Value
	t := toSettingType(e.Type)
	s.Type = &t
	return s, nil
}

func toSettingType(e setting.Type) biz.SettingType { return biz.SettingType(e) }

func toEntSettingType(s biz.SettingType) setting.Type { return setting.Type(s) }
