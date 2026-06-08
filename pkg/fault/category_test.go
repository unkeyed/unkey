package fault

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
)

func TestGetCategory(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected codes.Category
		found    bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
			found:    false,
		},
		{
			name:     "plain error has no category",
			err:      errors.New("plain"),
			expected: "",
			found:    false,
		},
		{
			name:     "derives category from the code's URN",
			err:      Wrap(errors.New("boom"), Code(codes.Frontline.Proxy.ServiceUnavailable.URN())),
			expected: codes.CategoryUpstream,
			found:    true,
		},
		{
			name:     "capacity URN resolves to capacity",
			err:      Wrap(errors.New("boom"), Code(codes.Frontline.Routing.NoRunningInstances.URN())),
			expected: codes.CategoryCapacity,
			found:    true,
		},
		{
			name: "explicit Category override wins over the code's category",
			err: Wrap(errors.New("boom"),
				Code(codes.User.BadRequest.RequestTimeout.URN()), // category bad_request
				Category(codes.CategoryUpstream),
			),
			expected: codes.CategoryUpstream,
			found:    true,
		},
		{
			name: "override on an outer wrap beats an inner code",
			err: Wrap(
				Wrap(errors.New("boom"), Code(codes.Frontline.Internal.InternalServerError.URN())),
				Category(codes.CategoryClient),
			),
			expected: codes.CategoryClient,
			found:    true,
		},
		{
			name:     "walks the chain past non-fault wrappers",
			err:      fmt.Errorf("ctx: %w", Wrap(errors.New("boom"), Code(codes.Frontline.Auth.InvalidKey.URN()))),
			expected: codes.CategoryClient,
			found:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cat, ok := GetCategory(tt.err)
			require.Equal(t, tt.found, ok)
			require.Equal(t, tt.expected, cat)
		})
	}
}

func TestCategoryFirstWins(t *testing.T) {
	t.Parallel()

	// Two Category wrappers in one Wrap call: the first non-empty wins, matching
	// Code's merge semantics.
	err := Wrap(errors.New("boom"),
		Category(codes.CategoryUpstream),
		Category(codes.CategoryClient),
	)
	cat, ok := GetCategory(err)
	require.True(t, ok)
	require.Equal(t, codes.CategoryUpstream, cat)
}
