package mysql

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLikeContains guarantees that LIKE wildcards in user input are escaped so
// a search can never widen its own match, and that an empty search yields an
// invalid NullString to disable the filter.
func TestLikeContains(t *testing.T) {
	testCases := []struct {
		name   string
		search string
		want   sql.NullString
	}{
		{
			name:   "empty search disables the filter",
			search: "",
			want:   sql.NullString{String: "", Valid: false},
		},
		{
			name:   "plain text is wrapped in wildcards",
			search: "user_123",
			want:   sql.NullString{String: `%user\_123%`, Valid: true},
		},
		{
			name:   "percent matches literally",
			search: "100%",
			want:   sql.NullString{String: `%100\%%`, Valid: true},
		},
		{
			name:   "underscore matches literally",
			search: "a_b",
			want:   sql.NullString{String: `%a\_b%`, Valid: true},
		},
		{
			name:   "backslash matches literally",
			search: `a\b`,
			want:   sql.NullString{String: `%a\\b%`, Valid: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, LikeContains(tc.search))
		})
	}
}
