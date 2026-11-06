package uid

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/validation"
)

func TestSlugIsAValidSlug(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{})
	for range 1000 {
		s := Slug()
		require.Regexp(t, `^[a-z]+-[a-z]+-[0-9]{4}$`, s)
		require.True(t, validation.ValidateSlug(s), s)
		seen[s] = struct{}{}
	}

	require.Greater(t, len(seen), 990)
}
