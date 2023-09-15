package metrics

import (
	"reflect"
)

func toMap(s any) map[string]any {
	obj := map[string]any{}
	if s == nil {
		return obj
	}
	v := reflect.TypeOf(s)
	reflectValue := reflect.ValueOf(s)
	reflectValue = reflect.Indirect(reflectValue)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for i := 0; i < v.NumField(); i++ {
		tag := v.Field(i).Tag.Get("json")
		field := reflectValue.Field(i).Interface()
		if tag != "" && tag != "-" {
			if v.Field(i).Type.Kind() == reflect.Struct {
				obj[tag] = toMap(field)
			} else {
				obj[tag] = field
			}
		}
	}
	return obj
}
