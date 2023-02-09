package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

type Setting struct {
	Id    int64        `json:"id,omitempty"`
	Name  *string      `json:"name,omitempty"`
	Value *string      `json:"value,omitempty"`
	Type  *SettingType `json:"type,omitempty"`
}

type SettingName string

const (
	RegisterEnable       SettingName = "register_enable"
	RegisterDefaultGroup SettingName = "register_default_group"
	RegisterMailActive   SettingName = "register_mail_active"
	// RegisterMailFilter Indicates the status of the mail filter. "off" indicates that the mail filter
	// is disabled, "blacklist" indicates the blacklist, and "whitelist" indicates the whitelist
	RegisterMailFilter     SettingName = "register_mail_filter"
	RegisterMailFilterList SettingName = "register_mail_filter_list"
)

type SettingType string

const (
	TypeBasic    SettingType = "basic"
	TypeRegister SettingType = "register"
	TypeLogin    SettingType = "login"
	TypeMail     SettingType = "mail"
	TypeCaptcha  SettingType = "captcha"
	TypePwa      SettingType = "pwa"
	TypeTimeout  SettingType = "timeout"
	TypeUpload   SettingType = "upload"
	TypeShare    SettingType = "share"
	TypeAvatar   SettingType = "avatar"
	TypePayment  SettingType = "payment"
	TypeScore    SettingType = "score"
	TypeTask     SettingType = "task"
	TypeAuth     SettingType = "auth"
	TypeCron     SettingType = "cron"
)

func (s SettingType) String() string {
	return string(s)
}

type SettingRepo interface {
	Create(ctx context.Context, s *Setting) (*Setting, error)
	Get(ctx context.Context, id int64) (*Setting, error)
	GetByName(ctx context.Context, name string) (*Setting, error)
	Update(ctx context.Context, s *Setting) (*Setting, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) (map[SettingName]*Setting, error)
	ListByType(ctx context.Context, t SettingType) (map[SettingName]*Setting, error)
	BatchCreate(ctx context.Context, settings []*Setting) ([]*Setting, error)
	BatchUpsert(ctx context.Context, settings []*Setting) error
}

type SettingUsecase struct {
	repo SettingRepo
	log  *log.Helper
}

func NewSettingUsecase(repo SettingRepo, logger log.Logger) *SettingUsecase {
	return &SettingUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}
