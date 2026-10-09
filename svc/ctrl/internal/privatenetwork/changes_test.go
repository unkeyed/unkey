package privatenetwork

import (
	"database/sql"
	"os"
	"regexp"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	binlog "vitess.io/vitess/go/vt/proto/binlogdata"
	"vitess.io/vitess/go/vt/proto/query"
)

// TestDecodeRows guarantees that CDC rows decode into the selected columns,
// keep NULL distinct from empty strings, and fail loudly on rows that do not
// match the watched selections.
func TestDecodeRows(t *testing.T) {
	event := func(table string, row *query.Row) *binlog.VEvent {
		return &binlog.VEvent{Type: binlog.VEventType_ROW, RowEvent: &binlog.RowEvent{
			TableName: table, RowChanges: []*binlog.RowChange{{Before: nil, After: row}},
		}}
	}

	rows, err := decodeRows(event("unkey.clusters", &query.Row{Lengths: []int64{1, 1, -1}, Values: []byte("cr")}))
	require.NoError(t, err)
	require.Equal(t, []rowChange{{table: "clusters", before: nil, after: image{
		"id": {String: "c", Valid: true}, "region_id": {String: "r", Valid: true}, "cell_id": {String: "", Valid: false},
	}}}, rows)

	rows, err = decodeRows(event("clusters", &query.Row{Lengths: []int64{1, 1, 0}, Values: []byte("cr")}))
	require.NoError(t, err)
	require.Equal(t, sql.NullString{String: "", Valid: true}, rows[0].after["cell_id"])

	for name, test := range map[string]*binlog.VEvent{
		"wrong column count": event("clusters", &query.Row{Lengths: []int64{1}, Values: []byte("c")}),
		"truncated row":      event("clusters", &query.Row{Lengths: []int64{1, 1, 5}, Values: []byte("cr")}),
		"unknown table":      event("keys", &query.Row{Lengths: []int64{1}, Values: []byte("k")}),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := decodeRows(test)
			require.Error(t, err)
		})
	}
}

// TestSelectionsCoverQueryColumns guarantees that skipping cluster and region
// rewrites with equal images is safe: the CDC selections contain every
// clusters and regions column the topology query reads, and the caller
// queries read only the regions columns that trigger a rebuild.
func TestSelectionsCoverQueryColumns(t *testing.T) {
	selected := map[string][]string{}
	for _, table := range watchedTables {
		selected[table.name] = table.columns
	}
	columns := func(file, alias string) []string {
		t.Helper()
		query, err := os.ReadFile("../db/queries/" + file)
		require.NoError(t, err)
		var found []string
		for _, match := range regexp.MustCompile(`\b`+alias+`\.([a-z_]+)\b`).FindAllStringSubmatch(string(query), -1) {
			found = append(found, match[1])
		}
		require.NotEmpty(t, found, "%s reads no %s columns", file, alias)
		return found
	}
	for _, column := range columns("private_network_list_clusters.sql", "c") {
		require.Contains(t, selected["clusters"], column, "the topology reads clusters.%s", column)
	}
	for _, column := range columns("private_network_list_clusters.sql", "r") {
		require.Contains(t, selected["regions"], column, "the topology reads regions.%s", column)
	}
	for file, alias := range map[string]string{
		"private_network_list_replicas.sql":    "r",
		"private_network_list_connections.sql": "target_region",
	} {
		for _, column := range columns(file, alias) {
			require.True(t, slices.Contains(regionCallerColumns, column), "%s reads regions.%s", file, column)
		}
	}
	for _, column := range regionCallerColumns {
		require.Contains(t, selected["regions"], column)
	}
}
