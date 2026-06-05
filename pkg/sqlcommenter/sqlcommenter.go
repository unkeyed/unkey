// Package sqlcommenter builds W3C sqlcommenter-formatted comments to append to
// outbound SQL queries. Planetscale's Query Insights parses these comments and
// attributes captured slow queries back to the originating service, route, and
// request, which is the whole point of using this package.
//
// Spec: https://google.github.io/sqlcommenter/spec/
package sqlcommenter

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

type ctxKey struct{}

type requestMeta struct {
	route     string
	requestID string
}

// WithRequest stashes the route pattern and request id on ctx so that
// FromContext can pick them up when a DB query runs later in the request.
// Call this once in HTTP middleware.
func WithRequest(ctx context.Context, route, requestID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, requestMeta{route: route, requestID: requestID})
}

// FromContext builds a sqlcommenter comment for ctx. app is the static service
// name (e.g. "api"). Returns "" only if there are no tags at all, which in
// practice should not happen because db_driver is always present.
func FromContext(ctx context.Context, app string) string {
	tags := map[string]string{
		"db_driver": "go-database-sql",
	}
	if app != "" {
		tags["application"] = app
	}
	if meta, ok := ctx.Value(ctxKey{}).(requestMeta); ok {
		if meta.route != "" {
			tags["route"] = meta.route
		}
		if meta.requestID != "" {
			tags["request_id"] = meta.requestID
		}
	}
	if tp := traceparent(ctx); tp != "" {
		tags["traceparent"] = tp
	}
	return Format(tags)
}

// Format returns a sqlcommenter comment containing the given tags. Keys are
// sorted alphabetically (deterministic output, easier to test and grep) and
// values are percent-encoded per the spec. Returns "" if tags is empty.
func Format(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString("/*")
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(encode(k))
		sb.WriteString("='")
		sb.WriteString(encode(tags[k]))
		sb.WriteByte('\'')
	}
	sb.WriteString("*/")
	return sb.String()
}

// Append returns query with comment appended. If query has a trailing semicolon
// the comment is inserted just before it so the statement remains valid. Empty
// comment is a no-op.
func Append(query, comment string) string {
	if comment == "" {
		return query
	}
	trimmed := strings.TrimRight(query, " \t\r\n")
	if strings.HasSuffix(trimmed, ";") {
		return strings.TrimSuffix(trimmed, ";") + " " + comment + ";"
	}
	return query + " " + comment
}

// encode percent-encodes a value per the sqlcommenter spec. url.QueryEscape
// uses '+' for spaces but the spec wants '%20', so we swap that back.
func encode(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

// traceparent returns a W3C traceparent header value built from the active span
// on ctx, or "" if there is no span.
func traceparent(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	flags := "00"
	if sc.IsSampled() {
		flags = "01"
	}
	return fmt.Sprintf("00-%s-%s-%s", sc.TraceID().String(), sc.SpanID().String(), flags)
}
