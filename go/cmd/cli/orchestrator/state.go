package orchestrator

import (
	"fmt"
	"reflect"
)

// SetState sets a value in the shared state
func (o *Orchestrator) SetState(key string, value any) {
	o.state[key] = value
}

// State gets a value from the shared state
func (o *Orchestrator) State(key string) (any, bool) {
	value, exists := o.state[key]
	return value, exists
}

// MustState gets a value and panics if not found
func (o *Orchestrator) MustState(key string) any {
	value, exists := o.state[key]
	if !exists {
		panic(fmt.Sprintf("required state key '%s' not found", key))
	}
	return value
}

// StateAs gets a value with automatic type assertion
func StateAs[T any](o *Orchestrator, key string) (T, bool) {
	var zero T
	value, exists := o.state[key]
	if !exists {
		return zero, false
	}

	// Direct type assertion
	if typed, ok := value.(T); ok {
		return typed, true
	}

	// Try reflection-based conversion for compatible types
	return convertValue[T](value)
}

// MustStateAs gets a value with type assertion and panics if not found or wrong type
func MustStateAs[T any](o *Orchestrator, key string) T {
	value, ok := StateAs[T](o, key)
	if !ok {
		var zero T
		panic(fmt.Sprintf("required state key '%s' not found or cannot convert to %T", key, zero))
	}
	return value
}

// convertValue attempts to convert a value to the target type using reflection
func convertValue[T any](value any) (T, bool) {
	var zero T
	targetType := reflect.TypeOf(zero)
	sourceValue := reflect.ValueOf(value)

	// If source is nil, return zero value
	if !sourceValue.IsValid() {
		return zero, false
	}

	sourceType := sourceValue.Type()

	// Same type - should have been caught by direct assertion, but just in case
	if sourceType == targetType {
		return value.(T), true
	}

	// Check if source is convertible to target
	if sourceType.ConvertibleTo(targetType) {
		converted := sourceValue.Convert(targetType)
		return converted.Interface().(T), true
	}

	// Handle pointer/non-pointer conversions
	if targetType.Kind() == reflect.Ptr && sourceType == targetType.Elem() {
		// Converting value to pointer
		ptr := reflect.New(sourceType)
		ptr.Elem().Set(sourceValue)
		return ptr.Interface().(T), true
	}

	if sourceType.Kind() == reflect.Ptr && sourceType.Elem() == targetType {
		// Converting pointer to value
		if sourceValue.IsNil() {
			return zero, false
		}
		return sourceValue.Elem().Interface().(T), true
	}

	// String conversions
	if targetType.Kind() == reflect.String {
		return reflect.ValueOf(fmt.Sprintf("%v", value)).Interface().(T), true
	}

	return zero, false
}

// HasState checks if a key exists in state
func (o *Orchestrator) HasState(key string) bool {
	_, exists := o.state[key]
	return exists
}

// RemoveState removes a specific key from state
func (o *Orchestrator) RemoveState(key string) {
	delete(o.state, key)
}

// ClearState clears all state
func (o *Orchestrator) ClearState() {
	o.state = make(map[string]any)
}

// StateKeys returns all keys in the state
func (o *Orchestrator) StateKeys() []string {
	keys := make([]string, 0, len(o.state))
	for key := range o.state {
		keys = append(keys, key)
	}
	return keys
}

// StateCount returns the number of items in state
func (o *Orchestrator) StateCount() int {
	return len(o.state)
}

// StateSnapshot returns a copy of the current state
func (o *Orchestrator) StateSnapshot() map[string]any {
	snapshot := make(map[string]any, len(o.state))
	for k, v := range o.state {
		snapshot[k] = v
	}
	return snapshot
}

// SetStateIf sets a value only if the key doesn't exist
func (o *Orchestrator) SetStateIf(key string, value any) bool {
	if !o.HasState(key) {
		o.SetState(key, value)
		return true
	}
	return false
}

// UpdateState updates a value using a function if the key exists
func (o *Orchestrator) UpdateState(key string, fn func(any) any) bool {
	if value, exists := o.state[key]; exists {
		o.state[key] = fn(value)
		return true
	}
	return false
}

// UpdateStateAs updates a value with type safety
func UpdateStateAs[T any](o *Orchestrator, key string, fn func(T) T) bool {
	if value, ok := StateAs[T](o, key); ok {
		o.SetState(key, fn(value))
		return true
	}
	return false
}
