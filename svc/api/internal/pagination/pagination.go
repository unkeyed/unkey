// Package pagination implements the cursor pagination idiom shared by the v2
// list endpoints:
//
//	p := pagination.Parse(req.Limit, req.Cursor, 100)
//
//	rows, err := db.Query.ListX(ctx, h.DB.RO(), db.ListXParams{
//		IDCursor: p.Cursor,
//		Limit:    p.FetchLimit(),
//	})
//
//	rows, pg := pagination.Paginate(rows, p, func(r db.ListXRow) string { return r.ID })
//
// Queries over-fetch one row beyond the requested page size so the extra row
// can reveal whether a next page exists and serve as its cursor. Pairs with
// the inclusive `id >= cursor` / `ORDER BY id ASC` convention of the v2 list
// queries: the returned cursor is the first row of the next page.
package pagination

import (
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Params carries the parsed pagination inputs of a list request so the same
// limit drives both the over-fetch (FetchLimit) and the trim (Paginate).
type Params struct {
	Limit  int
	Cursor string
}

// Parse applies defaults to the optional pagination fields of a list request.
// Range bounds are already enforced by the OpenAPI request validation.
func Parse(limit *int, cursor *string, defaultLimit int) Params {
	return Params{
		Limit:  ptr.SafeDeref(limit, defaultLimit),
		Cursor: ptr.SafeDeref(cursor, ""),
	}
}

// FetchLimit returns the query limit including the extra look-ahead row.
func (p Params) FetchLimit() int32 {
	return int32(p.Limit + 1) // nolint:gosec // request validation bounds Limit far below int32 max
}

// Paginate trims rows over-fetched with FetchLimit back to the requested page
// size and builds the response pagination. cursor extracts the value the next
// page's query resumes from.
func Paginate[T any](rows []T, p Params, cursor func(T) string) ([]T, openapi.Pagination) {
	hasMore := len(rows) > p.Limit
	var next *string
	if hasMore {
		next = ptr.P(cursor(rows[p.Limit]))
		rows = rows[:p.Limit]
	}

	return rows, openapi.Pagination{
		Cursor:  next,
		HasMore: hasMore,
	}
}
