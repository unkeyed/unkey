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
	"context"
	"errors"

	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

const scanLimit = 10_000

// ErrScanLimit is returned when authorization filtering cannot fill a page
// within the maximum number of examined rows.
var ErrScanLimit = errors.New("authorized pagination scan limit exceeded")

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

// FetchAuthorized fills a page plus its authorized lookahead while keeping
// denied row cursors out of the response. The fetch callback must return at
// most its requested limit in stable order, starting at the inclusive cursor.
// Params.Limit must be positive and cursor must return a nonempty row ID.
// For example, denied rows between two allowed projects do not shorten the
// page. Pass the returned rows to Paginate to create the response cursor.
// Fetch errors and scans requiring more than 10,000 examined rows return no rows.
func FetchAuthorized[T any](ctx context.Context, p Params, fetch func(context.Context, string, int32) ([]T, error), allowed func(T) bool, cursor func(T) string) ([]T, error) {
	wanted := p.Limit + 1
	rows := make([]T, 0, wanted)
	batchSize := wanted
	nextCursor := p.Cursor
	for scanned := 0; scanned < scanLimit; {
		batchSize = min(batchSize, scanLimit-scanned)
		batch, err := fetch(ctx, nextCursor, int32(batchSize+1)) // nolint:gosec // bounded by scanLimit
		if err != nil {
			return nil, err
		}

		nextCursor = ""
		if len(batch) > batchSize {
			nextCursor = cursor(batch[batchSize])
			batch = batch[:batchSize]
		}
		for _, item := range batch {
			scanned++
			if allowed(item) {
				rows = append(rows, item)
				if len(rows) == wanted {
					return rows, nil
				}
			}
		}
		if nextCursor == "" {
			return rows, nil
		}
		batchSize = max(batchSize*2, 100)
	}
	return nil, ErrScanLimit
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
