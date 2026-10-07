package cdc

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestNew_RejectsInvalidRulesWithAssertionErrors(t *testing.T) {
	for _, test := range []struct {
		name  string
		rules []Rule
	}{
		{name: "no rules"},
		{name: "table pattern", rules: []Rule{{Table: "/.*", Query: "select id from records"}}},
		{name: "empty query", rules: []Rule{{Table: "records"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			watcher, err := New(Config{
				Address: "localhost:33575", Keyspace: "unkey", Insecure: true, Rules: test.rules,
			})
			if watcher != nil {
				t.Cleanup(func() { require.NoError(t, watcher.Close()) })
			}
			require.Error(t, err)
			require.Nil(t, watcher)
			_, tagged := fault.GetCode(err)
			require.True(t, tagged, "invalid rules must return a tagged assertion error")
		})
	}
}
