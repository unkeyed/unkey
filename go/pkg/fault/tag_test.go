package fault

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetTag(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected Tag
	}{
		{
			name:     "nil error returns UNTAGGED",
			err:      nil,
			expected: UNTAGGED,
		},
		{
			name:     "untagged error returns UNTAGGED",
			err:      errors.New("plain error"),
			expected: UNTAGGED,
		},
		{
			name:     "tagged error returns correct tag",
			err:      WithTag(Tag("CUSTOM_TAG"))(errors.New("tagged error")),
			expected: Tag("CUSTOM_TAG"),
		},
		{
			name: "deeply wrapped error returns first tag encountered when unwrapping",
			err: WithTag(Tag("OUTER_TAG"))(
				WithTag(Tag("INNER_TAG"))(errors.New("inner error")),
			),
			expected: Tag("OUTER_TAG"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := GetTag(tt.err)
			require.Equal(t, tt.expected, tag)
		})
	}
}

func TestWithTag(t *testing.T) {
	tests := []struct {
		name   string
		tag    Tag
		err    error
		verify func(*testing.T, error)
	}{
		{
			name: "nil error returns nil",
			tag:  Tag("TEST"),
			err:  nil,
			verify: func(t *testing.T, err error) {
				require.Nil(t, err)
			},
		},
		{
			name: "adds tag to error",
			tag:  Tag("TEST_TAG"),
			err:  errors.New("base error"),
			verify: func(t *testing.T, err error) {
				wrapped, ok := err.(*wrapped)
				require.True(t, ok)
				require.Equal(t, Tag("TEST_TAG"), wrapped.tag)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WithTag(tt.tag)(tt.err)
			tt.verify(t, result)
		})
	}
}
