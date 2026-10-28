// Package logfilter turns the filters of the public logs endpoints into a
// ClickHouse WHERE clause and its server-side parameters.
//
// A request filters each field with an operator set: the fixed list of
// operators that field accepts, such as [Text] or [Int]. The builder combines
// them with one rule: the positive operators of a field (is, contains,
// startsWith, endsWith) are ORed, every negated operator (isNot, notContains)
// and every range bound (gt, gte, lt, lte) is ANDed, and different fields are
// ANDed. A field with no operators adds nothing.
//
// Column expressions and operator templates are Go constants, parameter names
// come from a counter, and every value travels as a {name:Type} server-side
// parameter. The SQL text therefore depends only on which operators were used
// and how many values each one holds, never on the values themselves.
//
// Usage:
//
//	b := logfilter.New()
//	b.Text(logfilter.Column("path"), logfilter.Text{StartsWith: []string{"/v2/keys"}})
//	b.Exact(logfilter.Column("host"), logfilter.Exact{IsNot: []string{"api.unkey.dev"}})
//	where, params := b.Build()
package logfilter
