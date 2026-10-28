package logfilter

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	ch "github.com/ClickHouse/clickhouse-go/v2"
)

// Column is the SQL expression of a filterable column, such as "path". It is
// written into the query text, so it must be a Go constant.
type Column string

// Text is the operator set for free-form string fields such as a path.
// Matching is case-sensitive, and % and _ match only themselves.
type Text struct {
	Is          []string
	IsNot       []string
	Contains    []string
	NotContains []string
	StartsWith  []string
	EndsWith    []string
}

// Exact is the operator set for fields matched only as a whole, such as a
// host, a request id, or a closed enum like an HTTP method.
type Exact struct {
	Is    []string
	IsNot []string
}

// Int is the operator set for integer fields such as a response status or a
// time. {Is: [200], Gte: 400} matches nothing. Nil leaves a bound unset.
// Values bind as Int64 whatever the column type, because a narrower parameter
// type wraps an out-of-range value instead of rejecting it, and 4294967700
// bound as Int32 would match status 404.
type Int struct {
	Is    []int64
	IsNot []int64
	Gt    *int64
	Gte   *int64
	Lt    *int64
	Lte   *int64
}

const (
	opIs          = "%s = %s"
	opIsNot       = "%s != %s"
	opContains    = "position(%s, %s) > 0"
	opNotContains = "position(%s, %s) = 0"
	opStartsWith  = "startsWith(%s, %s)"
	opEndsWith    = "endsWith(%s, %s)"
	opGt          = "%s > %s"
	opGte         = "%s >= %s"
	opLt          = "%s < %s"
	opLte         = "%s <= %s"
)

// Builder collects the filters of one request. Create it with [New], add one
// call per field, then read the result with [Builder.Build].
type Builder struct {
	clauses    []string
	params     ch.Parameters
	paramCount int
}

// New returns an empty [Builder].
func New() *Builder {
	return &Builder{
		clauses:    nil,
		params:     ch.Parameters{},
		paramCount: 0,
	}
}

// Text adds the filter of a [Text] field.
func (b *Builder) Text(col Column, f Text) {
	b.addField(
		slices.Concat(
			b.bindStrings(opIs, col, f.Is),
			b.bindStrings(opContains, col, f.Contains),
			b.bindStrings(opStartsWith, col, f.StartsWith),
			b.bindStrings(opEndsWith, col, f.EndsWith),
		),
		slices.Concat(
			b.bindStrings(opIsNot, col, f.IsNot),
			b.bindStrings(opNotContains, col, f.NotContains),
		),
	)
}

// Exact adds the filter of an [Exact] field.
func (b *Builder) Exact(col Column, f Exact) {
	b.addField(
		b.bindStrings(opIs, col, f.Is),
		b.bindStrings(opIsNot, col, f.IsNot),
	)
}

// Int adds the filter of an [Int] field.
func (b *Builder) Int(col Column, f Int) {
	b.addField(
		b.bindInts(opIs, col, f.Is),
		slices.Concat(
			b.bindInts(opIsNot, col, f.IsNot),
			b.bindInts(opGt, col, optional(f.Gt)),
			b.bindInts(opGte, col, optional(f.Gte)),
			b.bindInts(opLt, col, optional(f.Lt)),
			b.bindInts(opLte, col, optional(f.Lte)),
		),
	)
}

// Build returns the WHERE body of every field added so far and the parameters
// it reads. With no filters the body is "true", so a caller can always write
// "WHERE " + where. A caller may add its own parameters to the returned map
// under any name that does not start with "filter_".
func (b *Builder) Build() (string, ch.Parameters) {
	return cmp.Or(strings.Join(b.clauses, " AND "), "true"), b.params
}

func (b *Builder) addField(anyOf, allOf []string) {
	if len(anyOf) > 0 {
		b.clauses = append(b.clauses, "("+strings.Join(anyOf, " OR ")+")")
	}
	b.clauses = append(b.clauses, allOf...)
}

func (b *Builder) bindStrings(op string, col Column, values []string) []string {
	var predicates []string
	for _, v := range values {
		predicates = append(predicates, fmt.Sprintf(op, col, b.bind("String", stringEscaper.Replace(v))))
	}

	return predicates
}

func (b *Builder) bindInts(op string, col Column, values []int64) []string {
	var predicates []string
	for _, v := range values {
		predicates = append(predicates, fmt.Sprintf(op, col, b.bind("Int64", strconv.FormatInt(v, 10))))
	}

	return predicates
}

func (b *Builder) bind(typ, value string) string {
	name := "filter_" + strconv.Itoa(b.paramCount)
	b.paramCount++
	b.params[name] = value

	return "{" + name + ":" + typ + "}"
}

func optional(v *int64) []int64 {
	if v == nil {
		return nil
	}

	return []int64{*v}
}

// stringEscaper prepares a value for a String parameter. The server reads
// String parameters in its escaped format, where a backslash starts an escape
// sequence and a raw tab or line break is invalid, while the driver escapes
// only for the quoting layer around it.
var stringEscaper = strings.NewReplacer(`\`, `\\`, "\t", `\t`, "\n", `\n`, "\r", `\r`)
