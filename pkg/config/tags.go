package config

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type directive struct {
	name  string
	value string
}

func parseTag(tag string) []directive {
	if tag == "" {
		return nil
	}

	var directives []directive
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		name, value, _ := strings.Cut(part, "=")
		directives = append(directives, directive{name: name, value: value})
	}

	return directives
}

func applyDefaults(v any) error {
	rv := reflect.ValueOf(v).Elem()
	return applyDefaultsRecursive(rv)
}

func applyDefaultsRecursive(rv reflect.Value) error {
	rt := rv.Type()

	for i := range rt.NumField() {
		field := rv.Field(i)
		structField := rt.Field(i)

		if !structField.IsExported() {
			continue
		}

		// Recurse into nested structs.
		if field.Kind() == reflect.Struct {
			if err := applyDefaultsRecursive(field); err != nil {
				return err
			}
			continue
		}

		// Dereference pointer to struct and recurse.
		if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			if !field.IsNil() {
				if err := applyDefaultsRecursive(field.Elem()); err != nil {
					return err
				}
			}
			continue
		}

		tag := structField.Tag.Get("config")
		if tag == "" {
			continue
		}

		directives := parseTag(tag)
		for _, d := range directives {
			if d.name != "default" || d.value == "" {
				continue
			}

			if !field.CanSet() {
				continue
			}

			if field.Kind() == reflect.Slice {
				continue
			}

			if !field.IsZero() {
				continue
			}

			if err := setFieldFromString(field, d.value); err != nil {
				return fmt.Errorf("field %q: invalid default %q: %w", structField.Name, d.value, err)
			}
		}
	}

	return nil
}

func setFieldFromString(field reflect.Value, raw string) error {
	// Handle time.Duration specially before switching on kind.
	if field.Type() == reflect.TypeOf(time.Duration(0)) {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(d))
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		field.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		field.SetBool(b)
	default:
		return fmt.Errorf("unsupported kind %s", field.Kind())
	}

	return nil
}

func validate(v any) error {
	rv := reflect.ValueOf(v).Elem()
	errs := validateRecursive(rv, "")
	return errors.Join(errs...)
}

func validateRecursive(rv reflect.Value, prefix string) []error {
	rt := rv.Type()
	var errs []error

	for i := range rt.NumField() {
		field := rv.Field(i)
		structField := rt.Field(i)

		if !structField.IsExported() {
			continue
		}

		fieldPath := structField.Name
		if prefix != "" {
			fieldPath = prefix + "." + structField.Name
		}

		// Recurse into nested structs.
		if field.Kind() == reflect.Struct {
			// Skip time.Duration and other non-config structs — only recurse
			// if at least one field has a config tag.
			if hasConfigTags(field.Type()) {
				errs = append(errs, validateRecursive(field, fieldPath)...)
				continue
			}
		}

		// Dereference pointer to struct and recurse.
		if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			tag := structField.Tag.Get("config")
			directives := parseTag(tag)
			if hasDirective(directives, "required") && field.IsNil() {
				errs = append(errs, fmt.Errorf("field %q: required but not set", fieldPath))
				continue
			}
			if !field.IsNil() && hasConfigTags(field.Type().Elem()) {
				errs = append(errs, validateRecursive(field.Elem(), fieldPath)...)
			}
			continue
		}

		tag := structField.Tag.Get("config")
		if tag == "" {
			continue
		}

		directives := parseTag(tag)
		for _, d := range directives {
			if err := validateDirective(field, fieldPath, d); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errs
}

func validateDirective(field reflect.Value, fieldPath string, d directive) error {
	switch d.name {
	case "required":
		if isZero(field) {
			return fmt.Errorf("field %q: required but not set", fieldPath)
		}

	case "nonempty":
		switch field.Kind() {
		case reflect.String:
			if field.Len() == 0 {
				return fmt.Errorf("field %q: must not be empty", fieldPath)
			}
		case reflect.Slice, reflect.Map:
			if field.IsNil() || field.Len() == 0 {
				return fmt.Errorf("field %q: must not be empty", fieldPath)
			}
		}

	case "min":
		bound, err := strconv.ParseFloat(d.value, 64)
		if err != nil {
			return fmt.Errorf("field %q: invalid min value %q: %w", fieldPath, d.value, err)
		}
		switch field.Kind() {
		case reflect.String, reflect.Slice, reflect.Map:
			if field.Len() < int(bound) {
				return fmt.Errorf("field %q: length %d is less than minimum %v", fieldPath, field.Len(), formatBound(bound))
			}
		default:
			val, ok := numericValue(field)
			if ok && val < bound {
				return fmt.Errorf("field %q: value %v is less than minimum %v", fieldPath, formatNumeric(field), formatBound(bound))
			}
		}

	case "max":
		bound, err := strconv.ParseFloat(d.value, 64)
		if err != nil {
			return fmt.Errorf("field %q: invalid max value %q: %w", fieldPath, d.value, err)
		}
		switch field.Kind() {
		case reflect.String, reflect.Slice, reflect.Map:
			if field.Len() > int(bound) {
				return fmt.Errorf("field %q: length %d exceeds maximum %v", fieldPath, field.Len(), formatBound(bound))
			}
		default:
			val, ok := numericValue(field)
			if ok && val > bound {
				return fmt.Errorf("field %q: value %v exceeds maximum %v", fieldPath, formatNumeric(field), formatBound(bound))
			}
		}

	case "oneof":
		if field.Kind() == reflect.String {
			options := strings.Split(d.value, "|")
			val := field.String()
			found := false
			for _, opt := range options {
				if val == opt {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("field %q: value %q must be one of [%s]", fieldPath, val, strings.Join(options, ", "))
			}
		}
	}

	return nil
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.IsNil()
	default:
		return v.IsZero()
	}
}

func numericValue(v reflect.Value) (float64, bool) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return v.Float(), true
	default:
		return 0, false
	}
}

func formatNumeric(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

func formatBound(f float64) string {
	if f == math.Trunc(f) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func hasDirective(directives []directive, name string) bool {
	for _, d := range directives {
		if d.name == name {
			return true
		}
	}
	return false
}

// validateCustom calls [Validator].Validate on v and recursively on all
// nested structs that implement [Validator].
func validateCustom(v any) error {
	rv := reflect.ValueOf(v)
	errs := validateCustomRecursive(rv)
	return errors.Join(errs...)
}

func validateCustomRecursive(rv reflect.Value) []error {
	// Dereference pointers.
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil
	}

	var errs []error

	// Check if this struct itself implements Validator.
	if rv.CanAddr() {
		if v, ok := rv.Addr().Interface().(Validator); ok {
			if err := v.Validate(); err != nil {
				errs = append(errs, err)
			}
		}
	}

	// Recurse into exported struct fields.
	rt := rv.Type()
	for i := range rt.NumField() {
		field := rv.Field(i)
		if !rt.Field(i).IsExported() {
			continue
		}

		switch field.Kind() {
		case reflect.Struct:
			errs = append(errs, validateCustomRecursive(field)...)
		case reflect.Ptr:
			if !field.IsNil() && field.Type().Elem().Kind() == reflect.Struct {
				errs = append(errs, validateCustomRecursive(field)...)
			}
		}
	}

	return errs
}

func hasConfigTags(t reflect.Type) bool {
	for i := range t.NumField() {
		if t.Field(i).Tag.Get("config") != "" {
			return true
		}
		ft := t.Field(i).Type
		if ft.Kind() == reflect.Struct && hasConfigTags(ft) {
			return true
		}
	}
	return false
}
