package privatenetwork

import (
	"database/sql"
	"fmt"
	"maps"
	"strings"

	"github.com/unkeyed/unkey/pkg/cdc"
	binlog "vitess.io/vitess/go/vt/proto/binlogdata"
	"vitess.io/vitess/go/vt/proto/query"
)

// Filtered rows that stop matching still arrive with a before image.
var watchedTables = []struct {
	name    string
	columns []string
	filter  string
}{
	{name: "deployments", columns: []string{"id", "app_id"}, filter: "desired_state = 'running'"},
	{name: "deployment_topology", columns: []string{"deployment_id"}, filter: "desired_status = 'running'"},
	{name: "apps", columns: []string{"id"}, filter: ""},
	{name: "environments", columns: []string{"id", "app_id"}, filter: ""},
	{name: "workspaces", columns: []string{"id", "k8s_namespace"}, filter: ""},
	{name: "regions", columns: []string{"id", "platform", "name"}, filter: ""},
	{name: "clusters", columns: []string{"id", "region_id", "cell_id"}, filter: ""},
}

func cdcRules() []cdc.Rule {
	rules := make([]cdc.Rule, 0, len(watchedTables))
	for _, table := range watchedTables {
		query := "select " + strings.Join(table.columns, ", ") + " from " + table.name
		if table.filter != "" {
			query += " where " + table.filter
		}
		rules = append(rules, cdc.Rule{Table: table.name, Query: query})
	}
	return rules
}

var regionCallerColumns = []string{"id", "platform"}

func sameColumns(a, b image, columns []string) bool {
	for _, column := range columns {
		if a[column] != b[column] {
			return false
		}
	}
	return true
}

type set map[string]struct{}

func (s set) add(value string) {
	if value != "" {
		s[value] = struct{}{}
	}
}

type image map[string]sql.NullString

type rowChange struct {
	table  string
	before image
	after  image
}

func decodeRows(event *binlog.VEvent) ([]rowChange, error) {
	rowEvent := event.GetRowEvent()
	table := rowEvent.GetTableName()
	if i := strings.LastIndexByte(table, '.'); i >= 0 {
		table = table[i+1:]
	}
	rows := make([]rowChange, 0, len(rowEvent.GetRowChanges()))
	for _, change := range rowEvent.GetRowChanges() {
		before, err := rowValues(table, change.GetBefore())
		if err != nil {
			return nil, err
		}
		after, err := rowValues(table, change.GetAfter())
		if err != nil {
			return nil, err
		}
		rows = append(rows, rowChange{table: table, before: before, after: after})
	}
	return rows, nil
}

func rowValues(table string, row *query.Row) (image, error) {
	if row == nil {
		return nil, nil
	}
	var columns []string
	for _, watched := range watchedTables {
		if watched.name == table {
			columns = watched.columns
		}
	}
	if columns == nil {
		return nil, fmt.Errorf("unexpected private network CDC table %q", table)
	}
	if len(row.Lengths) != len(columns) {
		return nil, fmt.Errorf("private network CDC row for %s has %d columns, want %d", table, len(row.Lengths), len(columns))
	}
	values := make(image, len(columns))
	offset := int64(0)
	for i, length := range row.Lengths {
		if length < 0 {
			values[columns[i]] = sql.NullString{String: "", Valid: false}
			continue
		}
		if offset+length > int64(len(row.Values)) {
			return nil, fmt.Errorf("private network CDC row for %s is truncated", table)
		}
		values[columns[i]] = sql.NullString{String: string(row.Values[offset : offset+length]), Valid: true}
		offset += length
	}
	return values, nil
}

type changes struct {
	callers         set
	topologyCallers set
	targetApps      set
	callerApps      set
	workspaces      set
	topology        bool
	rebuild         bool
}

func newChanges() changes {
	return changes{
		callers:         set{},
		topologyCallers: set{},
		targetApps:      set{},
		callerApps:      set{},
		workspaces:      set{},
		topology:        false,
		rebuild:         false,
	}
}

func (c *changes) empty() bool {
	return len(c.callers) == 0 && len(c.topologyCallers) == 0 && len(c.targetApps) == 0 &&
		len(c.callerApps) == 0 && len(c.workspaces) == 0 && !c.topology && !c.rebuild
}

// Equal images still invalidate, because columns outside the selection can change.
func (c *changes) add(row rowChange) {
	switch row.table {
	case "clusters", "regions":
		if row.before != nil && row.after != nil && maps.Equal(row.before, row.after) {
			return
		}
		c.topology = true
		if row.table == "regions" && row.before != nil && (row.after == nil || !sameColumns(row.before, row.after, regionCallerColumns)) {
			c.rebuild = true
		}
		return
	}
	for _, values := range []image{row.before, row.after} {
		if values == nil {
			continue
		}
		switch row.table {
		case "deployments":
			c.callers.add(values["id"].String)
			c.targetApps.add(values["app_id"].String)
		case "deployment_topology":
			c.callers.add(values["deployment_id"].String)
			c.topologyCallers.add(values["deployment_id"].String)
		case "apps":
			c.targetApps.add(values["id"].String)
			c.callerApps.add(values["id"].String)
		case "environments":
			c.targetApps.add(values["app_id"].String)
			c.callerApps.add(values["app_id"].String)
		}
	}
	if row.table == "workspaces" && row.before != nil {
		c.workspaces.add(row.before["id"].String)
		if row.after != nil && row.before["k8s_namespace"].String == "" && row.after["k8s_namespace"].String != "" {
			c.rebuild = true
		}
	}
}
