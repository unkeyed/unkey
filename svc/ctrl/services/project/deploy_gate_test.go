package project

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeployEntitled(t *testing.T) {
	null := sql.NullString{Valid: false}
	empty := sql.NullString{Valid: true, String: ""}
	plan := func(s string) sql.NullString { return sql.NullString{Valid: true, String: s} }

	cases := []struct {
		name     string
		plan     sql.NullString
		override sql.NullString
		want     bool
	}{
		{name: "no plan, no override", plan: null, override: null, want: false},
		{name: "empty plan, no override", plan: empty, override: null, want: false},
		{name: "synced plan grants", plan: plan("pro"), override: null, want: true},
		{name: "override grants without plan", plan: null, override: plan("business"), want: true},
		{name: "empty override does not grant", plan: null, override: empty, want: false},
		{name: "both set grants", plan: plan("starter"), override: plan("business"), want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, deployEntitled(tc.plan, tc.override))
		})
	}
}
