package sqlcomment

import (
	"net/url"
	"strings"
	"sync"
)

// annotateCache memoizes annotated queries keyed by query text, static tags,
// mode, and dynamic tags. sqlc emits a fixed query string per operation, so
// repeated calls with the same tags reuse the result.
var annotateCache sync.Map

type annotateCacheKey struct {
	query   string
	mode    string
	static  Static
	dynamic Dynamic
}

// Annotate rewrites query for PlanetScale Insights:
//  1. Strips the sqlc `-- name:` header when present and emits operation=<name>.
//  2. Appends a SQLCommenter block with static, dynamic, and connection mode tags.
//
// When static.Enabled() is false, query is returned unchanged.
func Annotate(query string, static Static, mode string, dynamic Dynamic) string {
	if !static.Enabled() {
		return query
	}

	key := annotateCacheKey{query: query, mode: mode, static: static, dynamic: dynamic}
	if cached, ok := annotateCache.Load(key); ok {
		return cached.(string)
	}

	annotated := buildAnnotatedQuery(query, static, mode, dynamic)
	annotateCache.Store(key, annotated)
	return annotated
}

func buildAnnotatedQuery(query string, static Static, mode string, dynamic Dynamic) string {
	body, operation := stripSQLCHeader(query)
	comment := formatComment(static, mode, dynamic, operation)
	if comment == "" {
		return strings.TrimSpace(body)
	}
	return strings.TrimSpace(body) + " " + comment
}

func stripSQLCHeader(query string) (body, operation string) {
	if len(query) < len("-- name:") || !strings.HasPrefix(query, "-- name:") {
		return query, ""
	}

	rest := strings.TrimLeft(query[len("-- name:"):], " \t")
	if rest == "" {
		return query, ""
	}

	end := strings.IndexAny(rest, " \t\r\n")
	if end < 0 {
		return query, ""
	}
	operation = rest[:end]

	newline := strings.IndexByte(rest, '\n')
	if newline < 0 {
		return query, ""
	}
	return rest[newline+1:], operation
}

func formatComment(static Static, mode string, dynamic Dynamic, operation string) string {
	prefix := static.staticPrefix()
	capacity := len(prefix) + len(operation) + len(dynamic.Route) + len(dynamic.Source) + len(mode) + 48
	var b strings.Builder
	b.Grow(capacity)

	b.WriteString("/*")
	if prefix != "" {
		b.WriteString(prefix)
	}

	appendTag(&b, "operation", operation)
	appendTag(&b, "route", dynamic.Route)
	appendTag(&b, "source", dynamic.Source)
	appendTag(&b, "mode", mode)

	if b.Len() == len("/*") {
		return ""
	}
	b.WriteString("*/")
	return b.String()
}

func appendTag(b *strings.Builder, key, value string) {
	if value == "" {
		return
	}
	if b.Len() > len("/*") {
		b.WriteByte(',')
	}
	b.WriteString(key)
	b.WriteString("='")
	b.WriteString(escape(value))
	b.WriteByte('\'')
}

func escape(value string) string {
	return urlEncode(value)
}

// urlEncode percent-encodes tag values per the SQLCommenter spec. QueryEscape
// uses '+' for space; PlanetScale expects '%20'.
func urlEncode(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}
