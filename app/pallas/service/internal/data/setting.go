package data

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/sync/singleflight"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/setting"
)

var _ biz.SettingRepo = (*settingRepo)(nil)

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
	res, err, _ := r.sg.Do(fmt.Sprintf("get_setting_by_id_%d", id),
		func() (interface{}, error) {
			get, err := r.data.db.Setting.Get(ctx, int(id))
			switch {
			case err == nil:
				return toSetting(get)
			case ent.IsNotFound(err):
				return nil, v1.ErrorNotFoundError("not found: %s", err)
			default:
				return nil, v1.ErrorUnknownError("unknown error: %s", err)
			}
		})
	if err != nil {
		return nil, err
	}
	return res.(*biz.Setting), nil
}

func (r *settingRepo) GetByName(ctx context.Context, name string) (*biz.Setting, error) {
	res, err, _ := r.sg.Do(fmt.Sprintf("get_setting_by_name_%s", name),
		func() (interface{}, error) {
			get, err := r.data.db.Setting.Query().
				Where(setting.NameEQ(name)).
				Only(ctx)
			switch {
			case err == nil:
				return toSetting(get)
			case ent.IsNotFound(err):
				return nil, v1.ErrorNotFoundError("not found: %s", err)
			default:
				return nil, v1.ErrorUnknownError("unknown error: %s", err)
			}
		})
	if err != nil {
		return nil, err
	}
	return res.(*biz.Setting), nil
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
		u, er := toSetting(res)
		if er != nil {
			return nil, v1.ErrorInternalError("internal error: %s", er)
		}
		return u, nil
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
	err := r.data.db.Setting.DeleteOneID(int(id)).Exec(ctx)
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

func (r *settingRepo) List(ctx context.Context) ([]*biz.Setting, error) {
	listQuery := r.data.db.Setting.Query()
	entList, err := listQuery.All(ctx)
	switch {
	case err == nil:
		settingList, er := toSettingsList(entList)
		if er != nil {
			return nil, v1.ErrorInternalError("internal error: %s", er)
		}
		return settingList, nil
	default:
		r.log.Errorf("unknown err: %v", err)
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *settingRepo) ListByType(ctx context.Context, t biz.SettingType) ([]*biz.Setting, error) {
	listQuery := r.data.db.Setting.Query().
		Where(setting.TypeEQ(toEntSettingType(t)))
	entList, err := listQuery.All(ctx)
	switch {
	case err == nil:
		settingList, er := toSettingsList(entList)
		if er != nil {
			return nil, v1.ErrorInternalError("internal error: %s", er)
		}
		return settingList, nil
	default:
		r.log.Errorf("unknown err: %v", err)
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
	now := time.Now()
	m.SetCreatedAt(now)
	m.SetUpdatedAt(now)
	return m, nil
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
