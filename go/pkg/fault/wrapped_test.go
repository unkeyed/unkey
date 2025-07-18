package fault

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/codes"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		wrappers []Wrapper
		expected string
	}{
		{
			name:     "basic message",
			message:  "test error",
			wrappers: []Wrapper{},
			expected: "test error",
		},
		{
			name:    "with single wrapper",
			message: "base error",
			wrappers: []Wrapper{
				Internal("internal message"),
			},
			expected: "internal message: base error",
		},
		{
			name:    "with multiple wrappers",
			message: "base error",
			wrappers: []Wrapper{
				Internal("internal 1"),
				Public("public 1"),
				Internal("internal 2"),
				Public("public 2"),
			},
			expected: "internal 2: internal 1: base error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.message, tt.wrappers...)

			// Check error message
			require.Equal(t, tt.expected, err.Error())

		})
	}
}

func TestWrappedError(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() error
		expected string
	}{
		{
			name: "single error",
			setup: func() error {
				return New("test error")
			},
			expected: "test error",
		},
		{
			name: "wrapped with empty internal",
			setup: func() error {
				return Wrap(
					New("base error"),
					Internal(""),
					Public("public"),
				)
			},
			expected: "base error",
		},
		{
			name: "wrapped with empty base",
			setup: func() error {
				return Wrap(
					New(""),
					Internal("internal"),
					Public("public"),
				)
			},
			expected: "internal",
		},
		{
			name: "deeply nested",
			setup: func() error {
				base := New("base")
				wrapped1 := Wrap(base, Internal("level 1"))
				wrapped2 := Wrap(wrapped1, Internal("level 2"))
				return wrapped2
			},
			expected: "level 2: level 1: base",
		},
		{
			name: "mixed wrapped and unwrapped",
			setup: func() error {
				return Wrap(
					errors.New("standard error"),
					Internal("wrapped layer"),
				)
			},
			expected: "wrapped layer: standard error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setup()
			require.Equal(t, tt.expected, err.Error())
		})
	}
}

func TestGetLocation(t *testing.T) {
	location := getLocation()

	parts := strings.Split(location, ":")
	require.Len(t, parts, 2)

	file := parts[0]
	line := parts[1]

	require.True(t, strings.HasSuffix(file, ".go"))

	_, err := strconv.ParseInt(line, 10, 64)
	require.NoError(t, err)
}

func TestErrorChainUnwrapping(t *testing.T) {
	baseErr := errors.New("base error")
	wrapped1 := Wrap(baseErr, Internal("level 1"))
	wrapped2 := Wrap(wrapped1, Internal("level 2"))

	// Test unwrapping at each level
	unwrapped := errors.Unwrap(wrapped2)
	require.NotNil(t, unwrapped)
	require.Equal(t, "level 1: base error", unwrapped.Error())

	unwrapped = errors.Unwrap(unwrapped)
	require.NotNil(t, unwrapped)
	require.Equal(t, "base error", unwrapped.Error())

	unwrapped = errors.Unwrap(unwrapped)
	require.Nil(t, unwrapped)
}
func TestUserFacingMessage(t *testing.T) {

	t.Run("basic error chain", func(t *testing.T) {
		err := New("internal error",
			Internal("retry failed"),
			Public("Please try again later."),
			Internal("db connection failed"),
			Public("Service unavailable."),
		)
		msg := UserFacingMessage(err)
		expected := "Service unavailable. Please try again later."
		require.Equal(t, expected, msg)
	})

	t.Run("mixed public and internal messages", func(t *testing.T) {
		err := New("base error")
		err = Wrap(err,
			Internal("internal detail 1"),
			Public("Public message 1"),
			Code(codes.Auth.Authentication.KeyNotFound.URN()),
		)
		err = Wrap(err, Internal("internal detail 2"), Public("Public message 2"))

		msg := UserFacingMessage(err)
		require.Equal(t, "Public message 2 Public message 1", msg)
	})

	t.Run("edge cases", func(t *testing.T) {
		tests := []struct {
			name     string
			err      error
			expected string
		}{
			{
				name:     "nil error",
				err:      nil,
				expected: "",
			},
			{
				name:     "non-wrapped error",
				err:      fmt.Errorf("plain error"),
				expected: "",
			},
			{
				name:     "wrapped error without public message",
				err:      New("internal only"),
				expected: "",
			},
			{
				name:     "empty public message",
				err:      New("internal", Internal("internal detail"), Public("")),
				expected: "",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result := UserFacingMessage(tc.err)
				require.Equal(t, tc.expected, result)
			})
		}
	})
}
