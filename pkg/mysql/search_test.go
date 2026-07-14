package mysql

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSearchPatterns guarantees that LIKE wildcards in user input are escaped
// so a search can never widen its own match, that each helper anchors the
// pattern on the right side, and that an empty search yields an invalid
// NullString to disable the filter.
func TestSearchPatterns(t *testing.T) {
	testCases := []struct {
		name   string
		fn     func(string) sql.NullString
		search string
		want   sql.NullString
	}{
		{
			name:   "contains: empty search disables the filter",
			fn:     SearchContains,
			search: "",
			want:   sql.NullString{String: "", Valid: false},
		},
		{
			name:   "contains: plain text is wrapped in wildcards",
			fn:     SearchContains,
			search: "user_123",
			want:   sql.NullString{String: `%user\_123%`, Valid: true},
		},
		{
			name:   "contains: percent matches literally",
			fn:     SearchContains,
			search: "100%",
			want:   sql.NullString{String: `%100\%%`, Valid: true},
		},
		{
			name:   "contains: underscore matches literally",
			fn:     SearchContains,
			search: "a_b",
			want:   sql.NullString{String: `%a\_b%`, Valid: true},
		},
		{
			name:   "contains: backslash matches literally",
			fn:     SearchContains,
			search: `a\b`,
			want:   sql.NullString{String: `%a\\b%`, Valid: true},
		},
		{
			name:   "prefix: empty search disables the filter",
			fn:     SearchPrefix,
			search: "",
			want:   sql.NullString{String: "", Valid: false},
		},
		{
			name:   "prefix: anchors the start",
			fn:     SearchPrefix,
			search: "user_123",
			want:   sql.NullString{String: `user\_123%`, Valid: true},
		},
		{
			name:   "suffix: empty search disables the filter",
			fn:     SearchSuffix,
			search: "",
			want:   sql.NullString{String: "", Valid: false},
		},
		{
			name:   "suffix: anchors the end",
			fn:     SearchSuffix,
			search: "user_123",
			want:   sql.NullString{String: `%user\_123`, Valid: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.fn(tc.search))
		})
	}
}
