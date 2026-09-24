package pagination

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
)

type row struct {
	ID string
}

// TestFetchAuthorized_RefillsSparsePagesWithoutExposingDeniedRows guarantees
// separated allowed rows, such as 001 and 101, fill a page across denied rows.
func TestFetchAuthorized_RefillsSparsePagesWithoutExposingDeniedRows(t *testing.T) {
	all := make([]row, 250)
	for i := range all {
		all[i].ID = fmt.Sprintf("%03d", i)
	}
	allowedIDs := map[string]bool{"001": true, "101": true, "201": true, "220": true}
	var fetchLimits []int32
	fetch := func(_ context.Context, cursor string, limit int32) ([]row, error) {
		fetchLimits = append(fetchLimits, limit)
		start := 0
		if cursor != "" {
			_, err := fmt.Sscanf(cursor, "%03d", &start)
			require.NoError(t, err)
		}
		end := min(start+int(limit), len(all))
		return all[start:end], nil
	}

	rows, err := FetchAuthorized(context.Background(), Params{Limit: 3}, fetch,
		func(r row) bool { return allowedIDs[r.ID] }, func(r row) string { return r.ID })

	require.NoError(t, err)
	require.Equal(t, []row{{ID: "001"}, {ID: "101"}, {ID: "201"}, {ID: "220"}}, rows)
	require.Equal(t, []int32{5, 101, 201}, fetchLimits)
}

// TestFetchAuthorized_UsesOpaqueInclusiveCursors guarantees cursors need no
// lexical ordering: opaque:next resumes at that row without skipping it.
func TestFetchAuthorized_UsesOpaqueInclusiveCursors(t *testing.T) {
	pages := map[string][]row{
		"descending:start": {{ID: "z"}, {ID: "y"}, {ID: "x"}, {ID: "opaque:next"}},
		"opaque:next":      {{ID: "opaque:next"}, {ID: "a"}},
	}
	rows, err := FetchAuthorized(context.Background(), Params{Limit: 2, Cursor: "descending:start"},
		func(_ context.Context, cursor string, _ int32) ([]row, error) { return pages[cursor], nil },
		func(r row) bool { return r.ID == "y" || r.ID == "opaque:next" || r.ID == "a" },
		func(r row) string { return r.ID },
	)
	require.NoError(t, err)
	require.Equal(t, []row{{ID: "y"}, {ID: "opaque:next"}, {ID: "a"}}, rows)
}

// TestFetchAuthorized_ReturnsEmptyWhenAllRowsAreDenied guarantees a denied row,
// such as secret, never becomes a result or an authorized lookahead cursor.
func TestFetchAuthorized_ReturnsEmptyWhenAllRowsAreDenied(t *testing.T) {
	rows, err := FetchAuthorized(context.Background(), Params{Limit: 2},
		func(context.Context, string, int32) ([]row, error) { return []row{{ID: "secret"}}, nil },
		func(row) bool { return false }, func(r row) string { return r.ID })
	require.NoError(t, err)
	require.Empty(t, rows)
}

// TestFetchAuthorized_PropagatesFetchErrorsWithoutPartialRows guarantees a
// database failure returns the original error and no successful page.
func TestFetchAuthorized_PropagatesFetchErrorsWithoutPartialRows(t *testing.T) {
	fetchErr := errors.New("fetch failed")
	rows, err := FetchAuthorized(context.Background(), Params{Limit: 2},
		func(context.Context, string, int32) ([]row, error) { return nil, fetchErr },
		func(row) bool { return true }, func(r row) string { return r.ID })
	require.ErrorIs(t, err, fetchErr)
	require.Nil(t, rows)
}

// TestFetchAuthorized_ScanBudget guarantees scans stop after 10,000 rows.
// A finished 10,000-row scan succeeds; an unfinished 10,001-row scan fails
// without returning even the allowed row already found.
func TestFetchAuthorized_ScanBudget(t *testing.T) {
	for _, count := range []int{9999, 10000, 10001} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			all := make([]row, count)
			for i := range all {
				all[i].ID = fmt.Sprint(i)
			}
			examined := 0
			rows, err := FetchAuthorized(t.Context(), Params{Limit: 1},
				func(_ context.Context, cursor string, limit int32) ([]row, error) {
					start := 0
					if cursor != "" {
						_, scanErr := fmt.Sscan(cursor, &start)
						require.NoError(t, scanErr)
					}
					return all[start:min(start+int(limit), len(all))], nil
				}, func(r row) bool {
					examined++
					return r.ID == "0"
				}, func(r row) string { return r.ID })
			require.Equal(t, min(count, 10000), examined)
			if count > 10000 {
				require.ErrorIs(t, err, ErrScanLimit)
				require.Nil(t, rows)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []row{{ID: "0"}}, rows)
		})
	}
}

func TestParse(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		p := Parse(nil, nil, 100)
		require.Equal(t, Params{Limit: 100, Cursor: ""}, p)
	})

	t.Run("explicit values", func(t *testing.T) {
		p := Parse(ptr.P(20), ptr.P("KEBAP"), 100)
		require.Equal(t, Params{Limit: 20, Cursor: "KEBAP"}, p)
	})
}

func TestFetchLimit(t *testing.T) {
	require.Equal(t, int32(51), Params{Limit: 50}.FetchLimit())
}

func TestPaginate(t *testing.T) {
	id := func(r row) string { return r.ID }

	tests := []struct {
		name        string
		rows        []row
		limit       int
		wantIDs     []string
		wantCursor  *string
		wantHasMore bool
	}{
		{
			name:        "empty",
			rows:        []row{},
			limit:       10,
			wantIDs:     []string{},
			wantCursor:  nil,
			wantHasMore: false,
		},
		{
			name:        "fewer than limit",
			rows:        []row{{ID: "a"}, {ID: "KEBAP"}},
			limit:       10,
			wantIDs:     []string{"a", "KEBAP"},
			wantCursor:  nil,
			wantHasMore: false,
		},
		{
			name:        "exactly limit",
			rows:        []row{{ID: "a"}, {ID: "b"}, {ID: "c"}},
			limit:       3,
			wantIDs:     []string{"a", "b", "c"},
			wantCursor:  nil,
			wantHasMore: false,
		},
		{
			name:        "one over limit",
			rows:        []row{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "KEBAP"}},
			limit:       3,
			wantIDs:     []string{"a", "b", "c"},
			wantCursor:  ptr.P("KEBAP"),
			wantHasMore: true,
		},
		{
			name:        "several over limit",
			rows:        []row{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}, {ID: "e"}, {ID: "f"}},
			limit:       3,
			wantIDs:     []string{"a", "b", "c"},
			wantCursor:  ptr.P("d"),
			wantHasMore: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, pg := Paginate(tt.rows, Params{Limit: tt.limit}, id)

			gotIDs := make([]string, len(items))
			for i, r := range items {
				gotIDs[i] = r.ID
			}
			require.Equal(t, tt.wantIDs, gotIDs)
			require.Equal(t, tt.wantHasMore, pg.HasMore)
			require.Equal(t, tt.wantCursor, pg.Cursor)
		})
	}
}
