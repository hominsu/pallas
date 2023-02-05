package settings

import (
	"fmt"
	"reflect"
	"testing"
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
