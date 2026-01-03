package fault

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
)

func TestGetCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected codes.URN
	}{
		{
			name:     "nil error returns codes.Nil",
			err:      nil,
			expected: "",
		},
		{
			name:     "untagged error returns codes.Nil",
			err:      errors.New("plain error"),
			expected: "",
		},
		{
			name:     "tagged error returns correct tag",
			err:      Code(codes.URN("CUSTOM_TAG"))(errors.New("tagged error")),
			expected: codes.URN("CUSTOM_TAG"),
		},
		{
			name: "deeply wrapped error returns first tag encountered when unwrapping",
			err: Code(codes.URN("OUTER_TAG"))(
				Code(codes.URN("INNER_TAG"))(errors.New("inner error")),
			),
			expected: codes.URN("OUTER_TAG"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _ := GetCode(tt.err)
			require.Equal(t, tt.expected, code)
		})
	}
}

func TestCode(t *testing.T) {
	tests := []struct {
		name   string
		tag    codes.URN
		err    error
		verify func(*testing.T, error)
	}{
		{
			name: "nil error returns nil",
			tag:  codes.URN("TEST"),
			err:  nil,
			verify: func(t *testing.T, err error) {
				require.Nil(t, err)
			},
		},
		{
			name: "adds tag to error",
			tag:  codes.URN("TEST_TAG"),
			err:  errors.New("base error"),
			verify: func(t *testing.T, err error) {
				wrapped, ok := err.(*wrapped)
				require.True(t, ok)
				require.Equal(t, codes.URN("TEST_TAG"), wrapped.code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Code(tt.tag)(tt.err)
			tt.verify(t, result)
		})
	}
}
