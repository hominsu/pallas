package settings

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()

	typ := reflect.TypeOf(s).Elem()
	val := reflect.ValueOf(s).Elem()
	for i := 0; i < typ.NumField(); i++ {
		fmt.Printf("name: %s, value: %v, type: %s\n",
			typ.Field(i).Name,
			val.Field(i).Interface(),
			typ.Field(i).Tag.Get("type"),
		)
	}
}

func TestToSettings(t *testing.T) {
	entEntities := toEntSettings(DefaultSettings())
	s, err := ToSettings(entEntities)
	assert.NoError(t, err)

	typ := reflect.TypeOf(s).Elem()
	val := reflect.ValueOf(s).Elem()
	for i := 0; i < typ.NumField(); i++ {
		fmt.Printf("name: %s, value: %v, type: %s\n",
			typ.Field(i).Name,
			val.Field(i).Interface(),
			typ.Field(i).Tag.Get("type"),
		)
	}
}

func toEntSettings(s *Settings) []*ent.Setting {
	typ := reflect.TypeOf(s).Elem()
	val := reflect.ValueOf(s).Elem()

	entEntities := make([]*ent.Setting, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		entEntities[i] = &ent.Setting{
			Name:  typ.Field(i).Name,
			Value: fmt.Sprintf("%v", val.Field(i).Interface()),
			Type:  ToEntSettingType(SettingTypeValue[typ.Field(i).Tag.Get("type")]),
		}
	}

	return entEntities
}
