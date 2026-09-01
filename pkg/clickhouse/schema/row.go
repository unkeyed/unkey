package schema

//go:generate go run ./gen

// Row is implemented by every struct that maps to an insertable ClickHouse
// table row. The implementations live in columns_generated.go: put an
// //unkey:table directive in the struct's doc comment and run
// `mise run generate` after changing the directive or any `ch` tag.
//
// Naming insert columns explicitly (instead of letting the driver expand to
// the table's full column list) keeps writers compatible with tables that
// have gained columns the binary does not know about yet: omitted columns
// fall back to their server-side DEFAULT instead of failing AppendStruct
// with "missing destination name". The reverse direction still needs
// ordering: a column must exist in every environment before a binary that
// lists it is deployed.
type Row interface {
	// Table returns the fully qualified table this row is inserted into,
	// e.g. "default.key_verifications_raw_v2".
	Table() string

	// InsertColumns returns the backtick-quoted, comma-separated column list
	// matching what AppendStruct can populate for this struct.
	InsertColumns() string
}
