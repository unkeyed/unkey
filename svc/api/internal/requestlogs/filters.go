// Package requestlogs holds the steps that every endpoint reading the Unkey
// API request logs shares, such as turning the OpenAPI filters into the
// filters of the logs store.
package requestlogs

import (
	"github.com/unkeyed/unkey/pkg/clickhouse/logfilter"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Time returns the filter of a time field. A nil filter matches every row.
func Time(f *openapi.TimeFilter) logfilter.Int {
	v := ptr.SafeDeref(f)

	return logfilter.Int{
		Is:    nil,
		IsNot: nil,
		Gt:    v.Gt,
		Gte:   v.Gte,
		Lt:    v.Lt,
		Lte:   v.Lte,
	}
}

// Int returns the filter of an integer field. A nil filter matches every row.
func Int(f *openapi.IntFilter) logfilter.Int {
	v := ptr.SafeDeref(f)

	return logfilter.Int{
		Is:    ptr.SafeDeref(v.Is),
		IsNot: ptr.SafeDeref(v.IsNot),
		Gt:    v.Gt,
		Gte:   v.Gte,
		Lt:    v.Lt,
		Lte:   v.Lte,
	}
}

// Text returns the filter of a text field. A nil filter matches every row.
func Text(f *openapi.TextFilter) logfilter.Text {
	v := ptr.SafeDeref(f)

	return logfilter.Text{
		Is:          ptr.SafeDeref(v.Is),
		IsNot:       ptr.SafeDeref(v.IsNot),
		Contains:    ptr.SafeDeref(v.Contains),
		NotContains: ptr.SafeDeref(v.NotContains),
		StartsWith:  ptr.SafeDeref(v.StartsWith),
		EndsWith:    ptr.SafeDeref(v.EndsWith),
	}
}

// Exact returns the filter of a whole-value field. A nil filter matches
// every row.
func Exact(f *openapi.ExactFilter) logfilter.Exact {
	v := ptr.SafeDeref(f)

	return logfilter.Exact{
		Is:    ptr.SafeDeref(v.Is),
		IsNot: ptr.SafeDeref(v.IsNot),
	}
}

// Method returns the filter of the HTTP method field. A nil filter matches
// every row.
func Method(f *openapi.MethodFilter) logfilter.Exact {
	v := ptr.SafeDeref(f)

	return logfilter.Exact{
		Is:    methodStrings(ptr.SafeDeref(v.Is)),
		IsNot: methodStrings(ptr.SafeDeref(v.IsNot)),
	}
}

func methodStrings(methods []openapi.HttpMethod) []string {
	values := make([]string, len(methods))
	for i, m := range methods {
		values[i] = string(m)
	}

	return values
}
