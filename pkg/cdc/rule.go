package cdc

import (
	"errors"
	"regexp"

	binlog "vitess.io/vitess/go/vt/proto/binlogdata"
)

// Rule names one table and the SELECT query used to filter its rows and columns.
// Do not build Query from unchecked user input.
type Rule struct {
	Table string `json:"table"`
	Query string `json:"query"`
}

// tablePattern prevents Vitess from treating a table name as a regex.
var tablePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// vstreamFilter checks table names and queries before starting a watch.
func vstreamFilter(rules []Rule) (*binlog.Filter, error) {
	if len(rules) == 0 {
		return nil, errors.New("CDC requires at least one table rule")
	}
	filter := &binlog.Filter{Rules: nil} //nolint:exhaustruct // No replication workflow metadata is needed.
	for _, rule := range rules {
		if !tablePattern.MatchString(rule.Table) || rule.Query == "" {
			return nil, errors.New("CDC requires literal table names and nonempty queries")
		}
		filter.Rules = append(filter.Rules, &binlog.Rule{Match: rule.Table, Filter: rule.Query}) //nolint:exhaustruct // No replication workflow metadata is needed.
	}
	return filter, nil
}
