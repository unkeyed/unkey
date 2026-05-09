package codes

import (
	"fmt"
	"reflect"
)

// codeRegistry maps every Code declared in this package's namespace
// structs (User, App, Auth, Data, Frontline, Portal) back to the
// Code value. ParseCode consults this so that a URN reconstructed
// from the wire carries its full metadata (notably Kind), keeping
// HTTPStatus() resolution identical on the producing and consuming
// sides of an RPC boundary.
//
// The registry is built once at init via reflection over the
// namespace structs so adding a new Code requires no manual
// registration step.
var codeRegistry = buildCodeRegistry()

func buildCodeRegistry() map[URN]Code {
	roots := []any{User, App, Auth, Data, Frontline, Portal}
	out := make(map[URN]Code)
	codeType := reflect.TypeOf((*Code)(nil)).Elem()
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		if v.Kind() != reflect.Struct {
			return
		}
		if v.Type() == codeType {
			c := v.Interface().(Code)
			if existing, ok := out[c.URN()]; ok {
				panic(fmt.Sprintf(
					"codes: duplicate URN %q registered twice (kinds: %q and %q); "+
						"two Code values cannot share the same system/category/specific tuple",
					c.URN(), existing.Kind, c.Kind,
				))
			}
			out[c.URN()] = c
			return
		}
		for i := 0; i < v.NumField(); i++ {
			walk(v.Field(i))
		}
	}
	for _, r := range roots {
		walk(reflect.ValueOf(r))
	}
	return out
}
