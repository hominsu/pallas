package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/hominsu/pallas/app/pallas/service/internal/data/settings"
)

type Setting struct {
	Id    int64                `json:"id,omitempty"`
	Name  string               `json:"name,omitempty"`
	Value string               `json:"value,omitempty"`
	Type  settings.SettingType `json:"type,omitempty"`
}

type SettingRepo interface {
	Create(ctx context.Context, s *Setting) (*Setting, error)
	Get(ctx context.Context, id int64) (*Setting, error)
	GetByName(ctx context.Context, name string) (*Setting, error)
	Update(ctx context.Context, s *Setting) (*Setting, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]*Setting, error)
	ListByType(ctx context.Context, t settings.SettingType) ([]*Setting, error)
	BatchCreate(ctx context.Context, settings []*Setting) ([]*Setting, error)
}

type SettingUsecase struct {
	repo SettingRepo
	ss   *settings.Settings
	log  *log.Helper
}

func NewSettingUsecase(repo SettingRepo, ss *settings.Settings, logger log.Logger) *SettingUsecase {
	return &SettingUsecase{
		repo: repo,
		ss:   ss,
		log:  log.NewHelper(logger),
	}
}
