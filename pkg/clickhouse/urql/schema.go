// Package urql is the Unkey Query Language: a thin DSL layer over ClickHouse SQL
// that exposes logical tables, virtual columns, automatic time-bucket resolution,
// and column-format hints. Output is plain ClickHouse SQL that flows through the
// existing pkg/clickhouse/query-parser for security (workspace isolation, RBAC,
// time-range bounds, function allowlist) — URQL never bypasses that layer.
package urql

// Granularity describes a physical table variant for a logical table.
type Granularity int

const (
	GranularityRaw Granularity = iota
	GranularityPerMinute
	GranularityPerHour
	GranularityPerDay
	GranularityPerMonth
)

// Column declares a column on a LogicalTable.
type Column struct {
	// Name is the user-facing column name. Defaults to the map key if zero.
	Name string

	// Expression, when non-empty, replaces references to this column with the
	// given ClickHouse expression. Used for virtual columns like
	// `is_successful` = `outcome = 'VALID'`.
	Expression string

	// OnlyVariants restricts the column to specific physical variants. If empty,
	// the column is available on all variants of the table.
	OnlyVariants []Granularity

	// AllowedValues, when non-empty, restricts the literal values that may
	// appear in WHERE/HAVING comparisons against this column. Compile-time
	// rejection of bad values produces friendlier errors than ClickHouse's
	// runtime type complaints.
	AllowedValues []string
}

// LogicalTable is a user-facing table name that resolves to one of several
// physical ClickHouse tables based on query shape (presence of timeBucket(),
// time-range bounds in WHERE).
type LogicalTable struct {
	// Name is the user-facing table name (e.g. "key_verifications").
	Name string

	// TimeColumn is the column used for time-range detection and timeBucket
	// resolution. Must exist in Columns and be present on every variant.
	TimeColumn string

	// Variants maps each available granularity to its physical ClickHouse
	// table name (e.g. "default.key_verifications_per_hour_v3").
	Variants map[Granularity]string

	// Columns declares every column accessible via this logical table,
	// across all variants. Per-variant restrictions are expressed via
	// Column.OnlyVariants.
	Columns map[string]Column
}

// availableOn reports whether a column is accessible on a given granularity.
func (c Column) availableOn(g Granularity) bool {
	if len(c.OnlyVariants) == 0 {
		return true
	}
	for _, v := range c.OnlyVariants {
		if v == g {
			return true
		}
	}
	return false
}

// newColumn returns a fully-initialized zero-value Column. Construction
// helpers below build on this so schema definitions stay readable while
// satisfying exhaustruct.
func newColumn() Column {
	return Column{
		Name:          "",
		Expression:    "",
		OnlyVariants:  nil,
		AllowedValues: nil,
	}
}

func (c Column) withExpression(expr string) Column {
	c.Expression = expr
	return c
}

func (c Column) onlyOn(variants ...Granularity) Column {
	c.OnlyVariants = variants
	return c
}

func (c Column) withAllowedValues(values ...string) Column {
	c.AllowedValues = values
	return c
}

// Schema is the registry of logical tables URQL knows about.
type Schema struct {
	tables map[string]*LogicalTable
}

// NewSchema builds a Schema from a list of LogicalTables.
func NewSchema(tables ...*LogicalTable) *Schema {
	s := &Schema{tables: make(map[string]*LogicalTable, len(tables))}
	for _, t := range tables {
		s.tables[t.Name] = t
	}
	return s
}

// Lookup returns the logical table with the given name, or nil if unknown.
func (s *Schema) Lookup(name string) *LogicalTable {
	return s.tables[name]
}

// keyVerifications is the only logical table in Phase 1.
//
// Variant column shape (verified against pkg/clickhouse/schema/001..005):
//   - raw (key_verifications_raw_v2): request_id, time(Int64 milli), workspace_id,
//     key_space_id, identity_id, external_id, key_id, region, outcome, tags,
//     spent_credits, latency
//   - aggregated (per_minute/hour/day/month_v3): time(DateTime|Date), workspace_id,
//     key_space_id, identity_id, external_id, key_id, outcome, tags, count,
//     spent_credits, latency_avg|p75|p99
//
// Aggregated-only AggregateFunction columns (latency_avg etc.) are deliberately
// omitted from Phase 1 — reading them requires *Merge functions that aren't
// in the legacy parser's allowlist. Users wanting latency stats query the raw
// variant with avg(latency)/quantile(latency).
var keyVerifications = &LogicalTable{
	Name:       "key_verifications",
	TimeColumn: "time",
	Variants: map[Granularity]string{
		GranularityRaw:       "default.key_verifications_raw_v2",
		GranularityPerMinute: "default.key_verifications_per_minute_v3",
		GranularityPerHour:   "default.key_verifications_per_hour_v3",
		GranularityPerDay:    "default.key_verifications_per_day_v3",
		GranularityPerMonth:  "default.key_verifications_per_month_v3",
	},
	Columns: map[string]Column{
		"time":          newColumn(),
		"workspace_id":  newColumn(),
		"key_space_id":  newColumn(),
		"identity_id":   newColumn(),
		"external_id":   newColumn(),
		"key_id":        newColumn(),
		"outcome":       newColumn().withAllowedValues("VALID", "RATE_LIMITED", "EXPIRED", "DISABLED", "FORBIDDEN", "USAGE_EXCEEDED", "INSUFFICIENT_PERMISSIONS"),
		"tags":          newColumn(),
		"spent_credits": newColumn(),

		"request_id": newColumn().onlyOn(GranularityRaw),
		"region":     newColumn().onlyOn(GranularityRaw),
		"latency":    newColumn().onlyOn(GranularityRaw),

		"count": newColumn().onlyOn(GranularityPerMinute, GranularityPerHour, GranularityPerDay, GranularityPerMonth),

		"is_successful": newColumn().withExpression("outcome = 'VALID'"),
	},
}

// DefaultSchema is the URQL schema used by the analytics endpoint.
var DefaultSchema = NewSchema(keyVerifications)
