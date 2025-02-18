package assert

import (
	"strings"

	"github.com/unkeyed/unkey/go/pkg/fault"
)

func Equal[T comparable](a T, b T) error {
	if a != b {
		return fault.New("expected equal", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

func Nil(t any) error {
	if t != nil {
		return fault.New("expected nil", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

func NotNil(t any) error {
	if t == nil {
		return fault.New("expected not nil", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// True asserts that a boolean is true
func True(value bool, messages ...string) error {
	if !value {
		if len(messages) == 0 {
			messages = []string{"expected true but got false"}
		}
		return fault.New(messages[0], fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// False asserts that a boolean is false
func False(value bool) error {
	if value {
		return fault.New("expected false but got true", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// Empty asserts that a string/slice/map is empty
func Empty[T ~string | ~[]any | ~map[any]any](value T) error {
	if len(value) != 0 {
		return fault.New("value is not empty", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// NotEmpty asserts that a string/slice/map is not empty
func NotEmpty[T ~string | ~[]any | ~map[any]any](value T) error {
	if len(value) == 0 {
		return fault.New("value is empty", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// Contains asserts that a string contains a substring
func Contains(s, substr string) error {
	if !strings.Contains(s, substr) {
		return fault.New("string does not contain substring", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// Greater asserts that a is greater than b
func Greater[T ~int | ~float64](a, b T) error {
	if a <= b {
		return fault.New("value is not greater", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// Less asserts that a is less than b
func Less[T ~int | ~float64](a, b T) error {
	if a >= b {
		return fault.New("value is not less", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}

// InRange asserts that a value is within a range (inclusive)
func InRange[T ~int | ~float64](value, minimum, maximum T) error {
	if value < minimum || value > maximum {
		return fault.New("value is out of range", fault.WithTag(fault.ASSERTION_FAILED))
	}
	return nil
}
