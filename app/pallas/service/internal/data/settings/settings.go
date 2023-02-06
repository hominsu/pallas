package settings

import (
	"reflect"
	"strconv"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/setting"
)

type Settings struct {
	RegisterEnable         bool   `type:"register"`
	RegisterDefaultGroup   string `type:"register"`
	RegisterMailActive     bool   `type:"register"`
	RegisterMailFilter     bool   `type:"register"`
	RegisterMailFilterList string `type:"register"`
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

var SettingTypeValue = map[string]SettingType{
	"basic":    TypeBasic,
	"register": TypeRegister,
	"login":    TypeLogin,
	"mail":     TypeMail,
	"captcha":  TypeCaptcha,
	"pwa":      TypePwa,
	"timeout":  TypeTimeout,
	"upload":   TypeUpload,
	"share":    TypeShare,
	"avatar":   TypeAvatar,
	"payment":  TypePayment,
	"score":    TypeScore,
	"task":     TypeTask,
	"auth":     TypeAuth,
	"cron":     TypeCron,
}

func (s SettingType) String() string {
	return string(s)
}

func DefaultSettings() *Settings {
	s := &Settings{
		RegisterEnable:       true,
		RegisterDefaultGroup: "Anonymous",
		RegisterMailActive:   false,
		RegisterMailFilter:   false,
		RegisterMailFilterList: "126.com,163.com,gmail.com," +
			"outlook.com,qq.com,foxmail.com,yeah.net,sohu.com," +
			"sohu.cn,139.com,wo.cn,189.cn,hotmail.com,live.com,live.cn",
	}
	return s
}

func ToSettings(e []*ent.Setting) (*Settings, error) {
	s := &Settings{}
	typ := reflect.TypeOf(s).Elem()
	val := reflect.ValueOf(s).Elem()

	entEntities := make(map[string]*ent.Setting)
	for _, entEntity := range e {
		entEntities[entEntity.Name] = entEntity
	}

	// avoid other fields in the []*ent.Setting by traversing the reflection structure
	for i := 0; i < typ.NumField(); i++ {
		k := val.Field(i).Type().Kind()
		name := typ.Field(i).Name
		switch k {
		case reflect.String:
			val.Field(i).SetString(entEntities[name].Value)
		case reflect.Bool:
			b, err := strconv.ParseBool(entEntities[name].Value)
			if err != nil {
				return nil, v1.ErrorInternalError("boolean convert error")
			}
			val.FieldByName(entEntities[name].Name).SetBool(b)
		default:
			return nil, v1.ErrorInternalError("unknown type")
		}
	}

	return s, nil
}

func ToSettingType(e setting.Type) SettingType { return SettingType(e) }

func ToEntSettingType(s SettingType) setting.Type { return setting.Type(s) }
