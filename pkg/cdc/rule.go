package cdc

import (
	"regexp"

	"github.com/unkeyed/unkey/pkg/assert"
	binlog "vitess.io/vitess/go/vt/proto/binlogdata"
)

// Rule names one table and the SELECT query used to filter its rows and columns.
// Do not build Query from unchecked user input.
//
// For example, this rule copies only IDs of enabled records:
//
//	Rule{Table: "records", Query: "select id from records where enabled = 1"}
type Rule struct {
	Table string `json:"table"`
	Query string `json:"query"`
}

// tablePattern restricts rules to single tables. Vitess also accepts patterns
// such as /.*, but our resume-token checks compare exact table names.
var tablePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// vstreamFilter checks table names and queries before starting a watch.
func vstreamFilter(rules []Rule) (*binlog.Filter, error) {
	if err := assert.True(len(rules) > 0, "CDC requires at least one table rule"); err != nil {
		return nil, err
	}
	filter := &binlog.Filter{Rules: nil} //nolint:exhaustruct // No replication workflow metadata is needed.
	for _, rule := range rules {
		if err := assert.All(
			assert.True(tablePattern.MatchString(rule.Table), "CDC requires literal table names"),
			assert.NotEmpty(rule.Query, "CDC requires nonempty queries"),
		); err != nil {
			return nil, err
		}
		filter.Rules = append(filter.Rules, &binlog.Rule{Match: rule.Table, Filter: rule.Query}) //nolint:exhaustruct // No replication workflow metadata is needed.
	}
	return filter, nil
}
