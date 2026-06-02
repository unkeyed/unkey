package codes

import "reflect"

// CollectURNs walks an arbitrarily-nested struct of Code values and
// returns every URN it finds. The argument is typically a code-tree
// root like codes.Frontline or codes.Data; passing a single Code
// returns that Code's URN as a one-element slice.
//
// Intended for tests that need to enumerate every URN in a namespace
// — for example, asserting that every URN under codes.Frontline has
// an entry in some classification table. Exported so test packages
// across the repo share one implementation.
//
// Non-struct values and zero-valued struct fields are skipped.
// Recursion is depth-first; URN order matches struct field order.
func CollectURNs(root any) []URN {
	var out []URN
	// reflect.TypeOf((*Code)(nil)).Elem() gets the Code type without
	// constructing one — Code{} would trip exhaustruct, and a
	// constructed Code would force an unused empty value.
	codeType := reflect.TypeOf((*Code)(nil)).Elem()

	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		if v.Kind() != reflect.Struct {
			return
		}

		if v.Type() == codeType {
			out = append(out, v.Interface().(Code).URN())
			return
		}

		for i := 0; i < v.NumField(); i++ {
			walk(v.Field(i))
		}
	}

	walk(reflect.ValueOf(root))
	return out
}
