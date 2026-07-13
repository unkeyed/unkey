package pagination

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
)

type row struct {
	ID string
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
