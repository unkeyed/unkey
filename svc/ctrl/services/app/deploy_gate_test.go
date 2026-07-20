package app

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeployEntitled(t *testing.T) {
	null := sql.NullString{Valid: false}
	plan := func(value string) sql.NullString {
		return sql.NullString{Valid: true, String: value}
	}

	require.False(t, deployEntitled(null, null))
	require.False(t, deployEntitled(plan(""), null))
	require.True(t, deployEntitled(plan("pro"), null))
	require.True(t, deployEntitled(null, plan("business")))
}
