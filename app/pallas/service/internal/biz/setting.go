package biz

import "context"

type Setting struct {
	Id    int64       `json:"id,omitempty"`
	Name  string      `json:"name,omitempty"`
	Value string      `json:"value,omitempty"`
	Type  SettingType `json:"type,omitempty"`
}

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
	List(ctx context.Context) ([]*Setting, error)
	ListByType(ctx context.Context, t SettingType) ([]*Setting, error)
	BatchCreate(ctx context.Context, settings []*Setting) ([]*Setting, error)
}
